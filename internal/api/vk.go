package api

import "github.com/bloodf/g0router/internal/schemas"

// VKProviderConfig binds a provider to the models a virtual key may use.
// AllowAllKeys=true means any provider key is allowed: the gate returns no
// pinned KeyIDs and falls through to normal selection (bf-gov-1, D5).
// BlacklistedModels blocks models even when allowlisted (bf-gov-2, D3).
type VKProviderConfig struct {
	Provider          string
	AllowedModels     []string
	BlacklistedModels schemas.BlackList
	KeyIDs            []string
	AllowAllKeys      bool
}

// VKInfo is the subset of virtual key state needed by VKGate. The Team* fields
// carry the optional owning team's budget/RPM for the 2-level hierarchy
// (bf-gov-1, D3); they are zero/empty for un-teamed keys.
type VKInfo struct {
	Key           string
	Configs       []VKProviderConfig
	BudgetLimit   float64
	BudgetPeriod  string
	RateLimitRPM  int
	IsActive      bool

	TeamID           string
	TeamBudgetLimit  float64
	TeamBudgetPeriod string
	TeamRateLimitRPM int
}

// VKResolver resolves a virtual key header value to a VKInfo.
// Implementations live outside the api package (e.g., in internal/server)
// so the api layer stays free of store and governance imports.
type VKResolver interface {
	ResolveVK(key string) (*VKInfo, error)
}

// VKPinnedKeyResolver resolves a provider/model/keyIDs triple to a pinned
// connection ID and credential. Implementations live outside the api package.
type VKPinnedKeyResolver interface {
	ResolvePinned(providerID, model string, keyIDs []string) (connID, credential string, ok bool)
}

// VKQuotaChecker enforces budget and RPM limits for a virtual key.
// Implementations live outside the api package.
type VKQuotaChecker interface {
	Allow(vk *VKInfo, model string) (ok bool, status int, reason string)
}

// VKGate enforces x-g0-vk header routing, allowed-model checks, and quotas.
type VKGate struct {
	resolver VKResolver
	quota    VKQuotaChecker
}

// NewVKGate creates a virtual-key gate with the given resolver and quota checker.
func NewVKGate(resolver VKResolver, quota VKQuotaChecker) *VKGate {
	return &VKGate{resolver: resolver, quota: quota}
}

// AllowVK checks whether a request bearing x-g0-vk may proceed for the given model
// and resolved provider. It returns ok=true when the key is absent/unresolved, when
// an active config matches the provider and model, and when quota checks pass. On
// denial it returns an HTTP status (401, 403, or 429), a human-readable reason, and
// the matched config's KeyIDs (nil when no header / no match / config has none).
func (g *VKGate) AllowVK(key, model, providerID string) (ok bool, status int, reason string, keyIDs []string) {
	if g == nil || g.resolver == nil || key == "" {
		return true, 0, "", nil
	}
	vk, err := g.resolver.ResolveVK(key)
	if err != nil {
		return false, 500, "virtual key lookup failed", nil
	}
	if vk == nil {
		return false, 401, "unknown virtual key", nil
	}
	if !vk.IsActive {
		return false, 403, "virtual key inactive", nil
	}
	cfg, matched := matchProviderConfig(vk.Configs, model, providerID)
	if !matched {
		return false, 403, "provider/model not allowed for virtual key", nil
	}
	if cfg != nil && !cfg.AllowAllKeys {
		// AllowAllKeys=true returns no pin (any provider key allowed, D5);
		// otherwise the matched config's KeyIDs pin the eligible keys.
		keyIDs = cfg.KeyIDs
	}
	if g.quota != nil {
		// Quota denial hides KeyIDs per design.
		ok, status, reason := g.quota.Allow(vk, model)
		if !ok {
			return false, status, reason, nil
		}
	}
	return true, 0, "", keyIDs
}

// matchProviderConfig returns the first matching provider config and whether a match
// was found. It runs a two-pass filter per provider-matching config (bf-gov-2, D3):
//  1. Blacklist pass (FIRST): a model in cfg.BlacklistedModels is blocked, so the
//     config does not match — blacklist wins over allowlist.
//  2. Allowlist pass (SECOND): an empty OR nil AllowedModels is legacy match-all
//     (backward-compat VAR, D1 — the gate does NOT adopt empty=deny-all); a
//     non-empty AllowedModels restricts to the listed models via WhiteList.IsAllowed.
func matchProviderConfig(configs []VKProviderConfig, model, providerID string) (*VKProviderConfig, bool) {
	for i := range configs {
		cfg := &configs[i]
		if cfg.Provider != providerID {
			continue
		}
		// Blacklist pass first: a blocked model can never match this config.
		if cfg.BlacklistedModels.IsBlocked(model) {
			continue
		}
		// Allowlist pass: legacy match-all on empty/nil AllowedModels (VAR, D1).
		if len(cfg.AllowedModels) == 0 {
			return cfg, true
		}
		if schemas.WhiteList(cfg.AllowedModels).IsAllowed(model) {
			return cfg, true
		}
	}
	return nil, false
}
