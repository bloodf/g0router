package handlers

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func TestConnectionsCreateListUpdateDelete(t *testing.T) {
	s := newHandlerStore(t)

	createBody := `{"provider":"openai","name":"primary","auth_type":"api_key","api_key":"sk-test","is_active":true,"provider_specific_data":{"region":"us"},"model_locks":{"gpt-4o":123}}`
	ctx, body := runHandler(t, fasthttp.MethodPost, createBody, func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoCredentialFields(t, body)

	var created store.Connection
	decodeJSON(t, body, &created)
	if created.ID == "" || created.Name != "primary" || created.APIKey != nil {
		t.Fatalf("created connection = %+v", created)
	}
	stored, err := s.GetConnection(created.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if stored.APIKey == nil || *stored.APIKey != "sk-test" {
		t.Fatalf("stored API key = %v, want sk-test", stored.APIKey)
	}

	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoCredentialFields(t, body)
	var listed struct {
		Data []store.Connection `json:"data"`
	}
	decodeJSON(t, body, &listed)
	if len(listed.Data) != 1 || listed.Data[0].ID != created.ID {
		t.Fatalf("listed = %+v, want created connection", listed.Data)
	}

	updateBody := `{"provider":"openai","name":"renamed","auth_type":"api_key","api_key":"sk-test-2","is_active":false}`
	ctx, body = runHandler(t, fasthttp.MethodPut, updateBody, func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoCredentialFields(t, body)
	var updated store.Connection
	decodeJSON(t, body, &updated)
	if updated.Name != "renamed" || updated.IsActive || updated.APIKey != nil {
		t.Fatalf("updated = %+v", updated)
	}
	stored, err = s.GetConnection(created.ID)
	if err != nil {
		t.Fatalf("GetConnection after update: %v", err)
	}
	if stored.APIKey == nil || *stored.APIKey != "sk-test-2" {
		t.Fatalf("stored updated API key = %v, want sk-test-2", stored.APIKey)
	}

	ctx, body = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestConnectionsResponsesRedactCredentialsWithoutMutatingStore(t *testing.T) {
	s := newHandlerStore(t)

	createBody := `{"provider":"openai","name":"primary","auth_type":"oauth","access_token":"access-secret","refresh_token":"refresh-secret","api_key":"api-secret","is_active":true}`
	ctx, body := runHandler(t, fasthttp.MethodPost, createBody, func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoCredentialFields(t, body)

	var created store.Connection
	decodeJSON(t, body, &created)
	assertStoredCredentials(t, s, created.ID, "access-secret", "refresh-secret", "api-secret")

	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoCredentialFields(t, body)
	assertStoredCredentials(t, s, created.ID, "access-secret", "refresh-secret", "api-secret")

	updateBody := `{"provider":"openai","name":"renamed","auth_type":"oauth","access_token":"access-new","refresh_token":"refresh-new","api_key":"api-new","is_active":false}`
	ctx, body = runHandler(t, fasthttp.MethodPut, updateBody, func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoCredentialFields(t, body)
	assertStoredCredentials(t, s, created.ID, "access-new", "refresh-new", "api-new")
}

func TestConnectionsResponsesRedactProviderSpecificCredentialData(t *testing.T) {
	s := newHandlerStore(t)

	createBody := `{"provider":"openai","name":"primary","auth_type":"oauth","access_token":"access-secret","refresh_token":"refresh-secret","api_key":"api-secret","is_active":true,"provider_specific_data":{"region":"us","access_token":"provider-access","refresh_token":"provider-refresh","api_key":"provider-key","Authorization":"Bearer provider-token","nested":{"mode":"readonly","password":"provider-password"}}}`
	ctx, body := runHandler(t, fasthttp.MethodPost, createBody, func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoCredentialFields(t, body)
	for _, secret := range [][]byte{
		[]byte("provider-access"),
		[]byte("provider-refresh"),
		[]byte("provider-key"),
		[]byte("provider-token"),
		[]byte("provider-password"),
	} {
		if bytes.Contains(body, secret) {
			t.Fatalf("response leaked provider-specific secret %q: %s", secret, body)
		}
	}

	var created connectionResponse
	decodeJSON(t, body, &created)
	if created.ProviderSpecificData["region"] != "us" {
		t.Fatalf("region = %v, want us", created.ProviderSpecificData["region"])
	}
	nested, ok := created.ProviderSpecificData["nested"].(map[string]any)
	if !ok || nested["mode"] != "readonly" {
		t.Fatalf("nested = %+v, want mode preserved", created.ProviderSpecificData["nested"])
	}
	if _, ok := nested["password"]; ok {
		t.Fatalf("nested password was not redacted: %+v", nested)
	}

	stored, err := s.GetConnection(created.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if stored.ProviderSpecificData["access_token"] != "provider-access" {
		t.Fatalf("stored provider access token = %v, want preserved secret", stored.ProviderSpecificData["access_token"])
	}
}

func TestConnectionsCanonicalizesProviderAliases(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"provider":"codex","name":"codex","auth_type":"api_key","api_key":"sk-test","is_active":true}`, func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
	var created store.Connection
	decodeJSON(t, body, &created)
	if created.Provider != "openai" {
		t.Fatalf("created provider = %q, want openai", created.Provider)
	}
	if conns, err := s.GetConnections("codex"); err != nil || len(conns) != 0 {
		t.Fatalf("codex connections = %d, err=%v; want 0", len(conns), err)
	}
	openAIConnections, err := s.GetConnections("openai")
	if err != nil {
		t.Fatalf("GetConnections openai: %v", err)
	}
	if len(openAIConnections) != 1 {
		t.Fatalf("openai connections = %d, want 1", len(openAIConnections))
	}

	ctx, body = runHandler(t, fasthttp.MethodPut, `{"provider":"github","name":"copilot","auth_type":"oauth","access_token":"access","is_active":true}`, func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var updated store.Connection
	decodeJSON(t, body, &updated)
	if updated.Provider != "github-copilot" {
		t.Fatalf("updated provider = %q, want github-copilot", updated.Provider)
	}
}

func TestConnectionsListIncludesAuthOnlyProviders(t *testing.T) {
	s := newHandlerStore(t)
	apiKey := "minimax-key"
	if err := s.CreateConnection(&store.Connection{
		Provider: "minimax",
		Name:     "minimax",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoCredentialFields(t, body)
	var listed struct {
		Data []store.Connection `json:"data"`
	}
	decodeJSON(t, body, &listed)
	if len(listed.Data) != 1 || listed.Data[0].Provider != "minimax" {
		t.Fatalf("listed = %+v, want minimax connection", listed.Data)
	}
}

func TestConnectionsMissingReturnsNotFound(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "missing")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestConnectionsInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"provider":`, func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func assertNoCredentialFields(t *testing.T, body []byte) {
	t.Helper()

	for _, field := range [][]byte{
		[]byte(`"AccessToken":`),
		[]byte(`"RefreshToken":`),
		[]byte(`"APIKey":`),
		[]byte(`"access_token":`),
		[]byte(`"refresh_token":`),
		[]byte(`"api_key":`),
		[]byte(`"Authorization":`),
		[]byte(`"password":`),
	} {
		if bytes.Contains(body, field) {
			t.Fatalf("response serialized credential field %s: %s", field, body)
		}
	}
}

func assertStoredCredentials(t *testing.T, s *store.Store, id, accessToken, refreshToken, apiKey string) {
	t.Helper()

	conn, err := s.GetConnection(id)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if conn.AccessToken == nil || *conn.AccessToken != accessToken {
		t.Fatalf("stored access token = %v, want %s", conn.AccessToken, accessToken)
	}
	if conn.RefreshToken == nil || *conn.RefreshToken != refreshToken {
		t.Fatalf("stored refresh token = %v, want %s", conn.RefreshToken, refreshToken)
	}
	if conn.APIKey == nil || *conn.APIKey != apiKey {
		t.Fatalf("stored API key = %v, want %s", conn.APIKey, apiKey)
	}
}

func TestConnectionsStoreFailureDoesNotLeakInternals(t *testing.T) {
	s := newHandlerStore(t)
	// Close store to force all operations to fail with a store error.
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	for _, tc := range []struct {
		name   string
		method string
		body   string
		id     string
	}{
		{"list", fasthttp.MethodGet, "", ""},
		{"create", fasthttp.MethodPost, `{"provider":"openai","name":"x","auth_type":"api_key","api_key":"sk","is_active":true}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, body := runHandler(t, tc.method, tc.body, func(ctx *fasthttp.RequestCtx) {
				Connections(ctx, s, tc.id)
			})
			if ctx.Response.StatusCode() < 500 {
				t.Fatalf("status = %d, want 5xx; body=%s", ctx.Response.StatusCode(), body)
			}
			assertNoInternalDetail(t, body)
		})
	}
}

// assertNoInternalDetail checks the response body does not contain
// common internal error substrings that should never reach clients.
func assertNoInternalDetail(t *testing.T, body []byte) {
	t.Helper()
	for _, banned := range []string{"sql", ".db", "sqlite", "/tmp", "/var", "database", "UNIQUE", "no such"} {
		if bytes.Contains(bytes.ToLower(body), bytes.ToLower([]byte(banned))) {
			t.Fatalf("response body leaked internal detail %q: %s", banned, body)
		}
	}
}

func TestConnectionsBulkDisable(t *testing.T) {
	s := newHandlerStore(t)

	// Seed connections with quota.
	createConnWithQuota(t, s, "low", true, 100, 3)
	createConnWithQuota(t, s, "at-threshold", true, 100, 5)
	createConnWithQuota(t, s, "above", true, 100, 10)

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"threshold_percent":5}`, func(ctx *fasthttp.RequestCtx) {
		ConnectionsBulkDisable(ctx, s, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var resp bulkActionResponse
	decodeJSON(t, body, &resp)
	if len(resp.Affected) != 2 {
		t.Fatalf("affected = %d, want 2; ids=%v", len(resp.Affected), resp.Affected)
	}

	// Verify default threshold (5) when omitted.
	createConnWithQuota(t, s, "default-threshold", true, 100, 4)
	ctx, body = runHandler(t, fasthttp.MethodPost, `{}`, func(ctx *fasthttp.RequestCtx) {
		ConnectionsBulkDisable(ctx, s, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var resp2 bulkActionResponse
	decodeJSON(t, body, &resp2)
	if len(resp2.Affected) != 1 {
		t.Fatalf("default threshold affected = %d, want 1", len(resp2.Affected))
	}
}

func TestConnectionsBulkDisableInvalidThreshold(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"threshold_percent":101}`, func(ctx *fasthttp.RequestCtx) {
		ConnectionsBulkDisable(ctx, s, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}

	ctx, body = runHandler(t, fasthttp.MethodPost, `{"threshold_percent":-1}`, func(ctx *fasthttp.RequestCtx) {
		ConnectionsBulkDisable(ctx, s, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestConnectionsBulkDisableAudit(t *testing.T) {
	s := newHandlerStore(t)
	fakeAudit := &fakeConnectionAuditWriter{}

	createConnWithQuota(t, s, "audit-low", true, 100, 2)

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"threshold_percent":5}`, func(ctx *fasthttp.RequestCtx) {
		ConnectionsBulkDisable(ctx, s, fakeAudit)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	if len(fakeAudit.entries) != 1 {
		t.Fatalf("audit entries = %d, want 1", len(fakeAudit.entries))
	}
	if fakeAudit.entries[0].Action != "connection.bulk_disable" {
		t.Fatalf("action = %q, want connection.bulk_disable", fakeAudit.entries[0].Action)
	}
	if !strings.Contains(fakeAudit.entries[0].Details, "threshold=5") {
		t.Fatalf("details missing threshold: %q", fakeAudit.entries[0].Details)
	}
}

func TestConnectionsBulkEnable(t *testing.T) {
	s := newHandlerStore(t)

	createConnWithQuota(t, s, "has-quota", false, 100, 10)
	createConnWithQuota(t, s, "no-quota", false, 100, 0)
	createConnWithQuota(t, s, "already-active", true, 100, 10)

	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ConnectionsBulkEnable(ctx, s, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var resp bulkActionResponse
	decodeJSON(t, body, &resp)
	if len(resp.Affected) != 1 {
		t.Fatalf("affected = %d, want 1; ids=%v", len(resp.Affected), resp.Affected)
	}
}

func TestConnectionsBulkEnableAudit(t *testing.T) {
	s := newHandlerStore(t)
	fakeAudit := &fakeConnectionAuditWriter{}

	createConnWithQuota(t, s, "audit-enable", false, 100, 10)

	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ConnectionsBulkEnable(ctx, s, fakeAudit)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	if len(fakeAudit.entries) != 1 {
		t.Fatalf("audit entries = %d, want 1", len(fakeAudit.entries))
	}
	if fakeAudit.entries[0].Action != "connection.bulk_enable" {
		t.Fatalf("action = %q, want connection.bulk_enable", fakeAudit.entries[0].Action)
	}
}

func TestConnectionsBulkDisableNilStore(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodPost, `{}`, func(ctx *fasthttp.RequestCtx) {
		ConnectionsBulkDisable(ctx, nil, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestConnectionsBulkEnableNilStore(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ConnectionsBulkEnable(ctx, nil, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
}

func createConnWithQuota(t *testing.T, s *store.Store, name string, isActive bool, limit, remaining float64) {
	t.Helper()
	conn := &store.Connection{
		Provider:       "openai",
		Name:           name,
		AuthType:       store.AuthTypeAPIKey,
		IsActive:       isActive,
		QuotaLimit:     &limit,
		QuotaRemaining: &remaining,
	}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection %q: %v", name, err)
	}
}

type fakeConnectionAuditWriter struct {
	entries []store.AuditEntry
}

func (f *fakeConnectionAuditWriter) AppendAudit(entry store.AuditEntry) error {
	f.entries = append(f.entries, entry)
	return nil
}
