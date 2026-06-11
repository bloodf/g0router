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

// providerForModel searches the static model catalogs in fixed precedence order
// and returns the first provider whose catalog contains the exact model ID.
// If no catalog match is found, it falls back to prefix-based routing for
// anthropic/gemini and defaults to openai.
func providerForModel(model string) (providerID string, ok bool) {
	// Catalog lookup takes precedence.
	for _, id := range providerPrecedence {
		models := catalog.ModelsFor(id)
		for _, m := range models {
			if m.ID == model {
				return id, true
			}
		}
	}

	// Fallback to prefix-based routing for existing providers.
	switch {
	case strings.HasPrefix(model, "anthropic/") || strings.HasPrefix(model, "claude-"):
		return "anthropic", true
	case strings.HasPrefix(model, "gemini/") || strings.HasPrefix(model, "gemini-"):
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
