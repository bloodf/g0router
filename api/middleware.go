package api

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type APIKeyValidator interface {
	ValidateAPIKey(key, secret string) (bool, error)
}

type APIKeyIdentity struct {
	ID               string
	ExpiresAt        *int64
	Scopes           []string
	RateLimitRPM     *int
	RateLimitTPM     *int
	DailySpendCapUSD *float64
}

// Expired reports whether the key's expiry has passed relative to now.
func (i APIKeyIdentity) Expired(now time.Time) bool {
	return i.ExpiresAt != nil && now.Unix() >= *i.ExpiresAt
}

type APIKeyIdentityValidator interface {
	ValidateAPIKeyIdentity(key, secret string) (*APIKeyIdentity, bool, error)
}

const (
	requestIDHeader          = "X-Request-ID"
	requestAuthTypeKey       = "g0router.auth_type"
	requestAPIKeyIDKey       = "g0router.api_key_id"
	requestAPIKeyPolicyKey   = "g0router.api_key_policy"
	requestSessionUserIDKey  = "g0router.session_user_id"
	requestSessionRoleKey    = "g0router.session_role"
	requestVirtualKeyIDKey   = "g0router.virtual_key_id"
	requestVirtualKeyTeamIDKey = "g0router.virtual_key_team_id"
	requestAuthTypeAPIKey    = "api_key"
	requestAuthTypeSession   = "session"
)

func (s *Server) applyMiddleware(ctx *fasthttp.RequestCtx) bool {
	requestID, err := newRequestID()
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return false
	}

	ctx.Response.Header.Set(requestIDHeader, requestID)
	s.applyCORS(ctx)

	if string(ctx.Method()) == fasthttp.MethodOptions {
		ctx.SetStatusCode(fasthttp.StatusNoContent)
		return false
	}

	if !s.sourceAllowed(ctx) {
		ctx.SetStatusCode(fasthttp.StatusForbidden)
		return false
	}

	if s.requiresAuth(ctx) {
		ok, err := s.validAPIKey(ctx)
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			return false
		}
		if !ok {
			// validAPIKey may have already written a specific status (403/429)
			// for a virtual key governance rejection. Do not overwrite it.
			if ctx.Response.StatusCode() != fasthttp.StatusOK {
				return false
			}
			ok, err = s.validSession(ctx)
			if err != nil {
				ctx.SetStatusCode(fasthttp.StatusInternalServerError)
				return false
			}
			if !ok {
				ctx.SetStatusCode(fasthttp.StatusUnauthorized)
				return false
			}
		}
	}

	if s.requiresCSRFCheck(ctx) {
		if !s.originMatchesHost(ctx) {
			ctx.SetStatusCode(fasthttp.StatusForbidden)
			return false
		}
	}

	return true
}

func (s *Server) applyCORS(ctx *fasthttp.RequestCtx) {
	if origin := string(ctx.Request.Header.Peek("Origin")); isAllowedLocalOrigin(origin) {
		ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
	}
	ctx.Response.Header.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-API-Key")
	ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
}

func isAllowedLocalOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	switch parsed.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func (s *Server) requiresAuth(ctx *fasthttp.RequestCtx) bool {
	requestPath := strings.TrimRight(string(ctx.Path()), "/")
	if requestPath == "" {
		requestPath = "/"
	}
	// The inference proxy is always protected: a valid API key is mandatory for
	// every /v1/* request regardless of RequireAPIKey. With no keys minted,
	// validAPIKey returns false and the proxy answers 401.
	if strings.HasPrefix(requestPath, "/v1/") {
		return true
	}
	if isExemptRoute(requestPath) {
		return false
	}
	// When require_login is enabled, the management plane and metrics require
	// authentication (bearer or session).
	if s.runtimeSettings().RequireLogin {
		return requestPath == "/api" || strings.HasPrefix(requestPath, "/api/") || requestPath == "/metrics"
	}
	// The management plane stays gated by the RequireAPIKey toggle.
	if !s.config.RequireAPIKey {
		return false
	}
	return isProtectedManagementPath(requestPath)
}

