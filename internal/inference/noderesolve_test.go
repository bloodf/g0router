package inference

import (
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
)

// fakeNodeResolver is a deterministic NodeResolver for the prefix-override tests.
type fakeNodeResolver struct {
	prefixes map[string]nodeRoute
}

type nodeRoute struct {
	providerID string
	baseURL    string
	apiType    string
}

func (f *fakeNodeResolver) ResolveByPrefix(prefix string) (providerID, baseURL, apiType string, ok bool) {
	r, ok := f.prefixes[prefix]
	if !ok {
		return "", "", "", false
	}
	return r.providerID, r.baseURL, r.apiType, true
}

// TestNodePrefixOverridesStaticResolution proves a model "mn/x" whose prefix mn
// is a registered node routes to the node's provider + base URL, short-circuiting
// static alias/catalog resolution (PAR-ROUTE-009/040).
func TestNodePrefixOverridesStaticResolution(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	r.SetNodeResolver(&fakeNodeResolver{prefixes: map[string]nodeRoute{
		"mn": {providerID: "node-123", baseURL: "https://node.example.com/v1", apiType: "openai"},
	}})

	p, key, err := r.Resolve("mn/some-model")
	if err != nil {
		t.Fatalf("Resolve(mn/some-model) error: %v", err)
	}
	if got := string(p.GetProvider()); got != "node-123" {
		t.Fatalf("provider = %q, want the node provider node-123", got)
	}
	if key.Provider != "node-123" {
		t.Fatalf("key.Provider = %q, want node-123", key.Provider)
	}
}

// TestNodePrefixBareModelFallsThrough proves the SAME bare model WITHOUT the node
// prefix falls through to the existing static resolution (unchanged).
func TestNodePrefixBareModelFallsThrough(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	r.SetNodeResolver(&fakeNodeResolver{prefixes: map[string]nodeRoute{
		"mn": {providerID: "node-123", baseURL: "https://node.example.com/v1", apiType: "openai"},
	}})

	// "gpt-4" has no node prefix → static resolution → openai.
	p, key, err := r.Resolve("gpt-4")
	if err != nil {
		t.Fatalf("Resolve(gpt-4) error: %v", err)
	}
	if p.GetProvider() != schemas.ProviderOpenAI {
		t.Fatalf("provider = %q, want openai (static fallthrough)", p.GetProvider())
	}
	if key.Provider != "openai" {
		t.Fatalf("key.Provider = %q, want openai", key.Provider)
	}
}

// TestNodePrefixWinsOverStaticAlias proves a prefix matching BOTH a node and a
// static alias resolves to the NODE (the node override semantics).
func TestNodePrefixWinsOverStaticAlias(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	// "anthropic" is a known static provider alias; register it as a node too.
	r.SetNodeResolver(&fakeNodeResolver{prefixes: map[string]nodeRoute{
		"anthropic": {providerID: "node-anthropic", baseURL: "https://node.example.com/v1", apiType: "openai"},
	}})

	p, _, err := r.Resolve("anthropic/claude-3-5-sonnet")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if got := string(p.GetProvider()); got != "node-anthropic" {
		t.Fatalf("provider = %q, want node-anthropic (node wins over static alias)", got)
	}
}

// TestNilNodeResolverByteIdentical proves a nil nodeResolver leaves resolution
// byte-identical to the pre-hook behavior (the additive-hook proof).
func TestNilNodeResolverByteIdentical(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	// nodeResolver intentionally nil.
	p, key, err := r.Resolve("anthropic/claude-3-5-sonnet")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if p.GetProvider() != schemas.ProviderAnthropic {
		t.Fatalf("provider = %q, want anthropic (unchanged)", p.GetProvider())
	}
	if key.Provider != "anthropic" {
		t.Fatalf("key.Provider = %q, want anthropic", key.Provider)
	}
}
