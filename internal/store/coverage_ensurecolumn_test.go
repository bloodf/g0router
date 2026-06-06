package store

import (
	"testing"
)

func TestEnsureColumnInvalidTable(t *testing.T) {
	s := openTestStore(t)
	if err := s.ensureColumn("bad-table", "col", "TEXT"); err == nil {
		t.Fatal("expected error for invalid table name")
	}
}

func TestEnsureColumnInvalidColumn(t *testing.T) {
	s := openTestStore(t)
	if err := s.ensureColumn("combos", "bad-col", "TEXT"); err == nil {
		t.Fatal("expected error for invalid column name")
	}
}
