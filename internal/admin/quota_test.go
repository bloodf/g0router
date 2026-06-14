package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// seedProviderAndConnection creates a provider + connection and returns the
// connection id.
func seedProviderAndConnection(t *testing.T, env *testEnv, providerName, providerType, connName, kind, secret string) string {
	t.Helper()
	status, envl := call(t, env.handlers.CreateProvider, "POST", "/api/providers",
		fmt.Sprintf(`{"name":%q,"type":%q,"enabled":true}`, providerName, providerType), nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create provider %s status = %d", providerName, status)
	}
	providerID := dataField[map[string]any](t, envl)["id"].(string)

	body := fmt.Sprintf(`{"provider_id":%q,"name":%q,"kind":%q,"secret":%q}`, providerID, connName, kind, secret)
	status, envl = call(t, env.handlers.CreateConnection, "POST", "/api/connections", body, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create connection %s status = %d err = %q", connName, status, errMessage(t, envl))
	}
	return dataField[map[string]any](t, envl)["id"].(string)
}

func TestQuotaAggregation(t *testing.T) {
	env := newTestEnv(t)

	// An oauth connection (resolved through the fetcher) and a non-oauth one.
	oauthID := seedProviderAndConnection(t, env, "OpenAI", "openai", "OpenAI Prod", "oauth", "")
	apiKeyID := seedProviderAndConnection(t, env, "Groq", "groq", "Groq Fast", "api_key", "sk-groq-supersecret")

	qh := &QuotaHandler{
		Handlers: env.handlers,
		Fetcher: func(providerType string, conn *store.Connection, client *http.Client, baseURL ...string) (map[string]any, error) {
			// Deterministic per-connection usage; no network.
			return map[string]any{
				"used":          float64(45000),
				"limit":         float64(100000),
				"unit":          "tokens",
				"plan":          "pro",
				"account_label": "org-123",
				"reset_at":      "2026-07-01T00:00:00Z",
			}, nil
		},
	}

	status, envl := call(t, qh.GetQuota, "GET", "/api/quota", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("quota status = %d err = %q", status, errMessage(t, envl))
	}
	cards := dataField[[]map[string]any](t, envl)
	if len(cards) != 2 {
		t.Fatalf("cards len = %d, want 2", len(cards))
	}

	byConn := map[string]map[string]any{}
	for _, c := range cards {
		id, _ := c["connection_id"].(string)
		byConn[id] = c
	}

	oauthCard := byConn[oauthID]
	if oauthCard == nil {
		t.Fatalf("missing oauth card; cards = %v", cards)
	}
	if oauthCard["provider"] != "openai" || oauthCard["connection_name"] != "OpenAI Prod" {
		t.Fatalf("oauth card = %v", oauthCard)
	}
	if oauthCard["used"].(float64) != 45000 || oauthCard["limit"].(float64) != 100000 {
		t.Fatalf("oauth card used/limit = %v/%v", oauthCard["used"], oauthCard["limit"])
	}
	if oauthCard["unit"] != "tokens" || oauthCard["plan"] != "pro" {
		t.Fatalf("oauth card unit/plan = %v/%v", oauthCard["unit"], oauthCard["plan"])
	}

	apiKeyCard := byConn[apiKeyID]
	if apiKeyCard == nil {
		t.Fatalf("missing api_key card; cards = %v", cards)
	}
	// Non-oauth connections contribute a card with zeroed usage.
	if apiKeyCard["used"].(float64) != 0 || apiKeyCard["limit"].(float64) != 0 {
		t.Fatalf("api_key card used/limit = %v/%v, want 0/0", apiKeyCard["used"], apiKeyCard["limit"])
	}

	// No secret/token must appear anywhere in the response.
	raw, _ := json.Marshal(cards)
	for _, leak := range []string{"supersecret", "access_token", "refresh_token", "secret"} {
		if strings.Contains(string(raw), leak) {
			t.Fatalf("quota response leaks %q: %s", leak, raw)
		}
	}
}

func TestQuotaDefaultFetcherNil(t *testing.T) {
	// With no connections, GetQuota returns an empty array regardless of fetcher.
	env := newTestEnv(t)
	qh := &QuotaHandler{Handlers: env.handlers}
	status, envl := call(t, qh.GetQuota, "GET", "/api/quota", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("quota status = %d", status)
	}
	if len(dataField[[]map[string]any](t, envl)) != 0 {
		t.Fatalf("expected empty array with no connections")
	}
}
