package server

import (
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/admin"
	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/logging"
	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/providers/catalog"
	"github.com/bloodf/g0router/internal/store"
	"github.com/fasthttp/router"
)

// consoleLogCapacity bounds the in-process console-log ring buffer.
const consoleLogCapacity = 500

// consoleLogWriter adapts an io.Writer onto a ConsoleLog: each written log line
// is parsed into {level,message} and appended. Used with io.MultiWriter so the
// server's standard log output is captured for the console-logs SSE stream.
type consoleLogWriter struct {
	console *logging.ConsoleLog
}

func (w consoleLogWriter) Write(p []byte) (int, error) {
	line := strings.TrimRight(string(p), "\n")
	if line != "" {
		w.console.Append(consoleLogLevel(line), line)
	}
	return len(p), nil
}

// consoleLogLevel infers a coarse level from a log line's content.
func consoleLogLevel(line string) string {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "error"), strings.Contains(lower, "fatal"):
		return "error"
	case strings.Contains(lower, "warn"):
		return "warn"
	default:
		return "info"
	}
}

// defaultModelProber resolves a model against the static catalog and reports it
// reachable when known. It is a best-effort, network-free check; tests inject a
// fake via SetModelProber. Implements admin.ModelProber.
type defaultModelProber struct{}

func (defaultModelProber) Probe(provider, modelID string) (bool, int, error) {
	if provider != "" {
		if _, ok := catalog.ResolveModel(provider, modelID); ok {
			return true, 0, nil
		}
		return false, 0, nil
	}
	for p := range catalog.Models {
		if _, ok := catalog.ResolveModel(p, modelID); ok {
			return true, 0, nil
		}
	}
	return false, 0, nil
}

// sessionTTL is the dashboard session lifetime.
const sessionTTL = 7 * 24 * time.Hour

// NewAdminHandlers builds the management handler set from the store with
// the default session TTL and production OAuth flows.
// deps supplies the shared usage events/tracker/ring that the OpenAI-compatible
// API handlers also consume; admin stats must observe the same live instances.
func NewAdminHandlers(st *store.Store, deps admin.UsageDeps) *admin.Handlers {
	sessions := auth.NewSessions(st, sessionTTL)
	flows := map[string]*auth.OAuthFlow{
		"anthropic": auth.NewOAuthFlow(auth.AnthropicOAuth(), st, nil),
		// w7-prov-oauth: 8 provider OAuth flows. Redirect providers (claude/
		// codex/gemini-cli/iflow/cline) drive the existing /api/oauth/{provider}/
		// {start,callback} handlers; device-code providers (qwen/github/kilocode)
		// expose StartDevice/PollDevice — their admin transport is a follow-up
		// (ESC-DEVICE-ENDPOINT; see open-questions.md).
		"claude":     auth.NewOAuthFlow(auth.ClaudeOAuth(), st, nil),
		"codex":      auth.NewOAuthFlow(auth.CodexOAuth(), st, nil),
		"gemini-cli": auth.NewOAuthFlow(auth.GeminiCLIOAuth(), st, nil),
		"qwen":       auth.NewOAuthFlow(auth.QwenOAuth(), st, nil),
		"iflow":      auth.NewOAuthFlow(auth.IflowOAuth(), st, nil),
		"github":     auth.NewOAuthFlow(auth.GithubOAuth(), st, nil),
		"kilocode":   auth.NewOAuthFlow(auth.KilocodeOAuth(), st, nil),
		"cline":      auth.NewOAuthFlow(auth.ClineOAuth(), st, nil),
	}
	h := admin.New(st, sessions, flows)
	stats, resolver := admin.BuildUsageServices(st, deps)
	h.SetUsageServices(stats, resolver)
	// Wire the real MCP launcher (stdio spawn + bridge), OAuth engine, and tools
	// probe after New (mirrors SetUsageServices; ESC-BOOTSTRAP).
	h.SetMCPLauncher(mcp.NewLauncher(st))
	h.SetMCPEngine(mcp.NewEngine(st, nil))
	h.SetMCPProbe(mcp.NewProbe(nil))

	// Console-logs capture seam (w7-misc, ESC-CONSOLE-SRC): a bounded ring
	// buffer the SSE stream subscribes to, fed by the server's standard log
	// output via io.MultiWriter (additive forward; stderr still receives logs).
	console := logging.NewConsoleLog(consoleLogCapacity)
	h.SetConsoleLog(console)
	log.SetOutput(io.MultiWriter(os.Stderr, consoleLogWriter{console: console}))

	// Model reachability prober for POST /api/models/test (w7-misc). Default is
	// a network-free catalog resolve; tests inject a fake.
	h.SetModelProber(defaultModelProber{})
	return h
}

