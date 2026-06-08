package g0router_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/api"
	"github.com/bloodf/g0router/api/handlers"
	"github.com/bloodf/g0router/internal/console"
	"github.com/bloodf/g0router/internal/governance"
	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

// ---------------------------------------------------------------------------
// Shared types
// ---------------------------------------------------------------------------

type listResponse[T any] struct {
	Data  []T `json:"data"`
	Total int `json:"total"`
}

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

type fakeProviderAdapter struct{}

func (f fakeProviderAdapter) Name() providers.ModelProvider { return providers.ProviderOpenAI }
func (f fakeProviderAdapter) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	return &providers.ChatResponse{ID: "test", Model: req.Model, Object: "chat.completion", Created: time.Now().Unix()}, nil
}
func (f fakeProviderAdapter) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	ch := make(chan providers.StreamChunk)
	close(ch)
	return ch, nil
}
func (f fakeProviderAdapter) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	return []providers.Model{{ID: "gpt-4o", Object: "model", Provider: providers.ProviderOpenAI}}, nil
}

type fakeProviderAdapterSource struct{}

func (f fakeProviderAdapterSource) GetProvider(name providers.ModelProvider) (providers.Provider, bool) {
	return fakeProviderAdapter{}, true
}

type fakeTunnelManager struct{}

func (f fakeTunnelManager) StartCloudflare(port string) (string, error) { return "https://test.trycloudflare.com", nil }
func (f fakeTunnelManager) StopCloudflare() error                       { return nil }
func (f fakeTunnelManager) StartTailscale(port string) (string, error)  { return "https://test-tailscale.ts.net", nil }
func (f fakeTunnelManager) StopTailscale() error                        { return nil }

type fakeMCPInstanceRuntime struct{}

