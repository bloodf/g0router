package inference

import (
	"testing"

	"github.com/bloodf/g0router/internal/providers/anthropic"
	"github.com/bloodf/g0router/internal/providers/commandcode"
	"github.com/bloodf/g0router/internal/providers/gemini"
	"github.com/bloodf/g0router/internal/providers/generic"
	"github.com/bloodf/g0router/internal/providers/ollama"
	"github.com/bloodf/g0router/internal/providers/openai"
	"github.com/bloodf/g0router/internal/providers/urltemplate"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
)

func TestProviderForModelCatalog(t *testing.T) {
	tests := []struct {
		model      string
		wantProvID string
	}{
		{"deepseek-chat", "deepseek"},
		{"grok-4", "xai"},
		{"sonar", "perplexity"},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got, ok := providerForModel(tt.model)
			if !ok {
				t.Fatalf("providerForModel(%q) = _, false, want true", tt.model)
			}
			if got != tt.wantProvID {
				t.Errorf("providerForModel(%q) = %q, want %q", tt.model, got, tt.wantProvID)
			}
		})
	}
}

func TestProviderForModelPrefix(t *testing.T) {
	tests := []struct {
		model      string
		wantProvID string
	}{
		{"claude-3-opus-20240229", "anthropic"},
		{"gemini-1.5-pro", "gemini"},
		{"anthropic/claude-3-5-sonnet", "anthropic"},
		{"gemini/gemini-1.5-pro", "gemini"},
		{"some-unknown-model", "openai"},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got, ok := providerForModel(tt.model)
			if !ok {
				t.Fatalf("providerForModel(%q) = _, false, want true", tt.model)
			}
			if got != tt.wantProvID {
				t.Errorf("providerForModel(%q) = %q, want %q", tt.model, got, tt.wantProvID)
			}
		})
	}
}

func TestBuildProviderGeneric(t *testing.T) {
	reg := translation.NewRegistry()
	p, err := buildProvider("deepseek", reg)
	if err != nil {
		t.Fatalf("buildProvider(deepseek) error: %v", err)
	}
	if _, ok := p.(*generic.Provider); !ok {
		t.Fatalf("buildProvider(deepseek) type = %T, want *generic.Provider", p)
	}
	if p.GetProvider() != schemas.ProviderDeepSeek {
		t.Errorf("GetProvider() = %q, want deepseek", p.GetProvider())
	}
}

func TestBuildProviderOllama(t *testing.T) {
	reg := translation.NewRegistry()
	p, err := buildProvider("ollama", reg)
	if err != nil {
		t.Fatalf("buildProvider(ollama) error: %v", err)
	}
	if _, ok := p.(*ollama.Provider); !ok {
		t.Fatalf("buildProvider(ollama) type = %T, want *ollama.Provider", p)
	}
	if p.GetProvider() != schemas.ProviderOllama {
		t.Errorf("GetProvider() = %q, want ollama", p.GetProvider())
	}
}

func TestBuildProviderExisting(t *testing.T) {
	reg := translation.NewRegistry()

	p, err := buildProvider("openai", reg)
	if err != nil {
		t.Fatalf("buildProvider(openai) error: %v", err)
	}
	if _, ok := p.(*openai.Provider); !ok {
		t.Fatalf("buildProvider(openai) type = %T, want *openai.Provider", p)
	}

	p, err = buildProvider("anthropic", reg)
	if err != nil {
		t.Fatalf("buildProvider(anthropic) error: %v", err)
	}
	if _, ok := p.(*anthropic.Provider); !ok {
		t.Fatalf("buildProvider(anthropic) type = %T, want *anthropic.Provider", p)
	}

	p, err = buildProvider("gemini", reg)
	if err != nil {
		t.Fatalf("buildProvider(gemini) error: %v", err)
	}
	if _, ok := p.(*gemini.Provider); !ok {
		t.Fatalf("buildProvider(gemini) type = %T, want *gemini.Provider", p)
	}
}

// TestClaudeFormatProvidersDispatch (w7-prov-special-a) verifies the additive
// factory arm dispatching format:"claude" catalog providers to the anthropic
// adapter, constructed with the catalog base URL (not the hardcoded
// api.anthropic.com).
func TestClaudeFormatProvidersDispatch(t *testing.T) {
	reg := translation.NewRegistry()
	for _, id := range []string{"glm", "kimi", "minimax", "minimax-cn"} {
		t.Run(id, func(t *testing.T) {
			p, err := buildProvider(id, reg)
			if err != nil {
				t.Fatalf("buildProvider(%q) error: %v", id, err)
			}
			if _, ok := p.(*anthropic.Provider); !ok {
				t.Fatalf("buildProvider(%q) type = %T, want *anthropic.Provider", id, p)
			}
			if p.GetProvider() != schemas.ModelProvider(id) {
				t.Errorf("buildProvider(%q).GetProvider() = %q, want %q", id, p.GetProvider(), id)
			}
		})
	}
}

