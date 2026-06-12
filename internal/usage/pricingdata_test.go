package usage

import (
	"testing"
)

func TestPricingDataParity(t *testing.T) {
	if got := len(ModelPricing); got != 83 {
		t.Errorf("len(ModelPricing) = %d, want 83", got)
	}
	if got := len(PatternPricing); got != 49 {
		t.Errorf("len(PatternPricing) = %d, want 49", got)
	}
	if got := len(ProviderPricing); got != 1 {
		t.Errorf("len(ProviderPricing) = %d, want 1", got)
	}

	cases := map[string]Pricing{
		"claude-opus-4-6":            {Input: 5.00, Output: 25.00, Cached: 0.50, Reasoning: 25.00, CacheCreation: 6.25},
		"claude-sonnet-4-6":          {Input: 3.00, Output: 15.00, Cached: 0.30, Reasoning: 15.00, CacheCreation: 3.75},
		"gpt-4o-mini":                {Input: 0.15, Output: 0.60, Cached: 0.075, Reasoning: 0.90, CacheCreation: 0.15},
		"gpt-5.3-codex-spark":        {Input: 3.00, Output: 12.00, Cached: 0.30, Reasoning: 12.00, CacheCreation: 3.00},
		"gemini-2.5-flash-lite":      {Input: 0.15, Output: 1.25, Cached: 0.015, Reasoning: 1.875, CacheCreation: 0.15},
		"deepseek-chat":              {Input: 0.14, Output: 0.28, Cached: 0.0028, Reasoning: 0.28, CacheCreation: 0.14},
		"kimi-k2.5-thinking":         {Input: 1.80, Output: 7.20, Cached: 0.90, Reasoning: 10.80, CacheCreation: 1.80},
		"glm-4.6v":                   {Input: 0.75, Output: 3.00, Cached: 0.375, Reasoning: 4.50, CacheCreation: 0.75},
		"MiniMax-M3":                 {Input: 0.30, Output: 1.20, Cached: 0.06, Reasoning: 1.80, CacheCreation: 0.30},
		"auto":                       {Input: 2.00, Output: 8.00, Cached: 1.00, Reasoning: 12.00, CacheCreation: 2.00},
	}
	for model, want := range cases {
		got, ok := ModelPricing[model]
		if !ok {
			t.Fatalf("ModelPricing missing %q", model)
		}
		assertPricing(t, "ModelPricing["+model+"]", got, want)
	}

	gh, ok := ProviderPricing["gh"]
	if !ok {
		t.Fatal("ProviderPricing missing gh")
	}
	wantGH := Pricing{Input: 1.75, Output: 14.00, Cached: 0.175, Reasoning: 14.00, CacheCreation: 1.75}
	gotGH, ok := gh["gpt-5.3-codex"]
	if !ok {
		t.Fatal("ProviderPricing[gh] missing gpt-5.3-codex")
	}
	assertPricing(t, "ProviderPricing[gh][gpt-5.3-codex]", gotGH, wantGH)

	if PatternPricing[0].Pattern != "*-codex-xhigh" {
		t.Errorf("PatternPricing[0].Pattern = %q, want *-codex-xhigh", PatternPricing[0].Pattern)
	}
	wantFirst := Pricing{Input: 10.00, Output: 40.00, Cached: 5.00, Reasoning: 60.00, CacheCreation: 10.00}
	assertPricing(t, "PatternPricing[0]", PatternPricing[0].Pricing, wantFirst)

	if PatternPricing[48].Pattern != "grok-*" {
		t.Errorf("PatternPricing[48].Pattern = %q, want grok-*", PatternPricing[48].Pattern)
	}
	wantLast := Pricing{Input: 0.50, Output: 2.00, Cached: 0.25, Reasoning: 3.00, CacheCreation: 0.50}
	assertPricing(t, "PatternPricing[48]", PatternPricing[48].Pricing, wantLast)
}

func assertPricing(t *testing.T, label string, got, want Pricing) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %+v, want %+v", label, got, want)
	}
}
