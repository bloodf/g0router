package store

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func newComboAdminTestStore(t *testing.T) *Store {
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

func TestComboAdminCRUDAndStepOrder(t *testing.T) {
	st := newComboAdminTestStore(t)

	steps := []ComboStep{
		{Provider: "groq", Model: "llama-3-70b"},
		{Provider: "openai", Model: "gpt-4o-mini"},
	}
	created, err := st.CreateComboAdmin(&ComboAdmin{
		Name: "Fast + Cheap", Strategy: "fallback", Steps: steps, IsActive: true,
	})
	if err != nil {
		t.Fatalf("CreateComboAdmin: %v", err)
	}
	if created.ID == "" || created.CreatedAt == 0 || created.UpdatedAt == 0 {
		t.Fatalf("created = %+v", created)
	}

	got, err := st.GetComboAdminByID(created.ID)
	if err != nil {
		t.Fatalf("GetComboAdminByID: %v", err)
	}
	if !reflect.DeepEqual(got.Steps, steps) {
		t.Fatalf("steps = %+v, want %+v", got.Steps, steps)
	}

	if _, err := st.CreateComboAdmin(&ComboAdmin{Name: "Best Quality", Strategy: "fallback", IsActive: true}); err != nil {
		t.Fatalf("CreateComboAdmin 2: %v", err)
	}

	list, err := st.ListComboAdmins()
	if err != nil {
		t.Fatalf("ListComboAdmins: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len = %d, want 2", len(list))
	}

	// Reorder steps and persist.
	reordered := []ComboStep{
		{Provider: "openai", Model: "gpt-4o-mini"},
		{Provider: "groq", Model: "llama-3-70b"},
	}
	got.Steps = reordered
	got.Name = "Fast + Cheap (edited)"
	if err := st.UpdateComboAdmin(got); err != nil {
		t.Fatalf("UpdateComboAdmin: %v", err)
	}
	got, err = st.GetComboAdminByID(created.ID)
	if err != nil {
		t.Fatalf("GetComboAdminByID after update: %v", err)
	}
	if !reflect.DeepEqual(got.Steps, reordered) || got.Name != "Fast + Cheap (edited)" {
		t.Fatalf("updated = %+v", got)
	}

	if err := st.DeleteComboAdmin(created.ID); err != nil {
		t.Fatalf("DeleteComboAdmin: %v", err)
	}
	if _, err := st.GetComboAdminByID(created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
	if err := st.DeleteComboAdmin(created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound on double delete, got %v", err)
	}
	if err := st.UpdateComboAdmin(&ComboAdmin{ID: "missing", Name: "x"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound on update missing, got %v", err)
	}
}
