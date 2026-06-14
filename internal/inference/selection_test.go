package inference

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

// fakeConnStore implements ConnStore for selection tests.
type fakeConnStore struct {
	conns []*store.Connection
	locks map[string][]*store.ModelLock // connID → locks (active filtering by ExpiresAt)
}

func (f *fakeConnStore) ListConnections() ([]*store.Connection, error) {
	return f.conns, nil
}

func (f *fakeConnStore) ActiveLocks(connID string, now int64) ([]*store.ModelLock, error) {
	var out []*store.ModelLock
	for _, l := range f.locks[connID] {
		if l.ExpiresAt > now {
			out = append(out, l)
		}
	}
	return out, nil
}

// fakeSettingStore implements SettingStore for selection tests.
type fakeSettingStore struct {
	settings map[string]string
}

func (f *fakeSettingStore) GetSetting(key string) (string, error) {
	return f.settings[key], nil
}

// fakeCooldownForSelection implements Cooldown for selection tests.
type fakeCooldownForSelection struct {
	unavailCalls []struct {
		connID, providerID, model string
		verdict                   Verdict
	}
	successCalls []string
}

func (f *fakeCooldownForSelection) MarkUnavailable(connID, providerID, model string, verdict Verdict) error {
	f.unavailCalls = append(f.unavailCalls, struct {
		connID, providerID, model string
		verdict                   Verdict
	}{connID, providerID, model, verdict})
	return nil
}

func (f *fakeCooldownForSelection) MarkSuccess(connID string) error {
	f.successCalls = append(f.successCalls, connID)
	return nil
}

func (f *fakeCooldownForSelection) GroupRetryAfter(providerID, model string, now time.Time) (time.Time, bool, error) {
	return time.Time{}, false, nil
}

func makeConn(id, providerID string) *store.Connection {
	return &store.Connection{ID: id, ProviderID: providerID, Name: id, Kind: "api_key"}
}

func TestFillFirstDefault(t *testing.T) {
	cs := &fakeConnStore{
		conns: []*store.Connection{
			makeConn("c1", "p1"),
			makeConn("c2", "p1"),
			makeConn("c3", "p1"),
		},
	}
	engine := NewSelectionEngine(cs, &fakeSettingStore{}, &fakeCooldownForSelection{}, time.Now)

	conn, err := engine.SelectConnection("p1", "gpt-4", nil, "")
	if err != nil {
		t.Fatalf("SelectConnection: %v", err)
	}
	if conn.ID != "c1" {
		t.Errorf("fill-first: got %q, want c1", conn.ID)
	}

	// Excluding c1 → fill-first picks c2.
	conn, err = engine.SelectConnection("p1", "gpt-4", []string{"c1"}, "")
	if err != nil {
		t.Fatalf("SelectConnection after exclude: %v", err)
	}
	if conn.ID != "c2" {
		t.Errorf("fill-first after exclude: got %q, want c2", conn.ID)
	}
}

// TestWeightedSelectionRatio verifies PAR-ROUTE-027: with strategy "weighted"
// and two connections weighted 3:1 (via Connection.Metadata {"weight":N}), over
// four selects the heavier connection is picked 3× and the lighter 1×. The pick
// uses a deterministic smooth weighted round-robin accumulator (no math/rand),
// so the outcome is reproducible.
func TestWeightedSelectionRatio(t *testing.T) {
	cA := makeConn("cA", "p1")
	cA.Metadata = `{"weight":3}`
	cB := makeConn("cB", "p1")
	cB.Metadata = `{"weight":1}`
	cs := &fakeConnStore{conns: []*store.Connection{cA, cB}}
	ss := &fakeSettingStore{settings: map[string]string{
		"fallbackStrategy": "weighted",
	}}
	engine := NewSelectionEngine(cs, ss, &fakeCooldownForSelection{}, time.Now)

	counts := map[string]int{}
	for i := 0; i < 4; i++ {
		conn, err := engine.SelectConnection("p1", "gpt-4", nil, "")
		if err != nil {
			t.Fatalf("select %d: %v", i, err)
		}
		counts[conn.ID]++
	}
	if counts["cA"] != 3 || counts["cB"] != 1 {
		t.Errorf("weighted 3:1 over 4 selects = %v, want cA:3 cB:1", counts)
	}
}

