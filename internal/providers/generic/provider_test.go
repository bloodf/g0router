package generic

import (
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

func TestNewGenericKnownProvider(t *testing.T) {
	known := []string{"deepseek", "groq", "mistral", "cohere", "together", "fireworks", "openrouter", "xai", "perplexity"}
	for _, id := range known {
		p, err := New(id)
		if err != nil {
			t.Fatalf("New(%q) error: %v", id, err)
		}
		if p.GetProvider() != schemas.ModelProvider(id) {
			t.Errorf("GetProvider() = %q, want %q", p.GetProvider(), id)
		}
	}
}

func TestNewGenericUnknown(t *testing.T) {
	_, err := New("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
}

func TestNewGenericRejectsNonOpenAIFormat(t *testing.T) {
	_, err := New("ollama")
	if err == nil {
		t.Fatal("expected error for non-openai format provider, got nil")
	}
	_, err = New("ollama-local")
	if err == nil {
		t.Fatal("expected error for non-openai format provider, got nil")
	}
}

func TestGenericSetNetworkConfig(t *testing.T) {
	p, err := New("deepseek")
	if err != nil {
		t.Fatalf("New(deepseek) error: %v", err)
	}
	p.SetNetworkConfig(schemas.NetworkConfig{Timeout: 30, ProxyURL: "http://proxy"})
	if p.networkConfig.Timeout != 30 {
		t.Errorf("timeout = %d, want 30", p.networkConfig.Timeout)
	}
	if p.networkConfig.ProxyURL != "http://proxy" {
		t.Errorf("proxy = %q, want http://proxy", p.networkConfig.ProxyURL)
	}
}
