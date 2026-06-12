package admin

import (
	"net/http"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
	httprouter "github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

func TestConnectionUsageRoute404(t *testing.T) {
	env := newTestEnv(t)
	token := loginToken(t, env)
	authHeader := map[string]string{"Authorization": "Bearer " + token}

	stats, resolver := BuildUsageServices(env.store)
	env.handlers.SetUsageServices(stats, resolver)

	r := httprouter.New()
	r.GET("/api/usage/stats", env.handlers.RequireSession(env.handlers.GetUsageStats))
	r.GET("/api/usage/{connectionId}", env.handlers.RequireSession((&ConnectionUsageHandler{Handlers: env.handlers}).GetConnectionUsage))

	status, envl := call(t, r.Handler, "GET", "/api/usage/stats", "", nil, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("stats status = %d, want 200", status)
	}

	status, envl = call(t, r.Handler, "GET", "/api/usage/missing-id", "", nil, authHeader)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", status)
	}
	if msg := errMessage(t, envl); msg != "Connection not found" {
		t.Fatalf("error = %q", msg)
	}
}

func TestConnectionUsageNonOAuthMessage(t *testing.T) {
	env := newTestEnv(t)
	token := loginToken(t, env)
	authHeader := map[string]string{"Authorization": "Bearer " + token}

	status, envl := call(t, env.handlers.CreateProvider, "POST", "/api/providers",
		`{"name":"Anthropic","type":"anthropic","enabled":true}`, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create provider status = %d", status)
	}
	providerID := dataField[map[string]any](t, envl)["id"].(string)

	body := `{"provider_id":"` + providerID + `","name":"key","kind":"api_key","secret":"sk-ant"}`
	status, envl = call(t, env.handlers.CreateConnection, "POST", "/api/connections", body, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create connection status = %d", status)
	}
	connID := dataField[map[string]any](t, envl)["id"].(string)

	h := &ConnectionUsageHandler{Handlers: env.handlers}
	handler := env.handlers.RequireSession(h.GetConnectionUsage)

	status, envl = call(t, handler, "GET", "/api/usage/"+connID, "", map[string]any{"connectionId": connID}, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	data := dataField[map[string]any](t, envl)
	if data["message"] != "Usage not available for this connection" {
		t.Fatalf("data = %v", data)
	}
}

func TestConnectionUsageAuthExpiredRetryOnce(t *testing.T) {
	env := newTestEnv(t)
	env.withOAuth(t)
	token := loginToken(t, env)
	authHeader := map[string]string{"Authorization": "Bearer " + token}

	status, envl := call(t, env.handlers.CreateProvider, "POST", "/api/providers",
		`{"name":"Anthropic","type":"anthropic","enabled":true}`, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create provider status = %d", status)
	}
	providerID := dataField[map[string]any](t, envl)["id"].(string)

	conn := &store.Connection{
		ProviderID:   providerID,
		Name:         "claude oauth",
		Kind:         "oauth",
		AccessToken:  "at-initial",
		RefreshToken: "rt-1",
	}
	if err := env.store.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	calls := 0
	h := &ConnectionUsageHandler{
		Handlers:   env.handlers,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Fetcher: func(providerType string, c *store.Connection, client *http.Client, _ ...string) (map[string]any, error) {
			calls++
			if calls == 1 {
				return map[string]any{"message": "token expired"}, nil
			}
			return map[string]any{"plan": "Pro", "quotas": map[string]any{"model": c.AccessToken}}, nil
		},
	}
	handler := env.handlers.RequireSession(h.GetConnectionUsage)

	status, envl = call(t, handler, "GET", "/api/usage/"+conn.ID, "", map[string]any{"connectionId": conn.ID}, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	data := dataField[map[string]any](t, envl)
	if data["plan"] != "Pro" {
		t.Fatalf("data = %v", data)
	}
	quotas := data["quotas"].(map[string]any)
	if quotas["model"] != "at-refreshed" {
		t.Fatalf("retry did not use refreshed token: %v", quotas)
	}
	if calls != 2 {
		t.Fatalf("fetcher calls = %d, want 2", calls)
	}

	stored, err := env.store.GetConnection(conn.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if stored.AccessToken != "at-refreshed" {
		t.Fatalf("access token not refreshed: %q", stored.AccessToken)
	}
}
