package api

import (
	"crypto/rand"
	"fmt"
	"net/url"
	"strings"

	"github.com/valyala/fasthttp"
)

type APIKeyValidator interface {
	ValidateAPIKey(key, secret string) (bool, error)
}

func (s *Server) applyMiddleware(ctx *fasthttp.RequestCtx) bool {
	requestID, err := newRequestID()
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return false
	}

	ctx.Response.Header.Set("X-Request-ID", requestID)
	s.applyCORS(ctx)

	if string(ctx.Method()) == fasthttp.MethodOptions {
		ctx.SetStatusCode(fasthttp.StatusNoContent)
		return false
	}

	if s.requiresAuth(ctx) {
		ok, err := s.validAPIKey(ctx)
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			return false
		}
		if !ok {
			ctx.SetStatusCode(fasthttp.StatusUnauthorized)
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
	if !s.config.RequireAPIKey {
		return false
	}
	requestPath := strings.TrimRight(string(ctx.Path()), "/")
	if requestPath == "" {
		requestPath = "/"
	}
	return strings.HasPrefix(requestPath, "/v1/") || isProtectedManagementPath(requestPath)
}

func isProtectedManagementPath(requestPath string) bool {
	if requestPath == "/api/oauth/callback" || requestPath == "/api/mcp/oauth/callback" {
		return false
	}
	return requestPath == "/api" || strings.HasPrefix(requestPath, "/api/")
}

func (s *Server) validAPIKey(ctx *fasthttp.RequestCtx) (bool, error) {
	if s.config.APIKeyValidator == nil {
		return false, nil
	}

	key := bearerToken(string(ctx.Request.Header.Peek("Authorization")))
	if key == "" {
		key = string(ctx.Request.Header.Peek("X-API-Key"))
	}
	if key == "" {
		return false, nil
	}

	ok, err := s.config.APIKeyValidator.ValidateAPIKey(key, s.config.APIKeySecret)
	if err != nil {
		return false, fmt.Errorf("validate api key: %w", err)
	}
	return ok, nil
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
