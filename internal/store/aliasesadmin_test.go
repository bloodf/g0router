package store

import (
	"errors"
	"path/filepath"
	"testing"
)

func newAliasAdminTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	secret, err := LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	st, err := Open(filepath.Join(dir, "test.db"), secret)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestAliasRecordCRUD(t *testing.T) {
	st := newAliasAdminTestStore(t)

	created, err := st.CreateAliasRecord(&AliasRecord{Alias: "gpt4", Provider: "openai", Model: "gpt-4o"})
	if err != nil {
		t.Fatalf("CreateAliasRecord: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected generated id, got empty")
	}
	if created.Alias != "gpt4" || created.Provider != "openai" || created.Model != "gpt-4o" {
		t.Fatalf("created = %+v", created)
	}
	if created.CreatedAt == 0 || created.UpdatedAt == 0 {
		t.Fatalf("timestamps not set: %+v", created)
	}

	got, err := st.GetAliasRecordByID(created.ID)
	if err != nil {
		t.Fatalf("GetAliasRecordByID: %v", err)
	}
	if got.Alias != "gpt4" {
		t.Fatalf("got = %+v", got)
	}

	if _, err := st.CreateAliasRecord(&AliasRecord{Alias: "claude", Provider: "anthropic", Model: "claude-sonnet-4"}); err != nil {
		t.Fatalf("CreateAliasRecord 2: %v", err)
	}

	list, err := st.ListAliasRecords()
	if err != nil {
		t.Fatalf("ListAliasRecords: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}

	created.Alias = "gpt4-fast"
	created.Model = "gpt-4o-mini"
	if err := st.UpdateAliasRecord(created); err != nil {
		t.Fatalf("UpdateAliasRecord: %v", err)
	}
	got, err = st.GetAliasRecordByID(created.ID)
	if err != nil {
		t.Fatalf("GetAliasRecordByID after update: %v", err)
	}
	if got.Alias != "gpt4-fast" || got.Model != "gpt-4o-mini" {
		t.Fatalf("updated = %+v", got)
	}

	if err := st.DeleteAliasRecord(created.ID); err != nil {
		t.Fatalf("DeleteAliasRecord: %v", err)
	}
	if _, err := st.GetAliasRecordByID(created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
	if err := st.DeleteAliasRecord(created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound on double delete, got %v", err)
	}
	if err := st.UpdateAliasRecord(&AliasRecord{ID: "missing", Alias: "x"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound on update missing, got %v", err)
	}
}
