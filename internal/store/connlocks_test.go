package store

import (
	"testing"
	"time"
)

func TestModelLockCRUD(t *testing.T) {
	st := newTestStore(t)
	now := time.Now().Unix()
	future := now + 60

	if err := st.LockModel("conn1", "p1", "gpt-4", future); err != nil {
		t.Fatalf("LockModel: %v", err)
	}

	locks, err := st.ActiveLocks("conn1", now)
	if err != nil {
		t.Fatalf("ActiveLocks: %v", err)
	}
	if len(locks) != 1 || locks[0].Model != "gpt-4" || locks[0].ExpiresAt != future {
		t.Fatalf("ActiveLocks = %+v", locks)
	}

	// Expired lock is not returned.
	past := now - 1
	if err := st.LockModel("conn1", "p1", "gpt-3.5", past); err != nil {
		t.Fatalf("LockModel expired: %v", err)
	}
	locks, err = st.ActiveLocks("conn1", now)
	if err != nil {
		t.Fatalf("ActiveLocks after expired: %v", err)
	}
	if len(locks) != 1 || locks[0].Model != "gpt-4" {
		t.Fatalf("should only see active lock, got = %+v", locks)
	}

	// ClearLocks removes all locks for connection.
	if err := st.ClearLocks("conn1"); err != nil {
		t.Fatalf("ClearLocks: %v", err)
	}
	locks, err = st.ActiveLocks("conn1", now)
	if err != nil {
		t.Fatalf("ActiveLocks after clear: %v", err)
	}
	if len(locks) != 0 {
		t.Fatalf("after ClearLocks, got %d locks, want 0", len(locks))
	}
}

func TestAccountLockSentinel(t *testing.T) {
	st := newTestStore(t)
	now := time.Now().Unix()
	future := now + 120

	if err := st.LockAccount("conn2", "p1", future); err != nil {
		t.Fatalf("LockAccount: %v", err)
	}

	locks, err := st.ActiveLocks("conn2", now)
	if err != nil {
		t.Fatalf("ActiveLocks: %v", err)
	}
	if len(locks) != 1 || locks[0].Model != "__all" {
		t.Fatalf("account lock model sentinel = %q, want __all", locks[0].Model)
	}
}

func TestMigrationAdditiveRerun(t *testing.T) {
	st := newTestStore(t)

	// Running migrate a second time on the same DB must not error.
	if err := migrate(st.db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	// connection_model_locks table must exist after migration with provider_id column.
	if _, err := st.db.Exec("SELECT connection_id, provider_id, model, expires_at FROM connection_model_locks LIMIT 0"); err != nil {
		t.Fatalf("connection_model_locks table missing: %v", err)
	}

	// Additive connections columns must exist.
	for _, col := range []string{"backoff_level", "rate_limited_until", "last_error"} {
		if _, err := st.db.Exec("SELECT " + col + " FROM connections LIMIT 0"); err != nil {
			t.Fatalf("connections.%s missing: %v", col, err)
		}
	}
}

func TestEarliestExpiryAcrossConnections(t *testing.T) {
	st := newTestStore(t)
	now := time.Now().Unix()

	// Two connections with locks on the same model under the same provider.
	if err := st.LockModel("connA", "p1", "claude-3", now+200); err != nil {
		t.Fatalf("LockModel connA: %v", err)
	}
	if err := st.LockModel("connB", "p1", "claude-3", now+100); err != nil {
		t.Fatalf("LockModel connB: %v", err)
	}
	// One with a different model — must not affect claude-3 expiry.
	if err := st.LockModel("connC", "p1", "gpt-4", now+50); err != nil {
		t.Fatalf("LockModel connC gpt-4: %v", err)
	}

	earliest, ok, err := st.EarliestExpiry("p1", "claude-3", now)
	if err != nil {
		t.Fatalf("EarliestExpiry: %v", err)
	}
	if !ok {
		t.Fatal("EarliestExpiry ok=false, want true")
	}
	if earliest != now+100 {
		t.Fatalf("EarliestExpiry = %d, want %d", earliest, now+100)
	}

	// No active locks for unknown model under p1.
	_, ok, err = st.EarliestExpiry("p1", "unknown-model", now)
	if err != nil {
		t.Fatalf("EarliestExpiry unknown: %v", err)
	}
	if ok {
		t.Fatal("EarliestExpiry for unknown model should be ok=false")
	}

	// Account-level lock (__all) must be included in EarliestExpiry for any model.
	if err := st.LockAccount("connD", "p1", now+60); err != nil {
		t.Fatalf("LockAccount connD: %v", err)
	}
	earliest, ok, err = st.EarliestExpiry("p1", "any-model", now)
	if err != nil {
		t.Fatalf("EarliestExpiry __all: %v", err)
	}
	if !ok {
		t.Fatal("EarliestExpiry with __all lock ok=false, want true")
	}
	if earliest != now+60 {
		t.Fatalf("EarliestExpiry with __all = %d, want %d", earliest, now+60)
	}
}

func TestBackoffLevelRoundTrip(t *testing.T) {
	st := newTestStore(t)

	// Non-existent connection returns 0, no error.
	level, err := st.GetBackoffLevel("no-such-conn")
	if err != nil {
		t.Fatalf("GetBackoffLevel missing: %v", err)
	}
	if level != 0 {
		t.Fatalf("missing connection backoff = %d, want 0", level)
	}

	// Create a real connection so SetBackoffLevel has a row to UPDATE.
	p := &ProviderRecord{Name: "Test", Type: "openai", Enabled: true}
	if err := st.CreateProvider(p); err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}
	c := &Connection{ProviderID: p.ID, Name: "key", Kind: "api_key"}
	if err := st.CreateConnection(c); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	// Initial level is 0.
	level, err = st.GetBackoffLevel(c.ID)
	if err != nil {
		t.Fatalf("GetBackoffLevel initial: %v", err)
	}
	if level != 0 {
		t.Fatalf("initial backoff level = %d, want 0", level)
	}

	if err := st.SetBackoffLevel(c.ID, 3); err != nil {
		t.Fatalf("SetBackoffLevel: %v", err)
	}
	level, err = st.GetBackoffLevel(c.ID)
	if err != nil {
		t.Fatalf("GetBackoffLevel after set: %v", err)
	}
	if level != 3 {
		t.Fatalf("backoff level = %d, want 3", level)
	}
}

func TestSetRateLimitedUntil(t *testing.T) {
	st := newTestStore(t)

	p := &ProviderRecord{Name: "Test", Type: "openai", Enabled: true}
	if err := st.CreateProvider(p); err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}
	c := &Connection{ProviderID: p.ID, Name: "key", Kind: "api_key"}
	if err := st.CreateConnection(c); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	until := time.Now().Unix() + 120
	if err := st.SetRateLimitedUntil(c.ID, until); err != nil {
		t.Fatalf("SetRateLimitedUntil: %v", err)
	}

	var got int64
	if err := st.db.QueryRow("SELECT rate_limited_until FROM connections WHERE id = ?", c.ID).Scan(&got); err != nil {
		t.Fatalf("SELECT rate_limited_until: %v", err)
	}
	if got != until {
		t.Fatalf("rate_limited_until = %d, want %d", got, until)
	}
}
