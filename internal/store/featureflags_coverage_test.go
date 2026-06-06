package store

import (
	"testing"
)

func TestListFeatureFlagsDBError(t *testing.T) {
	s := openTestStore(t)
	s.db.Close()
	_, err := s.ListFeatureFlags()
	if err == nil {
		t.Fatal("expected error from closed DB")
	}
}

func TestToggleFeatureFlagDBError(t *testing.T) {
	s := openTestStore(t)
	s.db.Close()
	err := s.ToggleFeatureFlag(1, true)
	if err == nil {
		t.Fatal("expected error from closed DB")
	}
}

func TestSeedFeatureFlagsDBError(t *testing.T) {
	s := openTestStore(t)
	s.db.Close()
	err := s.seedFeatureFlags()
	if err == nil {
		t.Fatal("expected error from closed DB")
	}
}
