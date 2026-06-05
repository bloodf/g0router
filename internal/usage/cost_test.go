package usage

import (
	"testing"

	"github.com/bloodf/g0router/internal/modelcatalog"
	"github.com/bloodf/g0router/internal/providers"
)

func TestCalculateCostUSDUsesExtractedUsageAndPricing(t *testing.T) {
	resp := providers.ChatResponse{
		Model: "gpt-4o",
		Usage: &providers.Usage{
			PromptTokens:     1000,
			CompletionTokens: 500,
			TotalTokens:      1500,
			PromptTokensDetails: &providers.PromptTokensDetails{
				CachedTokens: 200,
			},
		},
	}
	extracted, ok := FromChatResponse(resp)
	if !ok {
		t.Fatal("expected usage")
	}

	got, err := CalculateCostUSD(modelcatalog.NewCatalog(), providers.ProviderOpenAI, resp.Model, &extracted)
	if err != nil {
		t.Fatalf("CalculateCostUSD: %v", err)
	}

	want := 0.00725
	if got != want {
		t.Fatalf("cost = %f, want %f", got, want)
	}
}

func TestCalculateCostUSDWithoutUsageReturnsZero(t *testing.T) {
	got, err := CalculateCostUSD(modelcatalog.NewCatalog(), providers.ProviderOpenAI, "gpt-4o", nil)
	if err != nil {
		t.Fatalf("CalculateCostUSD: %v", err)
	}
	if got != 0 {
		t.Fatalf("cost = %f, want 0", got)
	}
}

func TestCalculateCostUSDWithPricingOverridesUsesOverride(t *testing.T) {
	usage := Usage{
		InputTokens:  1000,
		OutputTokens: 500,
		TotalTokens:  1500,
	}
	overrides := fakePricingOverrides{
		providers.ProviderOpenAI.String() + "/gpt-4o": {
			InputCostPerToken:  0.00001,
			OutputCostPerToken: 0.00002,
		},
	}

	got, err := CalculateCostUSDWithOverrides(modelcatalog.NewCatalog(), overrides, providers.ProviderOpenAI, "gpt-4o", &usage)
	if err != nil {
		t.Fatalf("CalculateCostUSDWithOverrides: %v", err)
	}

	want := 0.02
	if got != want {
		t.Fatalf("cost = %f, want %f", got, want)
	}
}

func TestCalculateCostUSDWithPricingOverridesFallsBackToCatalog(t *testing.T) {
	usage := Usage{
		InputTokens:  1000,
		OutputTokens: 500,
		TotalTokens:  1500,
	}

	got, err := CalculateCostUSDWithOverrides(modelcatalog.NewCatalog(), fakePricingOverrides{}, providers.ProviderOpenAI, "gpt-4o", &usage)
	if err != nil {
		t.Fatalf("CalculateCostUSDWithOverrides: %v", err)
	}

	want := 0.0075
	if got != want {
		t.Fatalf("cost = %f, want %f", got, want)
	}
}

func TestCalculateCostUSDMissingPricingReturnsError(t *testing.T) {
	usage := Usage{
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
	}

	got, err := CalculateCostUSD(modelcatalog.NewCatalog(), providers.ProviderOpenAI, "missing-model", &usage)
	if err == nil {
		t.Fatal("expected error")
	}
	if got != 0 {
		t.Fatalf("cost = %f, want 0", got)
	}
}

// TestCalculateCostUSDCacheReadExceedsInput guards against providers that report
// InputTokens exclusive of cache reads (underflow => negative cost).
func TestCalculateCostUSDCacheReadExceedsInput(t *testing.T) {
	// CacheReadTokens > InputTokens: simulates a provider reporting exclusive counts.
	u := Usage{
		InputTokens:     50,
		OutputTokens:    100,
		TotalTokens:     150,
		CacheReadTokens: 200, // exceeds InputTokens intentionally
	}
	got, err := CalculateCostUSD(modelcatalog.NewCatalog(), providers.ProviderOpenAI, "gpt-4o", &u)
	if err != nil {
		t.Fatalf("CalculateCostUSD: %v", err)
	}
	if got < 0 {
		t.Fatalf("cost = %f, must be >= 0", got)
	}
	// With clamped inputTokens=0: cost = cachedInputCost + outputCost only.
	catalog := modelcatalog.NewCatalog()
	price, _ := catalog.Lookup(providers.ProviderOpenAI, "gpt-4o")
	want := float64(200)*price.CachedInputPerMillionUSD/1_000_000 + float64(100)*price.OutputPerMillionUSD/1_000_000
	if got != want {
		t.Fatalf("cost = %f, want %f", got, want)
	}
}

type fakePricingOverrides map[string]PricingOverride

func (f fakePricingOverrides) PricingOverride(provider, model string) (PricingOverride, bool, error) {
	override, ok := f[provider+"/"+model]
	return override, ok, nil
}
