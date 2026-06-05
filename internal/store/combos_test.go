package store

import (
	"errors"
	"testing"
)

func TestComboCreateAndGetByIDRoundTripsSteps(t *testing.T) {
	s := openTestStore(t)
	combo := &Combo{
		Name: "research-chain",
		Steps: []ComboStep{
			{Provider: "anthropic", Model: "claude-sonnet-4-20250514"},
			{Provider: "openai", Model: "gpt-4o"},
		},
		IsActive: true,
	}

	if err := s.CreateCombo(combo); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}
	if combo.ID == "" {
		t.Fatal("ID should be set after create")
	}
	if combo.CreatedAt == "" {
		t.Fatal("CreatedAt should be set after create")
	}
	if combo.UpdatedAt == "" {
		t.Fatal("UpdatedAt should be set after create")
	}

	got, err := s.GetCombo(combo.ID)
	if err != nil {
		t.Fatalf("GetCombo: %v", err)
	}
	if got.Name != "research-chain" {
		t.Fatalf("name = %q, want research-chain", got.Name)
	}
	if !got.IsActive {
		t.Fatal("combo should be active")
	}
	assertComboSteps(t, got.Steps, combo.Steps)
}

func TestComboGetByNameRequiresActive(t *testing.T) {
	s := openTestStore(t)
	active := &Combo{
		Name:     "active-chain",
		Steps:    []ComboStep{{Provider: "openai", Model: "gpt-4o"}},
		IsActive: true,
	}
	inactive := &Combo{
		Name:     "inactive-chain",
		Steps:    []ComboStep{{Provider: "anthropic", Model: "claude-haiku-4"}},
		IsActive: false,
	}
	if err := s.CreateCombo(active); err != nil {
		t.Fatalf("CreateCombo active: %v", err)
	}
	if err := s.CreateCombo(inactive); err != nil {
		t.Fatalf("CreateCombo inactive: %v", err)
	}

	got, err := s.GetActiveCombo("active-chain")
	if err != nil {
		t.Fatalf("GetActiveCombo active: %v", err)
	}
	if got.ID != active.ID {
		t.Fatalf("active combo ID = %q, want %q", got.ID, active.ID)
	}

	_, err = s.GetActiveCombo("inactive-chain")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for inactive combo, got %v", err)
	}
}

func TestComboListOrdersByCreation(t *testing.T) {
	s := openTestStore(t)
	for _, name := range []string{"first", "second"} {
		if err := s.CreateCombo(&Combo{
			Name:     name,
			Steps:    []ComboStep{{Provider: "openai", Model: "gpt-4o"}},
			IsActive: true,
		}); err != nil {
			t.Fatalf("CreateCombo %s: %v", name, err)
		}
	}

	combos, err := s.ListCombos()
	if err != nil {
		t.Fatalf("ListCombos: %v", err)
	}
	if len(combos) != 2 {
		t.Fatalf("len = %d, want 2", len(combos))
	}
	if combos[0].Name != "first" || combos[1].Name != "second" {
		t.Fatalf("unexpected order: %+v", combos)
	}
}

func TestComboUpdate(t *testing.T) {
	s := openTestStore(t)
	combo := &Combo{
		Name:     "chain",
		Steps:    []ComboStep{{Provider: "openai", Model: "gpt-4o"}},
		IsActive: true,
	}
	if err := s.CreateCombo(combo); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	combo.Name = "renamed"
	combo.Steps = []ComboStep{{Provider: "anthropic", Model: "claude-sonnet-4-20250514"}}
	combo.IsActive = false
	if err := s.UpdateCombo(combo); err != nil {
		t.Fatalf("UpdateCombo: %v", err)
	}

	got, err := s.GetCombo(combo.ID)
	if err != nil {
		t.Fatalf("GetCombo: %v", err)
	}
	if got.Name != "renamed" || got.IsActive {
		t.Fatalf("update failed: %+v", got)
	}
	assertComboSteps(t, got.Steps, combo.Steps)
}

func TestComboUpdateNotFound(t *testing.T) {
	s := openTestStore(t)

	err := s.UpdateCombo(&Combo{
		ID:       "missing",
		Name:     "missing",
		Steps:    []ComboStep{{Provider: "openai", Model: "gpt-4o"}},
		IsActive: true,
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestComboDelete(t *testing.T) {
	s := openTestStore(t)
	combo := &Combo{
		Name:     "chain",
		Steps:    []ComboStep{{Provider: "openai", Model: "gpt-4o"}},
		IsActive: true,
	}
	if err := s.CreateCombo(combo); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	if err := s.DeleteCombo(combo.ID); err != nil {
		t.Fatalf("DeleteCombo: %v", err)
	}
	_, err := s.GetCombo(combo.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestComboDeleteNotFound(t *testing.T) {
	s := openTestStore(t)

	err := s.DeleteCombo("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestComboStrategyRoundTrips(t *testing.T) {
	s := openTestStore(t)
	combo := &Combo{
		Name:     "rr-chain",
		Steps:    []ComboStep{{Provider: "openai", Model: "gpt-4o"}},
		Strategy: "round_robin",
		IsActive: true,
	}
	if err := s.CreateCombo(combo); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	got, err := s.GetCombo(combo.ID)
	if err != nil {
		t.Fatalf("GetCombo: %v", err)
	}
	if got.Strategy != "round_robin" {
		t.Fatalf("strategy = %q, want round_robin", got.Strategy)
	}

	got.Strategy = "least_used"
	if err := s.UpdateCombo(got); err != nil {
		t.Fatalf("UpdateCombo: %v", err)
	}
	reloaded, err := s.GetCombo(combo.ID)
	if err != nil {
		t.Fatalf("GetCombo: %v", err)
	}
	if reloaded.Strategy != "least_used" {
		t.Fatalf("strategy = %q, want least_used", reloaded.Strategy)
	}
}

func TestComboStrategyDefaultsToFallback(t *testing.T) {
	s := openTestStore(t)
	combo := &Combo{
		Name:     "default-chain",
		Steps:    []ComboStep{{Provider: "openai", Model: "gpt-4o"}},
		IsActive: true,
	}
	if err := s.CreateCombo(combo); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}
	if combo.Strategy != "fallback" {
		t.Fatalf("strategy = %q, want fallback after create", combo.Strategy)
	}

	got, err := s.GetActiveCombo("default-chain")
	if err != nil {
		t.Fatalf("GetActiveCombo: %v", err)
	}
	if got.Strategy != "fallback" {
		t.Fatalf("loaded strategy = %q, want fallback", got.Strategy)
	}
}

func TestComboStrategyInvalidRejected(t *testing.T) {
	s := openTestStore(t)
	err := s.CreateCombo(&Combo{
		Name:     "bad-chain",
		Steps:    []ComboStep{{Provider: "openai", Model: "gpt-4o"}},
		Strategy: "nonsense",
		IsActive: true,
	})
	if !errors.Is(err, ErrInvalidComboStrategy) {
		t.Fatalf("CreateCombo error = %v, want ErrInvalidComboStrategy", err)
	}

	combo := &Combo{
		Name:     "ok-chain",
		Steps:    []ComboStep{{Provider: "openai", Model: "gpt-4o"}},
		IsActive: true,
	}
	if err := s.CreateCombo(combo); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}
	combo.Strategy = "nonsense"
	if err := s.UpdateCombo(combo); !errors.Is(err, ErrInvalidComboStrategy) {
		t.Fatalf("UpdateCombo error = %v, want ErrInvalidComboStrategy", err)
	}
}

func assertComboSteps(t *testing.T, got, want []ComboStep) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("steps len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("step %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}
