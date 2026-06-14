package store

import (
	"fmt"
	"testing"
	"time"
)

func TestAuditLogInsertListCount(t *testing.T) {
	st := newTestStore(t)

	// Empty store.
	n, err := st.CountAuditEntries()
	if err != nil {
		t.Fatalf("CountAuditEntries: %v", err)
	}
	if n != 0 {
		t.Fatalf("initial count = %d, want 0", n)
	}

	// Insert 5 entries with increasing timestamps so ordering is deterministic.
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		e := AuditEntry{
			ID:        fmt.Sprintf("audit-%d", i),
			Timestamp: base.Add(time.Duration(i) * time.Minute).Format(time.RFC3339),
			Actor:     "admin",
			Action:    fmt.Sprintf("action_%d", i),
			Target:    fmt.Sprintf("target-%d", i),
			Details:   fmt.Sprintf("did action %d", i),
		}
		if err := st.InsertAuditEntry(e); err != nil {
			t.Fatalf("InsertAuditEntry %d: %v", i, err)
		}
	}

	n, err = st.CountAuditEntries()
	if err != nil {
		t.Fatalf("CountAuditEntries: %v", err)
	}
	if n != 5 {
		t.Fatalf("count = %d, want 5", n)
	}

	// List with limit returns newest-first, capped at limit.
	items, err := st.ListAuditEntries(2)
	if err != nil {
		t.Fatalf("ListAuditEntries: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Action != "action_4" || items[1].Action != "action_3" {
		t.Fatalf("not newest-first: %s, %s", items[0].Action, items[1].Action)
	}
	if items[0].Details != "did action 4" || items[0].Target != "target-4" || items[0].Actor != "admin" {
		t.Fatalf("entry fields not round-tripped: %+v", items[0])
	}

	// Limit larger than count returns all.
	all, err := st.ListAuditEntries(100)
	if err != nil {
		t.Fatalf("ListAuditEntries all: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("len(all) = %d, want 5", len(all))
	}
}
