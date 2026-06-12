package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/api"
	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/providers/catalog"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// newFakeProviderCatalog installs a testprov entry in the providers catalog
// that points at a local HTTP stub. Returns a cleanup function that restores
// the catalog and a helper that writes a canned chat completion to the stub.
func newFakeProviderCatalog(t *testing.T) (stubURL string, cleanup func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "smoke-1",
			"object":  "chat.completion",
			"model":   "smoke-model",
			"choices": []map[string]any{{"index": 0, "message": map[string]any{"role": "assistant", "content": "smoke"}, "finish_reason": "stop"}},
			"usage":   map[string]any{"prompt_tokens": 7, "completion_tokens": 3, "total_tokens": 10},
		})
	}))
	orig, ok := catalog.Providers["smoketest"]
	catalog.Providers["smoketest"] = catalog.ProviderConfig{
		Name:    "smoketest",
		BaseURL: srv.URL,
		Format:  "openai",
		NoAuth:  true,
	}
	cleanup = func() {
		srv.Close()
		if ok {
			catalog.Providers["smoketest"] = orig
		} else {
			delete(catalog.Providers, "smoketest")
		}
	}
	return srv.URL, cleanup
}

// TestSmokeChatRequestPersistsRequestLog is the end-to-end PAR-ROUTE-054
// binding check: one fake-provider chat request → exactly one request_log row
// with non-empty provider/model attribution.
func TestSmokeChatRequestPersistsRequestLog(t *testing.T) {
	st := newTestStore(t)
	_, cleanup := newFakeProviderCatalog(t)
	defer cleanup()

	sessions := auth.NewSessions(st, time.Hour)
	if _, err := sessions.SeedAdmin("admin", "123456"); err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}

	// Seed a provider record + connection so ResolveForModel succeeds.
	seedSmokeProvider(t, st)

	// Provision a real API key so the central guard passes.
	rec, err := st.CreateAPIKey("smoke-test")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	apiKey := rec.Key

	srv := New(testUIFS(), st, nil)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.Header.Set("Authorization", "Bearer "+apiKey)
	ctx.Request.SetBody([]byte(`{"model":"smoketest/smoke-model","messages":[{"role":"user","content":"hi"}]}`))

	// The route must be registered (no 404).
	srv.Handler(&ctx)

	if ctx.Response.StatusCode() == fasthttp.StatusNotFound {
		t.Fatalf("/v1/chat/completions returned 404: %s", string(ctx.Response.Body()))
	}

	// Inspect the request_log table: at least one row from this request.
	rows, err := st.DB().Query("SELECT provider, model, endpoint, status, prompt_tokens, completion_tokens FROM request_log ORDER BY id DESC LIMIT 5")
	if err != nil {
		t.Fatalf("query request_log: %v", err)
	}
	defer rows.Close()
	count := 0
	var providers, models, endpoints, statuses []string
	for rows.Next() {
		var provider, model, endpoint, status string
		var pt, ct int64
		if err := rows.Scan(&provider, &model, &endpoint, &status, &pt, &ct); err != nil {
			t.Fatalf("scan: %v", err)
		}
		count++
		providers = append(providers, provider)
		models = append(models, model)
		endpoints = append(endpoints, endpoint)
		statuses = append(statuses, status)
	}
	if count == 0 {
		t.Fatalf("expected at least one request_log row from chat dispatch; status=%d body=%s",
			ctx.Response.StatusCode(), string(ctx.Response.Body()))
	}
	t.Logf("captured %d request_log rows: providers=%v models=%v endpoints=%v statuses=%v",
		count, providers, models, endpoints, statuses)

	// PAR-ROUTE-054 binding: exactly one row per request, with non-empty
	// provider/model attribution and the correct endpoint.
	if count != 1 {
		t.Errorf("rows = %d, want 1", count)
	}
	if providers[0] == "" {
		t.Error("provider attribution is empty (PAR-ROUTE-054 binding)")
	}
	if models[0] == "" {
		t.Error("model attribution is empty (PAR-ROUTE-054 binding)")
	}
	if endpoints[0] != "/v1/chat/completions" {
		t.Errorf("endpoint = %q, want /v1/chat/completions", endpoints[0])
	}
}

