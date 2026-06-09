package inference

import (
	"fmt"

	"github.com/bloodf/g0router/internal/providers/openai"
	"github.com/bloodf/g0router/internal/schemas"
)

// Router resolves a model name to a provider and API key.
// This is a minimal Phase 4 implementation; full virtual-key routing
// arrives in Phase 8.
type Router struct {
	openaiProvider *openai.Provider
}

// NewRouter creates a router with the OpenAI provider wired in.
func NewRouter() *Router {
	return &Router{
		openaiProvider: openai.NewProvider(),
	}
}

// Resolve returns the provider and key for a given model request.
// In Phase 4 this always returns the OpenAI provider with a key
// drawn from the G0ROUTER_OPENAI_KEY environment variable.
func (r *Router) Resolve(model string) (schemas.Provider, schemas.Key, error) {
	// TODO(phase-8): virtual-key routing, weighted selection, fallbacks.
	key := schemas.Key{
		ID:       "openai-default",
		Provider: "openai",
		Value:    "", // populated from env by caller for now
	}
	return r.openaiProvider, key, nil
}

// ResolveForModel is an alias for Resolve that accepts a full ChatRequest.
func (r *Router) ResolveForModel(req *schemas.ChatRequest) (schemas.Provider, schemas.Key, error) {
	return r.Resolve(req.Model)
}

// ErrNoProvider is returned when no provider can handle a request.
var ErrNoProvider = fmt.Errorf("no provider available for model")
