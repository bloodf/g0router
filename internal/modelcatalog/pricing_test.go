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

func TestCatalogModelsReturnsCopy(t *testing.T) {
	catalog := NewCatalog()

	models := catalog.Models(providers.ProviderOpenAI)
	if len(models) != 2 {
		t.Fatalf("models len = %d, want 2", len(models))
	}

	delete(models, "gpt-4o")
	if _, ok := catalog.Lookup(providers.ProviderOpenAI, "gpt-4o"); !ok {
		t.Fatal("catalog should not be mutated by caller")
	}
}
