package inference

import (
	"fmt"
	"strings"

	"github.com/bloodf/g0router/internal/providers/anthropic"
	"github.com/bloodf/g0router/internal/providers/catalog"
	"github.com/bloodf/g0router/internal/providers/gemini"
	"github.com/bloodf/g0router/internal/providers/generic"
	"github.com/bloodf/g0router/internal/providers/ollama"
	"github.com/bloodf/g0router/internal/providers/openai"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
)

// providerPrecedence is the fixed search order for catalog model lookup.
// Defensive deterministic tie-break for any colliding model IDs.
var providerPrecedence = []string{
	"deepseek",
	"groq",
	"mistral",
	"together",
	"fireworks",
	"cohere",
	"xai",
	"openrouter",
	"perplexity",
	"ollama",
	"ollama-local",
}

// providerForModel resolves a model string to a provider ID.
// It first checks for an explicit "provider/model" or "alias/model" prefix,
// then falls back to catalog lookup, name-prefix inference, and the legacy
// anthropic/gemini prefix heuristics.
func providerForModel(model string) (providerID string, ok bool) {
	// 1. Parse optional provider/alias prefix.
	if prefix, bare := ParseModelPrefix(model); prefix != "" {
		// 2a. Resolve known provider alias to provider ID.
		if id, ok := catalog.ResolveProviderAlias(prefix); ok {
			return id, true
		}
		// PR-485: passthrough lookup by providerId.
		if _, ok := catalog.Lookup(prefix); ok {
			return prefix, true
		}
		// 2b. Name-prefix inference on the bare model name.
		if id, ok := InferProvider(bare); ok {
			return id, true
		}
		// Legacy prefix-based routing for existing providers.
		switch {
		case strings.HasPrefix(bare, "claude-"):
			return "anthropic", true
		case strings.HasPrefix(bare, "gemini-"):
			return "gemini", true
		}
		return "openai", true
	}

	// 3. No prefix: catalog lookup takes precedence.
	for _, id := range providerPrecedence {
		models := catalog.ModelsFor(id)
		for _, m := range models {
			if m.ID == model {
				return id, true
			}
		}
	}

	// 4. Name-prefix inference fallback for bare models.
	if id, ok := InferProvider(model); ok {
		return id, true
	}

	// 5. Legacy prefix-based routing for existing providers.
	switch {
	case strings.HasPrefix(model, "claude-"):
		return "anthropic", true
	case strings.HasPrefix(model, "gemini-"):
		return "gemini", true
	default:
		return "openai", true
	}
}

// buildProvider creates a provider instance for the given provider ID.
func buildProvider(providerID string, reg *translation.Registry) (schemas.Provider, error) {
	switch providerID {
	case "openai":
		return openai.NewProvider(), nil
	case "anthropic":
		return anthropic.NewProvider(), nil
	case "gemini":
		return gemini.NewProvider(), nil
	case "ollama", "ollama-local":
		return ollama.New(providerID, reg)
	default:
		if _, ok := catalog.Lookup(providerID); !ok {
			return nil, fmt.Errorf("unknown provider %q", providerID)
		}
		return generic.New(providerID)
	}
}