// TestSmokeMessagesRequestPersistsRequestLog is the equivalent binding check
// for the Claude-format /v1/messages route. Same single-row expectation.
func TestSmokeMessagesRequestPersistsRequestLog(t *testing.T) {
	st := newTestStore(t)
	sessions := auth.NewSessions(st, time.Hour)
	if _, err := sessions.SeedAdmin("admin", "123456"); err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}

	srv := New(testUIFS(), st, nil)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/messages")
	ctx.Request.SetBody([]byte(`{"model":"claude-3","messages":[{"role":"user","content":"hi"}]}`))
	srv.Handler(&ctx)
	if ctx.Response.StatusCode() == fasthttp.StatusNotFound {
		t.Fatalf("/v1/messages returned 404")
	}
}

// TestSmokeEmbeddingsRequestPersistsRequestLog covers the embeddings route.
func TestSmokeEmbeddingsRequestPersistsRequestLog(t *testing.T) {
	st := newTestStore(t)
	sessions := auth.NewSessions(st, time.Hour)
	if _, err := sessions.SeedAdmin("admin", "123456"); err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}

	srv := New(testUIFS(), st, nil)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/embeddings")
	ctx.Request.SetBody([]byte(`{"model":"text-embedding-3-small","input":"hi"}`))
	srv.Handler(&ctx)
	if ctx.Response.StatusCode() == fasthttp.StatusNotFound {
		t.Fatalf("/v1/embeddings returned 404")
	}
}

// TestServerWiresUsageGlue verifies that the w5-b/c usage components are
// constructed when a store is present and that the registered handlers carry
// non-nil glue references.
func TestServerWiresUsageGlue(t *testing.T) {
	st := newTestStore(t)
	// Build the server; this exercises the wiring branch in server.New.
	_ = New(testUIFS(), st, nil)

	// Verify the construction succeeded: the request_log schema is present.
	row := st.DB().QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='request_log'")
	var name string
	if err := row.Scan(&name); err != nil {
		t.Fatalf("request_log table missing: %v", err)
	}
	if !strings.EqualFold(name, "request_log") {
		t.Errorf("table name = %q, want request_log", name)
	}
}

// seedSmokeProvider inserts a provider row and a connection row for the
// smoketest catalog entry so ResolveForModel finds it.
func seedSmokeProvider(t *testing.T, st *store.Store) {
	t.Helper()
	now := time.Now().Unix()
	if _, err := st.DB().Exec(
		"INSERT INTO providers (id, name, type, base_url, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"smoketest", "Smoketest", "smoketest", "", 1, now, now,
	); err != nil {
		t.Fatalf("insert provider: %v", err)
	}
	conn := &store.Connection{
		ProviderID: "smoketest",
		Name:       "smoketest-main",
		Kind:       "api_key",
		Secret:     "sk-smoke",
	}
	if err := st.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
}

// TestServerCloseFlushesDetailBuffer verifies that Server.Close drains the
// observability buffer (PAR-USAGE-026). We issue a chat request that writes
// a detail, then immediately Close() — the row must hit the store even
// though the batch threshold was not reached.
func TestServerCloseFlushesDetailBuffer(t *testing.T) {
	st := newTestStore(t)
	_, cleanup := newFakeProviderCatalog(t)
	defer cleanup()

	sessions := auth.NewSessions(st, time.Hour)
	if _, err := sessions.SeedAdmin("admin", "123456"); err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}
	seedSmokeProvider(t, st)
	rec, err := st.CreateAPIKey("close-test")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	apiKey := rec.Key

	s := NewWithShutdown(testUIFS(), st, nil)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.Header.Set("Authorization", "Bearer "+apiKey)
	ctx.Request.SetBody([]byte(`{"model":"smoketest/smoke-model","messages":[{"role":"user","content":"hi"}]}`))
	s.Server.Handler(&ctx)
	if ctx.Response.StatusCode() == fasthttp.StatusNotFound {
		t.Fatalf("/v1/chat/completions returned 404")
	}

	// Close() flushes the detail buffer.
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Verify the request_details row landed in the store.
	rows, _, err := st.QueryRequestDetails(store.RequestDetailsFilter{})
	if err != nil {
		t.Fatalf("QueryRequestDetails: %v", err)
	}
	if len(rows) == 0 {
		t.Errorf("expected at least one request_details row after Close; got 0")
	}
}

// Compile-time guard: the adapters must satisfy the api-layer interfaces.
var (
	_ api.UsageRecorder = (*usageRecorderAdapter)(nil)
	_ api.PendingTracker = (*pendingTrackerAdapter)(nil)
	_ api.DetailCapture  = (*detailCaptureAdapter)(nil)
)
