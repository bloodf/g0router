package inference

import (
	"fmt"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
)

// Router resolves a model name to a provider and API key.
// This is a minimal Phase 5 implementation; full virtual-key routing
// arrives in Phase 8.
type Router struct {
	registry  *translation.Registry
	providers map[string]schemas.Provider
}

// NewRouter creates a router with all providers wired in.
func NewRouter(reg *translation.Registry) *Router {
	return &Router{
		registry:  reg,
		providers: make(map[string]schemas.Provider),
	}
}

// Resolve returns the provider and key for a given model request.
// Phase 5: prefix-based routing (anthropic/, gemini/, or default openai).
func (r *Router) Resolve(model string) (schemas.Provider, schemas.Key, error) {
	// TODO(phase-8): virtual-key routing, weighted selection, fallbacks.
	providerID, ok := providerForModel(model)
	if !ok {
		return nil, schemas.Key{}, fmt.Errorf("no provider available for model %q", model)
	}

	p, err := r.providerForID(providerID)
	if err != nil {
		return nil, schemas.Key{}, fmt.Errorf("resolve %q: %w", model, err)
	}

	return p, schemas.Key{
		ID:       providerID + "-default",
		Provider: providerID,
		Value:    "",
	}, nil
}

// providerForID returns a cached provider instance for the given provider ID,
// creating it lazily on first access.
func (r *Router) providerForID(providerID string) (schemas.Provider, error) {
	if p, ok := r.providers[providerID]; ok {
		return p, nil
	}
	p, err := buildProvider(providerID, r.registry)
	if err != nil {
		return nil, fmt.Errorf("build provider %q: %w", providerID, err)
	}
	r.providers[providerID] = p
	return p, nil
}

// ResolveForModel is an alias for Resolve that accepts a full ChatRequest.
func (r *Router) ResolveForModel(req *schemas.ChatRequest) (schemas.Provider, schemas.Key, error) {
	return r.Resolve(req.Model)
}

// ErrNoProvider is returned when no provider can handle a request.
var ErrNoProvider = fmt.Errorf("no provider available for model")
