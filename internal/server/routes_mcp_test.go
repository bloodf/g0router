package server

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/admin"
	"github.com/bloodf/g0router/internal/auth"
	httprouter "github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

// newMCPRouteHandlers builds an admin.Handlers over a fresh test store for the
// /mcp route-registration tests.
func newMCPRouteHandlers(t *testing.T) *admin.Handlers {
	t.Helper()
	st := newTestStore(t)
	sessions := auth.NewSessions(st, time.Hour)
	return admin.New(st, sessions, nil)
}

// TestMCPRoutesRegistered proves RegisterMCPRoutes wires POST /mcp + GET /mcp,
// and that POST /mcp returns a RAW JSON-RPC body (not the {data,error} admin
// envelope).
func TestMCPRoutesRegistered(t *testing.T) {
	r := httprouter.New()
	r.NotFound = func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString("not found")
	}
	RegisterMCPRoutes(r, newMCPRouteHandlers(t))

	// POST /mcp -> raw JSON-RPC tools/list.
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetRequestURI("/mcp")
	ctx.Request.SetBody([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	r.Handler(&ctx)
	if ctx.Response.StatusCode() == fasthttp.StatusNotFound {
		t.Fatalf("POST /mcp not registered (404)")
	}
	body := string(ctx.Response.Body())
	if strings.Contains(body, `"data"`) || strings.Contains(body, `"error":null`) {
		t.Fatalf("POST /mcp returned the {data,error} envelope, want raw JSON-RPC: %s", body)
	}
	var m map[string]any
	if err := json.Unmarshal(ctx.Response.Body(), &m); err != nil {
		t.Fatalf("POST /mcp body not JSON: %v\n%s", err, body)
	}
	if m["jsonrpc"] != "2.0" {
		t.Fatalf("POST /mcp not JSON-RPC 2.0: %v", m)
	}

	// GET /mcp registered (heartbeat SSE stream).
	var getCtx fasthttp.RequestCtx
	getCtx.Request.Header.SetMethod("GET")
	getCtx.Request.SetRequestURI("/mcp")
	r.Handler(&getCtx)
	if getCtx.Response.StatusCode() == fasthttp.StatusNotFound {
		t.Fatalf("GET /mcp not registered (404)")
	}
}

// TestMCPCompleteOAuthRouteRegistered proves the complete-oauth route is wired.
func TestMCPCompleteOAuthRouteRegistered(t *testing.T) {
	r := httprouter.New()
	r.NotFound = func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString("not found")
	}
	RegisterMCPRoutes(r, newMCPRouteHandlers(t))

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetRequestURI("/api/mcp/instances/abc/auth/complete")
	ctx.Request.SetBody([]byte(`{"state":"s","code":"c"}`))
	r.Handler(&ctx)
	if ctx.Response.StatusCode() == fasthttp.StatusNotFound {
		t.Fatalf("POST /api/mcp/instances/{id}/auth/complete not registered (404)")
	}
}

// TestMCPNotLocalOnly proves /mcp is NOT in LOCAL_ONLY_PATHS (D2 — it is the
// public VK-gated MCP-server surface, not a local-only admin path).
func TestMCPNotLocalOnly(t *testing.T) {
	for _, p := range LOCAL_ONLY_PATHS {
		if p == "/mcp" || p == "/mcp/" {
			t.Fatalf("/mcp must not be local-only: found %q in LOCAL_ONLY_PATHS", p)
		}
	}
}
