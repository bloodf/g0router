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
	wantRetry := time.Date(2026, 1, 1, 0, 5, 0, 0, time.UTC)
	cd := &fakeCooldownWithRetryAfter{retryAt: wantRetry}
	engine := NewSelectionEngine(cs, &fakeSettingStore{}, cd, time.Now)

	err := engine.WithAccountFallback("p1", "gpt-4", func(conn *store.Connection) (Verdict, error) {
		return VerdictRateLimit, nil // exhaust the single connection
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrAllUnavailable) {
		t.Errorf("expected ErrAllUnavailable, got: %v", err)
	}
	if got := err.Error(); !containsRetryTime(got, wantRetry) {
		t.Errorf("error %q should mention retry time %v", got, wantRetry)
	}
}

func containsRetryTime(msg string, t time.Time) bool {
	return strings.Contains(msg, "retry after") || strings.Contains(msg, t.Format("2006"))
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
