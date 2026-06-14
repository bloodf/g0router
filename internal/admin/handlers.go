package admin

import (
	"fmt"

	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/governance"
	"github.com/bloodf/g0router/internal/platform"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
)

// Handlers bundles the management API endpoints and their dependencies.
type Handlers struct {
	store        *store.Store
	sessions     *auth.Sessions
	flows        map[string]*auth.OAuthFlow
	limiter      *auth.LoginLimiter
	stats        *usage.StatsService
	resolver     *usage.Resolver
	audit        *governance.AuditService
	proxyPools   *platform.ProxyPoolService
	version      string
	buildDate    string
	shutdownFunc func()
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
	return &Handlers{
		store:      st,
		sessions:   sessions,
		flows:      flows,
		limiter:    auth.NewLoginLimiter(),
		audit:      governance.NewAuditService(st),
		proxyPools: platform.NewProxyPoolService(st),
	}
}

// auditService returns the audit service for recording administrative actions.
func (h *Handlers) auditService() *governance.AuditService {
	return h.audit
}

// SetUsageServices wires the usage stats service and pricing resolver.
// It is called by the server bootstrap after New.
func (h *Handlers) SetUsageServices(stats *usage.StatsService, resolver *usage.Resolver) {
	h.stats = stats
	h.resolver = resolver
}

// SetVersionInfo injects the binary's version and build date so GetVersion can
// report them. The binary's version lives in cmd/g0router package-level vars
// that are not reachable from internal/admin; the server bootstrap forwards
// them via this setter after New (mirroring SetUsageServices).
func (h *Handlers) SetVersionInfo(version, buildDate string) {
	h.version = version
	h.buildDate = buildDate
}

// SetShutdownFunc injects the graceful-shutdown hook the Shutdown handler
// triggers. It is nil-able: when unset, Shutdown reports 501 and never exits.
// The hook is invoked asynchronously, after the response is written, so the
// real os.Exit/server-close path is never reached inside the handler body.
func (h *Handlers) SetShutdownFunc(fn func()) {
	h.shutdownFunc = fn
}

// SetProxyProber injects the proxy connectivity prober used by TestProxyPool.
// Production wires the real proxied dial; tests inject a deterministic fake.
func (h *Handlers) SetProxyProber(p platform.Prober) {
	h.proxyPools.SetProber(p)
}

// pathID returns the {id} route parameter.
func pathID(v any) (string, bool) {
	s, ok := v.(string)
	return s, ok
}
