package ollama

import (
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
)

func TestNewOllamaProvider(t *testing.T) {
	reg := translation.NewRegistry()

	p, err := New("ollama", reg)
	if err != nil {
		t.Fatalf("New(ollama) error: %v", err)
	}
	if p.GetProvider() != schemas.ModelProvider("ollama") {
		t.Errorf("GetProvider() = %v, want ollama", p.GetProvider())
	}

	p2, err := New("ollama-local", reg)
	if err != nil {
		t.Fatalf("New(ollama-local) error: %v", err)
	}
	if p2.GetProvider() != schemas.ModelProvider("ollama-local") {
		t.Errorf("GetProvider() = %v, want ollama-local", p2.GetProvider())
	}
}

func TestNewOllamaRejectsNonOllama(t *testing.T) {
	reg := translation.NewRegistry()

	_, err := New("deepseek", reg)
	if err == nil {
		t.Fatal("expected error for deepseek id, got nil")
	}
}
