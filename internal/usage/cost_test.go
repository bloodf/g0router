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
