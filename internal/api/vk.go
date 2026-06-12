package api

// VKInfo is the subset of virtual key state needed by VKGate.
type VKInfo struct {
	Key           string
	AllowedModels []string
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

// AllowVK checks whether a request bearing x-g0-vk may proceed for the given model.
// It returns ok=true when the key is absent/unresolved, when the model is allowed,
// and when quota checks pass. On denial it returns an HTTP status (403 or 429) and
// a human-readable reason.
func (g *VKGate) AllowVK(key, model string) (ok bool, status int, reason string) {
	if g == nil || g.resolver == nil || key == "" {
		return true, 0, ""
	}
	vk, err := g.resolver.ResolveVK(key)
	if err != nil {
		return false, 500, "virtual key lookup failed"
	}
	if vk == nil {
		return true, 0, ""
	}
	if !vk.IsActive {
		return false, 403, "virtual key inactive"
	}
	if !modelAllowed(vk.AllowedModels, model) {
		return false, 403, "model not allowed for virtual key"
	}
	if g.quota != nil {
		return g.quota.Allow(vk, model)
	}
	return true, 0, ""
}

func modelAllowed(models []string, model string) bool {
	if len(models) == 0 {
		return true
	}
	for _, m := range models {
		if m == model {
			return true
		}
	}
	return false
}
