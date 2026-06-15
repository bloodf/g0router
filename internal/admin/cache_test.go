package admin

import (
	"testing"

	"github.com/valyala/fasthttp"
)

// seedSemanticCacheRow inserts a cache row directly so the admin GET/DELETE can
// be exercised without the chat hook (hermetic).
func seedSemanticCacheRow(t *testing.T, env *testEnv, key, model, responseJSON string, hits int64) {
	t.Helper()
	_, err := env.store.DB().Exec(
		`INSERT INTO semantic_cache (cache_key, embedding_json, model, response_json, expires_at, hit_count)
		 VALUES (?, '[]', ?, ?, NULL, ?)`,
		key, model, responseJSON, hits,
	)
	if err != nil {
		t.Fatalf("seed semantic_cache row: %v", err)
	}
}

func TestGetSemanticCache(t *testing.T) {
	env := newTestEnv(t)
	seedSemanticCacheRow(t, env, "k1", "gpt-4", `{"secret":"do-not-leak"}`, 3)
	seedSemanticCacheRow(t, env, "k2", "gpt-3.5", `{"secret":"also-secret"}`, 1)

	status, envl := call(t, env.handlers.GetSemanticCache, "GET", "/api/cache/semantic", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("get status = %d err = %q", status, errMessage(t, envl))
	}
	data := dataField[struct {
		Stats struct {
			Entries   int64 `json:"entries"`
			TotalHits int64 `json:"total_hits"`
		} `json:"stats"`
		Entries []map[string]any `json:"entries"`
	}](t, envl)

	if data.Stats.Entries != 2 {
		t.Fatalf("stats.entries = %d, want 2", data.Stats.Entries)
	}
	if data.Stats.TotalHits != 4 {
		t.Fatalf("stats.total_hits = %d, want 4", data.Stats.TotalHits)
	}
	if len(data.Entries) != 2 {
		t.Fatalf("entries len = %d, want 2", len(data.Entries))
	}
	first := data.Entries[0]
	if first["key"] != "k1" || first["model"] != "gpt-4" {
		t.Fatalf("first entry = %v", first)
	}
	if _, ok := first["hits"]; !ok {
		t.Fatalf("entry missing hits: %v", first)
	}
	if _, ok := first["expires"]; !ok {
		t.Fatalf("entry missing expires: %v", first)
	}
	// The GET must NOT leak the full response payload.
	if _, ok := first["response_json"]; ok {
		t.Fatalf("entry leaks response_json: %v", first)
	}
	if _, ok := first["response"]; ok {
		t.Fatalf("entry leaks response: %v", first)
	}
}

func TestClearSemanticCache(t *testing.T) {
	env := newTestEnv(t)
	seedSemanticCacheRow(t, env, "k1", "gpt-4", `{"x":1}`, 0)
	seedSemanticCacheRow(t, env, "k2", "gpt-4", `{"x":2}`, 0)

	admin, err := env.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}

	status, envl := call(t, env.handlers.ClearSemanticCache, "DELETE", "/api/cache/semantic", "",
		map[string]any{userKey: admin}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete status = %d err = %q", status, errMessage(t, envl))
	}

	// Table emptied.
	stats, err := env.store.SemanticCacheStats()
	if err != nil {
		t.Fatalf("SemanticCacheStats: %v", err)
	}
	if stats.Entries != 0 {
		t.Fatalf("after clear: entries = %d, want 0", stats.Entries)
	}

	// Audit row written.
	status, envl = call(t, env.handlers.GetAudit, "GET", "/api/audit", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("audit status = %d", status)
	}
	page := dataField[struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}](t, envl)
	if page.Total < 1 {
		t.Fatalf("expected an audit entry after clear, total=%d", page.Total)
	}
	if page.Items[0]["action"] != "semantic_cache.clear" {
		t.Fatalf("audit action = %v, want semantic_cache.clear", page.Items[0]["action"])
	}
}
