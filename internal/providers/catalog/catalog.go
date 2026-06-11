package catalog

import "strings"

// ProviderConfig holds the static configuration for a single provider.
type ProviderConfig struct {
	Name       string
	BaseURL    string
	Format     string
	Headers    map[string]string
	AuthHeader string
	NoAuth     bool
}

// Providers is the Go port of the reference PROVIDERS map
// (open-sse/config/providers.js:50-438). Only the 11 Stage-1 entries are
// included here; the rest are out of scope for this wave.
var Providers = map[string]ProviderConfig{
	"groq": {
		Name:    "groq",
		BaseURL: "https://api.groq.com/openai/v1/chat/completions",
		Format:  "openai",
	},
	"deepseek": {
		Name:    "deepseek",
		BaseURL: "https://api.deepseek.com/chat/completions",
		Format:  "openai",
	},
	"mistral": {
		Name:    "mistral",
		BaseURL: "https://api.mistral.ai/v1/chat/completions",
		Format:  "openai",
	},
	"cohere": {
		Name:    "cohere",
		BaseURL: "https://api.cohere.ai/v1/chat/completions",
		Format:  "openai",
	},
	"together": {
		Name:    "together",
		BaseURL: "https://api.together.xyz/v1/chat/completions",
		Format:  "openai",
	},
	"fireworks": {
		Name:    "fireworks",
		BaseURL: "https://api.fireworks.ai/inference/v1/chat/completions",
		Format:  "openai",
	},
	"openrouter": {
		Name:    "openrouter",
		BaseURL: "https://openrouter.ai/api/v1/chat/completions",
		Format:  "openai",
		Headers: map[string]string{
			"HTTP-Referer": "https://endpoint-proxy.local",
			"X-Title":      "Endpoint Proxy",
		},
	},
	"xai": {
		// providers.js:273-280 carries OAuth fields (clientId, tokenUrl, refreshUrl).
		// Stage-1 includes xai via its API-key (bearer) path only; OAuth is Wave-3.
		// Those fields are intentionally omitted from ProviderConfig here.
		Name:    "xai",
		BaseURL: "https://api.x.ai/v1/chat/completions",
		Format:  "openai",
	},
	"perplexity": {
		Name:    "perplexity",
		BaseURL: "https://api.perplexity.ai/chat/completions",
		Format:  "openai",
	},
	"ollama": {
		Name:    "ollama",
		BaseURL: "https://ollama.com/api/chat",
		Format:  "ollama",
		NoAuth:  true,
	},
	"ollama-local": {
		Name:    "ollama-local",
		BaseURL: "http://localhost:11434/api/chat",
		Format:  "ollama",
		NoAuth:  true,
	},
}

// Lookup returns the ProviderConfig for the given provider name.
func Lookup(provider string) (ProviderConfig, bool) {
	cfg, ok := Providers[provider]
	return cfg, ok
}

// ResolveOllamaHost is the Go port of resolveOllamaLocalHost
// (providers.js:442-445). It returns the trimmed override or the default
// host, with any trailing slash removed.
func ResolveOllamaHost(baseURLOverride string) string {
	raw := strings.TrimSpace(baseURLOverride)
	if raw == "" {
		raw = "http://localhost:11434"
	}
	return strings.TrimRight(raw, "/")
}
