package api

import (
	"net/http"
	"testing"

	"github.com/valyala/fasthttp"
)

// TestRoutesHandlerCoverage exercises the closure bodies in routes.go by
// calling each route handler directly with a context that has the expected
// method and path. This covers the tiny glue code (requireMethod, switch,
// handler calls) that is not exercised by the handler unit tests because
// those call the handler functions directly instead of going through the
// router.
func TestRoutesHandlerCoverage(t *testing.T) {
	s := newAPITestStore(t)
	srv := NewServer(ServerConfig{Port: 0, Store: s})

	for _, r := range srv.routes() {
		// Skip inference and UI routes that are harder to set up safely.
		switch r.pattern {
		case "/v1/chat/completions", "/v1/messages", "/v1/responses",
			"/v1/embeddings", "/v1/images/generations",
			"/v1/audio/transcriptions", "/v1/audio/speech",
			"/api/traffic/stream", "/*":
			continue
		}

		t.Run(r.pattern, func(t *testing.T) {
			ctx := makeCtxWithRoutes(r.method, r.pattern)
			r.handler(ctx)
			// We only care that the handler runs without panic and covers
			// the closure body; status codes may vary.
		})
	}
}

// TestRoutesWrongMethodCoverage exercises the early-return branches inside
// route closures (requireMethod and switch-default) by calling handlers
// with the wrong HTTP method.
func TestRoutesWrongMethodCoverage(t *testing.T) {
	s := newAPITestStore(t)
	srv := NewServer(ServerConfig{Port: 0, Store: s})

	wrongMethodCases := []struct {
		pattern string
		method  string
	}{
		{"/api/providers/test-batch", http.MethodGet},
		{"/api/providers/:id/models/:model/test", http.MethodGet},
		{"/api/providers/:id/connections", http.MethodPost},
		{"/api/providers/:id/suggested-models", http.MethodPost},
		{"/api/providers/:id", http.MethodPost},
		{"/api/connections/:id/test", http.MethodPost},
		{"/api/mcp/instances/:id/auth/start", http.MethodGet},
		{"/api/mcp/instances/:id/accounts", http.MethodPost},
		{"/api/mcp/oauth/callback", http.MethodPost},
		{"/api/mcp/instances/:id/oauth/complete", http.MethodGet},
		{"/api/oauth/callback", http.MethodPost},
		{"/api/oauth/:provider/authorize", http.MethodGet},
		{"/api/oauth/:provider/poll", http.MethodPost},
		{"/api/oauth/:provider/exchange", http.MethodGet},
		{"/api/proxy-pools/batch", http.MethodGet},
		{"/api/proxy-pools/:id/test", http.MethodGet},
		{"/api/auth/setup", http.MethodGet},
		{"/api/auth/login", http.MethodGet},
		{"/api/auth/logout", http.MethodGet},
		{"/api/auth/status", http.MethodPost},
		{"/api/auth/password", http.MethodGet},
		{"/api/auth/users", http.MethodPut},
		{"/api/auth/users/:id", http.MethodGet},
	}

	for _, c := range wrongMethodCases {
		t.Run(c.method+" "+c.pattern, func(t *testing.T) {
			for _, r := range srv.routes() {
				if r.pattern == c.pattern {
					ctx := makeCtxWithRoutes(c.method, c.pattern)
					r.handler(ctx)
					// Should get 405 or 404, not panic.
					if ctx.Response.StatusCode() == 0 {
						t.Fatalf("handler for %s did not set status", c.pattern)
					}
					return
				}
			}
			t.Fatalf("route not found: %s", c.pattern)
		})
	}
}

// TestRoutesSwitchDefaultCoverage exercises the default branches of
// method-switch routes by calling them with unsupported methods.
func TestRoutesSwitchDefaultCoverage(t *testing.T) {
	s := newAPITestStore(t)
	srv := NewServer(ServerConfig{Port: 0, Store: s})

	switchCases := []struct {
		pattern string
		method  string
	}{
		// /api/models/disabled supports GET, POST, DELETE
		{"/api/models/disabled", http.MethodPut},
		// /api/models/custom supports GET, POST
		{"/api/models/custom", http.MethodDelete},
		// /api/proxy-pools supports GET, POST
		{"/api/proxy-pools", http.MethodDelete},
		// /api/proxy-pools/:id supports GET, PUT, DELETE
		{"/api/proxy-pools/:id", http.MethodPost},
	}

	for _, c := range switchCases {
		t.Run(c.method+" "+c.pattern, func(t *testing.T) {
			for _, r := range srv.routes() {
				if r.pattern == c.pattern {
					ctx := makeCtxWithRoutes(c.method, c.pattern)
					r.handler(ctx)
					if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
						t.Fatalf("got %d, want 405", ctx.Response.StatusCode())
					}
					return
				}
			}
			t.Fatalf("route not found: %s", c.pattern)
		})
	}
}

// TestRoutesSwitchBranchCoverage exercises one remaining switch branch in
// routes.go (calling a handler with its supported method through the route
// closure). Each call covers a single statement in the switch body.
func TestRoutesSwitchBranchCoverage(t *testing.T) {
	s := newAPITestStore(t)
	srv := NewServer(ServerConfig{Port: 0, Store: s})

	cases := []struct {
		pattern string
		method  string
	}{
		{"/api/models/disabled", fasthttp.MethodPost},
		{"/api/models/disabled", fasthttp.MethodDelete},
		{"/api/models/custom", fasthttp.MethodPost},
		{"/api/proxy-pools", fasthttp.MethodPost},
		{"/api/proxy-pools/:id", fasthttp.MethodPut},
		{"/api/proxy-pools/:id", fasthttp.MethodDelete},
	}

	for _, c := range cases {
		t.Run(c.method+" "+c.pattern, func(t *testing.T) {
			for _, r := range srv.routes() {
				if r.pattern == c.pattern {
					ctx := makeCtxWithRoutes(c.method, c.pattern)
					r.handler(ctx)
					return
				}
			}
			t.Fatalf("route not found: %s", c.pattern)
		})
	}
}

func makeCtxWithRoutes(method, path string) *fasthttp.RequestCtx {
	var req fasthttp.Request
	req.Header.SetMethod(method)
	req.SetRequestURI(path)
	ctx := &fasthttp.RequestCtx{}
	ctx.Init(&req, nil, nil)
	return ctx
}
