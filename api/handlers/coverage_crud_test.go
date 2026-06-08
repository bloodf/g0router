package handlers

import (
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// crudCase exercises a CRUD-style handler that takes (*store.Store, id) and
// follows the standard nil-store / method-not-allowed / validation / 500 shape.
type crudCase struct {
	name string
	call func(ctx *fasthttp.RequestCtx, s *store.Store)
}

// runStoreClosed500 closes the store then runs the handler, asserting the
// response is a sanitized 5xx (or 404 for not-found mapping).
func runStoreClosed500(t *testing.T, method, body string, call func(ctx *fasthttp.RequestCtx, s *store.Store)) {
	t.Helper()
	s := newHandlerStore(t)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	ctx, respBody := runHandler(t, method, body, func(ctx *fasthttp.RequestCtx) { call(ctx, s) })
	if ctx.Response.StatusCode() < 500 && ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want >=500 or 404; body=%s", ctx.Response.StatusCode(), respBody)
	}
	assertNoInternalDetail(t, respBody)
}

func runNilStore503(t *testing.T, call func(ctx *fasthttp.RequestCtx, s *store.Store)) {
	t.Helper()
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) { call(ctx, nil) })
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("nil store status = %d, want 503", ctx.Response.StatusCode())
	}
}

func runMethodNotAllowed(t *testing.T, call func(ctx *fasthttp.RequestCtx, s *store.Store)) {
	t.Helper()
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPatch, "", func(ctx *fasthttp.RequestCtx) { call(ctx, s) })
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("method status = %d, want 405", ctx.Response.StatusCode())
	}
}

// --- Aliases ---

func TestAliasesNilMethodAndErrorPaths(t *testing.T) {
	call := func(ctx *fasthttp.RequestCtx, s *store.Store) { Aliases(ctx, s, "") }
	runNilStore503(t, call)
	runMethodNotAllowed(t, call)
	runStoreClosed500(t, fasthttp.MethodGet, "", call)
	runStoreClosed500(t, fasthttp.MethodPost, `{"alias":"a","provider":"openai","model":"gpt-4o"}`, call)

	// PUT missing id.
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"provider":"openai","model":"m"}`, func(ctx *fasthttp.RequestCtx) { Aliases(ctx, s, "") })
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("PUT missing id status = %d, want 400", ctx.Response.StatusCode())
	}
	// DELETE missing id.
	ctx, _ = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) { Aliases(ctx, s, "") })
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("DELETE missing id status = %d, want 400", ctx.Response.StatusCode())
	}
	// POST invalid JSON.
	ctx, _ = runHandler(t, fasthttp.MethodPost, `{`, func(ctx *fasthttp.RequestCtx) { Aliases(ctx, s, "") })
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("POST invalid json status = %d, want 400", ctx.Response.StatusCode())
	}
	// POST missing required fields.
	ctx, _ = runHandler(t, fasthttp.MethodPost, `{"alias":"a"}`, func(ctx *fasthttp.RequestCtx) { Aliases(ctx, s, "") })
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("POST missing fields status = %d, want 400", ctx.Response.StatusCode())
	}
	// PUT success path with id supplied overriding body alias.
	ctx, body := runHandler(t, fasthttp.MethodPut, `{"provider":"openai","model":"gpt-4o"}`, func(ctx *fasthttp.RequestCtx) { Aliases(ctx, s, "fast") })
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("PUT status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	// PUT store-failure path (close after setup).
	runStoreClosed500(t, fasthttp.MethodPut, `{"provider":"openai","model":"m"}`, func(ctx *fasthttp.RequestCtx, s *store.Store) { Aliases(ctx, s, "fast") })
	// DELETE store-failure path.
	runStoreClosed500(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx, s *store.Store) { Aliases(ctx, s, "fast") })
}

// --- Combos ---

func TestCombosNilMethodAndErrorPaths(t *testing.T) {
	call := func(ctx *fasthttp.RequestCtx, s *store.Store) { Combos(ctx, s, "") }
	runNilStore503(t, call)
	runMethodNotAllowed(t, call)
	runStoreClosed500(t, fasthttp.MethodGet, "", call)
	runStoreClosed500(t, fasthttp.MethodPost, `{"name":"c","steps":[],"is_active":true}`, call)

	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"c"}`, func(ctx *fasthttp.RequestCtx) { Combos(ctx, s, "") })
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("PUT missing id = %d, want 400", ctx.Response.StatusCode())
	}
	ctx, _ = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) { Combos(ctx, s, "") })
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("DELETE missing id = %d, want 400", ctx.Response.StatusCode())
	}
	ctx, _ = runHandler(t, fasthttp.MethodPut, `{`, func(ctx *fasthttp.RequestCtx) { Combos(ctx, s, "id") })
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("PUT invalid json = %d, want 400", ctx.Response.StatusCode())
	}
	ctx, _ = runHandler(t, fasthttp.MethodPut, `{"name":"c"}`, func(ctx *fasthttp.RequestCtx) { Combos(ctx, s, "missing") })
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("PUT missing combo = %d, want 404", ctx.Response.StatusCode())
	}
}

// --- Pricing ---

