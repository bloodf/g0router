package inference

import (
	"testing"

	"github.com/bloodf/g0router/internal/providers/anthropic"
	"github.com/bloodf/g0router/internal/providers/gemini"
	"github.com/bloodf/g0router/internal/providers/generic"
	"github.com/bloodf/g0router/internal/providers/ollama"
	"github.com/bloodf/g0router/internal/providers/openai"
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
