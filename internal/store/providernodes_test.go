package store

import (
	"errors"
	"testing"
)

// seedNode creates a provider-node row (a providers row with prefix/api_type) and
// returns it.
func seedNode(t *testing.T, st *Store, name, typ, prefix, apiType, baseURL string) *ProviderRecord {
	t.Helper()
	rec := &ProviderRecord{
		Name:    name,
		Type:    typ,
		BaseURL: baseURL,
		Enabled: true,
		Prefix:  prefix,
		APIType: apiType,
	}
	if err := st.CreateProvider(rec); err != nil {
		t.Fatalf("CreateProvider %q: %v", name, err)
	}
	return rec
}

// TestListProviderNodesFiltersNodeTypes proves ListProviderNodes returns only the
// three node types (openai-compatible / anthropic-compatible / custom-embedding)
// and excludes plain providers (w7-platnodes, PAR-PLAT-010).
func TestListProviderNodesFiltersNodeTypes(t *testing.T) {
	st := openTestStore(t)

	seedNode(t, st, "OpenAI", "openai", "", "", "https://api.openai.com/v1")
	seedNode(t, st, "Compat", "openai-compatible", "co", "openai", "https://compat.example.com/v1")
	seedNode(t, st, "Anthropic Node", "anthropic-compatible", "an", "anthropic", "https://anthropic.example.com")
	seedNode(t, st, "Embed", "custom-embedding", "em", "openai", "https://embed.example.com")

	nodes, err := st.ListProviderNodes()
	if err != nil {
		t.Fatalf("ListProviderNodes: %v", err)
	}
	if len(nodes) != 3 {
		t.Fatalf("ListProviderNodes len = %d, want 3 (excludes plain openai): %v", len(nodes), nodes)
	}
	for _, n := range nodes {
		if n.Type == "openai" {
			t.Fatalf("ListProviderNodes leaked a plain provider: %v", n)
		}
	}
}

// TestGetProviderNodeByPrefixHitMiss proves prefix lookup resolves the right node
// and returns ErrNotFound on a miss (w7-platnodes, PAR-ROUTE-009).
func TestGetProviderNodeByPrefixHitMiss(t *testing.T) {
	st := openTestStore(t)

	want := seedNode(t, st, "Compat", "openai-compatible", "co", "openai", "https://compat.example.com/v1")
	// A plain provider with an empty prefix must never be returned by prefix lookup.
	seedNode(t, st, "OpenAI", "openai", "", "", "https://api.openai.com/v1")

	got, err := st.GetProviderNodeByPrefix("co")
	if err != nil {
		t.Fatalf("GetProviderNodeByPrefix hit: %v", err)
	}
	if got.ID != want.ID || got.BaseURL != want.BaseURL || got.APIType != "openai" {
		t.Fatalf("GetProviderNodeByPrefix = %+v, want id=%s", got, want.ID)
	}

	if _, err := st.GetProviderNodeByPrefix("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetProviderNodeByPrefix miss err = %v, want ErrNotFound", err)
	}

	// An empty prefix never matches (a plain provider's '' must not resolve).
	if _, err := st.GetProviderNodeByPrefix(""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetProviderNodeByPrefix(\"\") err = %v, want ErrNotFound", err)
	}
}
