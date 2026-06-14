package server

import (
	"time"

	"github.com/bloodf/g0router/internal/admin"
	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/store"
	"github.com/fasthttp/router"
)

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
	}
	h := admin.New(st, sessions, flows)
	stats, resolver := admin.BuildUsageServices(st, deps)
	h.SetUsageServices(stats, resolver)
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

	r.GET("/api/provider-nodes", h.RequireSession(h.ListProviderNodes))
	r.POST("/api/provider-nodes", h.RequireSession(h.CreateProviderNode))
	r.POST("/api/provider-nodes/validate", h.RequireSession(h.ValidateProviderNode))

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

	r.GET("/api/oauth/{provider}/start", h.RequireSession(h.OAuthStart))
	r.POST("/api/oauth/{provider}/callback", h.RequireSession(h.OAuthCallback))

	r.GET("/api/models/disabled", h.RequireSession(h.GetDisabledModels))
	r.POST("/api/models/disabled", h.RequireSession(h.PostDisabledModels))
	r.DELETE("/api/models/disabled", h.RequireSession(h.DeleteDisabledModels))

	r.GET("/api/combos", h.RequireSession(h.ListCombos))
	r.POST("/api/combos", h.RequireSession(h.CreateCombo))
	r.PUT("/api/combos/{name}", h.RequireSession(h.UpdateCombo))
	r.DELETE("/api/combos/{name}", h.RequireSession(h.DeleteCombo))

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

	// Public UI preference endpoint (no session required).
	r.POST("/api/locale", h.PostLocale)
}
