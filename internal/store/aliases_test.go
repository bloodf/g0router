package store

import (
	"strings"
	"testing"
)

func TestAliasChainResolution(t *testing.T) {
	st := newTestStore(t)

	if err := st.CreateAlias("C", "claude-3-5-sonnet"); err != nil {
		t.Fatalf("CreateAlias C: %v", err)
	}
	if err := st.CreateAlias("B", "C"); err != nil {
		t.Fatalf("CreateAlias B: %v", err)
	}
	if err := st.CreateAlias("A", "B"); err != nil {
		t.Fatalf("CreateAlias A: %v", err)
	}

	got, err := st.ResolveChain("A")
	if err != nil {
		t.Fatalf("ResolveChain(A): %v", err)
	}
	if got != "claude-3-5-sonnet" {
		t.Errorf("ResolveChain(A) = %q, want claude-3-5-sonnet", got)
	}
}

func TestAliasCycleRejectedOnWrite(t *testing.T) {
	st := newTestStore(t)

	if err := st.CreateAlias("A", "B"); err != nil {
		t.Fatalf("CreateAlias A->B: %v", err)
	}

	err := st.CreateAlias("B", "A")
	if err == nil {
		t.Fatal("CreateAlias B->A should reject cycle")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("CreateAlias B->A error = %v, want error containing 'cycle'", err)
	}
}

func TestAliasMissingPassthrough(t *testing.T) {
	st := newTestStore(t)

	got, err := st.ResolveChain("unknown")
	if err != nil {
		t.Fatalf("ResolveChain(unknown): %v", err)
	}
	if got != "unknown" {
		t.Errorf("ResolveChain(unknown) = %q, want unknown", got)
	}
}

func TestMigrationAliasesAdditive(t *testing.T) {
	st := newTestStore(t)

	var count int
	if err := st.DB().QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'model_aliases'").Scan(&count); err != nil {
		t.Fatalf("count tables: %v", err)
	}
	if count != 1 {
		t.Fatalf("model_aliases table count = %d, want 1", count)
	}

	// Re-running migrate on an already-migrated DB must be a no-op.
	if err := migrate(st.DB()); err != nil {
		t.Fatalf("migrate second run: %v", err)
	}
}

func TestAliasListAndDelete(t *testing.T) {
	st := newTestStore(t)

	if err := st.CreateAlias("fast", "gpt-4o-mini"); err != nil {
		t.Fatalf("CreateAlias fast: %v", err)
	}
	if err := st.CreateAlias("smart", "claude-3-opus"); err != nil {
		t.Fatalf("CreateAlias smart: %v", err)
	}

	list, err := st.ListAliases()
	if err != nil {
		t.Fatalf("ListAliases: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}

	if err := st.DeleteAlias("fast"); err != nil {
		t.Fatalf("DeleteAlias: %v", err)
	}

	list, err = st.ListAliases()
	if err != nil {
		t.Fatalf("ListAliases after delete: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) after delete = %d, want 1", len(list))
	}
}

