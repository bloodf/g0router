package usage

import (
	"encoding/json"
	"fmt"
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
	// SetKV upserts a single key→value pair within a scope.
	SetKV(scope, key, value string) error
	// DeleteKV removes a single key from a scope.
	DeleteKV(scope, key string) error
	// ClearKVScope removes every key in a scope.
	ClearKVScope(scope string) error
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

const cacheTTLMs = 5000

// NewResolver creates a pricing resolver with the given override store and clock.
// clock returns the current time in milliseconds.
func NewResolver(store OverrideStore, clock func() int64) *Resolver {
	return &Resolver{store: store, clock: clock}
}

// Merged returns the merged provider→model→pricing view, caching it for 5s.
func (r *Resolver) Merged() (map[string]map[string]Pricing, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.clock()
	if r.cache != nil && r.cache.expiresAt > now {
		return copyMerged(r.cache.value), nil
	}

	user, err := r.store.UserPricing()
	if err != nil {
		return nil, fmt.Errorf("user pricing: %w", err)
	}

	merged := make(map[string]map[string]Pricing)

	// Seed canonical provider pricing.
	for provider, models := range ProviderPricing {
		out := make(map[string]Pricing, len(models))
		for model, p := range models {
			out[model] = p
		}
		merged[provider] = out
	}

	// Overlay user pricing.
	for provider, models := range user {
		out, ok := merged[provider]
		if !ok {
			out = make(map[string]Pricing)
			merged[provider] = out
		}
		for model, rates := range models {
			base := Pricing{}
			if existing, ok := out[model]; ok {
				base = existing
			}
			out[model] = overlayPricing(base, rates)
		}
	}

	r.cache = &resolvedCache{value: merged, expiresAt: now + cacheTTLMs}
	return copyMerged(merged), nil
}

// Invalidate clears the cached merged view so the next call re-reads the store.
func (r *Resolver) Invalidate() {
	r.mu.Lock()
	r.cache = nil
	r.mu.Unlock()
}

// Update merges the supplied provider→model→rates data into the existing user
// pricing overrides and invalidates the cache.
func (r *Resolver) Update(updates map[string]map[string]map[string]float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, err := r.store.UserPricing()
	if err != nil {
		return fmt.Errorf("read user pricing: %w", err)
	}
	if current == nil {
		current = make(map[string]map[string]map[string]float64)
	}

	for provider, models := range updates {
		merged, ok := current[provider]
		if !ok {
			merged = make(map[string]map[string]float64)
			current[provider] = merged
		}
		for model, rates := range models {
			merged[model] = rates
		}
		value, err := json.Marshal(merged)
		if err != nil {
			return fmt.Errorf("marshal pricing %s: %w", provider, err)
		}
		if err := r.store.SetKV("pricing", provider, string(value)); err != nil {
			return fmt.Errorf("set pricing %s: %w", provider, err)
		}
	}

	r.cache = nil
	return nil
}

// Reset removes a provider-level override, or a single model override when
// model is non-empty. If the provider becomes empty it is deleted.
func (r *Resolver) Reset(provider, model string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, err := r.store.UserPricing()
	if err != nil {
		return fmt.Errorf("read user pricing: %w", err)
	}

	if model == "" {
		if err := r.store.DeleteKV("pricing", provider); err != nil {
			return fmt.Errorf("delete pricing %s: %w", provider, err)
		}
		r.cache = nil
		return nil
	}

	models, ok := current[provider]
	if !ok {
		r.cache = nil
		return nil
	}
	delete(models, model)
	if len(models) == 0 {
		if err := r.store.DeleteKV("pricing", provider); err != nil {
			return fmt.Errorf("delete pricing %s: %w", provider, err)
		}
	} else {
		value, err := json.Marshal(models)
		if err != nil {
			return fmt.Errorf("marshal pricing %s: %w", provider, err)
		}
		if err := r.store.SetKV("pricing", provider, string(value)); err != nil {
			return fmt.Errorf("set pricing %s: %w", provider, err)
		}
	}

	r.cache = nil
	return nil
}

// ResetAll clears all user pricing overrides.
func (r *Resolver) ResetAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.store.ClearKVScope("pricing"); err != nil {
		return fmt.Errorf("clear pricing: %w", err)
	}
	r.cache = nil
	return nil
}

// overlayPricing applies present user rate keys over the canonical pricing.
func overlayPricing(base Pricing, rates map[string]float64) Pricing {
	if v, ok := rates["input"]; ok {
		base.Input = v
	}
	if v, ok := rates["output"]; ok {
		base.Output = v
	}
	if v, ok := rates["cached"]; ok {
		base.Cached = v
	}
	if v, ok := rates["reasoning"]; ok {
		base.Reasoning = v
	}
	if v, ok := rates["cache_creation"]; ok {
		base.CacheCreation = v
	}
	return base
}

// copyMerged returns a shallow copy of the merged view.
func copyMerged(src map[string]map[string]Pricing) map[string]map[string]Pricing {
	out := make(map[string]map[string]Pricing, len(src))
	for provider, models := range src {
		mcopy := make(map[string]Pricing, len(models))
		for model, p := range models {
			mcopy[model] = p
		}
		out[provider] = mcopy
	}
	return out
}

// PricingForModel resolves pricing for a provider/model, checking user overrides first.
// User overrides are matched by exact provider/model only (pricingRepo.js:51-56);
// vendor-prefix stripping is reserved for canonical constant resolution.
func (r *Resolver) PricingForModel(provider, model string) (Pricing, bool, error) {
	overrides, err := r.store.UserPricing()
	if err != nil {
		return Pricing{}, false, fmt.Errorf("user pricing: %w", err)
	}

	// User override checked before constants (pricingRepo.js:51-56).
	if provider != "" {
		if models, ok := overrides[provider]; ok {
			if u, ok := models[model]; ok {
				return overrideToPricing(u), true, nil
			}
		}
	}

	if p, ok := ResolvePricing(provider, model); ok {
		return p, true, nil
	}
	return Pricing{}, false, nil
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
