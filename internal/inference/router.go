package inference

import (
	"fmt"
	"strings"

	"github.com/bloodf/g0router/internal/providers/anthropic"
	"github.com/bloodf/g0router/internal/providers/gemini"
	"github.com/bloodf/g0router/internal/providers/openai"
	"github.com/bloodf/g0router/internal/schemas"
)

// Router resolves a model name to a provider and API key.
// This is a minimal Phase 5 implementation; full virtual-key routing
// arrives in Phase 8.
type Router struct {
	openaiProvider    *openai.Provider
	anthropicProvider *anthropic.Provider
	geminiProvider    *gemini.Provider
}

// NewRouter creates a router with all Phase 5 providers wired in.
func NewRouter() *Router {
	return &Router{
		openaiProvider:    openai.NewProvider(),
		anthropicProvider: anthropic.NewProvider(),
		geminiProvider:    gemini.NewProvider(),
	}
}

// Resolve returns the provider and key for a given model request.
// Phase 5: prefix-based routing (anthropic/, gemini/, or default openai).
func (r *Router) Resolve(model string) (schemas.Provider, schemas.Key, error) {
	// TODO(phase-8): virtual-key routing, weighted selection, fallbacks.
	switch {
	case strings.HasPrefix(model, "anthropic/") || strings.HasPrefix(model, "claude-"):
		return r.anthropicProvider, schemas.Key{
			ID:       "anthropic-default",
			Provider: "anthropic",
			Value:    "",
		}, nil
	case strings.HasPrefix(model, "gemini/") || strings.HasPrefix(model, "gemini-"):
		return r.geminiProvider, schemas.Key{
			ID:       "gemini-default",
			Provider: "gemini",
			Value:    "",
		}, nil
	default:
		return r.openaiProvider, schemas.Key{
			ID:       "openai-default",
			Provider: "openai",
			Value:    "",
		}, nil
	}
}

// ResolveForModel is an alias for Resolve that accepts a full ChatRequest.
func (r *Router) ResolveForModel(req *schemas.ChatRequest) (schemas.Provider, schemas.Key, error) {
	return r.Resolve(req.Model)
}

// ErrNoProvider is returned when no provider can handle a request.
var ErrNoProvider = fmt.Errorf("no provider available for model")
