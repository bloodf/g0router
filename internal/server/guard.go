package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// sessionCookieName must stay in sync with internal/admin/auth.go.
const sessionCookieName = "g0_session"

// Public API paths — no auth required (LLM API has its own key auth inside handler).
// Ported verbatim from dashboardGuard.js:22-32.
var PUBLIC_API_PATHS = []string{
	"/api/health",
	"/api/init",
	"/api/locale",
	"/api/auth/login",
	"/api/auth/logout",
	"/api/auth/status",
	"/api/auth/oidc",
	"/api/version",
	"/api/settings/require-login",
}

// Public top-level prefixes (LLM API endpoints with their own API key auth).
// In this plan they pass through unchanged; w3-d adds gating + validators.
var PUBLIC_LLM_PREFIXES = []string{"/v1", "/v1beta", "/api/v1", "/api/v1beta"}

// Always require a valid session or CLI token regardless of requireLogin setting.
// Stage-1 list is empty because none of the referenced routes exist in g0router yet.
// Full ref set from dashboardGuard.js:38-45:
//   "/api/shutdown", "/api/settings/database", "/api/version/shutdown",
//   "/api/version/update", "/api/oauth/cursor/auto-import", "/api/oauth/kiro/auto-import".
var ALWAYS_PROTECTED = []string{}

// Routes that spawn child processes or read host secrets — restrict to localhost.
// Stage-1 list contains only routes that exist today. Full ref set from
// dashboardGuard.js:69-81 includes tunnel/cli-tools/oauth entries added by later plans.
var LOCAL_ONLY_PATHS = []string{
	"/api/mcp/",
}

var loopbackHosts = map[string]bool{
	"localhost": true,
	"127.0.0.1": true,
	"::1":       true,
}

// settingsReader loads dashboard settings; satisfied by *store.Store.
type settingsReader interface {
	GetSettings() (map[string]string, error)
}

// sessionValidator is satisfied by *auth.Sessions; the interface keeps unit tests
// lightweight and free of mocks.
type sessionValidator interface {
	Validate(token string) (*store.User, error)
}

// Guard centralizes dashboard and /api authorization before route dispatch.
type Guard struct {
	Sessions          sessionValidator
	Settings          settingsReader
	CLITokenValidator func(*fasthttp.RequestCtx) bool
	APIKeyValidator   func(*fasthttp.RequestCtx) bool
}

// Wrap returns a handler that enforces the guard evaluation order and then calls next.
func (g *Guard) Wrap(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())

		// 1. Local-only gate for spawn-capable / host-secret routes.
		for _, p := range LOCAL_ONLY_PATHS {
			if strings.HasPrefix(path, p) {
				if g.cliTokenOK(ctx) || (isLocalRequest(ctx) && g.isAuthenticated(ctx)) {
					next(ctx)
					return
				}
				writeError(ctx, fasthttp.StatusForbidden, "Local only: CLI token required")
				return
			}
		}

		// 2. Always protected - require valid session or CLI token.
		for _, p := range ALWAYS_PROTECTED {
			if strings.HasPrefix(path, p) {
				if g.cliTokenOK(ctx) || g.sessionOK(ctx) {
					next(ctx)
					return
				}
				writeError(ctx, fasthttp.StatusUnauthorized, "Unauthorized")
				return
			}
		}

		// 3. Public LLM API - loopback is keyless; remote needs CLI token or API key.
		if isPublicLlmApi(path) {
			if isLocalRequest(ctx) {
				next(ctx)
				return
			}
			if g.cliTokenOK(ctx) {
				next(ctx)
				return
			}
			if g.apiKeyOK(ctx) {
				next(ctx)
				return
			}
			writeError(ctx, fasthttp.StatusUnauthorized, "API key required for remote API access")
			return
		}

		// 4. Deny-by-default for /api/* - public allow-list bypasses, everything else requires auth.
		if strings.HasPrefix(path, "/api/") {
			if isPublicApi(path) {
				next(ctx)
				return
			}
			if g.cliTokenOK(ctx) || g.isAuthenticated(ctx) {
				next(ctx)
				return
			}
			writeError(ctx, fasthttp.StatusUnauthorized, "Unauthorized")
			return
		}

		// 5. Protect all dashboard routes.
		if strings.HasPrefix(path, "/dashboard") {
			settings, _ := g.loadSettings()
			requireLogin := settings["requireLogin"] != "false"
			tunnelDashboardAccess := settings["tunnelDashboardAccess"] == "true"

			if !tunnelDashboardAccess {
				host := hostName(string(ctx.Host()))
				tunnelHost := urlHostname(settings["tunnelUrl"])
				tailscaleHost := urlHostname(settings["tailscaleUrl"])
				if (tunnelHost != "" && host == tunnelHost) || (tailscaleHost != "" && host == tailscaleHost) {
					ctx.Redirect("/login", fasthttp.StatusFound)
					return
				}
			}

			if !requireLogin {
				next(ctx)
				return
			}

			if g.sessionOK(ctx) {
				next(ctx)
				return
			}
			ctx.Redirect("/login", fasthttp.StatusFound)
			return
		}

		// 6. Redirect / to /dashboard.
		if path == "/" {
			ctx.Redirect("/dashboard", fasthttp.StatusFound)
			return
		}

		next(ctx)
	}
}

