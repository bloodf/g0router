package usage

import (
	"errors"
	"strings"
	"testing"
)

func TestMatchPattern(t *testing.T) {
	cases := []struct {
		pattern string
		model   string
		want    bool
	}{
		{"*-codex-xhigh", "gpt-5.3-codex-xhigh", true},
		{"*-codex-xhigh", "gpt-5.3-codex-high", false},
		{"gemini-*-flash-lite", "gemini-2.5-flash-lite", true},
		{"gemini-*-flash-lite", "gemini-2.5-flash", false},
		{"gpt-4o", "gpt-4o", true},
		{"gpt-4o", "gpt-4o-mini", false},
		// literal dots are escaped, not regex-active
		{"gpt-4.*", "gpt-4x", false},
		{"gpt-4.*", "gpt-4.1", true},
	}

	for _, tc := range cases {
		got := MatchPattern(tc.pattern, tc.model)
		if got != tc.want {
			t.Errorf("MatchPattern(%q, %q) = %v, want %v", tc.pattern, tc.model, got, tc.want)
		}
	}
}

func TestResolvePricing(t *testing.T) {
	// provider override wins
	p, ok := ResolvePricing("gh", "gpt-5.3-codex")
	if !ok {
		t.Fatal("ResolvePricing(gh, gpt-5.3-codex): expected hit")
	}
	wantGH := Pricing{Input: 1.75, Output: 14.00, Cached: 0.175, Reasoning: 14.00, CacheCreation: 1.75}
	if p != wantGH {
		t.Errorf("ResolvePricing(gh, gpt-5.3-codex) = %+v, want %+v", p, wantGH)
	}

	// vendor prefix strip
	p, ok = ResolvePricing("deepseek", "deepseek/deepseek-chat")
	if !ok {
		t.Fatal("ResolvePricing(deepseek, deepseek/deepseek-chat): expected hit")
	}
	wantDeepseek := ModelPricing["deepseek-chat"]
	if p != wantDeepseek {
		t.Errorf("ResolvePricing(deepseek, deepseek/deepseek-chat) = %+v, want %+v", p, wantDeepseek)
	}

	// exact canonical model
	p, ok = ResolvePricing("openai", "gpt-4o-mini")
	if !ok {
		t.Fatal("ResolvePricing(openai, gpt-4o-mini): expected hit")
	}
	if p != ModelPricing["gpt-4o-mini"] {
		t.Errorf("ResolvePricing(openai, gpt-4o-mini) = %+v, want %+v", p, ModelPricing["gpt-4o-mini"])
	}

	// pattern fallback order
	p, ok = ResolvePricing("", "some-codex-xhigh")
	if !ok {
		t.Fatal("expected pattern fallback")
	}
	wantPattern := PatternPricing[0].Pricing // *-codex-xhigh
	if p != wantPattern {
		t.Errorf("ResolvePricing pattern fallback = %+v, want %+v", p, wantPattern)
	}

	// unknown model
	_, ok = ResolvePricing("openai", "totally-unknown-model-xyz")
	if ok {
		t.Error("ResolvePricing unknown model: expected miss")
	}

	// empty model returns miss
	_, ok = ResolvePricing("openai", "")
	if ok {
		t.Error("ResolvePricing empty model: expected miss")
	}
}

func TestPricingForModelWrapsStoreError(t *testing.T) {
	store := &fakeOverrideStore{err: errPricingBoom}
	r := NewResolver(store, func() int64 { return 0 })

	_, _, err := r.PricingForModel("deepseek", "deepseek-chat")
	if err == nil {
		t.Fatal("expected error from failing store")
	}
	if !errors.Is(err, errPricingBoom) {
		t.Errorf("errors.Is(err, errPricingBoom) = false, err = %v", err)
	}
	if !strings.Contains(err.Error(), "user pricing:") {
		t.Errorf("error message = %q, want 'user pricing:' context", err.Error())
	}
}

func TestPricingForModelUserOverrideExactMatch(t *testing.T) {
	store := &fakeOverrideStore{
		data: map[string]map[string]map[string]float64{
			"deepseek": {
				"deepseek-chat": {"input": 9.9},
			},
		},
	}
	r := NewResolver(store, func() int64 { return 0 })

	// Exact provider+model uses the user override.
	p, ok, err := r.PricingForModel("deepseek", "deepseek-chat")
	if err != nil {
		t.Fatalf("exact match: %v", err)
	}
	if !ok {
		t.Fatal("exact match: expected hit")
	}
	if p.Input != 9.9 {
		t.Errorf("exact match input = %v, want 9.9", p.Input)
	}

	// Vendor-prefixed model does NOT match the user override (ref exact lookup).
	p, ok, err = r.PricingForModel("deepseek", "deepseek/deepseek-chat")
	if err != nil {
		t.Fatalf("prefixed match: %v", err)
	}
	if !ok {
		t.Fatal("prefixed match: expected hit from constants")
	}
	if p == (Pricing{}) {
		t.Fatal("prefixed match: got zero pricing")
	}
	if p.Input == 9.9 {
		t.Errorf("prefixed match used user override (%v), should have fallen through to constants", p)
	}
}
