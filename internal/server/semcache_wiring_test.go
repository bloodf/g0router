package server

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/providers/catalog"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// countingStubCatalog installs a smoketest provider whose stub increments a
// counter on every chat completion, so a cache short-circuit (the provider not
// being hit on the second identical request) is provable end-to-end.
func countingStubCatalog(t *testing.T, hits *int64) func() {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(hits, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"live","object":"chat.completion","model":"smoke-model","choices":[{"index":0,"message":{"role":"assistant","content":"live-content"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	orig, ok := catalog.Providers["smoketest"]
	catalog.Providers["smoketest"] = catalog.ProviderConfig{
		Name:    "smoketest",
		BaseURL: srv.URL,
		Format:  "openai",
		NoAuth:  true,
	}
	return func() {
		srv.Close()
		if ok {
			catalog.Providers["smoketest"] = orig
		} else {
			delete(catalog.Providers, "smoketest")
		}
	}
}

func chatRequest(t *testing.T, srv *fasthttp.Server, apiKey string) *fasthttp.RequestCtx {
	t.Helper()
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.Header.Set("Authorization", "Bearer "+apiKey)
	ctx.Request.SetBody([]byte(`{"model":"smoketest/smoke-model","messages":[{"role":"user","content":"hi"}]}`))
	srv.Handler(ctx)
	if ctx.Response.StatusCode() == fasthttp.StatusNotFound {
		t.Fatalf("/v1/chat/completions returned 404: %s", string(ctx.Response.Body()))
	}
	return ctx
}

// TestSemanticCacheShortCircuitsProviderWhenFlagOn proves the wired cache:
// with the semantic_cache flag ON, two identical chat requests hit the upstream
// provider only ONCE (the second is served from cache).
func TestSemanticCacheShortCircuitsProviderWhenFlagOn(t *testing.T) {
	st := newTestStore(t)
	var hits int64
	defer countingStubCatalog(t, &hits)()

	sessions := auth.NewSessions(st, time.Hour)
	if _, err := sessions.SeedAdmin("admin", "123456"); err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}
	seedSmokeProvider(t, st)
	enableSemanticCacheFlag(t, st)

	rec, err := st.CreateAPIKey("semcache-test")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	srv := New(testUIFS(), st, nil)

	first := chatRequest(t, srv, rec.Key)
	if first.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("first request status = %d: %s", first.Response.StatusCode(), first.Response.Body())
	}
	firstBody := string(first.Response.Body())

	second := chatRequest(t, srv, rec.Key)
	secondBody := string(second.Response.Body())

	if got := atomic.LoadInt64(&hits); got != 1 {
		t.Fatalf("provider hits = %d, want 1 (second request must be served from cache)", got)
	}
	if secondBody != firstBody {
		t.Fatalf("cached response body = %q, want first body %q", secondBody, firstBody)
	}
}

// TestSemanticCacheFlagOffReachesProvider proves the flag gate: with the flag
// OFF, both identical requests reach the provider (no caching).
func TestSemanticCacheFlagOffReachesProvider(t *testing.T) {
	st := newTestStore(t)
	var hits int64
	defer countingStubCatalog(t, &hits)()

	sessions := auth.NewSessions(st, time.Hour)
	if _, err := sessions.SeedAdmin("admin", "123456"); err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}
	seedSmokeProvider(t, st)
	// Flag left OFF (seeded disabled by the migration).

	rec, err := st.CreateAPIKey("semcache-test-off")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	srv := New(testUIFS(), st, nil)

	_ = chatRequest(t, srv, rec.Key)
	_ = chatRequest(t, srv, rec.Key)

	if got := atomic.LoadInt64(&hits); got != 2 {
		t.Fatalf("provider hits = %d, want 2 (flag off ⇒ no caching)", got)
	}
}

func enableSemanticCacheFlag(t *testing.T, st *store.Store) {
	t.Helper()
	if _, err := st.DB().Exec(
		"UPDATE feature_flags SET enabled = 1 WHERE key = 'semantic_cache'",
	); err != nil {
		t.Fatalf("enable semantic_cache flag: %v", err)
	}
}
