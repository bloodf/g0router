package catalog

import (
	"testing"
)

func TestLookupKnownProviders(t *testing.T) {
	wantBaseURL := map[string]string{
		"groq":         "https://api.groq.com/openai/v1/chat/completions",
		"deepseek":     "https://api.deepseek.com/chat/completions",
		"mistral":      "https://api.mistral.ai/v1/chat/completions",
		"cohere":       "https://api.cohere.ai/v1/chat/completions",
		"together":     "https://api.together.xyz/v1/chat/completions",
		"fireworks":    "https://api.fireworks.ai/inference/v1/chat/completions",
		"openrouter":   "https://openrouter.ai/api/v1/chat/completions",
		"xai":          "https://api.x.ai/v1/chat/completions",
		"perplexity":   "https://api.perplexity.ai/chat/completions",
		"ollama":       "https://ollama.com/api/chat",
		"ollama-local": "http://localhost:11434/api/chat",
	}
	known := []string{
		"groq", "deepseek", "mistral", "cohere",
		"together", "fireworks", "openrouter", "xai",
		"perplexity", "ollama", "ollama-local",
	}
	for _, name := range known {
		cfg, ok := Lookup(name)
		if !ok {
			t.Fatalf("Lookup(%q) returned ok=false", name)
		}
		if cfg.Name != name {
			t.Errorf("Lookup(%q).Name = %q, want %q", name, cfg.Name, name)
		}
		if got, want := cfg.BaseURL, wantBaseURL[name]; got != want {
			t.Errorf("Lookup(%q).BaseURL = %q, want %q", name, got, want)
		}
		if cfg.Format == "" {
			t.Errorf("Lookup(%q).Format is empty", name)
		}
	}
}

func TestLookupUnknown(t *testing.T) {
	_, ok := Lookup("nonexistent")
	if ok {
		t.Fatal("Lookup(\"nonexistent\") returned ok=true, want false")
	}
}

func TestOpenRouterHeaders(t *testing.T) {
	cfg, ok := Lookup("openrouter")
	if !ok {
		t.Fatal("Lookup(\"openrouter\") returned ok=false")
	}
	if got, want := cfg.Headers["HTTP-Referer"], "https://endpoint-proxy.local"; got != want {
		t.Errorf("openrouter HTTP-Referer = %q, want %q", got, want)
	}
	if got, want := cfg.Headers["X-Title"], "Endpoint Proxy"; got != want {
		t.Errorf("openrouter X-Title = %q, want %q", got, want)
	}
}

func TestOllamaConfig(t *testing.T) {
	for _, name := range []string{"ollama", "ollama-local"} {
		cfg, ok := Lookup(name)
		if !ok {
			t.Fatalf("Lookup(%q) returned ok=false", name)
		}
		if cfg.Format != "ollama" {
			t.Errorf("Lookup(%q).Format = %q, want %q", name, cfg.Format, "ollama")
		}
		if !cfg.NoAuth {
			t.Errorf("Lookup(%q).NoAuth = false, want true", name)
		}
	}
}

func TestResolveOllamaHost(t *testing.T) {
	// override trimmed
	if got := ResolveOllamaHost("  http://ollama.local:11434/  "); got != "http://ollama.local:11434" {
		t.Errorf("ResolveOllamaHost(trimmed) = %q", got)
	}
	// default
	if got := ResolveOllamaHost(""); got != "http://localhost:11434" {
		t.Errorf("ResolveOllamaHost(default) = %q, want %q", got, "http://localhost:11434")
	}
	// trailing slash stripped
	if got := ResolveOllamaHost("http://host:11434/"); got != "http://host:11434" {
		t.Errorf("ResolveOllamaHost(trailing slash) = %q, want %q", got, "http://host:11434")
	}
	// multiple trailing slashes stripped
	if got := ResolveOllamaHost("http://host:11434///"); got != "http://host:11434" {
		t.Errorf("ResolveOllamaHost(multiple slashes) = %q, want %q", got, "http://host:11434")
	}
}