// TestWeightedSelectionDefaultEqual verifies PAR-ROUTE-027: connections with no
// weight in metadata default to weight 1, so a 1:1 pair alternates evenly.
func TestWeightedSelectionDefaultEqual(t *testing.T) {
	cs := &fakeConnStore{conns: []*store.Connection{
		makeConn("c1", "p1"),
		makeConn("c2", "p1"),
	}}
	ss := &fakeSettingStore{settings: map[string]string{"fallbackStrategy": "weighted"}}
	engine := NewSelectionEngine(cs, ss, &fakeCooldownForSelection{}, time.Now)

	counts := map[string]int{}
	for i := 0; i < 4; i++ {
		conn, err := engine.SelectConnection("p1", "gpt-4", nil, "")
		if err != nil {
			t.Fatalf("select %d: %v", i, err)
		}
		counts[conn.ID]++
	}
	if counts["c1"] != 2 || counts["c2"] != 2 {
		t.Errorf("weighted default-equal over 4 = %v, want c1:2 c2:2", counts)
	}
}

// TestFreeConnInjectedWhenNoStoredConn verifies PAR-ROUTE-039: a free (noAuth)
// provider with zero stored connections yields a synthetic no-auth virtual
// connection rather than a "no eligible connections" error.
func TestFreeConnInjectedWhenNoStoredConn(t *testing.T) {
	cs := &fakeConnStore{conns: nil}
	engine := NewSelectionEngine(cs, &fakeSettingStore{}, &fakeCooldownForSelection{}, time.Now)

	conn, err := engine.SelectConnection("opencode", "some-model", nil, "")
	if err != nil {
		t.Fatalf("free provider selection: %v", err)
	}
	if conn == nil || conn.ProviderID != "opencode" {
		t.Fatalf("expected synthetic opencode conn, got %+v", conn)
	}
	if conn.Secret != "" {
		t.Errorf("synthetic free conn Secret = %q, want empty", conn.Secret)
	}
}

// TestNonFreeProviderNoStoredConnErrors verifies PAR-ROUTE-039 boundary: a
// non-free provider with zero connections keeps the existing "no eligible
// connections" error (no synthetic injection).
func TestNonFreeProviderNoStoredConnErrors(t *testing.T) {
	cs := &fakeConnStore{conns: nil}
	engine := NewSelectionEngine(cs, &fakeSettingStore{}, &fakeCooldownForSelection{}, time.Now)

	if _, err := engine.SelectConnection("deepseek", "deepseek-chat", nil, ""); err == nil {
		t.Fatal("expected 'no eligible connections' for non-free provider, got nil")
	}
}

// TestRealConnPreferredOverSyntheticFree verifies a free provider that DOES have
// a real eligible connection uses it (synthetic injection only fills the gap).
func TestRealConnPreferredOverSyntheticFree(t *testing.T) {
	cs := &fakeConnStore{conns: []*store.Connection{makeConn("real1", "opencode")}}
	engine := NewSelectionEngine(cs, &fakeSettingStore{}, &fakeCooldownForSelection{}, time.Now)

	conn, err := engine.SelectConnection("opencode", "m", nil, "")
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if conn.ID != "real1" {
		t.Errorf("got %q, want the real connection real1 (not synthetic)", conn.ID)
	}
}

func TestPinnedPreferred(t *testing.T) {
	cs := &fakeConnStore{
		conns: []*store.Connection{
			makeConn("c1", "p1"),
			makeConn("c2", "p1"),
		},
	}
	engine := NewSelectionEngine(cs, &fakeSettingStore{}, &fakeCooldownForSelection{}, time.Now)

	conn, err := engine.SelectConnection("p1", "gpt-4", nil, "c2")
	if err != nil {
		t.Fatalf("SelectConnection pinned: %v", err)
	}
	if conn.ID != "c2" {
		t.Errorf("pinned: got %q, want c2", conn.ID)
	}
}

func TestPinnedExcludedFallsThrough(t *testing.T) {
	cs := &fakeConnStore{
		conns: []*store.Connection{
			makeConn("c1", "p1"),
			makeConn("c2", "p1"),
		},
	}
	engine := NewSelectionEngine(cs, &fakeSettingStore{}, &fakeCooldownForSelection{}, time.Now)

	// If preferredConnID is excluded, fall through to fill-first.
	conn, err := engine.SelectConnection("p1", "gpt-4", []string{"c2"}, "c2")
	if err != nil {
		t.Fatalf("pinned+excluded: %v", err)
	}
	if conn.ID != "c1" {
		t.Errorf("pinned+excluded fallthrough: got %q, want c1", conn.ID)
	}
}

