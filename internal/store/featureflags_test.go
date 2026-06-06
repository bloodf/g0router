package store

import (
	"errors"
	"testing"
)

func TestFeatureFlagsSeeded(t *testing.T) {
	s := openTestStore(t)

	flags, err := s.ListFeatureFlags()
	if err != nil {
		t.Fatalf("ListFeatureFlags: %v", err)
	}
	if len(flags) != 5 {
		t.Fatalf("len = %d, want 5", len(flags))
	}

	want := map[string]bool{
		"semantic_cache":  false,
		"guardrails":      false,
		"pii_redaction":   false,
		"websocket_chat":  false,
		"mitm_proxy":      false,
	}
	got := make(map[string]bool)
	for _, f := range flags {
		got[f.Key] = f.Enabled
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("flag %q = %v, want %v", k, got[k], v)
		}
	}
}

func TestFeatureFlagsGetByID(t *testing.T) {
	s := openTestStore(t)

	flags, err := s.ListFeatureFlags()
	if err != nil {
		t.Fatalf("ListFeatureFlags: %v", err)
	}

	got, err := s.GetFeatureFlag(flags[0].ID)
	if err != nil {
		t.Fatalf("GetFeatureFlag: %v", err)
	}
	if got.Key != flags[0].Key {
		t.Fatalf("key = %q, want %q", got.Key, flags[0].Key)
	}
}

func TestFeatureFlagsGetByKey(t *testing.T) {
	s := openTestStore(t)

	got, err := s.GetFeatureFlagByKey("semantic_cache")
	if err != nil {
		t.Fatalf("GetFeatureFlagByKey: %v", err)
	}
	if got.Key != "semantic_cache" {
		t.Fatalf("key = %q", got.Key)
	}
	if got.Enabled {
		t.Fatal("expected disabled")
	}
}

func TestFeatureFlagsToggle(t *testing.T) {
	s := openTestStore(t)

	flag, err := s.GetFeatureFlagByKey("semantic_cache")
	if err != nil {
		t.Fatalf("GetFeatureFlagByKey: %v", err)
	}

	if err := s.ToggleFeatureFlag(flag.ID, true); err != nil {
		t.Fatalf("ToggleFeatureFlag: %v", err)
	}

	updated, err := s.GetFeatureFlag(flag.ID)
	if err != nil {
		t.Fatalf("GetFeatureFlag: %v", err)
	}
	if !updated.Enabled {
		t.Fatal("expected enabled")
	}
}

func TestFeatureFlagsGetNotFound(t *testing.T) {
	s := openTestStore(t)

	_, err := s.GetFeatureFlag(9999)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestFeatureFlagsGetByKeyNotFound(t *testing.T) {
	s := openTestStore(t)

	_, err := s.GetFeatureFlagByKey("unknown_flag")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestFeatureFlagsToggleNotFound(t *testing.T) {
	s := openTestStore(t)

	err := s.ToggleFeatureFlag(9999, true)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
