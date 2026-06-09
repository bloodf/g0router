package inference

import (
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

func TestResolveOpenAI(t *testing.T) {
	r := NewRouter()
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
	r := NewRouter()
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
	r := NewRouter()
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
	r := NewRouter()
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
	r := NewRouter()
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
