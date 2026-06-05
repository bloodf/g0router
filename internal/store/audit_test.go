package store

import (
	"path/filepath"
	"strings"
	"testing"
)

func newAuditStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestAuditMigrationCreatesTable(t *testing.T) {
	s := newAuditStore(t)
	// A SELECT against the table must succeed, proving the migration ran.
	if _, _, err := s.ListAudit(AuditFilter{}); err != nil {
		t.Fatalf("ListAudit on fresh store: %v", err)
	}
}

func TestAppendAndListAudit(t *testing.T) {
	s := newAuditStore(t)

	for _, e := range []AuditEntry{
		{ActorAPIKeyID: "key-1", Action: "POST /api/keys", Target: "abc", Details: "created api key"},
		{ActorAPIKeyID: "key-1", Action: "DELETE /api/keys/abc", Target: "abc", Details: "revoked api key"},
		{ActorAPIKeyID: "key-2", Action: "PUT /api/settings", Target: "settings", Details: "updated settings"},
	} {
		if err := s.AppendAudit(e); err != nil {
			t.Fatalf("AppendAudit: %v", err)
		}
	}

	entries, total, err := s.ListAudit(AuditFilter{})
	if err != nil {
		t.Fatalf("ListAudit: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if len(entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3", len(entries))
	}
	// Newest first.
	if entries[0].Action != "PUT /api/settings" {
		t.Fatalf("newest action = %q, want PUT /api/settings", entries[0].Action)
	}
	if entries[0].ID == 0 {
		t.Fatalf("expected non-zero ID")
	}
	if entries[0].Timestamp.IsZero() {
		t.Fatalf("expected non-zero timestamp")
	}
}

func TestListAuditPagination(t *testing.T) {
	s := newAuditStore(t)
	for i := 0; i < 5; i++ {
		if err := s.AppendAudit(AuditEntry{ActorAPIKeyID: "k", Action: "POST /api/combos", Target: "c", Details: "x"}); err != nil {
			t.Fatalf("AppendAudit: %v", err)
		}
	}

	entries, total, err := s.ListAudit(AuditFilter{Limit: 2, Offset: 1})
	if err != nil {
		t.Fatalf("ListAudit: %v", err)
	}
	if total != 5 {
		t.Fatalf("total = %d, want 5", total)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
}

func TestListAuditFilter(t *testing.T) {
	s := newAuditStore(t)
	_ = s.AppendAudit(AuditEntry{ActorAPIKeyID: "alice", Action: "POST /api/keys", Target: "1", Details: "x"})
	_ = s.AppendAudit(AuditEntry{ActorAPIKeyID: "bob", Action: "PUT /api/settings", Target: "settings", Details: "x"})
	_ = s.AppendAudit(AuditEntry{ActorAPIKeyID: "alice", Action: "PUT /api/settings", Target: "settings", Details: "x"})

	byActor, total, err := s.ListAudit(AuditFilter{Actor: strPtr("alice")})
	if err != nil {
		t.Fatalf("ListAudit actor: %v", err)
	}
	if total != 2 || len(byActor) != 2 {
		t.Fatalf("actor filter total=%d len=%d, want 2/2", total, len(byActor))
	}

	byAction, total, err := s.ListAudit(AuditFilter{Action: strPtr("PUT /api/settings")})
	if err != nil {
		t.Fatalf("ListAudit action: %v", err)
	}
	if total != 2 || len(byAction) != 2 {
		t.Fatalf("action filter total=%d len=%d, want 2/2", total, len(byAction))
	}

	both, total, err := s.ListAudit(AuditFilter{Actor: strPtr("alice"), Action: strPtr("PUT /api/settings")})
	if err != nil {
		t.Fatalf("ListAudit both: %v", err)
	}
	if total != 1 || len(both) != 1 {
		t.Fatalf("combined filter total=%d len=%d, want 1/1", total, len(both))
	}
}

// TestAuditDetailsAreCallerSanitized documents that the store never inspects or
// strips secrets: it stores exactly what the caller passes. The contract is
// that callers pass a sanitized summary, never a request body.
func TestAuditDetailsAreCallerSanitized(t *testing.T) {
	s := newAuditStore(t)
	if err := s.AppendAudit(AuditEntry{ActorAPIKeyID: "k", Action: "POST /api/connections", Target: "c1", Details: "created connection"}); err != nil {
		t.Fatalf("AppendAudit: %v", err)
	}
	entries, _, err := s.ListAudit(AuditFilter{})
	if err != nil {
		t.Fatalf("ListAudit: %v", err)
	}
	if strings.Contains(entries[0].Details, "token") || strings.Contains(entries[0].Details, "secret") {
		t.Fatalf("details leaked secret-like content: %q", entries[0].Details)
	}
	if entries[0].Details != "created connection" {
		t.Fatalf("details = %q, want exact caller value", entries[0].Details)
	}
}