// TestCommandCodeDispatch (w7-prov-special-a) verifies the additive factory arm
// dispatching the commandcode custom-JSON provider to its adapter.
func TestCommandCodeDispatch(t *testing.T) {
	reg := translation.NewRegistry()
	p, err := buildProvider("commandcode", reg)
	if err != nil {
		t.Fatalf("buildProvider(commandcode) error: %v", err)
	}
	if _, ok := p.(*commandcode.Provider); !ok {
		t.Fatalf("buildProvider(commandcode) type = %T, want *commandcode.Provider", p)
	}
	if p.GetProvider() != schemas.ModelProvider("commandcode") {
		t.Errorf("GetProvider() = %q, want commandcode", p.GetProvider())
	}
}

// TestURLTemplateDispatch (w7-prov-special-a) verifies the additive factory arm
// dispatching the URL-template/build openai providers (cloudflare-ai, azure,
// xiaomi-tokenplan) to the urltemplate adapter. qoder is DEFERRED (ESC-A3:
// opaque COSY signing).
func TestURLTemplateDispatch(t *testing.T) {
	reg := translation.NewRegistry()
	for _, id := range []string{"cloudflare-ai", "azure", "xiaomi-tokenplan"} {
		t.Run(id, func(t *testing.T) {
			p, err := buildProvider(id, reg)
			if err != nil {
				t.Fatalf("buildProvider(%q) error: %v", id, err)
			}
			if _, ok := p.(*urltemplate.Provider); !ok {
				t.Fatalf("buildProvider(%q) type = %T, want *urltemplate.Provider", id, p)
			}
			if p.GetProvider() != schemas.ModelProvider(id) {
				t.Errorf("buildProvider(%q).GetProvider() = %q, want %q", id, p.GetProvider(), id)
			}
		})
	}
}

// TestVertexDispatch (w7-prov-special-a) verifies the additive factory arm
// dispatching vertex (partner-openai path) to the urltemplate adapter. The
// native gemini-on-vertex format is deferred (ESC-A1).
func TestVertexDispatch(t *testing.T) {
	reg := translation.NewRegistry()
	p, err := buildProvider("vertex", reg)
	if err != nil {
		t.Fatalf("buildProvider(vertex) error: %v", err)
	}
	if _, ok := p.(*urltemplate.Provider); !ok {
		t.Fatalf("buildProvider(vertex) type = %T, want *urltemplate.Provider", p)
	}
	if p.GetProvider() != schemas.ModelProvider("vertex") {
		t.Errorf("GetProvider() = %q, want vertex", p.GetProvider())
	}
}

func TestProviderForModelDeterministic(t *testing.T) {
	// Run multiple times and assert stable result.
	for i := 0; i < 5; i++ {
		got, ok := providerForModel("deepseek-chat")
		if !ok {
			t.Fatalf("iteration %d: providerForModel(deepseek-chat) = _, false", i)
		}
		if got != "deepseek" {
			t.Fatalf("iteration %d: providerForModel(deepseek-chat) = %q, want deepseek", i, got)
		}
	}
}

func TestBuildProviderUnknownErrors(t *testing.T) {
	reg := translation.NewRegistry()
	p, err := buildProvider("not-a-real-provider", reg)
	if err == nil {
		t.Fatalf("buildProvider(not-a-real-provider) error = nil, want error; provider = %T", p)
	}
	if p != nil {
		t.Fatalf("buildProvider(not-a-real-provider) provider = %v, want nil", p)
	}
}

func TestPassthroughLookupByProviderID(t *testing.T) {
	// "ollama" is not in ProviderAliases, but it is a valid provider ID.
	got, ok := providerForModel("ollama/llama3")
	if !ok {
		t.Fatalf("providerForModel(ollama/llama3) = _, false, want true")
	}
	if got != "ollama" {
		t.Errorf("providerForModel(ollama/llama3) = %q, want ollama", got)
	}
}

// TestProviderAliasNonBuildableIdFallsThrough verifies that a model like "cc/claude-3-5-sonnet"
// (alias "cc" → "claude", which is not a Stage-1 provider) falls through to the legacy heuristic
// and routes to "anthropic" via the "claude-" prefix.
func TestProviderAliasNonBuildableIdFallsThrough(t *testing.T) {
	id, ok := providerForModel("cc/claude-3-5-sonnet")
	if !ok {
		t.Fatal("expected a provider, got none")
	}
	if id != "anthropic" {
		t.Errorf("providerForModel(cc/claude-3-5-sonnet) = %q, want anthropic", id)
	}
}

func TestFactoryCatalogPrecedenceUnchanged(t *testing.T) {
	// Catalog lookup must still win over prefix-based inference.
	got, ok := providerForModel("deepseek-chat")
	if !ok || got != "deepseek" {
		t.Errorf("providerForModel(deepseek-chat) = (%q, %v), want (deepseek, true)", got, ok)
	}

	// Existing prefix-based fallbacks for anthropic/gemini remain effective.
	for _, tc := range []struct{ model, want string }{
		{"claude-3-opus-20240229", "anthropic"},
		{"gemini-1.5-pro", "gemini"},
		{"anthropic/claude-3-5-sonnet", "anthropic"},
		{"gemini/gemini-1.5-pro", "gemini"},
	} {
		got, ok := providerForModel(tc.model)
		if !ok {
			t.Errorf("providerForModel(%q) = _, false, want true", tc.model)
			continue
		}
		if got != tc.want {
			t.Errorf("providerForModel(%q) = %q, want %q", tc.model, got, tc.want)
		}
	}
}
