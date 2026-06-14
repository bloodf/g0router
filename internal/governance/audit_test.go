package governance

import (
	"path/filepath"
	"testing"

	"github.com/bloodf/g0router/internal/store"
)

func newAuditTestStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	secret, err := store.LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	st, err := store.Open(filepath.Join(dir, "audit.db"), secret)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestAuditServiceWriteAndList(t *testing.T) {
	st := newAuditTestStore(t)
	svc := NewAuditService(st)

	if err := svc.WriteAudit("admin", "create_team", "Engineering", "Created team Engineering"); err != nil {
		t.Fatalf("WriteAudit: %v", err)
	}
	if err := svc.WriteAudit("admin", "delete_team", "Data Science", "Deleted team Data Science"); err != nil {
		t.Fatalf("WriteAudit second: %v", err)
	}

	items, total, err := svc.List(10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	// Newest-first.
	if items[0].Action != "delete_team" || items[1].Action != "create_team" {
		t.Fatalf("not newest-first: %s, %s", items[0].Action, items[1].Action)
	}
	if items[0].ID == "" || items[0].Timestamp == "" {
		t.Fatalf("id/timestamp not generated: %+v", items[0])
	}
	if items[1].Actor != "admin" || items[1].Target != "Engineering" || items[1].Details != "Created team Engineering" {
		t.Fatalf("fields not round-tripped: %+v", items[1])
	}

	// Limit caps results but total reflects all entries.
	items, total, err = svc.List(1)
	if err != nil {
		t.Fatalf("List(1): %v", err)
	}
	if len(items) != 1 || total != 2 {
		t.Fatalf("List(1) items=%d total=%d, want 1 and 2", len(items), total)
	}
}