func isExemptRoute(requestPath string) bool {
	switch requestPath {
	case "/api/oauth/callback", "/api/mcp/oauth/callback", "/api/auth/setup", "/api/auth/login", "/api/auth/status":
		return true
	}
	return false
}

func isProtectedManagementPath(requestPath string) bool {
	if isExemptRoute(requestPath) {
		return false
	}
	// /metrics is a management path: scrapers pass the API key via bearer just
	// like /api/* clients.
	if requestPath == "/metrics" {
		return true
	}
	return requestPath == "/api" || strings.HasPrefix(requestPath, "/api/")
}

// sourceAllowed enforces the connection-source policy for /v1/* and /api/*
// requests. /healthz and any other path bypass the check so health and
// orchestrator probes always work. The client IP is taken from the transport
// (ctx.RemoteIP), never from X-Forwarded-For.
func (s *Server) sourceAllowed(ctx *fasthttp.RequestCtx) bool {
	requestPath := strings.TrimRight(string(ctx.Path()), "/")
	if requestPath == "" {
		requestPath = "/"
	}
	if !strings.HasPrefix(requestPath, "/v1/") && requestPath != "/api" && !strings.HasPrefix(requestPath, "/api/") && requestPath != "/metrics" {
		return true
	}

	allowed := s.runtimeSettings().AllowedSources
	if len(allowed) == 0 {
		return true
	}

	class := classifySourceIP(ctx.RemoteIP())
	for _, source := range allowed {
		// "public" is a superset that permits every class.
		if source == "public" || source == class {
			return true
		}
	}
	return false
}

// classifySourceIP maps a client IP to a connection-source class: "local"
// (loopback), "tailscale" (CGNAT 100.64.0.0/10), "lan" (private/link-local), or
// "public" (everything else).
func classifySourceIP(ip net.IP) string {
	if ip == nil {
		return "public"
	}
	if ip.IsLoopback() {
		return "local"
	}
	if tailscaleCGNAT.Contains(ip) {
		return "tailscale"
	}
	if ip.IsPrivate() || ip.IsLinkLocalUnicast() {
		return "lan"
	}
	return "public"
}

// tailscaleCGNAT is the 100.64.0.0/10 Carrier-Grade NAT range Tailscale assigns
// to nodes.
var _, tailscaleCGNAT, _ = net.ParseCIDR("100.64.0.0/10")

func (s *Server) validAPIKey(ctx *fasthttp.RequestCtx) (bool, error) {
	key := bearerToken(string(ctx.Request.Header.Peek("Authorization")))
	if key == "" {
		key = string(ctx.Request.Header.Peek("X-API-Key"))
	}
	if key == "" {
		return false, nil
	}

	// Virtual keys are checked before regular API keys (prefix-distinguished: gvk-).
	if strings.HasPrefix(key, "gvk-") {
		return s.validVirtualKey(ctx, key)
	}

	if s.config.APIKeyValidator == nil {
		return false, nil
	}

	if identityValidator, ok := s.config.APIKeyValidator.(APIKeyIdentityValidator); ok {
		identity, ok, err := identityValidator.ValidateAPIKeyIdentity(key, s.config.APIKeySecret)
		if err != nil {
			return false, fmt.Errorf("validate api key: %w", err)
		}
		if ok {
			// Expired keys are rejected as if invalid: no auth context is set
			// and the request is denied (401 via the caller).
			if identity != nil && identity.Expired(time.Now()) {
				return false, nil
			}
			ctx.SetUserValue(requestAuthTypeKey, requestAuthTypeAPIKey)
			if identity != nil && identity.ID != "" {
				ctx.SetUserValue(requestAPIKeyIDKey, identity.ID)
			}
			if identity != nil {
				ctx.SetUserValue(requestAPIKeyPolicyKey, *identity)
			}
		}
		return ok, nil
	}

	ok, err := s.config.APIKeyValidator.ValidateAPIKey(key, s.config.APIKeySecret)
	if err != nil {
		return false, fmt.Errorf("validate api key: %w", err)
	}
	if ok {
		ctx.SetUserValue(requestAuthTypeKey, requestAuthTypeAPIKey)
	}
	return ok, nil
}

