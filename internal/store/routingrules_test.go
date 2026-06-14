package store

import (
	"errors"
	"path/filepath"
	"testing"
)

func newRoutingRuleTestStore(t *testing.T) *Store {
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

func TestRoutingRuleCRUDAndOrder(t *testing.T) {
	st := newRoutingRuleTestStore(t)

	// Insert priority 2 first, then priority 1, to prove ListRoutingRules orders by priority ASC.
	if _, err := st.CreateRoutingRule(&RoutingRule{
		Name: "second", Priority: 2, CondField: "model", CondOperator: "equals",
		CondValue: "claude", TargetProvider: "anthropic", IsActive: true,
	}); err != nil {
		t.Fatalf("CreateRoutingRule second: %v", err)
	}
	first, err := st.CreateRoutingRule(&RoutingRule{
		Name: "first", Priority: 1, CondField: "model", CondOperator: "equals",
		CondValue: "gpt-4o", TargetProvider: "openai", IsActive: true,
	})
	if err != nil {
		t.Fatalf("CreateRoutingRule first: %v", err)
	}
	if first.ID == "" || first.CreatedAt == 0 || first.UpdatedAt == 0 {
		t.Fatalf("first = %+v", first)
	}

	list, err := st.ListRoutingRules()
	if err != nil {
		t.Fatalf("ListRoutingRules: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len = %d, want 2", len(list))
	}
	if list[0].Name != "first" || list[1].Name != "second" {
		t.Fatalf("order = %s, %s; want first, second", list[0].Name, list[1].Name)
	}

	got, err := st.GetRoutingRuleByID(first.ID)
	if err != nil {
		t.Fatalf("GetRoutingRuleByID: %v", err)
	}
	if got.TargetProvider != "openai" || !got.IsActive {
		t.Fatalf("got = %+v", got)
	}

	first.Name = "first-updated"
	first.IsActive = false
	if err := st.UpdateRoutingRule(first); err != nil {
		t.Fatalf("UpdateRoutingRule: %v", err)
	}
	got, err = st.GetRoutingRuleByID(first.ID)
	if err != nil {
		t.Fatalf("GetRoutingRuleByID after update: %v", err)
	}
	if got.Name != "first-updated" || got.IsActive {
		t.Fatalf("updated = %+v", got)
	}

	if err := st.DeleteRoutingRule(first.ID); err != nil {
		t.Fatalf("DeleteRoutingRule: %v", err)
	}
	if _, err := st.GetRoutingRuleByID(first.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
	if err := st.DeleteRoutingRule(first.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound on double delete, got %v", err)
	}
	if err := st.UpdateRoutingRule(&RoutingRule{ID: "missing", Name: "x"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound on update missing, got %v", err)
	}
}
