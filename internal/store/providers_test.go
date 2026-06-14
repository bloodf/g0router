package store

import (
	"path/filepath"
	"testing"
)

// openTestStore opens a fresh store in a temp dir for provider tests.
func openTestStore(t *testing.T) *Store {
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

// TestProviderPrefixAPITypeRoundTrip proves the additive prefix/api_type columns
// persist and round-trip through create/get/update (w7-platnodes, PAR-PLAT-014).
func TestProviderPrefixAPITypeRoundTrip(t *testing.T) {
	st := openTestStore(t)

	rec := &ProviderRecord{
		Name:    "My Node",
		Type:    "openai-compatible",
		BaseURL: "https://node.example.com/v1",
		Enabled: true,
		Prefix:  "mn",
		APIType: "openai",
	}
	if err := st.CreateProvider(rec); err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	got, err := st.GetProvider(rec.ID)
	if err != nil {
		t.Fatalf("GetProvider: %v", err)
	}
	if got.Prefix != "mn" || got.APIType != "openai" {
		t.Fatalf("create round-trip = {prefix:%q api_type:%q}, want {mn openai}", got.Prefix, got.APIType)
	}

	got.Prefix = "mn2"
	got.APIType = "anthropic"
	if err := st.UpdateProvider(got); err != nil {
		t.Fatalf("UpdateProvider: %v", err)
	}
	after, err := st.GetProvider(rec.ID)
	if err != nil {
		t.Fatalf("GetProvider after update: %v", err)
	}
	if after.Prefix != "mn2" || after.APIType != "anthropic" {
		t.Fatalf("update round-trip = {prefix:%q api_type:%q}, want {mn2 anthropic}", after.Prefix, after.APIType)
	}
}

// TestProviderDefaultsEmptyPrefixAPIType proves a plain (non-node) provider keeps
// the '' defaults — the additive columns do not disturb existing callers.
func TestProviderDefaultsEmptyPrefixAPIType(t *testing.T) {
	st := openTestStore(t)

	rec := &ProviderRecord{
		Name:    "OpenAI",
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		Enabled: true,
	}
	if err := st.CreateProvider(rec); err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}
	got, err := st.GetProvider(rec.ID)
	if err != nil {
		t.Fatalf("GetProvider: %v", err)
	}
	if got.Prefix != "" || got.APIType != "" {
		t.Fatalf("plain provider = {prefix:%q api_type:%q}, want both empty", got.Prefix, got.APIType)
	}
}
