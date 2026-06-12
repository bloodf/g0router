package usage

import (
	"math"
	"testing"
)

func TestCalculateCost(t *testing.T) {
	sonnet := ModelPricing["claude-sonnet-4-6"]

	// claude-sonnet-4-6: 1M in, 200k cached, 100k out, 50k reasoning, 10k cache_creation
	// input cost = 0.8M * 3.00 = 2.40
	// cached cost = 0.2M * 0.30 = 0.06
	// output cost = 0.1M * 15.00 = 1.50
	// reasoning cost = 0.05M * 15.00 = 0.75
	// cache_creation cost = 0.01M * 3.75 = 0.0375
	// total = 4.7475
	tokens := TokenSet{Prompt: 1_000_000, Cached: 200_000, Completion: 100_000, Reasoning: 50_000, CacheCreation: 10_000}
	got := CalculateCost(tokens, sonnet)
	want := 4.7475
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("CalculateCost = %v, want %v", got, want)
	}

	// Synonym inputs produce identical cost.
	synonyms := NormalizeTokens(map[string]int64{
		"input_tokens":                1_000_000,
		"cache_read_input_tokens":     200_000,
		"output_tokens":               100_000,
		"reasoning_tokens":            50_000,
		"cache_creation_input_tokens": 10_000,
	})
	gotSyn := CalculateCost(synonyms, sonnet)
	if math.Abs(gotSyn-want) > 1e-9 {
		t.Errorf("CalculateCost synonyms = %v, want %v", gotSyn, want)
	}

	// nil/empty tokens → 0
	if got := CalculateCost(TokenSet{}, sonnet); got != 0 {
		t.Errorf("CalculateCost empty = %v, want 0", got)
	}

	// cached > input clamps to 0 non-cached input
	cachedOnly := TokenSet{Prompt: 100_000, Cached: 200_000}
	if got := CalculateCost(cachedOnly, sonnet); got < 0 {
		t.Errorf("CalculateCost cached>input = %v, want >= 0", got)
	}

	// Fallback rates: reasoning falls back to output, cache_creation to input, cached to input.
	partial := Pricing{Input: 3.00, Output: 15.00}
	partialTokens := TokenSet{Prompt: 1_000_000, Cached: 100_000, Completion: 100_000, Reasoning: 100_000, CacheCreation: 100_000}
	gotPartial := CalculateCost(partialTokens, partial)
	// noncached=900k*3=2.70; cached=100k*3=0.30; output=100k*15=1.50; reasoning=100k*15=1.50; creation=100k*3=0.30
	wantPartial := 2.70 + 0.30 + 1.50 + 1.50 + 0.30
	if math.Abs(gotPartial-wantPartial) > 1e-9 {
		t.Errorf("CalculateCost partial = %v, want %v", gotPartial, wantPartial)
	}

	// Zero pricing returns 0.
	if got := CalculateCost(tokens, Pricing{}); got != 0 {
		t.Errorf("CalculateCost zero pricing = %v, want 0", got)
	}
}

func TestResolverCostFor(t *testing.T) {
	store := &fakeOverrideStore{data: nil}
	r := NewResolver(store, func() int64 { return 0 })

	// known model
	tokens := TokenSet{Prompt: 1_000_000}
	got := r.CostFor("anthropic", "claude-sonnet-4-6", tokens)
	want := 3.00 // 1M * $3 / 1M
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("CostFor claude-sonnet-4-6 = %v, want %v", got, want)
	}

	// unknown model returns 0
	if got := r.CostFor("openai", "unknown-model-xyz", tokens); got != 0 {
		t.Errorf("CostFor unknown = %v, want 0", got)
	}
}
