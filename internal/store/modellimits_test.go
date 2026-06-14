package store

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func newModelLimitTestStore(t *testing.T) *Store {
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

func TestModelLimitCRUDAndKeyIDsRoundTrip(t *testing.T) {
	st := newModelLimitTestStore(t)

	created, err := st.CreateModelLimit(&ModelLimit{
		Model: "gpt-4o", MaxTokens: 128000, MaxRPM: 1000, AllowedKeyIDs: []string{"key-1"},
	})
	if err != nil {
		t.Fatalf("CreateModelLimit: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("expected autoincrement id > 0, got %d", created.ID)
	}
	if created.CreatedAt == 0 || created.UpdatedAt == 0 {
		t.Fatalf("timestamps not set: %+v", created)
	}

	second, err := st.CreateModelLimit(&ModelLimit{
		Model: "claude-sonnet-4", MaxTokens: 200000, MaxRPM: 500, AllowedKeyIDs: []string{"key-1", "key-2"},
	})
	if err != nil {
		t.Fatalf("CreateModelLimit second: %v", err)
	}
	if second.ID == created.ID {
		t.Fatalf("expected distinct autoincrement ids, both %d", created.ID)
	}

	got, err := st.GetModelLimitByID(created.ID)
	if err != nil {
		t.Fatalf("GetModelLimitByID: %v", err)
	}
	if got.Model != "gpt-4o" || !reflect.DeepEqual(got.AllowedKeyIDs, []string{"key-1"}) {
		t.Fatalf("got = %+v", got)
	}

	list, err := st.ListModelLimits()
	if err != nil {
		t.Fatalf("ListModelLimits: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len = %d, want 2", len(list))
	}

	got.Model = "gpt-4o-mini"
	got.MaxTokens = 64000
	got.AllowedKeyIDs = []string{"key-3"}
	if err := st.UpdateModelLimit(got); err != nil {
		t.Fatalf("UpdateModelLimit: %v", err)
	}
	got, err = st.GetModelLimitByID(created.ID)
	if err != nil {
		t.Fatalf("GetModelLimitByID after update: %v", err)
	}
	if got.Model != "gpt-4o-mini" || got.MaxTokens != 64000 || !reflect.DeepEqual(got.AllowedKeyIDs, []string{"key-3"}) {
		t.Fatalf("updated = %+v", got)
	}

	if err := st.DeleteModelLimit(created.ID); err != nil {
		t.Fatalf("DeleteModelLimit: %v", err)
	}
	if _, err := st.GetModelLimitByID(created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
	if err := st.DeleteModelLimit(created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound on double delete, got %v", err)
	}
	if err := st.UpdateModelLimit(&ModelLimit{ID: 99999, Model: "x"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound on update missing, got %v", err)
	}
}
