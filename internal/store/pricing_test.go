package store

import (
	"errors"
	"testing"
)

func TestPricingOverrideSetAndGet(t *testing.T) {
	s := openTestStore(t)

	override := PricingOverride{
		Provider:           "openai",
		Model:              "gpt-4o-mini",
		InputCostPerToken:  0.00000015,
		OutputCostPerToken: 0.0000006,
	}
	if err := s.SetPricingOverride(override); err != nil {
		t.Fatalf("SetPricingOverride: %v", err)
	}

	got, err := s.GetPricingOverride("openai", "gpt-4o-mini")
	if err != nil {
		t.Fatalf("GetPricingOverride: %v", err)
	}
	if got != override {
		t.Fatalf("override = %+v, want %+v", got, override)
	}
}

func TestPricingOverrideSetReplacesExisting(t *testing.T) {
	s := openTestStore(t)

	if err := s.SetPricingOverride(PricingOverride{
		Provider:           "openai",
		Model:              "gpt-4o-mini",
		InputCostPerToken:  0.00000015,
		OutputCostPerToken: 0.0000006,
	}); err != nil {
		t.Fatalf("first SetPricingOverride: %v", err)
	}
	want := PricingOverride{
		Provider:           "openai",
		Model:              "gpt-4o-mini",
		InputCostPerToken:  0.00000010,
		OutputCostPerToken: 0.0000004,
	}
	if err := s.SetPricingOverride(want); err != nil {
		t.Fatalf("second SetPricingOverride: %v", err)
	}

	got, err := s.GetPricingOverride("openai", "gpt-4o-mini")
	if err != nil {
		t.Fatalf("GetPricingOverride: %v", err)
	}
	if got != want {
		t.Fatalf("override = %+v, want %+v", got, want)
	}
}

func TestPricingOverrideListOrdersByProviderAndModel(t *testing.T) {
	s := openTestStore(t)

	for _, override := range []PricingOverride{
		{Provider: "openai", Model: "gpt-4o-mini", InputCostPerToken: 0.00000015, OutputCostPerToken: 0.0000006},
		{Provider: "anthropic", Model: "claude-sonnet-4-20250514", InputCostPerToken: 0.000003, OutputCostPerToken: 0.000015},
	} {
		if err := s.SetPricingOverride(override); err != nil {
			t.Fatalf("SetPricingOverride %s/%s: %v", override.Provider, override.Model, err)
		}
	}

	overrides, err := s.ListPricingOverrides()
	if err != nil {
		t.Fatalf("ListPricingOverrides: %v", err)
	}
	want := []PricingOverride{
		{Provider: "anthropic", Model: "claude-sonnet-4-20250514", InputCostPerToken: 0.000003, OutputCostPerToken: 0.000015},
		{Provider: "openai", Model: "gpt-4o-mini", InputCostPerToken: 0.00000015, OutputCostPerToken: 0.0000006},
	}
	if len(overrides) != len(want) {
		t.Fatalf("len = %d, want %d", len(overrides), len(want))
	}
	for i := range want {
		if overrides[i] != want[i] {
			t.Fatalf("override %d = %+v, want %+v", i, overrides[i], want[i])
		}
	}
}

func TestPricingOverrideDelete(t *testing.T) {
	s := openTestStore(t)

	if err := s.SetPricingOverride(PricingOverride{
		Provider:           "openai",
		Model:              "gpt-4o-mini",
		InputCostPerToken:  0.00000015,
		OutputCostPerToken: 0.0000006,
	}); err != nil {
		t.Fatalf("SetPricingOverride: %v", err)
	}
	if err := s.DeletePricingOverride("openai", "gpt-4o-mini"); err != nil {
		t.Fatalf("DeletePricingOverride: %v", err)
	}

	_, err := s.GetPricingOverride("openai", "gpt-4o-mini")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestPricingOverrideNotFound(t *testing.T) {
	s := openTestStore(t)

	_, err := s.GetPricingOverride("openai", "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	err = s.DeletePricingOverride("openai", "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound deleting missing override, got %v", err)
	}
}
