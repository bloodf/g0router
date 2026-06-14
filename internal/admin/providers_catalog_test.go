package admin

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

// seedProviderWithConnections creates a provider and the given number of
// connections for it, returning the provider id.
func seedProviderWithConnections(t *testing.T, env *testEnv, name, typ string, connCount int) string {
	t.Helper()
	status, envl := call(t, env.handlers.CreateProvider, "POST", "/api/providers",
		fmt.Sprintf(`{"name":%q,"type":%q,"enabled":true}`, name, typ), nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create provider %s status = %d", name, status)
	}
	id := dataField[map[string]any](t, envl)["id"].(string)
	for i := 0; i < connCount; i++ {
		body := fmt.Sprintf(`{"provider_id":%q,"name":"conn-%d","kind":"api_key","secret":"sk-secret-%d"}`, id, i, i)
		status, _ := call(t, env.handlers.CreateConnection, "POST", "/api/connections", body, nil, nil)
		if status != fasthttp.StatusCreated {
			t.Fatalf("create connection for %s status = %d", name, status)
		}
	}
	return id
}

func TestListProviderCatalog(t *testing.T) {
	env := newTestEnv(t)
	// type "openai" is a known provider in the static catalog; give it 2 connections.
	pid := seedProviderWithConnections(t, env, "OpenAI", "openai", 2)
	_ = pid

	status, envl := call(t, env.handlers.ListProviderCatalog, "GET", "/api/providers/catalog", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("catalog list status = %d err = %q", status, errMessage(t, envl))
	}
	list := dataField[[]map[string]any](t, envl)
	if len(list) == 0 {
		t.Fatalf("catalog list is empty")
	}

	var openai map[string]any
	for _, p := range list {
		if p["type"] == "openai" || p["name"] == "OpenAI" {
			openai = p
			break
		}
	}
	if openai == nil {
		t.Fatalf("openai provider not in catalog list: %v", list)
	}
	if openai["display_name"] == nil || openai["display_name"] == "" {
		t.Fatalf("missing display_name: %v", openai)
	}
	if _, ok := openai["auth_types"]; !ok {
		t.Fatalf("missing auth_types: %v", openai)
	}
	if _, ok := openai["capabilities"]; !ok {
		t.Fatalf("missing capabilities: %v", openai)
	}
	if cc, _ := openai["connection_count"].(float64); cc != 2 {
		t.Fatalf("connection_count = %v, want 2", openai["connection_count"])
	}
	if openai["status"] != "active" {
		t.Fatalf("status = %v, want active (has active connections)", openai["status"])
	}
}

func TestGetProviderCatalog(t *testing.T) {
	env := newTestEnv(t)
	pid := seedProviderWithConnections(t, env, "Anthropic", "anthropic", 1)

	status, envl := call(t, env.handlers.GetProviderCatalog, "GET", "/api/providers/"+pid+"/catalog", "",
		map[string]any{"id": pid}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("get catalog status = %d err = %q", status, errMessage(t, envl))
	}
	p := dataField[map[string]any](t, envl)
	if p["id"] != pid {
		t.Fatalf("id = %v, want %v", p["id"], pid)
	}
	if cc, _ := p["connection_count"].(float64); cc != 1 {
		t.Fatalf("connection_count = %v, want 1", p["connection_count"])
	}

	// Not found → 404.
	status, _ = call(t, env.handlers.GetProviderCatalog, "GET", "/api/providers/missing/catalog", "",
		map[string]any{"id": "missing"}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("missing provider catalog status = %d, want 404", status)
	}
}

func TestGetProviderConnectionsCatalogMasksSecrets(t *testing.T) {
	env := newTestEnv(t)
	pid := seedProviderWithConnections(t, env, "OpenAI", "openai", 2)

	status, envl := call(t, env.handlers.GetProviderConnections, "GET", "/api/providers/"+pid+"/connections", "",
		map[string]any{"id": pid}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("connections status = %d err = %q", status, errMessage(t, envl))
	}
	conns := dataField[[]map[string]any](t, envl)
	if len(conns) != 2 {
		t.Fatalf("connections = %d, want 2", len(conns))
	}
	c := conns[0]
	// UI-shaped fields (ESCALATION-2): provider, auth_type, is_active, needs_reauth.
	if c["provider"] != pid {
		t.Fatalf("provider = %v, want %v", c["provider"], pid)
	}
	if _, ok := c["auth_type"]; !ok {
		t.Fatalf("missing auth_type: %v", c)
	}
	if _, ok := c["is_active"]; !ok {
		t.Fatalf("missing is_active: %v", c)
	}
	if _, ok := c["needs_reauth"]; !ok {
		t.Fatalf("missing needs_reauth: %v", c)
	}
	// No secret material leaks.
	raw, _ := json.Marshal(conns)
	if strings.Contains(string(raw), "sk-secret") {
		t.Fatalf("connections leak secret: %s", raw)
	}

	// Not found → empty list (no provider, no connections).
	status, envl = call(t, env.handlers.GetProviderConnections, "GET", "/api/providers/missing/connections", "",
		map[string]any{"id": "missing"}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("missing connections status = %d", status)
	}
	empty := dataField[[]map[string]any](t, envl)
	if len(empty) != 0 {
		t.Fatalf("missing provider connections = %v, want empty", empty)
	}
}