func (g *Guard) cliTokenOK(ctx *fasthttp.RequestCtx) bool {
	if g.CLITokenValidator == nil {
		return false
	}
	return g.CLITokenValidator(ctx)
}

func (g *Guard) apiKeyOK(ctx *fasthttp.RequestCtx) bool {
	if g.APIKeyValidator == nil {
		return false
	}
	return g.APIKeyValidator(ctx)
}

func (g *Guard) sessionOK(ctx *fasthttp.RequestCtx) bool {
	if g.Sessions == nil {
		return false
	}
	_, err := g.Sessions.Validate(requestToken(ctx))
	return err == nil
}

func (g *Guard) isAuthenticated(ctx *fasthttp.RequestCtx) bool {
	if g.sessionOK(ctx) {
		return true
	}
	settings, err := g.loadSettings()
	if err == nil && settings["requireLogin"] == "false" {
		return true
	}
	return false
}

func (g *Guard) loadSettings() (map[string]string, error) {
	if g.Settings == nil {
		return map[string]string{}, nil
	}
	return g.Settings.GetSettings()
}

// requestToken extracts the opaque session token from the Authorization header
// or the session cookie, matching the contract used by internal/admin/auth.go.
func requestToken(ctx *fasthttp.RequestCtx) string {
	header := string(ctx.Request.Header.Peek("Authorization"))
	if after, ok := strings.CutPrefix(header, "Bearer "); ok && after != "" {
		return after
	}
	return string(ctx.Request.Header.Cookie(sessionCookieName))
}

// urlHostname extracts the hostname from a URL setting value (e.g. https://host:port).
func urlHostname(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func isLoopbackHostname(h string) bool {
	if h == "" {
		return false
	}
	return loopbackHosts[h]
}

func hostName(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		return strings.ToLower(h)
	}
	h := strings.ToLower(host)
	h = strings.TrimPrefix(h, "[")
	h = strings.TrimSuffix(h, "]")
	return h
}

func isLocalRequest(ctx *fasthttp.RequestCtx) bool {
	if !isLoopbackHostname(hostName(string(ctx.Host()))) {
		return false
	}
	origin := string(ctx.Request.Header.Peek("Origin"))
	if origin != "" {
		host, err := originHostname(origin)
		if err != nil || !isLoopbackHostname(host) {
			return false
		}
	}
	return true
}

func originHostname(origin string) (string, error) {
	u, err := url.Parse(origin)
	if err != nil {
		return "", fmt.Errorf("parse origin: %w", err)
	}
	return u.Hostname(), nil
}

func isPublicLlmApi(path string) bool {
	for _, p := range PUBLIC_LLM_PREFIXES {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}

func isPublicApi(path string) bool {
	if isPublicLlmApi(path) {
		return true
	}
	for _, p := range PUBLIC_API_PATHS {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}

func writeError(ctx *fasthttp.RequestCtx, status int, message string) {
	b, err := json.Marshal(map[string]any{
		"data":  nil,
		"error": map[string]string{"message": message},
	})
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentType("application/json")
		ctx.SetBodyString(`{"data":null,"error":{"message":"encode response"}}`)
		return
	}
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json")
	ctx.SetBody(b)
}
