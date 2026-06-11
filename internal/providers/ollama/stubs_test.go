package ollama

import (
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
)

func TestOllamaSatisfiesProviderInterface(t *testing.T) {
	// Compile-time assertion is in stubs.go: var _ schemas.Provider = (*Provider)(nil)
	// This test ensures the provider can be instantiated and identified.
	p, err := New("ollama", translation.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if p.GetProvider() != schemas.ModelProvider("ollama") {
		t.Errorf("GetProvider() = %v, want ollama", p.GetProvider())
	}
}

func TestOllamaEmbeddingNotImplemented(t *testing.T) {
	p, _ := New("ollama", translation.NewRegistry())
	_, perr := p.Embedding(&schemas.GatewayContext{}, schemas.Key{Value: ""}, &schemas.EmbeddingRequest{})
	if perr == nil {
		t.Fatal("expected 501 error, got nil")
	}
	if perr.StatusCode != 501 {
		t.Errorf("status code = %d, want 501", perr.StatusCode)
	}
	if perr.Type != "not_implemented" {
		t.Errorf("error type = %q, want not_implemented", perr.Type)
	}
}
