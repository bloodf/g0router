package store

import (
	"strings"
	"testing"
)

func TestCreateAPIKeyReturnsRawKey(t *testing.T) {
	s := openTestStore(t)

	key, raw, err := s.CreateAPIKey("default", "test-secret")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	if key.ID == "" {
		t.Error("ID should be set")
	}
	if key.Name != "default" {
		t.Errorf("Name = %q, want default", key.Name)
	}
	if !strings.HasPrefix(raw, "g0r_") {
		t.Fatalf("raw key = %q, want g0r_ prefix", raw)
	}
	if key.Prefix != raw[:8] {
		t.Errorf("Prefix = %q, want %q", key.Prefix, raw[:8])
	}
	if !key.IsActive {
		t.Error("IsActive should be true")
	}
	if key.CreatedAt == "" {
		t.Error("CreatedAt should be set")
	}
}

func TestValidateAPIKeyCorrect(t *testing.T) {
	s := openTestStore(t)

	created, raw, err := s.CreateAPIKey("default", "test-secret")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	got, ok, err := s.ValidateAPIKey(raw, "test-secret")
	if err != nil {
		t.Fatalf("ValidateAPIKey: %v", err)
	}
	if !ok {
		t.Fatal("ValidateAPIKey should return ok")
	}
	if got == nil {
		t.Fatal("ValidateAPIKey returned nil key")
	}
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
	if got.Name != "default" {
		t.Errorf("Name = %q, want default", got.Name)
	}
	if got.LastUsedAt == nil || *got.LastUsedAt == "" {
		t.Fatal("LastUsedAt should be updated")
	}
}

func TestValidateAPIKeyWrong(t *testing.T) {
	s := openTestStore(t)

	if _, _, err := s.CreateAPIKey("default", "test-secret"); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	got, ok, err := s.ValidateAPIKey("g0r_wrong", "test-secret")
	if err != nil {
		t.Fatalf("ValidateAPIKey: %v", err)
	}
	if ok {
		t.Fatal("ValidateAPIKey should not return ok")
	}
	if got != nil {
		t.Fatalf("key = %+v, want nil", got)
	}
}

func TestValidateAPIKeyAfterDelete(t *testing.T) {
	s := openTestStore(t)

	key, raw, err := s.CreateAPIKey("default", "test-secret")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if err := s.DeleteAPIKey(key.ID); err != nil {
		t.Fatalf("DeleteAPIKey: %v", err)
	}

	got, ok, err := s.ValidateAPIKey(raw, "test-secret")
	if err != nil {
		t.Fatalf("ValidateAPIKey: %v", err)
	}
	if ok {
		t.Fatal("ValidateAPIKey should not return ok")
	}
	if got != nil {
		t.Fatalf("key = %+v, want nil", got)
	}
}

func TestListAPIKeys(t *testing.T) {
	s := openTestStore(t)

	created, raw, err := s.CreateAPIKey("default", "test-secret")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	keys, err := s.ListAPIKeys()
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("len(keys) = %d, want 1", len(keys))
	}

	got := keys[0]
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
	if got.Name != "default" {
		t.Errorf("Name = %q, want default", got.Name)
	}
	if got.Prefix != raw[:8] {
		t.Errorf("Prefix = %q, want %q", got.Prefix, raw[:8])
	}
	if strings.Contains(got.Prefix, raw[8:]) {
		t.Error("listed key should not expose raw key material")
	}
}

func TestCreateAPIKeyDuplicateName(t *testing.T) {
	s := openTestStore(t)

	if _, _, err := s.CreateAPIKey("default", "test-secret"); err != nil {
		t.Fatalf("first CreateAPIKey: %v", err)
	}
	if _, _, err := s.CreateAPIKey("default", "test-secret"); err == nil {
		t.Fatal("second CreateAPIKey should fail")
	}
}

func TestRegenerateAPIKey(t *testing.T) {
	s := openTestStore(t)

	key, raw, err := s.CreateAPIKey("regen-test", "test-secret")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	updated, newRaw, err := s.RegenerateAPIKey(key.ID, "test-secret")
	if err != nil {
		t.Fatalf("RegenerateAPIKey: %v", err)
	}
	if updated == nil {
		t.Fatal("RegenerateAPIKey returned nil key")
	}
	if updated.ID != key.ID {
		t.Errorf("ID = %q, want %q", updated.ID, key.ID)
	}
	if updated.Name != key.Name {
		t.Errorf("Name = %q, want %q", updated.Name, key.Name)
	}
	if !strings.HasPrefix(newRaw, "g0r_") {
		t.Fatalf("new raw key = %q, want g0r_ prefix", newRaw)
	}
	if newRaw == raw {
		t.Fatal("new raw key should differ from old raw key")
	}
	if updated.Prefix != newRaw[:8] {
		t.Errorf("Prefix = %q, want %q", updated.Prefix, newRaw[:8])
	}

	// Old key should no longer validate
	_, ok, err := s.ValidateAPIKey(raw, "test-secret")
	if err != nil {
		t.Fatalf("ValidateAPIKey old: %v", err)
	}
	if ok {
		t.Fatal("old raw key should no longer validate")
	}

	// New key should validate
	validated, ok, err := s.ValidateAPIKey(newRaw, "test-secret")
	if err != nil {
		t.Fatalf("ValidateAPIKey new: %v", err)
	}
	if !ok {
		t.Fatal("new raw key should validate")
	}
	if validated.ID != key.ID {
		t.Errorf("validated ID = %q, want %q", validated.ID, key.ID)
	}
}

func TestRegenerateAPIKeyNotFound(t *testing.T) {
	s := openTestStore(t)

	_, _, err := s.RegenerateAPIKey("nonexistent-id", "test-secret")
	if err == nil {
		t.Fatal("RegenerateAPIKey for missing key should fail")
	}
}
