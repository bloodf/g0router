package inference

import (
	"testing"
	"time"
)

type fakeLock struct {
	connID     string
	providerID string
	model      string
	expiresAt  int64
}

type fakeLockStore struct {
	locks      []fakeLock
	backoffs   map[string]int
	rateLimits map[string]int64
}

func newFakeLockStore() *fakeLockStore {
	return &fakeLockStore{
		backoffs:   make(map[string]int),
		rateLimits: make(map[string]int64),
	}
}

func (f *fakeLockStore) LockModel(connID, providerID, model string, expiresAt int64) error {
	for i, l := range f.locks {
		if l.connID == connID && l.model == model {
			f.locks[i].expiresAt = expiresAt
			f.locks[i].providerID = providerID
			return nil
		}
	}
	f.locks = append(f.locks, fakeLock{connID: connID, providerID: providerID, model: model, expiresAt: expiresAt})
	return nil
}

func (f *fakeLockStore) LockAccount(connID, providerID string, expiresAt int64) error {
	return f.LockModel(connID, providerID, "__all", expiresAt)
}

func (f *fakeLockStore) ClearLocks(connID string) error {
	var out []fakeLock
	for _, l := range f.locks {
		if l.connID != connID {
			out = append(out, l)
		}
	}
	f.locks = out
	return nil
}

func (f *fakeLockStore) EarliestExpiry(providerID, model string, now int64) (int64, bool, error) {
	var min int64
	found := false
	for _, l := range f.locks {
		if l.providerID == providerID && (l.model == model || l.model == "__all") && l.expiresAt > now {
			if !found || l.expiresAt < min {
				min = l.expiresAt
				found = true
			}
		}
	}
	return min, found, nil
}

func (f *fakeLockStore) SetBackoffLevel(connID string, level int) error {
	f.backoffs[connID] = level
	return nil
}

func (f *fakeLockStore) GetBackoffLevel(connID string) (int, error) {
	return f.backoffs[connID], nil
}

func (f *fakeLockStore) SetRateLimitedUntil(connID string, until int64) error {
	f.rateLimits[connID] = until
	return nil
}

// activeLocks is a test helper — not part of LockStore interface.
func (f *fakeLockStore) activeLocks(connID string, now int64) []fakeLock {
	var out []fakeLock
	for _, l := range f.locks {
		if l.connID == connID && l.expiresAt > now {
			out = append(out, l)
		}
	}
	return out
}

func TestBackoffSchedule(t *testing.T) {
	// BACKOFF_CONFIG.base = 2000ms, max = 5*60*1000ms = 300000ms, maxLevel = 15.
	// getQuotaCooldown(level): level = max(0, backoffLevel-1); cooldown = base * 2^level capped at max.
	cases := []struct {
		backoffLevel int
		wantMs       int64
	}{
		{1, 2000},   // level=0 → 2000*1 = 2s
		{2, 4000},   // level=1 → 2000*2 = 4s
		{3, 8000},   // level=2 → 2000*4 = 8s
		{4, 16000},  // level=3 → 16s
		{5, 32000},  // level=4 → 32s
		{8, 256000}, // level=7 → 256s
		{9, 300000}, // level=8 → 512s → capped at 300s
		{15, 300000},
	}
	for _, tc := range cases {
		got := quotaCooldown(tc.backoffLevel)
		if got != tc.wantMs {
			t.Errorf("quotaCooldown(%d) = %dms, want %dms", tc.backoffLevel, got, tc.wantMs)
		}
	}
}

func TestMarkUnavailableRateLimit(t *testing.T) {
	st := newFakeLockStore()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	engine := NewCooldownEngine(st, func() time.Time { return now })

	// First unavailable: backoffLevel 0 → 1, cooldown = 2000ms = 2s.
	if err := engine.MarkUnavailable("conn1", "p1", "gpt-4", VerdictRateLimit); err != nil {
		t.Fatalf("MarkUnavailable: %v", err)
	}
	if st.backoffs["conn1"] != 1 {
		t.Fatalf("backoff level = %d, want 1", st.backoffs["conn1"])
	}
	wantExpiry := now.Add(2 * time.Second).Unix()
	// rate_limited_until must be written.
	if got := st.rateLimits["conn1"]; got != wantExpiry {
		t.Fatalf("rate_limited_until = %d, want %d", got, wantExpiry)
	}
	found := false
	for _, l := range st.locks {
		if l.connID == "conn1" && l.model == "gpt-4" && l.expiresAt == wantExpiry {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected lock gpt-4 expiresAt=%d, locks=%+v", wantExpiry, st.locks)
	}
}

func TestSuccessResets(t *testing.T) {
	st := newFakeLockStore()
	now := time.Now()
	engine := NewCooldownEngine(st, func() time.Time { return now })

	st.backoffs["conn1"] = 5
	_ = st.LockModel("conn1", "p1", "gpt-4", now.Unix()+100)

	if err := engine.MarkSuccess("conn1"); err != nil {
		t.Fatalf("MarkSuccess: %v", err)
	}
	if st.backoffs["conn1"] != 0 {
		t.Fatalf("backoff after success = %d, want 0", st.backoffs["conn1"])
	}
	if len(st.locks) != 0 {
		t.Fatalf("locks after success = %+v, want empty", st.locks)
	}
}

func TestGroupRetryAfter(t *testing.T) {
	st := newFakeLockStore()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	engine := NewCooldownEngine(st, func() time.Time { return now })

	_, ok := engine.GroupRetryAfter("p1", "gpt-4", now)
	if ok {
		t.Fatal("GroupRetryAfter with no locks: ok=true, want false")
	}

	_ = st.LockModel("connA", "p1", "gpt-4", now.Unix()+200)
	_ = st.LockModel("connB", "p1", "gpt-4", now.Unix()+100)

	retryAt, ok := engine.GroupRetryAfter("p1", "gpt-4", now)
	if !ok {
		t.Fatal("GroupRetryAfter ok=false, want true")
	}
	want := time.Unix(now.Unix()+100, 0)
	if !retryAt.Equal(want) {
		t.Fatalf("GroupRetryAfter = %v, want %v", retryAt, want)
	}
}

func TestPermanentAndAuthNotLocked(t *testing.T) {
	st := newFakeLockStore()
	now := time.Now()
	engine := NewCooldownEngine(st, func() time.Time { return now })

	for _, verdict := range []Verdict{VerdictAuth, VerdictPermanent} {
		if err := engine.MarkUnavailable("conn1", "p1", "gpt-4", verdict); err != nil {
			t.Fatalf("MarkUnavailable %v: %v", verdict, err)
		}
	}
	active := st.activeLocks("conn1", now.Unix())
	if len(active) != 0 {
		t.Fatalf("Auth/Permanent verdict created locks: %+v", active)
	}
}
