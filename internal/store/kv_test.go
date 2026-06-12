package store

import (
	"strings"
	"testing"
)

func TestKVRoundTrip(t *testing.T) {
	st := newTestStore(t)

	scope := "test"
	if err := st.SetKV(scope, "k1", "v1"); err != nil {
		t.Fatalf("SetKV: %v", err)
	}
	if err := st.SetKV(scope, "k2", "v2"); err != nil {
		t.Fatalf("SetKV: %v", err)
	}

	v, err := st.GetKV(scope, "k1")
	if err != nil {
		t.Fatalf("GetKV k1: %v", err)
	}
	if v != "v1" {
		t.Errorf("GetKV k1 = %q, want v1", v)
	}

	// Missing key returns ("", nil), matching settings.go ErrNoRows convention.
	missing, err := st.GetKV(scope, "no-such-key")
	if err != nil {
		t.Fatalf("GetKV missing: expected nil err, got %v", err)
	}
	if missing != "" {
		t.Errorf("GetKV missing = %q, want empty", missing)
	}

	all, err := st.ListKV(scope)
	if err != nil {
		t.Fatalf("ListKV: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("ListKV len = %d, want 2", len(all))
	}
	if all["k1"] != "v1" || all["k2"] != "v2" {
		t.Errorf("ListKV = %v, want k1=v1, k2=v2", all)
	}

	// Upsert updates existing key.
	if err := st.SetKV(scope, "k1", "v1-updated"); err != nil {
		t.Fatalf("SetKV update: %v", err)
	}
	v, err = st.GetKV(scope, "k1")
	if err != nil {
		t.Fatalf("GetKV updated k1: %v", err)
	}
	if v != "v1-updated" {
		t.Errorf("GetKV updated k1 = %q, want v1-updated", v)
	}

	// Different scopes are isolated.
	if err := st.SetKV("other", "k1", "v-other"); err != nil {
		t.Fatalf("SetKV other scope: %v", err)
	}
	sameScope, err := st.ListKV(scope)
	if err != nil {
		t.Fatalf("ListKV scope after other: %v", err)
	}
	if sameScope["k1"] != "v1-updated" {
		t.Errorf("ListKV scope k1 = %q, want v1-updated", sameScope["k1"])
	}
}

func TestUserPricingReadsKV(t *testing.T) {
	st := newTestStore(t)

	// Empty scope returns an empty non-nil map.
	got, err := st.UserPricing()
	if err != nil {
		t.Fatalf("UserPricing empty: %v", err)
	}
	if got == nil {
		t.Fatal("UserPricing empty returned nil map")
	}
	if len(got) != 0 {
		t.Errorf("UserPricing empty len = %d, want 0", len(got))
	}

	// Seed a provider pricing JSON blob.
	if err := st.SetKV("pricing", "gh", `{"gpt-5.3-codex":{"input":2.0}}`); err != nil {
		t.Fatalf("SetKV pricing: %v", err)
	}
	got, err = st.UserPricing()
	if err != nil {
		t.Fatalf("UserPricing seeded: %v", err)
	}
	if got["gh"]["gpt-5.3-codex"]["input"] != 2.0 {
		t.Errorf("UserPricing = %v, want gh/gpt-5.3-codex/input=2.0", got)
	}

	// Corrupt JSON surfaces the offending provider key.
	if err := st.SetKV("pricing", "bad", `not json`); err != nil {
		t.Fatalf("SetKV bad: %v", err)
	}
	_, err = st.UserPricing()
	if err == nil {
		t.Fatal("UserPricing corrupt: expected error")
	}
	if !strings.Contains(err.Error(), "bad") {
		t.Errorf("UserPricing corrupt error = %v, want mention of provider key 'bad'", err)
	}
}

func TestKVDeleteAndClear(t *testing.T) {
	st := newTestStore(t)

	if err := st.SetKV("pricing", "openai", `{"gpt-4o":{"input":1}}`); err != nil {
		t.Fatalf("SetKV openai: %v", err)
	}
	if err := st.SetKV("pricing", "anthropic", `{"claude":{"input":2}}`); err != nil {
		t.Fatalf("SetKV anthropic: %v", err)
	}

	if err := st.DeleteKV("pricing", "openai"); err != nil {
		t.Fatalf("DeleteKV: %v", err)
	}
	v, err := st.GetKV("pricing", "openai")
	if err != nil {
		t.Fatalf("GetKV after delete: %v", err)
	}
	if v != "" {
		t.Errorf("GetKV after delete = %q, want empty", v)
	}
	v, err = st.GetKV("pricing", "anthropic")
	if err != nil {
		t.Fatalf("GetKV anthropic: %v", err)
	}
	if v == "" {
		t.Errorf("GetKV anthropic = empty after deleting openai")
	}

	if err := st.ClearKVScope("pricing"); err != nil {
		t.Fatalf("ClearKVScope: %v", err)
	}
	all, err := st.ListKV("pricing")
	if err != nil {
		t.Fatalf("ListKV after clear: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("ListKV after clear = %v, want empty", all)
	}
}