func TestStrategyOverridePerProvider(t *testing.T) {
	cs := &fakeConnStore{
		conns: []*store.Connection{
			makeConn("c1", "p1"),
			makeConn("c2", "p1"),
		},
	}
	ss := &fakeSettingStore{settings: map[string]string{
		"providerStrategies": `{"p1":{"fallbackStrategy":"round-robin","stickyRoundRobinLimit":1}}`,
	}}
	engine := NewSelectionEngine(cs, ss, &fakeCooldownForSelection{}, time.Now)

	// stickyLimit=1 → second call must rotate to a different connection.
	first, err := engine.SelectConnection("p1", "gpt-4", nil, "")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := engine.SelectConnection("p1", "gpt-4", nil, "")
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if first.ID == second.ID {
		t.Errorf("round-robin stickyLimit=1: both calls returned %q, want different", first.ID)
	}
}

func TestGlobalFallbackStrategyDefault(t *testing.T) {
	cs := &fakeConnStore{
		conns: []*store.Connection{
			makeConn("c1", "p1"),
			makeConn("c2", "p1"),
		},
	}
	// No strategy configured → fill-first default.
	engine := NewSelectionEngine(cs, &fakeSettingStore{}, &fakeCooldownForSelection{}, time.Now)

	conn, err := engine.SelectConnection("p1", "model", nil, "")
	if err != nil {
		t.Fatalf("%v", err)
	}
	if conn.ID != "c1" {
		t.Errorf("default fill-first: got %q, want c1", conn.ID)
	}
}

func TestSkipsLockedConnections(t *testing.T) {
	now := time.Now()
	cs := &fakeConnStore{
		conns: []*store.Connection{
			makeConn("c1", "p1"),
			makeConn("c2", "p1"),
		},
		locks: map[string][]*store.ModelLock{
			"c1": {{ConnID: "c1", Model: "gpt-4", ExpiresAt: now.Unix() + 100}},
		},
	}
	engine := NewSelectionEngine(cs, &fakeSettingStore{}, &fakeCooldownForSelection{}, func() time.Time { return now })

	conn, err := engine.SelectConnection("p1", "gpt-4", nil, "")
	if err != nil {
		t.Fatalf("%v", err)
	}
	if conn.ID != "c2" {
		t.Errorf("skip locked c1: got %q, want c2", conn.ID)
	}
}

func TestRoundRobinSticky(t *testing.T) {
	cs := &fakeConnStore{
		conns: []*store.Connection{
			makeConn("c1", "p1"),
			makeConn("c2", "p1"),
		},
	}
	ss := &fakeSettingStore{settings: map[string]string{
		"fallbackStrategy":      "round-robin",
		"stickyRoundRobinLimit": "3",
	}}
	engine := NewSelectionEngine(cs, ss, &fakeCooldownForSelection{}, time.Now)

	// First 3 calls → same connection (sticky).
	var seen []string
	for i := 0; i < 3; i++ {
		conn, err := engine.SelectConnection("p1", "gpt-4", nil, "")
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		seen = append(seen, conn.ID)
	}
	if seen[0] != seen[1] || seen[1] != seen[2] {
		t.Errorf("sticky: first 3 calls should return same conn, got %v", seen)
	}

	// 4th call → rotates to a different connection.
	fourth, err := engine.SelectConnection("p1", "gpt-4", nil, "")
	if err != nil {
		t.Fatalf("4th call: %v", err)
	}
	if fourth.ID == seen[0] {
		t.Errorf("round-robin: 4th call should rotate away from %q, got same", seen[0])
	}
}

// slowConn wraps fakeConnStore and introduces a small delay in ListConnections
// to expose concurrent access if the mutex were absent.
type slowConnStore struct {
	*fakeConnStore
	inFlight int64 // accessed via sync/atomic
}

func (s *slowConnStore) ListConnections() ([]*store.Connection, error) {
	n := atomic.AddInt64(&s.inFlight, 1)
	if n > 1 {
		// This path must never be reached if selectionMu works.
		atomic.AddInt64(&s.inFlight, -1)
		return nil, fmt.Errorf("concurrent ListConnections: %d goroutines in flight", n)
	}
	time.Sleep(time.Millisecond) // hold the slot long enough for others to queue
	atomic.AddInt64(&s.inFlight, -1)
	return s.fakeConnStore.ListConnections()
}

func TestSelectionGlobalMutexSerializes(t *testing.T) {
	slow := &slowConnStore{
		fakeConnStore: &fakeConnStore{
			conns: []*store.Connection{
				makeConn("c1", "p1"),
			},
		},
	}
	engine := NewSelectionEngine(slow, &fakeSettingStore{}, &fakeCooldownForSelection{}, time.Now)

	const N = 10
	var wg sync.WaitGroup
	errCh := make(chan error, N)
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			_, err := engine.SelectConnection("p1", "gpt-4", nil, "")
			if err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent access detected: %v", err)
	}
}

