package inference

import (
	"testing"

	"github.com/bloodf/g0router/internal/providers/generic"
	"github.com/bloodf/g0router/internal/providers/ollama"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
)

func TestResolveOpenAI(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	p, key, err := r.Resolve("gpt-4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetProvider() != schemas.ProviderOpenAI {
		t.Errorf("provider = %q, want openai", p.GetProvider())
	}
	if key.Provider != "openai" {
		t.Errorf("key provider = %q, want openai", key.Provider)
	}
}

func TestResolveAnthropic(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	p, key, err := r.Resolve("anthropic/claude-3-5-sonnet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetProvider() != schemas.ProviderAnthropic {
		t.Errorf("provider = %q, want anthropic", p.GetProvider())
	}
	if key.Provider != "anthropic" {
		t.Errorf("key provider = %q, want anthropic", key.Provider)
	}
}

func TestResolveAnthropicByModelName(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	p, key, err := r.Resolve("claude-3-opus-20240229")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetProvider() != schemas.ProviderAnthropic {
		t.Errorf("provider = %q, want anthropic", p.GetProvider())
	}
	if key.Provider != "anthropic" {
		t.Errorf("key provider = %q, want anthropic", key.Provider)
	}
}

func TestResolveGemini(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	p, key, err := r.Resolve("gemini/gemini-1.5-pro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetProvider() != schemas.ProviderGemini {
		t.Errorf("provider = %q, want gemini", p.GetProvider())
	}
	if key.Provider != "gemini" {
		t.Errorf("key provider = %q, want gemini", key.Provider)
	}
}

func TestResolveGeminiByModelName(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	p, key, err := r.Resolve("gemini-1.5-flash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetProvider() != schemas.ProviderGemini {
		t.Errorf("provider = %q, want gemini", p.GetProvider())
	}
	if key.Provider != "gemini" {
		t.Errorf("key provider = %q, want gemini", key.Provider)
	}
}

func TestResolveDeepSeekRoutesToGeneric(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	p, key, err := r.Resolve("deepseek-chat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := p.(*generic.Provider); !ok {
		t.Fatalf("provider type = %T, want *generic.Provider", p)
	}
	if p.GetProvider() != schemas.ProviderDeepSeek {
		t.Errorf("GetProvider() = %q, want deepseek", p.GetProvider())
	}
	if key.Provider != "deepseek" {
		t.Errorf("key provider = %q, want deepseek", key.Provider)
	}
}

func TestResolveOllamaRoutesToOllama(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	p, key, err := r.Resolve("gpt-oss:120b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := p.(*ollama.Provider); !ok {
		t.Fatalf("provider type = %T, want *ollama.Provider", p)
	}
	if p.GetProvider() != schemas.ProviderOllama {
		t.Errorf("GetProvider() = %q, want ollama", p.GetProvider())
	}
	if key.Provider != "ollama" {
		t.Errorf("key provider = %q, want ollama", key.Provider)
	}
}

func TestResolveClaudePrefixUnchanged(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	p, key, err := r.Resolve("claude-3-5-sonnet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetProvider() != schemas.ProviderAnthropic {
		t.Errorf("provider = %q, want anthropic", p.GetProvider())
	}
	if key.Provider != "anthropic" {
		t.Errorf("key provider = %q, want anthropic", key.Provider)
	}
}

func TestResolveUnknownDefaultsOpenAI(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	p, key, err := r.Resolve("totally-unknown-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetProvider() != schemas.ProviderOpenAI {
		t.Errorf("provider = %q, want openai", p.GetProvider())
	}
	if key.Provider != "openai" {
		t.Errorf("key provider = %q, want openai", key.Provider)
	}
}