func TestGetProviderModelsCatalog(t *testing.T) {
	env := newTestEnv(t)
	pid := seedProviderWithConnections(t, env, "OpenAI", "openai", 1)

	status, envl := call(t, env.handlers.GetProviderModels, "GET", "/api/providers/"+pid+"/models", "",
		map[string]any{"id": pid}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("models status = %d err = %q", status, errMessage(t, envl))
	}
	models := dataField[[]map[string]any](t, envl)
	if len(models) == 0 {
		t.Fatalf("openai should have catalog models")
	}
	m := models[0]
	if m["provider"] != pid {
		t.Fatalf("model provider = %v, want %v", m["provider"], pid)
	}
	if _, ok := m["context_window"]; !ok {
		t.Fatalf("missing context_window: %v", m)
	}

	// Suggested models is a trimmed {id,name} list.
	status, envl = call(t, env.handlers.GetProviderSuggestedModels, "GET", "/api/providers/"+pid+"/suggested-models", "",
		map[string]any{"id": pid}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("suggested status = %d err = %q", status, errMessage(t, envl))
	}
	suggested := dataField[[]map[string]any](t, envl)
	if len(suggested) == 0 {
		t.Fatalf("suggested models empty")
	}
	if _, ok := suggested[0]["name"]; !ok {
		t.Fatalf("suggested missing name: %v", suggested[0])
	}
	if len(suggested) > 5 {
		t.Fatalf("suggested should be capped at 5, got %d", len(suggested))
	}
}

func TestTestProvidersBatch(t *testing.T) {
	env := newTestEnv(t)
	active := seedProviderWithConnections(t, env, "OpenAI", "openai", 1)
	inactive := seedProviderWithConnections(t, env, "Cohere", "cohere", 0)

	status, envl := call(t, env.handlers.TestProvidersBatch, "POST", "/api/providers/test-batch", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("test-batch status = %d err = %q", status, errMessage(t, envl))
	}
	out := dataField[map[string][]map[string]any](t, envl)
	results, ok := out["results"]
	if !ok {
		t.Fatalf("missing results: %v", out)
	}
	byProvider := map[string]bool{}
	for _, r := range results {
		p, _ := r["provider"].(string)
		okFlag, _ := r["ok"].(bool)
		byProvider[p] = okFlag
		if _, has := r["latency_ms"]; !has {
			t.Fatalf("result missing latency_ms: %v", r)
		}
	}
	if !byProvider[active] {
		t.Fatalf("provider with an active connection should be ok=true: %v", byProvider)
	}
	if byProvider[inactive] {
		t.Fatalf("provider without connections should be ok=false: %v", byProvider)
	}
}

// TestCatalogRouteDisambiguation proves the fasthttp/router matcher resolves the
// static /api/providers/catalog and /api/providers/test-batch routes distinctly
// from the /api/providers/{id}/... param routes (plan §8 ESCALATION-3).
func TestCatalogRouteDisambiguation(t *testing.T) {
	env := newTestEnv(t)
	r := router.New()
	r.GET("/api/providers/catalog", env.handlers.ListProviderCatalog)
	r.GET("/api/providers/{id}/catalog", env.handlers.GetProviderCatalog)
	r.GET("/api/providers/{id}/connections", env.handlers.GetProviderConnections)
	r.GET("/api/providers/{id}/models", env.handlers.GetProviderModels)
	r.GET("/api/providers/{id}/suggested-models", env.handlers.GetProviderSuggestedModels)
	r.POST("/api/providers/test-batch", env.handlers.TestProvidersBatch)

	// Static /catalog resolves to the LIST handler (returns an array).
	var listCtx fasthttp.RequestCtx
	listCtx.Request.Header.SetMethod("GET")
	listCtx.Request.SetRequestURI("/api/providers/catalog")
	r.Handler(&listCtx)
	if listCtx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("/api/providers/catalog status = %d", listCtx.Response.StatusCode())
	}
	var listEnv struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(listCtx.Response.Body(), &listEnv); err != nil {
		t.Fatalf("/api/providers/catalog body is not a list envelope: %v\n%s", err, listCtx.Response.Body())
	}

	// Param /{id}/catalog resolves to the DETAIL handler (returns an object, here 404 for unknown id).
	var detailCtx fasthttp.RequestCtx
	detailCtx.Request.Header.SetMethod("GET")
	detailCtx.Request.SetRequestURI("/api/providers/some-id/catalog")
	r.Handler(&detailCtx)
	if detailCtx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("/api/providers/some-id/catalog status = %d, want 404", detailCtx.Response.StatusCode())
	}

	// Static /test-batch resolves to the batch handler.
	var batchCtx fasthttp.RequestCtx
	batchCtx.Request.Header.SetMethod("POST")
	batchCtx.Request.SetRequestURI("/api/providers/test-batch")
	r.Handler(&batchCtx)
	if batchCtx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("/api/providers/test-batch status = %d", batchCtx.Response.StatusCode())
	}
}