func TestFallbackAdvancesOnFailure(t *testing.T) {
	cs := &fakeConnStore{
		conns: []*store.Connection{
			makeConn("c1", "p1"),
			makeConn("c2", "p1"),
		},
	}
	cd := &fakeCooldownForSelection{}
	engine := NewSelectionEngine(cs, &fakeSettingStore{}, cd, time.Now)

	var called []string
	callCount := 0
	err := engine.WithAccountFallback("p1", "gpt-4", func(conn *store.Connection) (Verdict, error) {
		called = append(called, conn.ID)
		callCount++
		if callCount == 1 {
			return VerdictRateLimit, nil
		}
		return VerdictUnknown, nil
	})
	if err != nil {
		t.Fatalf("WithAccountFallback: %v", err)
	}
	if len(called) != 2 {
		t.Fatalf("called %d times, want 2 (advance on failure)", len(called))
	}
	if called[0] == called[1] {
		t.Errorf("fallback should use different connection, got %q twice", called[0])
	}
	if len(cd.unavailCalls) == 0 {
		t.Error("MarkUnavailable should have been called")
	}
}

func TestFallbackTerminatesAllExcluded(t *testing.T) {
	cs := &fakeConnStore{
		conns: []*store.Connection{
			makeConn("c1", "p1"),
			makeConn("c2", "p1"),
		},
	}
	engine := NewSelectionEngine(cs, &fakeSettingStore{}, &fakeCooldownForSelection{}, time.Now)

	callCount := 0
	err := engine.WithAccountFallback("p1", "gpt-4", func(conn *store.Connection) (Verdict, error) {
		callCount++
		return VerdictRateLimit, nil // always fail
	})
	if err == nil {
		t.Fatal("expected error when all accounts excluded, got nil")
	}
	if !errors.Is(err, ErrAllUnavailable) {
		t.Errorf("expected ErrAllUnavailable, got: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected exactly 2 calls (one per connection), got %d", callCount)
	}
}

// fakeCooldownWithRetryAfter returns a specific retry-after time from GroupRetryAfter.
type fakeCooldownWithRetryAfter struct {
	fakeCooldownForSelection
	retryAt time.Time
}

func (f *fakeCooldownWithRetryAfter) GroupRetryAfter(providerID, model string, now time.Time) (time.Time, bool, error) {
	return f.retryAt, true, nil
}

func TestFallbackExhaustionReturnsGroupRetryAfter(t *testing.T) {
	cs := &fakeConnStore{
		conns: []*store.Connection{makeConn("c1", "p1")},
	}
	// Use a distinctive time whose formatted representation we can assert on.
	wantRetry := time.Date(2026, 3, 15, 12, 30, 0, 0, time.UTC)
	cd := &fakeCooldownWithRetryAfter{retryAt: wantRetry}
	engine := NewSelectionEngine(cs, &fakeSettingStore{}, cd, time.Now)

	err := engine.WithAccountFallback("p1", "gpt-4", func(conn *store.Connection) (Verdict, error) {
		return VerdictRateLimit, nil // exhaust the single connection
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrAllUnavailable) {
		t.Errorf("expected errors.Is(err, ErrAllUnavailable), got: %v", err)
	}
	// The error message must contain the exact retry time so callers can parse it.
	wantSubstr := wantRetry.UTC().Format("2006-01-02 15:04:05")
	if got := err.Error(); !strings.Contains(got, wantSubstr) {
		t.Errorf("error %q must contain retry time substring %q", got, wantSubstr)
	}
}

func TestFallbackSuccessMarksReset(t *testing.T) {
	cs := &fakeConnStore{
		conns: []*store.Connection{makeConn("c1", "p1")},
	}
	cd := &fakeCooldownForSelection{}
	engine := NewSelectionEngine(cs, &fakeSettingStore{}, cd, time.Now)

	err := engine.WithAccountFallback("p1", "gpt-4", func(conn *store.Connection) (Verdict, error) {
		return VerdictUnknown, nil // success
	})
	if err != nil {
		t.Fatalf("WithAccountFallback success: %v", err)
	}
	if len(cd.successCalls) != 1 || cd.successCalls[0] != "c1" {
		t.Errorf("MarkSuccess not called correctly: %v", cd.successCalls)
	}
}