func TestPricingNilMethodAndErrorPaths(t *testing.T) {
	call := func(ctx *fasthttp.RequestCtx, s *store.Store) { Pricing(ctx, s, "", "") }
	runNilStore503(t, call)
	runMethodNotAllowed(t, call)
	runStoreClosed500(t, fasthttp.MethodGet, "", call)
	runStoreClosed500(t, fasthttp.MethodPost, `{"provider":"openai","model":"gpt-4o"}`, call)

	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{}`, func(ctx *fasthttp.RequestCtx) { Pricing(ctx, s, "", "") })
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("PUT missing provider/model = %d, want 400", ctx.Response.StatusCode())
	}
	ctx, _ = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) { Pricing(ctx, s, "", "") })
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("DELETE missing provider/model = %d, want 400", ctx.Response.StatusCode())
	}
	ctx, _ = runHandler(t, fasthttp.MethodPost, `{`, func(ctx *fasthttp.RequestCtx) { Pricing(ctx, s, "", "") })
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("POST invalid json = %d, want 400", ctx.Response.StatusCode())
	}
	ctx, _ = runHandler(t, fasthttp.MethodPost, `{"provider":"openai"}`, func(ctx *fasthttp.RequestCtx) { Pricing(ctx, s, "", "") })
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("POST missing model = %d, want 400", ctx.Response.StatusCode())
	}
	// PUT success with path provider/model override.
	ctx, body := runHandler(t, fasthttp.MethodPut, `{"input_cost_per_token":0.1,"output_cost_per_token":0.2}`, func(ctx *fasthttp.RequestCtx) { Pricing(ctx, s, "openai", "gpt-4o") })
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("PUT status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	runStoreClosed500(t, fasthttp.MethodPut, `{}`, func(ctx *fasthttp.RequestCtx, s *store.Store) { Pricing(ctx, s, "openai", "gpt-4o") })
	runStoreClosed500(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx, s *store.Store) { Pricing(ctx, s, "openai", "gpt-4o") })
}

// --- APIKeys ---

func TestAPIKeysNilMethodAndErrorPaths(t *testing.T) {
	call := func(ctx *fasthttp.RequestCtx, s *store.Store) { APIKeys(ctx, s, "secret", "") }
	runNilStore503(t, call)
	runMethodNotAllowed(t, call)
	runStoreClosed500(t, fasthttp.MethodGet, "", call)
	runStoreClosed500(t, fasthttp.MethodPost, `{"name":"k"}`, call)

	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{`, func(ctx *fasthttp.RequestCtx) { APIKeys(ctx, s, "secret", "") })
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("POST invalid json = %d, want 400", ctx.Response.StatusCode())
	}
	ctx, _ = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) { APIKeys(ctx, s, "secret", "") })
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("DELETE missing id = %d, want 400", ctx.Response.StatusCode())
	}
	// Create then delete success.
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"name":"k"}`, func(ctx *fasthttp.RequestCtx) { APIKeys(ctx, s, "secret", "") })
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("POST status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
	var created apiKeyView
	decodeJSON(t, body, &created)
	ctx, _ = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) { APIKeys(ctx, s, "secret", created.ID) })
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("DELETE status = %d, want 204", ctx.Response.StatusCode())
	}
	runStoreClosed500(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx, s *store.Store) { APIKeys(ctx, s, "secret", "some-id") })
}

// --- Settings ---

func TestSettingsNilMethodAndErrorPaths(t *testing.T) {
	call := func(ctx *fasthttp.RequestCtx, s *store.Store) { Settings(ctx, s) }
	runNilStore503(t, call)
	runMethodNotAllowed(t, call)
	runStoreClosed500(t, fasthttp.MethodGet, "", call)
	runStoreClosed500(t, fasthttp.MethodPut, `{}`, call)

	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{`, func(ctx *fasthttp.RequestCtx) { Settings(ctx, s) })
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("PUT invalid json = %d, want 400", ctx.Response.StatusCode())
	}
}

// --- Usage / UsageSummary / Logs ---

func TestUsageNilAndErrorPaths(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) { Usage(ctx, nil) })
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("nil usage store = %d, want 503", ctx.Response.StatusCode())
	}
	runStoreClosed500ForUsage(t, func(ctx *fasthttp.RequestCtx, s *store.Store) { Usage(ctx, s) })
}

func TestUsageSummaryNilInvalidAndErrorPaths(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) { UsageSummary(ctx, nil) })
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("nil store = %d, want 503", ctx.Response.StatusCode())
	}
	// Invalid filter (bad limit) -> 400.
	s := newHandlerStore(t)
	c := newHandlerCtx(fasthttp.MethodGet, "/api/usage/summary?limit=abc")
	UsageSummary(c, s)
	if c.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("bad limit = %d, want 400", c.Response.StatusCode())
	}
	// Negative offset -> 400.
	c = newHandlerCtx(fasthttp.MethodGet, "/api/usage/summary?offset=-1")
	UsageSummary(c, s)
	if c.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("negative offset = %d, want 400", c.Response.StatusCode())
	}
	// Invalid 'to' date -> 400.
	c = newHandlerCtx(fasthttp.MethodGet, "/api/usage/summary?to=nope")
	UsageSummary(c, s)
	if c.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("bad to date = %d, want 400", c.Response.StatusCode())
	}
	runStoreClosed500ForUsage(t, func(ctx *fasthttp.RequestCtx, s *store.Store) { UsageSummary(ctx, s) })
}

func TestLogsNilInvalidAndErrorPaths(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) { Logs(ctx, nil) })
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("nil store = %d, want 503", ctx.Response.StatusCode())
	}
	s := newHandlerStore(t)
	c := newHandlerCtx(fasthttp.MethodGet, "/api/logs?from=bad")
	Logs(c, s)
	if c.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("bad from = %d, want 400", c.Response.StatusCode())
	}
	runStoreClosed500ForUsage(t, func(ctx *fasthttp.RequestCtx, s *store.Store) { Logs(ctx, s) })
}

func runStoreClosed500ForUsage(t *testing.T, call func(ctx *fasthttp.RequestCtx, s *store.Store)) {
	t.Helper()
	s := newHandlerStore(t)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) { call(ctx, s) })
	if ctx.Response.StatusCode() < 500 {
		t.Fatalf("status = %d, want >=500; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoInternalDetail(t, body)
}
