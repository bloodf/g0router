package admin

import (
	"fmt"
	"time"

	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/governance"
	"github.com/bloodf/g0router/internal/logging"
	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/platform"
	"github.com/bloodf/g0router/internal/platform/mitm"
	"github.com/bloodf/g0router/internal/platform/tunnel"
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
	audit         *governance.AuditService
	proxyPools    *platform.ProxyPoolService
	providerNodes *platform.ProviderNodeService
	tunnels       *tunnel.Service
	mitm         *mitm.Service
	mcpLauncher  *mcp.Launcher
	mcpEngine    *mcp.Engine
	mcpProbe     *mcp.Probe
	mcpSSEBeat   time.Duration
	console      *logging.ConsoleLog
	modelProber  ModelProber
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
		audit:         governance.NewAuditService(st),
		proxyPools:    platform.NewProxyPoolService(st),
		providerNodes: platform.NewProviderNodeService(st),
		tunnels:       tunnel.NewService(st),
		mitm:          mitm.NewService(st),
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

// SetNodeProber injects the provider-node validation probe used by
// ValidateProviderNode. Production wires the real /models→/chat/completions dial;
// tests inject a deterministic fake. Mirrors SetProxyProber.
func (h *Handlers) SetNodeProber(p platform.NodeProber) {
	h.providerNodes.SetProber(p)
}

// SetNodeResolver injects the DNS resolver used by the provider-node validation
// SSRF guard. Production uses the system resolver; tests inject a deterministic
// fake so validation runs without network access.
func (h *Handlers) SetNodeResolver(r platform.IPResolver) {
	h.providerNodes.SetResolver(r)
}

// SetTunnelRunner overrides the runner for a tunnel type. Production uses the
// real cloudflared/tailscale runners constructed in New; tests inject a
// deterministic fake so the tunnel admin API runs without spawning a process.
// Mirrors SetProxyProber.
func (h *Handlers) SetTunnelRunner(typ string, r tunnel.Runner) {
	h.tunnels.SetRunner(typ, r)
}

// SetMitmProxy overrides the MITM proxy listener. Production uses the real
// TLS-intercepting listener constructed lazily on enable; tests inject a
// deterministic fake so the MITM admin API runs without binding a port or
// performing a real TLS handshake. Mirrors SetTunnelRunner.
func (h *Handlers) SetMitmProxy(p mitm.MitmProxy) {
	h.mitm.SetProxy(p)
}

// SetMCPLauncher injects the MCP plugin launcher used for instance start/stop
// and the tool-execute bridge path. Production wires the real NewLauncher(st);
// tests inject a real launcher with a fake ProcessRunner (no real spawn).
// Mirrors SetMitmProxy. Nil-able: an unset launcher degrades execute/start to a
// typed 503 while list reads still work from the store.
func (h *Handlers) SetMCPLauncher(l *mcp.Launcher) {
	h.mcpLauncher = l
}

// SetMCPEngine injects the MCP-server OAuth engine used by StartInstanceAuth.
// Production wires the real NewEngine(st, nil); tests inject an engine over a
// fake-transport HTTP client (no real network). Nil-able: an unset engine makes
// auth/start return a typed 503.
func (h *Handlers) SetMCPEngine(e *mcp.Engine) {
	h.mcpEngine = e
}

// SetMCPProbe injects the MCP tools probe used by the tools-list aggregation.
// Nil-able: an unset probe degrades the tools list to the static catalog.
func (h *Handlers) SetMCPProbe(p *mcp.Probe) {
	h.mcpProbe = p
}

// SetMCPServerHeartbeat injects the GET /mcp SSE heartbeat interval (bf-mcp-1,
// D5). Production wires the default mcpSSEHeartbeatInterval; an integration
// deployment may override it. A non-positive value falls back to the default.
// Mirrors SetMCPProbe. The unit tests drive serveMCPSSE's injected tick channel
// directly, so the real ticker constructed in MCPServerSSE is never exercised by
// a test (hermetic).
func (h *Handlers) SetMCPServerHeartbeat(d time.Duration) {
	h.mcpSSEBeat = d
}

// SetConsoleLog injects the in-process console-log capture buffer that
// ConsoleLogStream subscribes to. Production wires the real ring buffer fed by
// the server's log output; tests inject a buffer and drive Append directly.
// Nil-able: an unset buffer makes ConsoleLogStream report 501.
func (h *Handlers) SetConsoleLog(c *logging.ConsoleLog) {
	h.console = c
}

// SetModelProber injects the model reachability prober used by TestModel.
// Production wires a best-effort reachability check; tests inject a
// deterministic fake. Mirrors SetNodeProber. Nil-able: an unset prober makes
// TestModel report 501.
func (h *Handlers) SetModelProber(p ModelProber) {
	h.modelProber = p
}

// pathID returns the {id} route parameter.
func pathID(v any) (string, bool) {
	s, ok := v.(string)
	return s, ok
}