func (s *Server) validVirtualKey(ctx *fasthttp.RequestCtx, raw string) (bool, error) {
	if s.config.Store == nil || s.config.Governance == nil {
		return false, nil
	}
	key, ok, err := s.config.Store.ValidateVirtualKey(raw)
	if err != nil {
		return false, fmt.Errorf("validate virtual key: %w", err)
	}
	if !ok {
		return false, nil
	}
	result := s.config.Governance.Check(key)
	if !result.Allowed {
		writePolicyError(ctx, result.Status, result.Reason)
		return false, nil
	}
	ctx.SetUserValue(requestAuthTypeKey, requestAuthTypeAPIKey)
	ctx.SetUserValue(requestVirtualKeyIDKey, strconv.FormatInt(key.ID, 10))
	if key.TeamID != nil {
		ctx.SetUserValue(requestVirtualKeyTeamIDKey, strconv.FormatInt(*key.TeamID, 10))
	}
	return true, nil
}

func (s *Server) validSession(ctx *fasthttp.RequestCtx) (bool, error) {
	if s.config.Store == nil {
		return false, nil
	}
	rawToken := string(ctx.Request.Header.Cookie("g0router_session"))
	if rawToken == "" {
		return false, nil
	}
	session, err := s.config.Store.GetDashboardSessionByRawToken(rawToken)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("get dashboard session: %w", err)
	}
	expiresAt, err := time.Parse(time.RFC3339, session.ExpiresAt)
	if err != nil {
		return false, nil
	}
	if time.Now().UTC().After(expiresAt) {
		return false, nil
	}
	user, err := s.config.Store.GetDashboardUser(strconv.FormatInt(session.UserID, 10))
	if err != nil {
		return false, nil
	}
	ctx.SetUserValue(requestAuthTypeKey, requestAuthTypeSession)
	ctx.SetUserValue(requestSessionUserIDKey, strconv.FormatInt(session.UserID, 10))
	ctx.SetUserValue(requestSessionRoleKey, user.Role)
	if err := s.config.Store.TouchDashboardSession(session.TokenHash); err != nil {
		return false, fmt.Errorf("touch dashboard session: %w", err)
	}
	return true, nil
}

func (s *Server) requiresCSRFCheck(ctx *fasthttp.RequestCtx) bool {
	authType, _ := ctx.UserValue(requestAuthTypeKey).(string)
	if authType != requestAuthTypeSession {
		return false
	}
	method := string(ctx.Method())
	switch method {
	case fasthttp.MethodPost, fasthttp.MethodPut, fasthttp.MethodDelete, fasthttp.MethodPatch:
		return true
	}
	return false
}

func (s *Server) originMatchesHost(ctx *fasthttp.RequestCtx) bool {
	origin := string(ctx.Request.Header.Peek("Origin"))
	if origin == "" {
		origin = string(ctx.Request.Header.Peek("Referer"))
	}
	if origin == "" {
		return false
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return parsed.Host == string(ctx.Request.Header.Peek("Host"))
}

func (s *Server) clientIP(ctx *fasthttp.RequestCtx) string {
	if s.runtimeSettings().TrustProxyHeaders {
		xff := string(ctx.Request.Header.Peek("X-Forwarded-For"))
		if xff != "" {
			if idx := strings.Index(xff, ","); idx != -1 {
				return strings.TrimSpace(xff[:idx])
			}
			return strings.TrimSpace(xff)
		}
	}
	return ctx.RemoteIP().String()
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func newRequestID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate request id: %w", err)
	}

	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		b[0:4],
		b[4:6],
		b[6:8],
		b[8:10],
		b[10:16],
	), nil
}
