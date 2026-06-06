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
		"GET /healthz",
		"GET /metrics",
		"POST /v1/chat/completions",
		"POST /v1/messages",
		"POST /v1/responses",
		"POST /v1/embeddings",
		"POST /v1/images/generations",
		"POST /v1/audio/transcriptions",
		"POST /v1/audio/speech",
		"GET /v1/models",
		"/api/providers",
		"/api/providers/:provider/models",
		"/api/connections",
		"/api/connections/:id",
		"/api/connections/:id/test",
		"/api/settings",
		"/api/keys",
		"/api/keys/:id",
		"/api/combos",
		"/api/combos/:id",
		"/api/aliases",
		"/api/aliases/:id",
		"/api/pricing",
		"/api/pricing/:provider/:model",
		"GET /api/oauth/callback",
		"POST /api/oauth/:provider/authorize",
		"GET /api/oauth/:provider/poll",
		"POST /api/oauth/:provider/exchange",
		"/api/usage",
		"/api/usage/summary",
		"/api/usage/quota/*",
		"/api/logs",
		"/api/audit",
		"/api/traffic/stream",
		"/api/mcp/clients",
		"/api/mcp/clients/:id",
		"/api/mcp/instances",
		"/api/mcp/instances/:id",
		"POST /api/mcp/instances/:id/auth/start",
		"GET /api/mcp/instances/:id/accounts",
		"/api/mcp/tools",
		"/api/mcp/tools/:id/execute",
		"GET /api/mcp/oauth/callback",
		"POST /api/mcp/instances/:id/oauth/complete",
		"POST /api/auth/setup",
		"POST /api/auth/login",
		"POST /api/auth/logout",
		"GET /api/auth/status",
		"/*",
	}
	sort.Strings(want)

	if len(gotPairs) != len(want) {
		t.Fatalf("route count mismatch: got %d, want %d\ngot:\n%v\nwant:\n%v", len(gotPairs), len(want), gotPairs, want)
	}
	for i := range gotPairs {
		if gotPairs[i] != want[i] {
			t.Fatalf("route mismatch at %d:\ngot:  %q\nwant: %q", i, gotPairs[i], want[i])
		}
	}
}