// RegisterAdminRoutes adds the /api/* management routes to the router.
// Everything except login is protected by the session middleware.
func RegisterAdminRoutes(r *router.Router, h *admin.Handlers) {
	// Public.
	r.POST("/api/auth/login", h.Login)
	r.GET("/api/auth/oidc/start", h.OIDCStart)
	r.GET("/api/auth/oidc/callback", h.OIDCCallback)
	r.POST("/api/auth/oidc/test", h.OIDCTest)

	// Public first-user onboarding (self-guards on CountUsers()==0).
	r.POST("/api/auth/setup", h.AuthSetup)

	// Protected.
	r.POST("/api/auth/logout", h.RequireSession(h.Logout))
	r.GET("/api/auth/me", h.RequireSession(h.Me))

	// Protected user-management.
	r.PUT("/api/auth/password", h.RequireSession(h.ChangePassword))
	r.GET("/api/auth/users", h.RequireSession(h.ListUsers))
	r.POST("/api/auth/users", h.RequireSession(h.CreateUser))
	r.DELETE("/api/auth/users/{id}", h.RequireSession(h.DeleteUser))

	// Teams CRUD (static collection before {id}).
	r.GET("/api/teams", h.RequireSession(h.ListTeams))
	r.POST("/api/teams", h.RequireSession(h.CreateTeam))
	r.GET("/api/teams/{id}", h.RequireSession(h.GetTeam))
	r.PUT("/api/teams/{id}", h.RequireSession(h.UpdateTeam))
	r.DELETE("/api/teams/{id}", h.RequireSession(h.DeleteTeam))

	// Audit read.
	r.GET("/api/audit", h.RequireSession(h.GetAudit))

	// Feature flags (GET list + PUT toggle only — no create/delete).
	r.GET("/api/feature-flags", h.RequireSession(h.ListFeatureFlags))
	r.GET("/api/feature-flags/{id}", h.RequireSession(h.GetFeatureFlag))
	r.PUT("/api/feature-flags/{id}", h.RequireSession(h.ToggleFeatureFlag))

	// Prompt templates CRUD (+ test). Static /test registered before {id}.
	r.GET("/api/prompt-templates", h.RequireSession(h.ListPromptTemplates))
	r.POST("/api/prompt-templates", h.RequireSession(h.CreatePromptTemplate))
	r.POST("/api/prompt-templates/test", h.RequireSession(h.TestPromptTemplate))
	r.GET("/api/prompt-templates/{id}", h.RequireSession(h.GetPromptTemplate))
	r.PUT("/api/prompt-templates/{id}", h.RequireSession(h.UpdatePromptTemplate))
	r.DELETE("/api/prompt-templates/{id}", h.RequireSession(h.DeletePromptTemplate))

	// Guardrails (singleton config — no list/{id}). Static /test before the bare PUT/GET.
	r.GET("/api/guardrails", h.RequireSession(h.GetGuardrails))
	r.PUT("/api/guardrails", h.RequireSession(h.UpdateGuardrails))
	r.POST("/api/guardrails/test", h.RequireSession(h.TestGuardrails))

	// Alert channels CRUD (+ per-channel test). Static collection before {id}; {id}/test deeper.
	r.GET("/api/alert-channels", h.RequireSession(h.ListAlertChannels))
	r.POST("/api/alert-channels", h.RequireSession(h.CreateAlertChannel))
	r.POST("/api/alert-channels/{id}/test", h.RequireSession(h.TestAlertChannel))
	r.GET("/api/alert-channels/{id}", h.RequireSession(h.GetAlertChannel))
	r.PUT("/api/alert-channels/{id}", h.RequireSession(h.UpdateAlertChannel))
	r.DELETE("/api/alert-channels/{id}", h.RequireSession(h.DeleteAlertChannel))

	// MCP admin (clients/instances/tools/tool-groups + oauth start). LOCAL_ONLY
	// via guard.go (/api/mcp/ entry — consumed, not edited). Static collections
	// and deeper sub-paths registered before the bare {id} routes.
	r.GET("/api/mcp/clients", h.RequireSession(h.ListClients))
	r.GET("/api/mcp/clients/{id}", h.RequireSession(h.GetClient))
	r.GET("/api/mcp/instances", h.RequireSession(h.ListInstances))
	r.POST("/api/mcp/instances", h.RequireSession(h.CreateInstance))
	r.GET("/api/mcp/instances/{id}/accounts", h.RequireSession(h.ListInstanceAccounts))
	r.POST("/api/mcp/instances/{id}/auth/start", h.RequireSession(h.StartInstanceAuth))
	r.GET("/api/mcp/instances/{id}", h.RequireSession(h.GetInstance))
	r.DELETE("/api/mcp/instances/{id}", h.RequireSession(h.DeleteInstance))
	r.GET("/api/mcp/tools", h.RequireSession(h.ListTools))
	r.POST("/api/mcp/tools/{name}/execute", h.RequireSession(h.ExecuteTool))
	r.GET("/api/mcp/tool-groups", h.RequireSession(h.ListToolGroups))
	r.POST("/api/mcp/tool-groups", h.RequireSession(h.CreateToolGroup))
	r.GET("/api/mcp/tool-groups/{id}", h.RequireSession(h.GetToolGroup))
	r.PUT("/api/mcp/tool-groups/{id}", h.RequireSession(h.UpdateToolGroup))
	r.DELETE("/api/mcp/tool-groups/{id}", h.RequireSession(h.DeleteToolGroup))

	// MCP server mode (bf-mcp-1, serial slot): NEW public POST/GET /mcp JSON-RPC
	// + SSE surface, plus the session-gated complete-oauth route.
	RegisterMCPRoutes(r, h)

	// Skills catalog (read-only). NOT under /api/mcp/, so a normal session route.
	r.GET("/api/skills", h.RequireSession(h.ListSkills))

	r.GET("/api/settings", h.RequireSession(h.GetSettings))
	r.PUT("/api/settings", h.RequireSession(h.PutSettings))

	r.GET("/api/version", h.RequireSession(h.GetVersion))
	r.POST("/api/version/shutdown", h.RequireSession(h.Shutdown))

	r.GET("/api/providers", h.RequireSession(h.ListProviders))
	r.POST("/api/providers", h.RequireSession(h.CreateProvider))
	r.PUT("/api/providers/{id}", h.RequireSession(h.UpdateProvider))
	r.DELETE("/api/providers/{id}", h.RequireSession(h.DeleteProvider))

	// Provider-shaped read overlay (w6-e). Static catalog/test-batch routes match
	// before the {id} param routes (fasthttp/router static-segment precedence).
	r.GET("/api/providers/catalog", h.RequireSession(h.ListProviderCatalog))
	r.POST("/api/providers/test-batch", h.RequireSession(h.TestProvidersBatch))
	r.GET("/api/providers/{id}/catalog", h.RequireSession(h.GetProviderCatalog))
	r.GET("/api/providers/{id}/connections", h.RequireSession(h.GetProviderConnections))
	r.GET("/api/providers/{id}/models", h.RequireSession(h.GetProviderModels))
	r.GET("/api/providers/{id}/suggested-models", h.RequireSession(h.GetProviderSuggestedModels))

	// Provider-nodes prefix-routing engine (w7-platnodes). Static collection and
	// /validate registered before the {id} param routes (fasthttp/router
	// static-segment precedence).
	r.GET("/api/provider-nodes", h.RequireSession(h.ListProviderNodes))
	r.POST("/api/provider-nodes", h.RequireSession(h.CreateProviderNode))
	r.POST("/api/provider-nodes/validate", h.RequireSession(h.ValidateProviderNode))
	r.GET("/api/provider-nodes/{id}", h.RequireSession(h.GetProviderNode))
	r.PUT("/api/provider-nodes/{id}", h.RequireSession(h.UpdateProviderNode))
	r.DELETE("/api/provider-nodes/{id}", h.RequireSession(h.DeleteProviderNode))

	r.GET("/api/connections", h.RequireSession(h.ListConnections))
	r.POST("/api/connections", h.RequireSession(h.CreateConnection))
	r.PUT("/api/connections/{id}", h.RequireSession(h.UpdateConnection))
	r.DELETE("/api/connections/{id}", h.RequireSession(h.DeleteConnection))
	r.POST("/api/connections/{id}/refresh", h.RequireSession(h.RefreshConnection))

	r.GET("/api/keys", h.RequireSession(h.ListAPIKeys))
	r.POST("/api/keys", h.RequireSession(h.CreateAPIKey))
	r.GET("/api/keys/{id}", h.RequireSession(h.GetAPIKey))
	r.PUT("/api/keys/{id}", h.RequireSession(h.UpdateAPIKey))
	r.DELETE("/api/keys/{id}", h.RequireSession(h.DeleteAPIKey))

	r.GET("/api/virtual-keys", h.RequireSession(h.ListVirtualKeys))
	r.POST("/api/virtual-keys", h.RequireSession(h.CreateVirtualKey))
	r.GET("/api/virtual-keys/{id}", h.RequireSession(h.GetVirtualKey))
	r.PUT("/api/virtual-keys/{id}", h.RequireSession(h.UpdateVirtualKey))
	r.DELETE("/api/virtual-keys/{id}", h.RequireSession(h.DeleteVirtualKey))

	// Proxy-pools CRUD (static collection + batch before {id}; {id}/test deepest).
	r.GET("/api/proxy-pools", h.RequireSession(h.ListProxyPools))
	r.POST("/api/proxy-pools", h.RequireSession(h.CreateProxyPool))
	r.POST("/api/proxy-pools/batch", h.RequireSession(h.BatchProxyPools))
	r.GET("/api/proxy-pools/{id}", h.RequireSession(h.GetProxyPool))
	r.PUT("/api/proxy-pools/{id}", h.RequireSession(h.UpdateProxyPool))
	r.DELETE("/api/proxy-pools/{id}", h.RequireSession(h.DeleteProxyPool))
	r.POST("/api/proxy-pools/{id}/test", h.RequireSession(h.TestProxyPool))

	// Tunnels (static collection + /health before the {type} param route).
	r.GET("/api/tunnels", h.RequireSession(h.ListTunnels))
	r.GET("/api/tunnels/health", h.RequireSession(h.TunnelHealth))
	r.POST("/api/tunnels/{type}", h.RequireSession(h.EnableTunnel))
	r.DELETE("/api/tunnels/{type}", h.RequireSession(h.DisableTunnel))

	// MITM (status/toggle/ca-cert static; tools/{id} param). ca-cert serves raw PEM.
	r.GET("/api/mitm/status", h.RequireSession(h.MitmStatus))
	r.POST("/api/mitm/toggle", h.RequireSession(h.MitmToggle))
	r.GET("/api/mitm/ca-cert", h.RequireSession(h.MitmCACert))
	r.POST("/api/mitm/tools/{id}", h.RequireSession(h.MitmToolToggle))

	r.GET("/api/oauth/{provider}/start", h.RequireSession(h.OAuthStart))
	r.POST("/api/oauth/{provider}/callback", h.RequireSession(h.OAuthCallback))

	r.GET("/api/models/disabled", h.RequireSession(h.GetDisabledModels))
	r.POST("/api/models/disabled", h.RequireSession(h.PostDisabledModels))
	r.DELETE("/api/models/disabled", h.RequireSession(h.DeleteDisabledModels))

	// Console-logs SSE (real server log stream; mirrors usagestream SSE).
	r.GET("/api/console-logs/stream", h.RequireSession((&admin.ConsoleStreamHandler{Handlers: h}).ConsoleLogStream))

	// Translator (load sample + translate over the translation registry).
	r.GET("/api/translator/load", h.RequireSession(h.TranslatorLoad))
	r.POST("/api/translator/translate", h.RequireSession(h.TranslatorTranslate))

	// Models test/availability/custom (static before {id}).
	r.POST("/api/models/test", h.RequireSession(h.TestModel))
	r.GET("/api/models/availability", h.RequireSession(h.ModelAvailability))
	r.GET("/api/models/custom", h.RequireSession(h.ListCustomModels))
	r.POST("/api/models/custom", h.RequireSession(h.CreateCustomModel))
	r.DELETE("/api/models/custom/{id}", h.RequireSession(h.DeleteCustomModel))

	// Combos admin (w7-route-a, ESC-COMBOS Option A). The id-keyed admin combos
	// surface OWNS /api/combos[/{id}] serving the frozen UI shape
	// {id,name,strategy,steps[{provider,model}],is_active}; it REPLACES the
	// engine's {name,models[]} /api/combos HTTP routes (verified: only the admin
	// page consumes those routes — the /v1/models lister reads the store
	// directly). The engine combos store table + /v1/models lister stay intact,
	// fed by the admin handlers' best-effort mirror-write.
	r.GET("/api/combos", h.RequireSession(h.ListCombosAdmin))
	r.POST("/api/combos", h.RequireSession(h.CreateComboAdmin))
	r.GET("/api/combos/{id}", h.RequireSession(h.GetComboAdmin))
	r.PUT("/api/combos/{id}", h.RequireSession(h.UpdateComboAdmin))
	r.DELETE("/api/combos/{id}", h.RequireSession(h.DeleteComboAdmin))

	// Aliases admin CRUD (w7-route-a; static collection before {id}).
	r.GET("/api/aliases", h.RequireSession(h.ListAliases))
	r.POST("/api/aliases", h.RequireSession(h.CreateAlias))
	r.GET("/api/aliases/{id}", h.RequireSession(h.GetAlias))
	r.PUT("/api/aliases/{id}", h.RequireSession(h.UpdateAlias))
	r.DELETE("/api/aliases/{id}", h.RequireSession(h.DeleteAlias))

	// Routing-rules admin CRUD (w7-route-a; admin CRUD only — not yet applied to
	// live inference).
	r.GET("/api/routing-rules", h.RequireSession(h.ListRoutingRules))
	r.POST("/api/routing-rules", h.RequireSession(h.CreateRoutingRule))
	r.GET("/api/routing-rules/{id}", h.RequireSession(h.GetRoutingRule))
	r.PUT("/api/routing-rules/{id}", h.RequireSession(h.UpdateRoutingRule))
	r.DELETE("/api/routing-rules/{id}", h.RequireSession(h.DeleteRoutingRule))

	// Model-limits admin CRUD (w7-route-a; numeric INTEGER-PK ids, ESC-IDTYPE).
	r.GET("/api/model-limits", h.RequireSession(h.ListModelLimits))
	r.POST("/api/model-limits", h.RequireSession(h.CreateModelLimit))
	r.GET("/api/model-limits/{id}", h.RequireSession(h.GetModelLimit))
	r.PUT("/api/model-limits/{id}", h.RequireSession(h.UpdateModelLimit))
	r.DELETE("/api/model-limits/{id}", h.RequireSession(h.DeleteModelLimit))

	// Quota aggregation over per-connection usage (w7-route-a).
	r.GET("/api/quota", h.RequireSession((&admin.QuotaHandler{Handlers: h}).GetQuota))

	r.GET("/api/usage/stats", h.RequireSession(h.GetUsageStats))
	r.GET("/api/usage/chart", h.RequireSession(h.GetUsageChart))
	r.GET("/api/usage/request-logs", h.RequireSession(h.GetUsageRequestLogs))
	r.GET("/api/usage/logs", h.RequireSession(h.GetUsageRequestLogs))
	r.GET("/api/usage/request-details", h.RequireSession(h.GetRequestDetails))
	r.GET("/api/usage/stream", h.RequireSession((&admin.UsageStreamHandler{Handlers: h}).UsageStream))
	r.GET("/api/usage/{connectionId}", h.RequireSession((&admin.ConnectionUsageHandler{Handlers: h}).GetConnectionUsage))

	r.GET("/api/pricing", h.RequireSession(h.GetPricing))
	r.PATCH("/api/pricing", h.RequireSession(h.PatchPricing))
	r.DELETE("/api/pricing", h.RequireSession(h.DeletePricing))

	// Semantic cache admin (bf-core-2). Static collection: GET returns stats +
	// entry metadata (never full responses), DELETE clears it (audited). This is
	// the routes_admin serial-chain terminus (bf-mcp-1 -> bf-mcp-2 -> bf-core-2);
	// bf-core-2 releases to nobody.
	r.GET("/api/cache/semantic", h.RequireSession(h.GetSemanticCache))
	r.DELETE("/api/cache/semantic", h.RequireSession(h.ClearSemanticCache))

	// Public UI preference endpoint (no session required).
	r.POST("/api/locale", h.PostLocale)
}
