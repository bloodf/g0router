package generic

import (
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

// TestGenericSatisfiesProviderInterface verifies the compile-time assertion
// that Provider implements schemas.Provider.
func TestGenericSatisfiesProviderInterface(t *testing.T) {
	p, err := New("deepseek")
	if err != nil {
		t.Fatalf("New(deepseek) error: %v", err)
	}
	var _ schemas.Provider = p
}

func TestGenericEmbeddingNotImplemented(t *testing.T) {
	p, err := New("deepseek")
	if err != nil {
		t.Fatalf("New(deepseek) error: %v", err)
	}
	_, perr := p.Embedding(&schemas.GatewayContext{}, schemas.Key{}, nil)
	if perr == nil {
		t.Fatal("expected error for Embedding, got nil")
	}
	if perr.Type != "not_implemented" {
		t.Errorf("type = %q, want not_implemented", perr.Type)
	}
	if perr.StatusCode != 501 {
		t.Errorf("status = %d, want 501", perr.StatusCode)
	}
}
