package api

import (
	"sort"
	"testing"
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
		"/api/keys",
		"/api/keys/:id",
		"/api/logs",
		"/api/mcp/clients",
		"/api/mcp/clients/:id",
		"/api/mcp/instances",
		"/api/mcp/instances/:id",
		"/api/mcp/tools",
		"/api/mcp/tools/:id/execute",
		"/api/models/custom",
		"/api/models/custom/:id",
		"/api/models/disabled",
		"/api/pricing",
		"/api/pricing/:provider/:model",
		"/api/providers",
		"/api/providers/:id",
		"/api/providers/:id/connections",
		"/api/providers/:id/models/:model/test",
		"/api/providers/:id/suggested-models",
		"/api/providers/:provider/models",
		"/api/proxy-pools",
		"/api/proxy-pools/:id",
		"/api/settings",
		"/api/traffic/stream",
		"/api/usage",
		"/api/usage/quota/*",
		"/api/usage/summary",
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
		"POST /api/mcp/instances/:id/auth/start",
		"POST /api/mcp/instances/:id/oauth/complete",
		"POST /api/oauth/:provider/authorize",
		"POST /api/oauth/:provider/exchange",
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
