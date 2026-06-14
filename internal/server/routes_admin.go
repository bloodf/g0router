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

	// Protected.
	r.POST("/api/auth/logout", h.RequireSession(h.Logout))
	r.GET("/api/auth/me", h.RequireSession(h.Me))

	r.GET("/api/settings", h.RequireSession(h.GetSettings))
	r.PUT("/api/settings", h.RequireSession(h.PutSettings))

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
