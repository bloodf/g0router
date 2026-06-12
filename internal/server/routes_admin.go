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
func NewAdminHandlers(st *store.Store) *admin.Handlers {
	sessions := auth.NewSessions(st, sessionTTL)
	flows := map[string]*auth.OAuthFlow{
		"anthropic": auth.NewOAuthFlow(auth.AnthropicOAuth(), st, nil),
	}
	h := admin.New(st, sessions, flows)
	stats, resolver := admin.BuildUsageServices(st)
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

	r.GET("/api/pricing", h.RequireSession(h.GetPricing))
	r.PATCH("/api/pricing", h.RequireSession(h.PatchPricing))
	r.DELETE("/api/pricing", h.RequireSession(h.DeletePricing))
}
