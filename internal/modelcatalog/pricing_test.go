package modelcatalog

import (
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

func TestCatalogLookupOpenAIPricing(t *testing.T) {
	catalog := NewCatalog()

	price, ok := catalog.Lookup(providers.ProviderOpenAI, "gpt-4o")
	if !ok {
		t.Fatal("expected gpt-4o pricing")
	}
	if price.InputPerMillionUSD != 2.50 {
		t.Fatalf("input price = %f, want 2.50", price.InputPerMillionUSD)
	}
	if price.CachedInputPerMillionUSD != 1.25 {
		t.Fatalf("cached input price = %f, want 1.25", price.CachedInputPerMillionUSD)
	}
	if price.OutputPerMillionUSD != 10.00 {
		t.Fatalf("output price = %f, want 10.00", price.OutputPerMillionUSD)
	}
}

func TestCatalogLookupAnthropicPricing(t *testing.T) {
	catalog := NewCatalog()

	price, ok := catalog.Lookup(providers.ProviderAnthropic, "claude-sonnet-4")
	if !ok {
		t.Fatal("expected claude-sonnet-4 pricing")
	}
	if price.InputPerMillionUSD != 3.00 {
		t.Fatalf("input price = %f, want 3.00", price.InputPerMillionUSD)
	}
	if price.CachedInputPerMillionUSD != 0.30 {
		t.Fatalf("cached input price = %f, want 0.30", price.CachedInputPerMillionUSD)
	}
	if price.OutputPerMillionUSD != 15.00 {
		t.Fatalf("output price = %f, want 15.00", price.OutputPerMillionUSD)
	}
}

func TestCatalogLookupUnknownModel(t *testing.T) {
	catalog := NewCatalog()

	_, ok := catalog.Lookup(providers.ProviderOpenAI, "missing-model")
	if ok {
		t.Fatal("expected missing model")
	}
}

func TestCatalogProviderForModel(t *testing.T) {
	catalog := NewCatalog()

	provider, ok := catalog.ProviderForModel("claude-sonnet-4")
	if !ok {
		t.Fatal("expected claude-sonnet-4 provider")
	}
	if provider != providers.ProviderAnthropic {
		t.Fatalf("provider = %q, want anthropic", provider)
	}

	provider, ok = catalog.ProviderForModel("gpt-4o")
	if !ok {
		t.Fatal("expected gpt-4o provider")
	}
	if provider != providers.ProviderOpenAI {
		t.Fatalf("provider = %q, want openai", provider)
	}

	_, ok = catalog.ProviderForModel("missing-model")
	if ok {
		t.Fatal("expected missing model provider")
	}
}

func TestCatalogProviderForModelUsesDeterministicLookupForSharedModelIDs(t *testing.T) {
	catalog := NewCatalog()

	provider, ok := catalog.ProviderForModel("gemini-2.5-flash")
	if !ok {
		t.Fatal("expected gemini-2.5-flash provider")
	}
	if provider != providers.ProviderGemini {
		t.Fatalf("provider = %q, want gemini", provider)
	}
}

func TestCatalogIncludesRepresentativeWave7IProviderCoverage(t *testing.T) {
	catalog := NewCatalog()

	tests := []struct {
		provider providers.ModelProvider
		model    string
		want     Pricing
	}{
		{providers.ProviderOpenAI, "gpt-4o-mini", Pricing{InputPerMillionUSD: 0.15, CachedInputPerMillionUSD: 0.075, OutputPerMillionUSD: 0.60}},
		{providers.ProviderAnthropic, "claude-3-5-haiku-20241022", Pricing{InputPerMillionUSD: 0.80, CachedInputPerMillionUSD: 0.08, OutputPerMillionUSD: 4.00}},
		{providers.ProviderGemini, "gemini-2.5-flash", Pricing{InputPerMillionUSD: 0.30, CachedInputPerMillionUSD: 0.03, OutputPerMillionUSD: 2.50}},
		{providers.ProviderCohere, "command-r-08-2024", Pricing{InputPerMillionUSD: 0.15, CachedInputPerMillionUSD: 0.15, OutputPerMillionUSD: 0.60}},
		{providers.ProviderGroq, "llama-3.3-70b-versatile", Pricing{InputPerMillionUSD: 0.59, CachedInputPerMillionUSD: 0.59, OutputPerMillionUSD: 0.79}},
		{providers.ProviderMistral, "mistral-small-latest", Pricing{InputPerMillionUSD: 0.10, CachedInputPerMillionUSD: 0.10, OutputPerMillionUSD: 0.30}},
		{providers.ProviderOpenRouter, "openai/gpt-4o-mini", Pricing{InputPerMillionUSD: 0.15, CachedInputPerMillionUSD: 0.15, OutputPerMillionUSD: 0.60}},
		{providers.ProviderDeepSeek, "deepseek-reasoner", Pricing{InputPerMillionUSD: 0.55, CachedInputPerMillionUSD: 0.14, OutputPerMillionUSD: 2.19}},
		{providers.ProviderPerplexity, "sonar-pro", Pricing{InputPerMillionUSD: 3.00, CachedInputPerMillionUSD: 3.00, OutputPerMillionUSD: 15.00}},
		{providers.ProviderMiniMax, "MiniMax-M3", Pricing{InputPerMillionUSD: 0.30, CachedInputPerMillionUSD: 0.06, OutputPerMillionUSD: 1.20}},
		{providers.ProviderQwen, "qwen3.6-plus", Pricing{InputPerMillionUSD: 0.50, CachedInputPerMillionUSD: 0.50, OutputPerMillionUSD: 3.00}},
		{providers.ProviderXAI, "grok-4.3", Pricing{InputPerMillionUSD: 1.25, CachedInputPerMillionUSD: 0.20, OutputPerMillionUSD: 2.50}},
		{providers.ProviderCerebras, "llama3.1-8b", Pricing{InputPerMillionUSD: 0.10, CachedInputPerMillionUSD: 0.10, OutputPerMillionUSD: 0.10}},
		{providers.ProviderNebius, "meta-llama/Llama-3.3-70B-Instruct", Pricing{InputPerMillionUSD: 0.13, CachedInputPerMillionUSD: 0.13, OutputPerMillionUSD: 0.40}},
		{providers.ProviderHuggingFace, "meta-llama/Llama-3.3-70B-Instruct:groq", Pricing{InputPerMillionUSD: 0.59, CachedInputPerMillionUSD: 0.59, OutputPerMillionUSD: 0.79}},
		{providers.ProviderFireworks, "accounts/fireworks/models/llama-v3p1-70b-instruct", Pricing{InputPerMillionUSD: 0.30, CachedInputPerMillionUSD: 0.15, OutputPerMillionUSD: 1.20}},
		{providers.ProviderTogether, "meta-llama/Llama-3.3-70B-Instruct-Turbo", Pricing{InputPerMillionUSD: 1.04, CachedInputPerMillionUSD: 1.04, OutputPerMillionUSD: 1.04}},
		{providers.ProviderOllama, "llama3.1:8b", Pricing{}},
		{providers.ProviderVertex, "gemini-2.5-flash", Pricing{InputPerMillionUSD: 0.30, CachedInputPerMillionUSD: 0.03, OutputPerMillionUSD: 2.50}},
	}

	for _, tt := range tests {
		t.Run(tt.provider.String()+"/"+tt.model, func(t *testing.T) {
			got, ok := catalog.Lookup(tt.provider, tt.model)
			if !ok {
				t.Fatalf("expected pricing for %s/%s", tt.provider, tt.model)
			}
			if got != tt.want {
				t.Fatalf("pricing = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestCatalogHostedModelsHaveExplicitNonZeroRates(t *testing.T) {
	catalog := NewCatalog()

	for _, provider := range catalog.providerNames() {
		for model, price := range catalog.Models(provider) {
			if provider == providers.ProviderOllama {
				if price != (Pricing{}) {
					t.Fatalf("ollama model %s pricing = %#v, want zero local pricing", model, price)
				}
				continue
			}

			if price.InputPerMillionUSD <= 0 {
				t.Fatalf("%s/%s input price = %f, want non-zero", provider, model, price.InputPerMillionUSD)
			}
			if price.CachedInputPerMillionUSD <= 0 {
				t.Fatalf("%s/%s cached input price = %f, want non-zero", provider, model, price.CachedInputPerMillionUSD)
			}
			if price.OutputPerMillionUSD <= 0 {
				t.Fatalf("%s/%s output price = %f, want non-zero", provider, model, price.OutputPerMillionUSD)
			}
		}
	}
}

func TestCatalogOmitsProvidersWithoutDefensibleEmbeddedPricing(t *testing.T) {
	catalog := NewCatalog()

	for _, provider := range []providers.ModelProvider{
		providers.ProviderAzure,
		providers.ProviderBedrock,
		providers.ProviderNVIDIA,
		providers.ProviderCursor,
		providers.ProviderGitHubCopilot,
	} {
		if models := catalog.Models(provider); len(models) != 0 {
			t.Fatalf("%s models len = %d, want 0", provider, len(models))
		}
	}
}

func TestCatalogModelsReturnsCopy(t *testing.T) {
	catalog := NewCatalog()

	models := catalog.Models(providers.ProviderOpenAI)
	if len(models) < 2 {
		t.Fatalf("models len = %d, want at least 2", len(models))
	}

	delete(models, "gpt-4o")
	if _, ok := catalog.Lookup(providers.ProviderOpenAI, "gpt-4o"); !ok {
		t.Fatal("catalog should not be mutated by caller")
	}

	groqModels := catalog.Models(providers.ProviderGroq)
	delete(groqModels, "llama-3.3-70b-versatile")
	if _, ok := catalog.Lookup(providers.ProviderGroq, "llama-3.3-70b-versatile"); !ok {
		t.Fatal("expanded provider catalog should not be mutated by caller")
	}
}
