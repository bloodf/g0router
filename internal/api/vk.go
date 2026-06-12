package api

// VKProviderConfig binds a provider to the models a virtual key may use.
type VKProviderConfig struct {
	Provider      string
	AllowedModels []string
	KeyIDs        []string
}

// VKInfo is the subset of virtual key state needed by VKGate.
type VKInfo struct {
	Key           string
	Configs       []VKProviderConfig
	BudgetLimit   float64
	BudgetPeriod  string
	RateLimitRPM  int
	IsActive      bool
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
	if cfg != nil {
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
// was found.
func matchProviderConfig(configs []VKProviderConfig, model, providerID string) (*VKProviderConfig, bool) {
	for i := range configs {
		cfg := &configs[i]
		if cfg.Provider != providerID {
			continue
		}
		if len(cfg.AllowedModels) == 0 {
			return cfg, true
		}
		for _, m := range cfg.AllowedModels {
			if m == model {
				return cfg, true
			}
		}
	}
	return nil, false
}
