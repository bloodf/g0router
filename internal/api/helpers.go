package api

import (
	"os"

	"github.com/bloodf/g0router/internal/schemas"
)

// resolveAPIKey returns the environment variable API key for the given provider.
func resolveAPIKey(provider schemas.Provider) string {
	switch provider.GetProvider() {
	case schemas.ProviderAnthropic:
		return os.Getenv("G0ROUTER_ANTHROPIC_KEY")
	case schemas.ProviderGemini:
		return os.Getenv("G0ROUTER_GEMINI_KEY")
	default:
		return os.Getenv("G0ROUTER_OPENAI_KEY")
	}
}
