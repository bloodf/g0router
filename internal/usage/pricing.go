package usage

import (
	"regexp"
	"strings"
	"sync"
)

// MatchPattern reports whether model matches the glob-style pattern.
// "*" matches any substring; all other regex metacharacters are escaped.
func MatchPattern(pattern, model string) bool {
	parts := strings.Split(pattern, "*")
	escaped := make([]string, len(parts))
	for i, p := range parts {
		escaped[i] = regexp.QuoteMeta(p)
	}
	re, err := regexp.Compile("^" + strings.Join(escaped, ".*") + "$")
	if err != nil {
		return false
	}
	return re.MatchString(model)
}

// ResolvePricing resolves pricing using the 3-step fallback chain:
//   1. ProviderPricing[provider][model]
//   2. ModelPricing[model] (with vendor prefix stripped)
//   3. PatternPricing (first glob match against baseModel or model)
func ResolvePricing(provider, model string) (Pricing, bool) {
	if model == "" {
		return Pricing{}, false
	}

	baseModel := model
	if idx := strings.LastIndex(model, "/"); idx >= 0 {
		baseModel = model[idx+1:]
	}

	// 1. Provider-specific override.
	if provider != "" {
		if models, ok := ProviderPricing[provider]; ok {
			if p, ok := models[model]; ok {
				return p, true
			}
			if p, ok := models[baseModel]; ok {
				return p, true
			}
		}
	}

	// 2. Canonical model pricing.
	if p, ok := ModelPricing[baseModel]; ok {
		return p, true
	}
	if p, ok := ModelPricing[model]; ok {
		return p, true
	}

	// 3. Pattern match.
	for _, pp := range PatternPricing {
		if MatchPattern(pp.Pattern, baseModel) || MatchPattern(pp.Pattern, model) {
			return pp.Pricing, true
		}
	}

	return Pricing{}, false
}

// OverrideStore is the layering-mandated seam that supplies user pricing overrides.
// The store package implements this over the kv table (scope='pricing').
type OverrideStore interface {
	// UserPricing returns provider → model → rate-name → value.
	// Only present keys are included; absent keys inherit the canonical value.
	UserPricing() (map[string]map[string]map[string]float64, error)
}

// Resolver resolves pricing with user overrides and a 5s TTL cache.
type Resolver struct {
	store OverrideStore
	clock func() int64

	mu     sync.Mutex
	cache  *resolvedCache
}

type resolvedCache struct {
	value     map[string]map[string]Pricing
	expiresAt int64
}

// NewResolver creates a pricing resolver with the given override store and clock.
// clock returns the current time in milliseconds.
func NewResolver(store OverrideStore, clock func() int64) *Resolver {
	return &Resolver{store: store, clock: clock}
}

// PricingForModel resolves pricing for a provider/model, checking user overrides first.
func (r *Resolver) PricingForModel(provider, model string) (Pricing, bool) {
	overrides, err := r.store.UserPricing()
	if err != nil {
		return Pricing{}, false
	}

	baseModel := model
	if idx := strings.LastIndex(model, "/"); idx >= 0 {
		baseModel = model[idx+1:]
	}

	// User override checked before constants (pricingRepo.js:51-56).
	if provider != "" {
		if models, ok := overrides[provider]; ok {
			if u, ok := models[model]; ok {
				return overrideToPricing(u), true
			}
			if u, ok := models[baseModel]; ok {
				return overrideToPricing(u), true
			}
		}
	}

	return ResolvePricing(provider, model)
}

// overrideToPricing converts a present-key rate map into a Pricing struct.
// Absent rate keys resolve to 0.
func overrideToPricing(u map[string]float64) Pricing {
	return Pricing{
		Input:         u["input"],
		Output:        u["output"],
		Cached:        u["cached"],
		Reasoning:     u["reasoning"],
		CacheCreation: u["cache_creation"],
	}
}
