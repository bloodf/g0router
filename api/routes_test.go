package api

import (
	"sort"
	"testing"

	"github.com/bloodf/g0router/internal/proxy"
	"github.com/valyala/fasthttp"
)

func TestRoutesSnapshot(t *testing.T) {
	s := NewServer(ServerConfig{})

	got := s.routes()
	var gotPairs []string
	for _, r := range got {
		if r.method != "" {
			gotPairs = append(gotPairs, r.method+" "+r.pattern)
		} else {
			gotPairs = append(gotPairs, r.pattern)
		}
	}
	sort.Strings(gotPairs)

	want := []string{
		"/*",
		"/api/aliases",
		"/api/aliases/:id",
		"/api/audit",
		"/api/auth/users/:id",
		"/api/chat-sessions",
		"/api/chat-sessions/:id",
		"/api/combos",
		"/api/combos/:id",
		"/api/connections",
		"/api/connections/:id",
		"/api/connections/:id/test",
		"/api/console-logs",
		"/api/console-logs/stream",
		"/api/guardrails",
		"/api/keys",
		"/api/keys/:id",
		"/api/logs",
		"/api/mcp/clients",
		"/api/mcp/clients/:id",
		"/api/mcp/instances",
		"/api/mcp/instances/:id",
		"/api/mcp/tools",
		"/api/mcp/tools/:id/execute",
		"/api/model-limits",
		"/api/model-limits/:id",
		"/api/models/custom",
		"/api/models/custom/:id",
		"/api/models/disabled",
		"/api/pricing",
		"/api/pricing/:provider/:model",
		"/api/prompt-templates",
		"/api/prompt-templates/:id",
		"/api/providers",
		"/api/providers/:id",
		"/api/providers/:id/connections",
		"/api/providers/:id/models/:model/test",
		"/api/providers/:id/suggested-models",
		"/api/providers/:provider/models",
		"/api/proxy-pools",
		"/api/proxy-pools/:id",
		"/api/routing-rules",
		"/api/routing-rules/:id",
		"/api/settings",
		"/api/teams",
		"/api/teams/:id",
		"/api/traffic/stream",
		"/api/usage",
		"/api/usage/chart",
		"/api/usage/quota/*",
		"/api/usage/summary",
		"/api/virtual-keys",
		"/api/virtual-keys/:id",
		"DELETE /api/tunnels/cloudflare",
		"DELETE /api/tunnels/tailscale",
		"GET /api/auth/status",
		"GET /api/auth/users",
		"GET /api/mcp/instances/:id/accounts",
		"GET /api/mcp/oauth/callback",
		"GET /api/oauth/:provider/poll",
		"GET /api/oauth/callback",
		"GET /api/tunnels",
		"GET /api/tunnels/health",
		"GET /healthz",
		"GET /metrics",
		"GET /v1/models",
		"POST /api/auth/login",
		"POST /api/auth/logout",
		"POST /api/auth/setup",
		"POST /api/auth/users",
		"POST /api/connections/bulk-disable",
		"POST /api/connections/bulk-enable",
		"POST /api/guardrails/test",
		"POST /api/mcp/instances/:id/auth/start",
		"POST /api/mcp/instances/:id/oauth/complete",
		"POST /api/oauth/:provider/authorize",
		"POST /api/oauth/:provider/exchange",
		"POST /api/prompt-templates/test",
		"POST /api/providers/test-batch",
		"POST /api/proxy-pools/:id/test",
		"POST /api/proxy-pools/batch",
		"POST /api/settings/proxy-test",
		"POST /api/tunnels/cloudflare",
		"POST /api/tunnels/tailscale",
		"POST /v1/audio/speech",
		"POST /v1/audio/transcriptions",
		"POST /v1/chat/completions",
		"POST /v1/embeddings",
		"POST /v1/images/generations",
		"POST /v1/messages",
		"POST /v1/responses",
		"PUT /api/auth/password",
	}

	if len(gotPairs) != len(want) {
		t.Fatalf("route count mismatch: got %d, want %d\ngot:\n%v\nwant:\n%v", len(gotPairs), len(want), gotPairs, want)
	}
	for i := range gotPairs {
		if gotPairs[i] != want[i] {
			t.Fatalf("route mismatch at %d:\ngot:  %q\nwant: %q", i, gotPairs[i], want[i])
		}
	}
}

type fakeEngineForInvalidation struct {
	proxy.Engine
	invalidated bool
}

func (f *fakeEngineForInvalidation) InvalidateRoutingRules() {
	f.invalidated = true
}

func TestWithRoutingInvalidation(t *testing.T) {
	eng := &fakeEngineForInvalidation{}
	s := NewServer(ServerConfig{InferenceEngine: eng})

	cases := []struct {
		method      string
		status      int
		wantInvalid bool
	}{
		{fasthttp.MethodPost, 201, true},
		{fasthttp.MethodPut, 200, true},
		{fasthttp.MethodDelete, 204, true},
		{fasthttp.MethodPost, 400, false},
		{fasthttp.MethodGet, 200, false},
	}

	for _, tc := range cases {
		eng.invalidated = false
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.Header.SetMethod(tc.method)
		ctx.Response.SetStatusCode(tc.status)

		handlerCalled := false
		s.withRoutingInvalidation(func(ctx *fasthttp.RequestCtx) {
			handlerCalled = true
		})(ctx)

		if !handlerCalled {
			t.Fatalf("%s: handler not called", tc.method)
		}
		if eng.invalidated != tc.wantInvalid {
			t.Fatalf("%s status=%d: invalidated=%v, want %v", tc.method, tc.status, eng.invalidated, tc.wantInvalid)
		}
	}
}
