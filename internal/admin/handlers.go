package admin

import (
	"fmt"

	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
)

// Handlers bundles the management API endpoints and their dependencies.
type Handlers struct {
	store      *store.Store
	sessions   *auth.Sessions
	flows      map[string]*auth.OAuthFlow
	limiter    *auth.LoginLimiter
	stats      *usage.StatsService
	resolver   *usage.Resolver
}

// New creates the admin handler set. flows maps provider type → OAuth flow
// and may be nil when no OAuth providers are configured.
func New(st *store.Store, sessions *auth.Sessions, flows map[string]*auth.OAuthFlow) *Handlers {
	if flows == nil {
		flows = map[string]*auth.OAuthFlow{}
	}
	// Wire auth-derived key generation into the store to keep the store
	// package free of the auth→store import cycle.
	st.SetAPIKeyGenerator(func(dataDir string) (string, string, error) {
		machineID, err := auth.MachineID(dataDir, "")
		if err != nil {
			return "", "", fmt.Errorf("derive machine id: %w", err)
		}
		key, _, err := auth.GenerateAPIKey(machineID)
		if err != nil {
			return "", "", fmt.Errorf("generate api key: %w", err)
		}
		return key, machineID, nil
	})
	return &Handlers{store: st, sessions: sessions, flows: flows, limiter: auth.NewLoginLimiter()}
}

// SetUsageServices wires the usage stats service and pricing resolver.
// It is called by the server bootstrap after New.
func (h *Handlers) SetUsageServices(stats *usage.StatsService, resolver *usage.Resolver) {
	h.stats = stats
	h.resolver = resolver
}

// pathID returns the {id} route parameter.
func pathID(v any) (string, bool) {
	s, ok := v.(string)
	return s, ok
}