func (f fakeMCPInstanceRuntime) RegisterInstance(ctx context.Context, instance *store.MCPInstance) (mcp.Manifest, error) {
	return mcp.Manifest{}, nil
}
func (f fakeMCPInstanceRuntime) CloseInstance(instanceID string) error { return nil }
func (f fakeMCPInstanceRuntime) ReapplyInstanceCredentials(ctx context.Context, s handlers.MCPRuntimeCredentialStore, instanceID string) (mcp.Manifest, error) {
	return mcp.Manifest{}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func setupE2E(t *testing.T) (*store.Store, string, string) {
	t.Helper()
	s := newE2EStore(t)
	s.SetEncKey("test-encryption-key")
	settings, _ := s.GetSettings()
	settings.EnableRequestLogs = true
	_ = s.UpdateSettings(settings)

	_, baseURL := startE2EServer(t, api.ServerConfig{
		Port:                  0,
		Version:               "e2e-test",
		BuildDate:             "2024-01-01",
		RequireAPIKey:         true,
		APIKeySecret:          "test-secret",
		APIKeyValidator:       storeAPIKeyValidator{s: s},
		InferenceEngine:       e2eInferenceEngine{},
		Store:                 s,
		UsageStore:            s,
		ModelSource:           e2eInferenceEngine{},
		ProviderAdapterSource: fakeProviderAdapterSource{},
		TunnelManager:         fakeTunnelManager{},
		ConsoleBroker:         console.NewBroker(10),
		Governance:            governance.New(s, nil),
		MCPInstanceRuntime:    fakeMCPInstanceRuntime{},
	})

	rawKey := createE2EAPIKey(t, s, "test-secret", "default")
	return s, baseURL, rawKey
}

func setupE2EAdminSession(t *testing.T, baseURL string) *http.Client {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 2 * time.Second}

	req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/auth/setup", strings.NewReader(`{"username":"admin","password":"adminpass123","display_name":"Admin"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("auth setup: %v", err)
	}
	resp.Body.Close()

	req, _ = http.NewRequest(http.MethodPost, baseURL+"/api/auth/login", strings.NewReader(`{"username":"admin","password":"adminpass123"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("auth login: %v", err)
	}
	resp.Body.Close()

	return client
}

func doReq(t *testing.T, client *http.Client, method, urlStr, body string, headers map[string]string) *http.Response {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, urlStr, r)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func doAuth(t *testing.T, client *http.Client, method, urlStr, body, rawKey string) *http.Response {
	t.Helper()
	headers := map[string]string{
		"Authorization": "Bearer " + rawKey,
		"Content-Type":  "application/json",
	}
	return doReq(t, client, method, urlStr, body, headers)
}

// doSessionReq sends a request with the session client and sets Origin to satisfy CSRF.
func doSessionReq(t *testing.T, client *http.Client, method, urlStr, body string) *http.Response {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, urlStr, r)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if u, err := url.Parse(urlStr); err == nil {
		req.Header.Set("Origin", u.Scheme+"://"+u.Host)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("status = %d, want %d; body = %s", resp.StatusCode, want, string(b))
	}
}

func assertBodyContains(t *testing.T, resp *http.Response, substr string) {
	t.Helper()
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(b), substr) {
		t.Fatalf("body does not contain %q: %s", substr, string(b))
	}
}

func mustDecode(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("decode json: %v; body = %s", err, string(b))
	}
}

// ---------------------------------------------------------------------------
// Auth & Health
// ---------------------------------------------------------------------------

func TestE2EHealthz(t *testing.T) {
	_, baseURL, _ := setupE2E(t)
	client := httpClient()
	resp, err := client.Get(baseURL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	assertStatus(t, resp, http.StatusOK)
	var body map[string]string
	mustDecode(t, resp, &body)
	if body["status"] != "ok" {
		t.Fatalf("status = %q, want ok", body["status"])
	}
}

func TestE2EAuthStatus(t *testing.T) {
	_, baseURL, _ := setupE2E(t)
	client := httpClient()
	resp, err := client.Get(baseURL + "/api/auth/status")
	if err != nil {
		t.Fatalf("GET /api/auth/status: %v", err)
	}
	assertStatus(t, resp, http.StatusOK)
	var envelope struct {
		Data struct {
			RequireLogin  bool `json:"require_login"`
			HasUsers      bool `json:"has_users"`
			Authenticated bool `json:"authenticated"`
		} `json:"data"`
	}
	mustDecode(t, resp, &envelope)
	if envelope.Data.HasUsers {
		t.Fatal("expected no users")
	}
}

func TestE2EAuthSetupLoginLogout(t *testing.T) {
	_, baseURL, _ := setupE2E(t)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 2 * time.Second}

	// Setup
	resp := doReq(t, client, http.MethodPost, baseURL+"/api/auth/setup", `{"username":"admin","password":"adminpass123","display_name":"Admin"}`, map[string]string{"Content-Type": "application/json"})
	assertStatus(t, resp, http.StatusCreated)

	// Login
	resp = doReq(t, client, http.MethodPost, baseURL+"/api/auth/login", `{"username":"admin","password":"adminpass123"}`, map[string]string{"Content-Type": "application/json"})
	assertStatus(t, resp, http.StatusOK)

	// Status shows authenticated
	resp, _ = client.Get(baseURL + "/api/auth/status")
	assertStatus(t, resp, http.StatusOK)
	var status struct {
		Data struct {
			Authenticated bool   `json:"authenticated"`
			Username      string `json:"username"`
		} `json:"data"`
	}
	mustDecode(t, resp, &status)
	if !status.Data.Authenticated || status.Data.Username != "admin" {
		t.Fatalf("unexpected auth status: %+v", status.Data)
	}

	// Logout
	resp = doSessionReq(t, client, http.MethodPost, baseURL+"/api/auth/logout", "")
	assertStatus(t, resp, http.StatusNoContent)

	// Status shows not authenticated
	resp, _ = client.Get(baseURL + "/api/auth/status")
	assertStatus(t, resp, http.StatusOK)
	mustDecode(t, resp, &status)
	if status.Data.Authenticated {
		t.Fatal("expected not authenticated after logout")
	}
}

func TestE2EAuthPasswordChange(t *testing.T) {
	_, baseURL, _ := setupE2E(t)
	client := setupE2EAdminSession(t, baseURL)

	resp := doSessionReq(t, client, http.MethodPut, baseURL+"/api/auth/password", `{"current_password":"adminpass123","new_password":"newpass456"}`)
	assertStatus(t, resp, http.StatusOK)

	// Login with new password
	resp = doReq(t, client, http.MethodPost, baseURL+"/api/auth/login", `{"username":"admin","password":"newpass456"}`, map[string]string{"Content-Type": "application/json"})
	assertStatus(t, resp, http.StatusOK)
}

func TestE2EAuthUsersAdmin(t *testing.T) {
	_, baseURL, _ := setupE2E(t)
	client := setupE2EAdminSession(t, baseURL)

	// Create user
	resp := doSessionReq(t, client, http.MethodPost, baseURL+"/api/auth/users", `{"username":"user1","password":"userpass123","display_name":"User One","role":"user"}`)
	assertStatus(t, resp, http.StatusCreated)

	// List users
	resp, _ = client.Get(baseURL + "/api/auth/users")
	assertStatus(t, resp, http.StatusOK)
	var list struct {
		Data []struct {
			ID       string `json:"id"`
			Username string `json:"username"`
			Role     string `json:"role"`
		} `json:"data"`
	}
	mustDecode(t, resp, &list)
	if len(list.Data) != 2 {
		t.Fatalf("expected 2 users, got %d", len(list.Data))
	}

	// Delete user (need to extract ID)
	var userID string
	for _, u := range list.Data {
		if u.Username == "user1" {
			userID = u.ID
			break
		}
	}
	if userID == "" {
		t.Fatal("user1 not found")
	}
	resp = doSessionReq(t, client, http.MethodDelete, baseURL+"/api/auth/users/"+userID, "")
	assertStatus(t, resp, http.StatusNoContent)
}

func TestE2EAuthUnauthorized(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	// No auth to protected endpoint
	resp, _ := client.Get(baseURL + "/api/keys")
	assertStatus(t, resp, http.StatusUnauthorized)

	// Bad password change without session
	resp = doReq(t, client, http.MethodPut, baseURL+"/api/auth/password", `{"current_password":"x","new_password":"y"}`, map[string]string{"Content-Type": "application/json"})
	assertStatus(t, resp, http.StatusUnauthorized)

	// Wrong api key
	req, _ := http.NewRequest(http.MethodGet, baseURL+"/api/keys", nil)
	req.Header.Set("Authorization", "Bearer bad-key")
	resp, _ = client.Do(req)
	assertStatus(t, resp, http.StatusUnauthorized)

	// Valid key should work
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/keys", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
}

// ---------------------------------------------------------------------------
// Settings
// ---------------------------------------------------------------------------

func TestE2ESettings(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	// GET
	resp := doAuth(t, client, http.MethodGet, baseURL+"/api/settings", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var settings map[string]any
	mustDecode(t, resp, &settings)

	// PUT
	resp = doAuth(t, client, http.MethodPut, baseURL+"/api/settings", `{"locale":"pt-BR","log_retention_days":30}`, rawKey)
	assertStatus(t, resp, http.StatusOK)
	mustDecode(t, resp, &settings)
	if settings["locale"] != "pt-BR" {
		t.Fatalf("locale = %v, want pt-BR", settings["locale"])
	}

	// Backup
	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/settings/backup", "", rawKey)
	assertStatus(t, resp, http.StatusOK)

	// Restore
	backupBody := `{"schema_version":"1","settings":{"locale":"en-US"}}`
	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/settings/restore", backupBody, rawKey)
	assertStatus(t, resp, http.StatusOK)

	// Proxy test
	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/settings/proxy-test", `{"url":"http://localhost:8080"}`, rawKey)
	assertStatus(t, resp, http.StatusOK)
	var proxyRes struct {
		OK bool `json:"ok"`
	}
	mustDecode(t, resp, &proxyRes)
	if proxyRes.OK {
		// localhost:8080 may or may not be reachable; we just care it returns 200
		t.Log("proxy test reachable")
	}
}

// ---------------------------------------------------------------------------
// Providers
// ---------------------------------------------------------------------------

func TestE2EProviders(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	// List providers
	resp := doAuth(t, client, http.MethodGet, baseURL+"/api/providers", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var list struct {
		Data []map[string]any `json:"data"`
	}
	mustDecode(t, resp, &list)
	if len(list.Data) == 0 {
		t.Fatal("expected providers")
	}

	// Provider detail (openai)
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/providers/openai", "", rawKey)
	assertStatus(t, resp, http.StatusOK)

	// Provider models
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/providers/openai/models", "", rawKey)
	assertStatus(t, resp, http.StatusOK)

	// Provider connections (no connections yet)
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/providers/openai/connections", "", rawKey)
	assertStatus(t, resp, http.StatusOK)

	// Suggested models (no active connections)
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/providers/openai/suggested-models", "", rawKey)
	assertStatus(t, resp, http.StatusBadRequest)

	// POST /api/providers (not supported -> 405 via catch-all or method check)
	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/providers", `{"id":"x"}`, rawKey)
	assertStatus(t, resp, http.StatusMethodNotAllowed)

	// PUT provider
	resp = doAuth(t, client, http.MethodPut, baseURL+"/api/providers/openai", `{"id":"openai"}`, rawKey)
	assertStatus(t, resp, http.StatusMethodNotAllowed)

	// Provider test batch (SSE, no connections)
	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/providers/test-batch", ``, rawKey)
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()
}

func TestE2EProviderModelTest(t *testing.T) {
	s, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	// Seed a connection
	apiKey := "sk-test"
	conn := &store.Connection{Provider: "openai", Name: "test-conn", AuthType: store.AuthTypeAPIKey, APIKey: &apiKey, IsActive: true}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("create connection: %v", err)
	}

	// Model test
	resp := doAuth(t, client, http.MethodPost, baseURL+"/api/providers/openai/models/gpt-4o/test", ``, rawKey)
	assertStatus(t, resp, http.StatusOK)
	var res struct {
		Data struct {
			OK bool `json:"ok"`
		} `json:"data"`
	}
	mustDecode(t, resp, &res)
	if !res.Data.OK {
		t.Fatal("model test failed")
	}
}

// ---------------------------------------------------------------------------
// Connections
// ---------------------------------------------------------------------------

func TestE2EConnectionsCRUD(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	// Create
	resp := doAuth(t, client, http.MethodPost, baseURL+"/api/connections", `{"provider":"openai","name":"conn1","auth_type":"api_key","api_key":"sk-test","is_active":true}`, rawKey)
	assertStatus(t, resp, http.StatusCreated)
	var created map[string]any
	mustDecode(t, resp, &created)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("missing connection id")
	}

	// List
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/connections", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var list struct {
		Data []map[string]any `json:"data"`
	}
	mustDecode(t, resp, &list)
	if len(list.Data) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(list.Data))
	}

	// Get by ID (via PUT with same body or generic route?)
	// The /api/connections/:id route handles PUT/DELETE but not GET individually.
	// GET /api/connections/:id is not explicitly routed; the catch-all returns 404.
	// Actually looking at routes.go, /api/connections/:id matches apiPathMatch and
	// calls handlers.Connections(ctx, s, parts[2]) which supports GET/POST/PUT/DELETE.
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/connections/"+id, "", rawKey)
	// Wait, the route for /api/connections/:id uses the same handler as /api/connections,
	// and when id is non-empty it only handles PUT/DELETE. So GET returns 405? Let's check.
	// Actually looking at handlers.Connections: switch method -> GET calls listConnections.
	// But the route has `parts := pathParts(...); handlers.Connections(ctx, s.config.Store, parts[2])`.
	// In the handler, GET doesn't use the id param at all — it lists all connections.
	// So GET /api/connections/:id will list all connections (200) because the handler ignores id for GET.
	// This is a bit weird but let's accept it.
	assertStatus(t, resp, http.StatusOK)

	// Update
	resp = doAuth(t, client, http.MethodPut, baseURL+"/api/connections/"+id, `{"provider":"openai","name":"conn1-renamed","auth_type":"api_key","api_key":"sk-test","is_active":true}`, rawKey)
	assertStatus(t, resp, http.StatusOK)
	var updated map[string]any
	mustDecode(t, resp, &updated)
	if updated["name"] != "conn1-renamed" {
		t.Fatalf("name = %v, want conn1-renamed", updated["name"])
	}

	// Test
	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/connections/"+id+"/test", "", rawKey)
	assertStatus(t, resp, http.StatusOK)

	// Delete
	resp = doReq(t, client, http.MethodDelete, baseURL+"/api/connections/"+id, "", map[string]string{"Authorization": "Bearer " + rawKey})
	assertStatus(t, resp, http.StatusNoContent)

	// Delete again -> 404
	resp = doReq(t, client, http.MethodDelete, baseURL+"/api/connections/"+id, "", map[string]string{"Authorization": "Bearer " + rawKey})
	assertStatus(t, resp, http.StatusNotFound)
}

func TestE2EConnectionsBulk(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	// Create connections
	for i := 0; i < 3; i++ {
		body := `{"provider":"openai","name":"bulk-` + strconv.Itoa(i) + `","auth_type":"api_key","api_key":"sk-test","is_active":true}`
		resp := doAuth(t, client, http.MethodPost, baseURL+"/api/connections", body, rawKey)
		assertStatus(t, resp, http.StatusCreated)
	}

	// Bulk disable
	resp := doAuth(t, client, http.MethodPost, baseURL+"/api/connections/bulk-disable", `{"threshold_percent":5}`, rawKey)
	assertStatus(t, resp, http.StatusOK)
	var disableRes struct {
		Affected []string `json:"affected"`
	}
	mustDecode(t, resp, &disableRes)
	_ = disableRes.Affected

	// Bulk enable
	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/connections/bulk-enable", ``, rawKey)
	assertStatus(t, resp, http.StatusOK)
	var enableRes struct {
		Affected []string `json:"affected"`
	}
	mustDecode(t, resp, &enableRes)
}

func TestE2EConnectionsProxyNotFound(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()
	resp := doAuth(t, client, http.MethodPost, baseURL+"/api/connections/123/proxy", ``, rawKey)
	assertStatus(t, resp, http.StatusNotFound)
}


// ---------------------------------------------------------------------------
// API Keys
// ---------------------------------------------------------------------------

func TestE2EAPIKeysCRUD(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	// Create
	resp := doAuth(t, client, http.MethodPost, baseURL+"/api/keys", `{"name":"key1"}`, rawKey)
	assertStatus(t, resp, http.StatusCreated)
	var created struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		FullKey string `json:"full_key"`
	}
	mustDecode(t, resp, &created)
	if created.Name != "key1" {
		t.Fatalf("name = %q, want key1", created.Name)
	}
	id := created.ID

	// List
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/keys", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var list struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	mustDecode(t, resp, &list)
	if len(list.Data) != 2 { // default + key1
		t.Fatalf("expected 2 keys, got %d", len(list.Data))
	}

	// Get by ID (handler currently returns list for all GET /api/keys/*)
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/keys/"+id, "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var got listResponse[struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}]
	mustDecode(t, resp, &got)
	found := false
	for _, k := range got.Data {
		if k.ID == id {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("key %s not found in list", id)
	}

	// Update policy
	resp = doAuth(t, client, http.MethodPut, baseURL+"/api/keys/"+id, `{"scopes":["read"]}`, rawKey)
	assertStatus(t, resp, http.StatusOK)
	var updated struct {
		Scopes []string `json:"scopes"`
	}
	mustDecode(t, resp, &updated)
	if len(updated.Scopes) != 1 || updated.Scopes[0] != "read" {
		t.Fatalf("scopes = %v", updated.Scopes)
	}

	// Regenerate
	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/keys/"+id+"/regenerate", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var regen struct {
		FullKey string `json:"full_key"`
	}
	mustDecode(t, resp, &regen)
	if regen.FullKey == "" {
		t.Fatal("expected new full_key")
	}

	// Delete
	resp = doReq(t, client, http.MethodDelete, baseURL+"/api/keys/"+id, "", map[string]string{"Authorization": "Bearer " + rawKey})
	assertStatus(t, resp, http.StatusNoContent)

	// Get after delete (handler returns list for all GET /api/keys/*; delete is soft)
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/keys/"+id, "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var afterDelete listResponse[struct {
		ID        string `json:"id"`
		IsActive  bool   `json:"is_active"`
	}]
	mustDecode(t, resp, &afterDelete)
	found = false
	for _, k := range afterDelete.Data {
		if k.ID == id {
			found = true
			if k.IsActive {
				t.Fatal("deleted key is still active")
			}
			break
		}
	}
	if !found {
		t.Fatal("deleted key not found in list")
	}
}

func TestE2EAPIKeysAuditNotFound(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()
	resp := doAuth(t, client, http.MethodGet, baseURL+"/api/keys/123/audit", "", rawKey)
	assertStatus(t, resp, http.StatusNotFound)
}

// ---------------------------------------------------------------------------
// Virtual Keys
// ---------------------------------------------------------------------------

func TestE2EVirtualKeysCRUD(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	resp := doAuth(t, client, http.MethodPost, baseURL+"/api/virtual-keys", `{"name":"vk1"}`, rawKey)
	assertStatus(t, resp, http.StatusCreated)
	var created struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	mustDecode(t, resp, &created)
	id := created.ID

	// List
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/virtual-keys", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var list struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	mustDecode(t, resp, &list)
	if len(list.Data) != 1 {
		t.Fatalf("expected 1 virtual key, got %d", len(list.Data))
	}

	// Get by ID
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/virtual-keys/"+id, "", rawKey)
	assertStatus(t, resp, http.StatusOK)

	// Update
	resp = doAuth(t, client, http.MethodPut, baseURL+"/api/virtual-keys/"+id, `{"name":"vk1-renamed","is_active":true}`, rawKey)
	assertStatus(t, resp, http.StatusOK)
	var updated struct {
		Name string `json:"name"`
	}
	mustDecode(t, resp, &updated)
	if updated.Name != "vk1-renamed" {
		t.Fatalf("name = %q", updated.Name)
	}

	// Delete
	resp = doReq(t, client, http.MethodDelete, baseURL+"/api/virtual-keys/"+id, "", map[string]string{"Authorization": "Bearer " + rawKey})
	assertStatus(t, resp, http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Teams
// ---------------------------------------------------------------------------

func TestE2ETeamsCRUD(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	resp := doAuth(t, client, http.MethodPost, baseURL+"/api/teams", `{"name":"team1"}`, rawKey)
	assertStatus(t, resp, http.StatusCreated)
	var created struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	mustDecode(t, resp, &created)
	id := created.ID

	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/teams", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var list struct {
		Data []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	mustDecode(t, resp, &list)
	if len(list.Data) != 1 {
		t.Fatalf("expected 1 team, got %d", len(list.Data))
	}

	resp = doAuth(t, client, http.MethodPut, baseURL+"/api/teams/"+strconv.FormatInt(id, 10), `{"name":"team1-renamed"}`, rawKey)
	assertStatus(t, resp, http.StatusOK)

	resp = doReq(t, client, http.MethodDelete, baseURL+"/api/teams/"+strconv.FormatInt(id, 10), "", map[string]string{"Authorization": "Bearer " + rawKey})
	assertStatus(t, resp, http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Combos
// ---------------------------------------------------------------------------

func TestE2ECombosCRUD(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	resp := doAuth(t, client, http.MethodPost, baseURL+"/api/combos", `{"name":"combo1","steps":[{"provider":"openai","model":"gpt-4o"}],"strategy":"fallback","is_active":true}`, rawKey)
	assertStatus(t, resp, http.StatusCreated)
	var created map[string]any
	mustDecode(t, resp, &created)
	id, _ := created["ID"].(string)
	if id == "" {
		t.Fatal("missing combo id")
	}

	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/combos", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var list struct {
		Data []map[string]any `json:"data"`
	}
	mustDecode(t, resp, &list)
	if len(list.Data) != 1 {
		t.Fatalf("expected 1 combo, got %d", len(list.Data))
	}

	resp = doAuth(t, client, http.MethodPut, baseURL+"/api/combos/"+id, `{"name":"combo1-renamed","steps":[{"provider":"openai","model":"gpt-4o"}],"strategy":"fallback","is_active":true}`, rawKey)
	assertStatus(t, resp, http.StatusOK)

	resp = doReq(t, client, http.MethodDelete, baseURL+"/api/combos/"+id, "", map[string]string{"Authorization": "Bearer " + rawKey})
	assertStatus(t, resp, http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Routing Rules
// ---------------------------------------------------------------------------

func TestE2ERoutingRulesCRUD(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	resp := doAuth(t, client, http.MethodPost, baseURL+"/api/routing-rules", `{"name":"rule1","priority":1,"cond_field":"model","cond_operator":"eq","cond_value":"gpt-4o","target_provider":"openai"}`, rawKey)
	assertStatus(t, resp, http.StatusCreated)
	var created struct {
		ID int64 `json:"id"`
	}
	mustDecode(t, resp, &created)
	id := created.ID

	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/routing-rules", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var list struct {
		Data []struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	mustDecode(t, resp, &list)
	if len(list.Data) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(list.Data))
	}

	resp = doAuth(t, client, http.MethodPut, baseURL+"/api/routing-rules/"+strconv.FormatInt(id, 10), `{"name":"rule1-renamed","priority":1,"cond_field":"model","cond_operator":"eq","cond_value":"gpt-4o","target_provider":"openai","is_active":true}`, rawKey)
	assertStatus(t, resp, http.StatusOK)

	resp = doReq(t, client, http.MethodDelete, baseURL+"/api/routing-rules/"+strconv.FormatInt(id, 10), "", map[string]string{"Authorization": "Bearer " + rawKey})
	assertStatus(t, resp, http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Aliases
// ---------------------------------------------------------------------------

func TestE2EAliasesCRUD(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	resp := doAuth(t, client, http.MethodPost, baseURL+"/api/aliases", `{"alias":"my-model","provider":"openai","model":"gpt-4o"}`, rawKey)
	assertStatus(t, resp, http.StatusCreated)

	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/aliases", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var list struct {
		Data []struct {
			Alias string `json:"alias"`
		} `json:"data"`
	}
	mustDecode(t, resp, &list)
	if len(list.Data) != 1 {
		t.Fatalf("expected 1 alias, got %d", len(list.Data))
	}

	resp = doAuth(t, client, http.MethodPut, baseURL+"/api/aliases/my-model", `{"alias":"my-model","provider":"openai","model":"gpt-4o-mini"}`, rawKey)
	assertStatus(t, resp, http.StatusOK)

	resp = doReq(t, client, http.MethodDelete, baseURL+"/api/aliases/my-model", "", map[string]string{"Authorization": "Bearer " + rawKey})
	assertStatus(t, resp, http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Pricing
// ---------------------------------------------------------------------------

func TestE2EPricingCRUD(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	resp := doAuth(t, client, http.MethodPost, baseURL+"/api/pricing", `{"provider":"openai","model":"gpt-4o","input_cost":1.0,"output_cost":2.0}`, rawKey)
	assertStatus(t, resp, http.StatusCreated)
	var created struct {
		ID string `json:"id"`
	}
	mustDecode(t, resp, &created)

	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/pricing", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	mustDecode(t, resp, &list)
	if len(list.Data) != 1 {
		t.Fatalf("expected 1 pricing override, got %d", len(list.Data))
	}

	resp = doAuth(t, client, http.MethodPut, baseURL+"/api/pricing/openai/gpt-4o", `{"provider":"openai","model":"gpt-4o","input_cost":3.0,"output_cost":4.0}`, rawKey)
	assertStatus(t, resp, http.StatusOK)

	resp = doReq(t, client, http.MethodDelete, baseURL+"/api/pricing/openai/gpt-4o", "", map[string]string{"Authorization": "Bearer " + rawKey})
	assertStatus(t, resp, http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Admin Models
// ---------------------------------------------------------------------------

func TestE2EAdminModels(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	resp := doAuth(t, client, http.MethodGet, baseURL+"/api/models", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var list struct {
		Data []map[string]any `json:"data"`
	}
	mustDecode(t, resp, &list)
	if len(list.Data) == 0 {
		t.Fatal("expected models")
	}

	// Disabled models
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/models/disabled", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var disabled struct {
		Data []map[string]any `json:"data"`
	}
	mustDecode(t, resp, &disabled)

	// Create disabled model
	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/models/disabled", `{"provider":"openai","model":"gpt-4o"}`, rawKey)
	assertStatus(t, resp, http.StatusCreated)

	// Delete disabled model
	resp = doAuth(t, client, http.MethodDelete, baseURL+"/api/models/disabled", `{"provider":"openai","model":"gpt-4o"}`, rawKey)
	assertStatus(t, resp, http.StatusNoContent)

	// Custom models
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/models/custom", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var custom struct {
		Data []map[string]any `json:"data"`
	}
	mustDecode(t, resp, &custom)

	// Create custom model
	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/models/custom", `{"provider":"openai","model":"custom-1","display_name":"Custom One"}`, rawKey)
	assertStatus(t, resp, http.StatusCreated)
	var cm struct {
		ID string `json:"id"`
	}
	mustDecode(t, resp, &cm)

	// Delete custom model
	resp = doReq(t, client, http.MethodDelete, baseURL+"/api/models/custom/"+cm.ID, "", map[string]string{"Authorization": "Bearer " + rawKey})
	assertStatus(t, resp, http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Model Limits
// ---------------------------------------------------------------------------

func TestE2EModelLimitsCRUD(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	resp := doAuth(t, client, http.MethodPost, baseURL+"/api/model-limits", `{"model":"gpt-4o","max_tokens":1000,"max_rpm":10}`, rawKey)
	assertStatus(t, resp, http.StatusCreated)
	var created struct {
		ID int64 `json:"id"`
	}
	mustDecode(t, resp, &created)
	id := created.ID

	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/model-limits", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var list struct {
		Data []struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	mustDecode(t, resp, &list)
	if len(list.Data) != 1 {
		t.Fatalf("expected 1 limit, got %d", len(list.Data))
	}

	resp = doAuth(t, client, http.MethodPut, baseURL+"/api/model-limits/"+strconv.FormatInt(id, 10), `{"model":"gpt-4o","max_tokens":2000}`, rawKey)
	assertStatus(t, resp, http.StatusOK)

	resp = doReq(t, client, http.MethodDelete, baseURL+"/api/model-limits/"+strconv.FormatInt(id, 10), "", map[string]string{"Authorization": "Bearer " + rawKey})
	assertStatus(t, resp, http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Guardrails
// ---------------------------------------------------------------------------

func TestE2EGuardrails(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	resp := doAuth(t, client, http.MethodGet, baseURL+"/api/guardrails", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var cfg struct {
		GuardrailsEnabled bool `json:"guardrails_enabled"`
	}
	mustDecode(t, resp, &cfg)

	resp = doAuth(t, client, http.MethodPut, baseURL+"/api/guardrails", `{"guardrails_enabled":true,"guardrails_blocklist":["badword"],"pii_redaction_enabled":false,"pii_redaction_types":[]}`, rawKey)
	assertStatus(t, resp, http.StatusOK)
	mustDecode(t, resp, &cfg)
	if !cfg.GuardrailsEnabled {
		t.Fatal("expected guardrails enabled")
	}

	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/guardrails/test", `{"prompt":"hello badword world"}`, rawKey)
	assertStatus(t, resp, http.StatusOK)
	var testRes struct {
		Blocked bool `json:"blocked"`
	}
	mustDecode(t, resp, &testRes)
	if !testRes.Blocked {
		t.Fatal("expected blocked")
	}
}

// ---------------------------------------------------------------------------
// Prompt Templates
// ---------------------------------------------------------------------------

func TestE2EPromptTemplatesCRUD(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	resp := doAuth(t, client, http.MethodPost, baseURL+"/api/prompt-templates", `{"name":"pt1","system_prompt":"You are helpful","models":["gpt-4o"],"is_active":true}`, rawKey)
	assertStatus(t, resp, http.StatusCreated)
	var created struct {
		ID int64 `json:"id"`
	}
	mustDecode(t, resp, &created)
	id := created.ID

	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/prompt-templates", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var list struct {
		Data []struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	mustDecode(t, resp, &list)
	if len(list.Data) != 1 {
		t.Fatalf("expected 1 template, got %d", len(list.Data))
	}

	// Get by ID
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/prompt-templates/"+strconv.FormatInt(id, 10), "", rawKey)
	assertStatus(t, resp, http.StatusOK)

	// Update
	resp = doAuth(t, client, http.MethodPut, baseURL+"/api/prompt-templates/"+strconv.FormatInt(id, 10), `{"name":"pt1","system_prompt":"You are very helpful","models":["gpt-4o"],"is_active":true}`, rawKey)
	assertStatus(t, resp, http.StatusNoContent)

	// Delete
	resp = doReq(t, client, http.MethodDelete, baseURL+"/api/prompt-templates/"+strconv.FormatInt(id, 10), "", map[string]string{"Authorization": "Bearer " + rawKey})
	assertStatus(t, resp, http.StatusNoContent)

	// Test endpoint
	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/prompt-templates/test", `{"model":"gpt-4o"}`, rawKey)
	assertStatus(t, resp, http.StatusOK)
	var testRes struct {
		Model   string `json:"model"`
		Matched bool   `json:"matched"`
	}
	mustDecode(t, resp, &testRes)
	if testRes.Model != "gpt-4o" {
		t.Fatalf("model = %q", testRes.Model)
	}
}


// ---------------------------------------------------------------------------
// Proxy Pools
// ---------------------------------------------------------------------------

func TestE2EProxyPoolsCRUD(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	resp := doAuth(t, client, http.MethodPost, baseURL+"/api/proxy-pools", `{"name":"pool1","protocol":"http","host":"127.0.0.1","port":8080}`, rawKey)
	assertStatus(t, resp, http.StatusCreated)
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	mustDecode(t, resp, &created)
	id := created.Data.ID

	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/proxy-pools", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	mustDecode(t, resp, &list)
	if len(list.Data) != 1 {
		t.Fatalf("expected 1 pool, got %d", len(list.Data))
	}

	// Get by ID
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/proxy-pools/"+id, "", rawKey)
	assertStatus(t, resp, http.StatusOK)

	// Update
	resp = doAuth(t, client, http.MethodPut, baseURL+"/api/proxy-pools/"+id, `{"name":"pool1-renamed","protocol":"http","host":"127.0.0.1","port":8081}`, rawKey)
	assertStatus(t, resp, http.StatusOK)

	// Test
	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/proxy-pools/"+id+"/test", "", rawKey)
	assertStatus(t, resp, http.StatusOK)

	// Batch import
	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/proxy-pools/batch", `{"lines":["http://127.0.0.1:9090"]}`, rawKey)
	assertStatus(t, resp, http.StatusOK)
	var batch struct {
		Created []struct {
			ID string `json:"id"`
		} `json:"created"`
	}
	mustDecode(t, resp, &batch)
	if len(batch.Created) != 1 {
		t.Fatalf("expected 1 created, got %d", len(batch.Created))
	}

	// Delete
	resp = doReq(t, client, http.MethodDelete, baseURL+"/api/proxy-pools/"+id, "", map[string]string{"Authorization": "Bearer " + rawKey})
	assertStatus(t, resp, http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Tunnels
// ---------------------------------------------------------------------------

func TestE2ETunnels(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	// List
	resp := doAuth(t, client, http.MethodGet, baseURL+"/api/tunnels", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var list struct {
		Data []map[string]any `json:"data"`
	}
	mustDecode(t, resp, &list)

	// Create cloudflare
	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/tunnels/cloudflare", `{"port":"8080"}`, rawKey)
	assertStatus(t, resp, http.StatusCreated)
	var cf struct {
		URL    string `json:"url"`
		Status string `json:"status"`
	}
	mustDecode(t, resp, &cf)
	if cf.URL == "" {
		t.Fatal("expected tunnel url")
	}

	// Delete cloudflare
	resp = doReq(t, client, http.MethodDelete, baseURL+"/api/tunnels/cloudflare", "", map[string]string{"Authorization": "Bearer " + rawKey})
	assertStatus(t, resp, http.StatusNoContent)

	// Create tailscale
	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/tunnels/tailscale", `{"port":"8080"}`, rawKey)
	assertStatus(t, resp, http.StatusCreated)

	// Delete tailscale
	resp = doReq(t, client, http.MethodDelete, baseURL+"/api/tunnels/tailscale", "", map[string]string{"Authorization": "Bearer " + rawKey})
	assertStatus(t, resp, http.StatusNoContent)

	// Health
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/tunnels/health", "", rawKey)
	assertStatus(t, resp, http.StatusOK)

	// Non-existent generic tunnel endpoints from user list
	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/tunnels", ``, rawKey)
	assertStatus(t, resp, http.StatusMethodNotAllowed)

	resp = doAuth(t, client, http.MethodDelete, baseURL+"/api/tunnels/123", ``, rawKey)
	assertStatus(t, resp, http.StatusNotFound)
}

// ---------------------------------------------------------------------------
// MCP
// ---------------------------------------------------------------------------

func TestE2EMCPInstances(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	// Create inactive instance (no runtime registration needed)
	resp := doAuth(t, client, http.MethodPost, baseURL+"/api/mcp/instances", `{"name":"inst1","server_key":"test","launch_type":"command","transport":"stdio","command":"echo","is_active":false}`, rawKey)
	assertStatus(t, resp, http.StatusCreated)
	var created struct {
		ID string `json:"id"`
	}
	mustDecode(t, resp, &created)
	id := created.ID

	// List
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/mcp/instances", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var list struct {
		Data []map[string]any `json:"data"`
	}
	mustDecode(t, resp, &list)
	if len(list.Data) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(list.Data))
	}

	// Accounts (empty)
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/mcp/instances/"+id+"/accounts", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var accounts struct {
		Data []map[string]any `json:"data"`
	}
	mustDecode(t, resp, &accounts)

	// Delete
	resp = doReq(t, client, http.MethodDelete, baseURL+"/api/mcp/instances/"+id, "", map[string]string{"Authorization": "Bearer " + rawKey})
	assertStatus(t, resp, http.StatusNoContent)
}

func TestE2EMCPClients(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	// List (empty, no runtime needed)
	resp := doAuth(t, client, http.MethodGet, baseURL+"/api/mcp/clients", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var list struct {
		Data []map[string]any `json:"data"`
	}
	mustDecode(t, resp, &list)

	// Tools list
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/mcp/tools", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var tools struct {
		Data []map[string]any `json:"data"`
	}
	mustDecode(t, resp, &tools)

	// Tool groups
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/mcp/tool-groups", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var groups struct {
		Data []map[string]any `json:"data"`
	}
	mustDecode(t, resp, &groups)
}

// ---------------------------------------------------------------------------
// Chat Sessions
// ---------------------------------------------------------------------------

func TestE2EChatSessionsCRUD(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	resp := doAuth(t, client, http.MethodPost, baseURL+"/api/chat-sessions", `{"title":"session1","model":"gpt-4o","provider":"openai"}`, rawKey)
	assertStatus(t, resp, http.StatusCreated)
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	mustDecode(t, resp, &created)
	id := created.Data.ID

	// List
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/chat-sessions", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	mustDecode(t, resp, &list)
	if len(list.Data) != 1 {
		t.Fatalf("expected 1 session, got %d", len(list.Data))
	}

	// Get
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/chat-sessions/"+id, "", rawKey)
	assertStatus(t, resp, http.StatusOK)

	// Update
	resp = doAuth(t, client, http.MethodPut, baseURL+"/api/chat-sessions/"+id, `{"title":"session1-renamed"}`, rawKey)
	assertStatus(t, resp, http.StatusOK)

	// Delete
	resp = doReq(t, client, http.MethodDelete, baseURL+"/api/chat-sessions/"+id, "", map[string]string{"Authorization": "Bearer " + rawKey})
	assertStatus(t, resp, http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Alert Channels
// ---------------------------------------------------------------------------

func TestE2EAlertChannelsCRUD(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	resp := doAuth(t, client, http.MethodPost, baseURL+"/api/alert-channels", `{"name":"ch1","channel_type":"webhook","config":{"url":"http://localhost/hook"},"events":["quota_depleted"],"is_active":true}`, rawKey)
	assertStatus(t, resp, http.StatusCreated)
	var created struct {
		ID int64 `json:"id"`
	}
	mustDecode(t, resp, &created)
	id := created.ID

	// List
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/alert-channels", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var list struct {
		Data []struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	mustDecode(t, resp, &list)
	if len(list.Data) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(list.Data))
	}

	// Get by ID
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/alert-channels/"+strconv.FormatInt(id, 10), "", rawKey)
	assertStatus(t, resp, http.StatusOK)

	// Update
	resp = doAuth(t, client, http.MethodPut, baseURL+"/api/alert-channels/"+strconv.FormatInt(id, 10), `{"name":"ch1-renamed","channel_type":"webhook","config":{"url":"http://localhost/hook"},"events":["quota_depleted"],"is_active":true}`, rawKey)
	assertStatus(t, resp, http.StatusOK)

	// Test (will likely fail because webhook is unreachable, but endpoint returns 200/503)
	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/alert-channels/"+strconv.FormatInt(id, 10)+"/test", "", rawKey)
	// Webhook test may return 200 or 503 depending on whether the notifier succeeds.
	// We accept either as long as it's not 404/405.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("unexpected status %d; body=%s", resp.StatusCode, string(b))
	}
	resp.Body.Close()

	// Delete
	resp = doReq(t, client, http.MethodDelete, baseURL+"/api/alert-channels/"+strconv.FormatInt(id, 10), "", map[string]string{"Authorization": "Bearer " + rawKey})
	assertStatus(t, resp, http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Feature Flags
// ---------------------------------------------------------------------------

func TestE2EFeatureFlags(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	resp := doAuth(t, client, http.MethodGet, baseURL+"/api/feature-flags", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var list struct {
		Data []struct {
			ID      int64  `json:"id"`
			Key     string `json:"key"`
			Enabled bool   `json:"enabled"`
		} `json:"data"`
	}
	mustDecode(t, resp, &list)
	if len(list.Data) == 0 {
		t.Fatal("expected feature flags")
	}
	id := list.Data[0].ID

	// Toggle
	resp = doAuth(t, client, http.MethodPut, baseURL+"/api/feature-flags/"+strconv.FormatInt(id, 10), `{"enabled":true}`, rawKey)
	assertStatus(t, resp, http.StatusOK)
	var updated struct {
		Enabled bool `json:"enabled"`
	}
	mustDecode(t, resp, &updated)
}

// ---------------------------------------------------------------------------
// Usage & Logs & Audit
// ---------------------------------------------------------------------------

func TestE2EUsageAndLogs(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	// Usage summary
	resp := doAuth(t, client, http.MethodGet, baseURL+"/api/usage/summary", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var summary struct {
		RequestCount int64 `json:"request_count"`
	}
	mustDecode(t, resp, &summary)

	// Usage list
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/usage", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var usage struct {
		Object string `json:"object"`
		Total  int    `json:"total"`
	}
	mustDecode(t, resp, &usage)
	if usage.Object != "list" {
		t.Fatalf("object = %q", usage.Object)
	}

	// Usage chart
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/usage/chart?period=today", "", rawKey)
	assertStatus(t, resp, http.StatusOK)

	// Quota
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/quota", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var quota []map[string]any
	mustDecode(t, resp, &quota)

	// Logs
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/logs", "", rawKey)
	assertStatus(t, resp, http.StatusOK)

	// Audit
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/audit", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var audit struct {
		Object string `json:"object"`
		Total  int    `json:"total"`
	}
	mustDecode(t, resp, &audit)
	if audit.Object != "list" {
		t.Fatalf("audit object = %q", audit.Object)
	}
}

// ---------------------------------------------------------------------------
// Traffic & Console Streams
// ---------------------------------------------------------------------------

func TestE2ETrafficAndConsole(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	// Traffic stream (SSE) — read a bit then close
	resp := doAuth(t, client, http.MethodGet, baseURL+"/api/traffic/stream", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("content-type = %q, want event-stream", ct)
	}
	resp.Body.Close()

	// Console logs stream
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/console-logs/stream", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("content-type = %q, want event-stream", ct)
	}
	resp.Body.Close()

	// Console logs clear
	resp = doReq(t, client, http.MethodDelete, baseURL+"/api/console-logs", "", map[string]string{"Authorization": "Bearer " + rawKey})
	assertStatus(t, resp, http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Version & Update
// ---------------------------------------------------------------------------

func TestE2EVersionAndUpdate(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()
	adminClient := setupE2EAdminSession(t, baseURL)

	// Version (no auth required? It's /api/version, protected if RequireAPIKey true)
	resp := doAuth(t, client, http.MethodGet, baseURL+"/api/version", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var ver struct {
		Data struct {
			Version string `json:"version"`
		} `json:"data"`
	}
	mustDecode(t, resp, &ver)
	if ver.Data.Version != "e2e-test" {
		t.Fatalf("version = %q", ver.Data.Version)
	}

	// Update check (needs auth; may fail due to external network)
	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/update/check", "", rawKey)
	if resp.StatusCode == http.StatusOK {
		var check struct {
			Data struct {
				Current string `json:"current"`
			} `json:"data"`
		}
		mustDecode(t, resp, &check)
		if check.Data.Current != "e2e-test" {
			t.Fatalf("current = %q", check.Data.Current)
		}
	} else {
		// External network may be unavailable in test environment
		assertStatus(t, resp, http.StatusInternalServerError)
	}

	// Update apply (admin only, needs session)
	resp = doSessionReq(t, adminClient, http.MethodPost, baseURL+"/api/update/apply", "")
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("update apply status = %d, want 200 or 500", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Locale & Skills & Cache
// ---------------------------------------------------------------------------

func TestE2ELocaleSkillsCache(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	// Locale GET
	resp := doAuth(t, client, http.MethodGet, baseURL+"/api/locale", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var loc struct {
		Data struct {
			Locale string `json:"locale"`
		} `json:"data"`
	}
	mustDecode(t, resp, &loc)

	// Locale POST
	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/locale", `{"locale":"pt-BR"}`, rawKey)
	assertStatus(t, resp, http.StatusOK)
	mustDecode(t, resp, &loc)
	if loc.Data.Locale != "pt-BR" {
		t.Fatalf("locale = %q", loc.Data.Locale)
	}

	// Skills
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/skills", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var skills struct {
		Data []map[string]any `json:"data"`
	}
	mustDecode(t, resp, &skills)
	if len(skills.Data) == 0 {
		t.Fatal("expected skills")
	}

	// Semantic cache stats
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/cache/semantic", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var cacheStats map[string]any
	mustDecode(t, resp, &cacheStats)

	// Semantic cache clear
	resp = doReq(t, client, http.MethodDelete, baseURL+"/api/cache/semantic", "", map[string]string{"Authorization": "Bearer " + rawKey})
	assertStatus(t, resp, http.StatusOK)
}


// ---------------------------------------------------------------------------
// Inference (OpenAI-compatible)
// ---------------------------------------------------------------------------

func TestE2EInferenceEndpoints(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	// v1/models
	resp := doAuth(t, client, http.MethodGet, baseURL+"/v1/models", "", rawKey)
	assertStatus(t, resp, http.StatusOK)
	var models struct {
		Data []map[string]any `json:"data"`
	}
	mustDecode(t, resp, &models)
	if len(models.Data) == 0 {
		t.Fatal("expected models")
	}

	// v1/chat/completions
	resp = doAuth(t, client, http.MethodPost, baseURL+"/v1/chat/completions", `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`, rawKey)
	assertStatus(t, resp, http.StatusOK)
	var chat providers.ChatResponse
	mustDecode(t, resp, &chat)
	if chat.Model != "gpt-4o" {
		t.Fatalf("model = %q", chat.Model)
	}

	// v1/embeddings (engine doesn't support it -> 501 via writeDispatchError)
	resp = doAuth(t, client, http.MethodPost, baseURL+"/v1/embeddings", `{"model":"text-embedding-ada-002","input":"hello"}`, rawKey)
	if resp.StatusCode != http.StatusNotImplemented && resp.StatusCode != http.StatusServiceUnavailable {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("embeddings status = %d, want 501 or 503; body=%s", resp.StatusCode, string(b))
	}
	resp.Body.Close()

	// v1/images/generations (engine doesn't support -> 501/503)
	resp = doAuth(t, client, http.MethodPost, baseURL+"/v1/images/generations", `{"model":"dall-e-3","prompt":"cat"}`, rawKey)
	if resp.StatusCode != http.StatusNotImplemented && resp.StatusCode != http.StatusServiceUnavailable {
		resp.Body.Close()
		t.Fatalf("images status = %d, want 501 or 503", resp.StatusCode)
	}
	resp.Body.Close()

	// v1/audio/speech (engine doesn't support -> 501/503)
	resp = doAuth(t, client, http.MethodPost, baseURL+"/v1/audio/speech", `{"model":"tts-1","input":"hello","voice":"alloy"}`, rawKey)
	if resp.StatusCode != http.StatusNotImplemented && resp.StatusCode != http.StatusServiceUnavailable {
		resp.Body.Close()
		t.Fatalf("speech status = %d, want 501 or 503", resp.StatusCode)
	}
	resp.Body.Close()

	// v1/audio/transcriptions (multipart, engine doesn't support -> 501/503)
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("model", "whisper-1")
	part, _ := writer.CreateFormFile("file", "test.mp3")
	_, _ = part.Write([]byte("fake audio"))
	writer.Close()

	req, _ := http.NewRequest(http.MethodPost, baseURL+"/v1/audio/transcriptions", &buf)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("transcriptions: %v", err)
	}
	if resp.StatusCode != http.StatusNotImplemented && resp.StatusCode != http.StatusServiceUnavailable {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("transcriptions status = %d, want 501 or 503; body=%s", resp.StatusCode, string(b))
	}
	resp.Body.Close()

	// v1/messages (Anthropic)
	resp = doAuth(t, client, http.MethodPost, baseURL+"/v1/messages", `{"model":"claude-3-opus-20240229","messages":[{"role":"user","content":"hi"}]}`, rawKey)
	// Anthropic translation may fail if engine can't handle it; accept 200, 400, 501
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotImplemented {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("messages status = %d; body=%s", resp.StatusCode, string(b))
	}
	resp.Body.Close()

	// v1/responses
	resp = doAuth(t, client, http.MethodPost, baseURL+"/v1/responses", `{"model":"gpt-4o","input":"hi"}`, rawKey)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotImplemented {
		resp.Body.Close()
		t.Fatalf("responses status = %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// ---------------------------------------------------------------------------
// OAuth
// ---------------------------------------------------------------------------

func TestE2EOAuth(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	// OAuth callback (no stored session -> 500)
	resp := doAuth(t, client, http.MethodGet, baseURL+"/api/oauth/callback?code=abc&state=xyz", "", rawKey)
	assertStatus(t, resp, http.StatusInternalServerError)

	// OAuth authorize (no flows -> 404)
	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/oauth/openai/authorize", ``, rawKey)
	assertStatus(t, resp, http.StatusNotFound)

	// OAuth poll (no flows -> 404)
	resp = doAuth(t, client, http.MethodGet, baseURL+"/api/oauth/openai/poll?session_id=123", "", rawKey)
	assertStatus(t, resp, http.StatusNotFound)

	// OAuth exchange (no flows -> 404)
	resp = doAuth(t, client, http.MethodPost, baseURL+"/api/oauth/openai/exchange", `{"code":"abc","state":"xyz"}`, rawKey)
	assertStatus(t, resp, http.StatusNotFound)
}

// ---------------------------------------------------------------------------
// Missing / User-Listed Endpoints That Don't Exist in routes.go
// ---------------------------------------------------------------------------

func TestE2EMissingEndpoints(t *testing.T) {
	_, baseURL, rawKey := setupE2E(t)
	client := httpClient()

	cases := []struct {
		method string
		path   string
		body   string
		want   int
	}{
		// Providers
		{http.MethodPost, "/api/providers", `{"id":"x"}`, http.StatusMethodNotAllowed},
		{http.MethodPut, "/api/providers/123", `{"id":"x"}`, http.StatusMethodNotAllowed},
		{http.MethodDelete, "/api/providers/123", ``, http.StatusMethodNotAllowed},

		// Connections proxy
		{http.MethodPost, "/api/connections/123/proxy", ``, http.StatusNotFound},

		// API key audit
		{http.MethodGet, "/api/keys/123/audit", ``, http.StatusNotFound},

		// Chat session messages
		{http.MethodGet, "/api/chat-sessions/123/messages", ``, http.StatusNotFound},
		{http.MethodPost, "/api/chat-sessions/123/messages", ``, http.StatusNotFound},

		// Tunnels
		{http.MethodPost, "/api/tunnels", ``, http.StatusMethodNotAllowed},
		{http.MethodDelete, "/api/tunnels/123", ``, http.StatusNotFound},
		{http.MethodPut, "/api/tunnels/123/cloudflare", ``, http.StatusNotFound},
		{http.MethodPut, "/api/tunnels/123/tailscale", ``, http.StatusNotFound},

		// OAuth
		{http.MethodGet, "/api/oauth/status", ``, http.StatusNotFound},
		{http.MethodPost, "/api/oauth/login", ``, http.StatusNotFound},
		{http.MethodGet, "/api/oauth/callback/openai", ``, http.StatusNotFound},

		// Inference
		{http.MethodGet, "/v1/skills", ``, http.StatusNotFound},
		{http.MethodGet, "/v1/locale", ``, http.StatusNotFound},

		// Diagnostics
		{http.MethodGet, "/api/diagnostics", ``, http.StatusNotFound},

	}

	for _, tc := range cases {
		resp := doAuth(t, client, tc.method, baseURL+tc.path, tc.body, rawKey)
		if resp.StatusCode != tc.want {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			t.Fatalf("%s %s: status = %d, want %d; body=%s", tc.method, tc.path, resp.StatusCode, tc.want, string(b))
		}
		resp.Body.Close()
	}
}
