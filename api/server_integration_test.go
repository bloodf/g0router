package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/api/handlers"
	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/providers/openai"
	"github.com/bloodf/g0router/internal/proxy"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
)

func TestIntegrationAuthenticatedAPIServerWithFakeUpstream(t *testing.T) {
	const (
		apiKeySecret      = "test-secret"
		upstreamModelID   = "gpt-4o"
		upstreamReplyText = "integration reply"
	)
	providerSecret := integrationSecret(t)

	upstream := newIntegrationFakeOpenAI(t, providerSecret, upstreamModelID, upstreamReplyText)
	defer upstream.Close()

	s := newAPITestStore(t)
	_, rawAPIKey, err := s.CreateAPIKey("integration", apiKeySecret)
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "fake-openai",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   stringPtr(providerSecret),
		IsActive: true,
		ProviderSpecificData: map[string]any{
			"region":        "local",
			"Authorization": "Bearer " + providerSecret,
		},
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.RequireAPIKey = true
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	engine := proxy.NewEngine(s)
	engine.Register(openai.New(upstream.URL))

	mcpClient := &routeMCPClient{tools: []mcp.Tool{
		{Name: "search", Description: "Search local docs", InputSchema: json.RawMessage(`{"type":"object"}`)},
	}}
	clientManager := mcp.NewClientManager(routeMCPConnector{client: mcpClient})
	toolManager := mcp.NewToolManager()

	_, baseURL := startTestServer(t, ServerConfig{
		Port:             0,
		Version:          "integration-test",
		Store:            s,
		UsageStore:       s,
		RequireAPIKey:    true,
		APIKeySecret:     apiKeySecret,
		APIKeyValidator:  integrationStoreAPIKeyValidator{s: s},
		InferenceEngine:  engine,
		ModelSource:      engine,
		MCPClientManager: clientManager,
		MCPToolManager:   toolManager,
	})

	assertAuthenticatedGETStatus(t, baseURL, rawAPIKey, "/api/providers", http.StatusOK)
	assertAuthenticatedGETStatus(t, baseURL, rawAPIKey, "/api/settings", http.StatusOK)
	assertAuthenticatedConnectionsRedactSecrets(t, baseURL, rawAPIKey)

	createMCPClient(t, baseURL, rawAPIKey)
	assertAuthenticatedGETStatus(t, baseURL, rawAPIKey, "/api/mcp/tools", http.StatusOK)

	assertAuthenticatedModels(t, baseURL, rawAPIKey, upstreamModelID)
	assertAuthenticatedChatCompletion(t, baseURL, rawAPIKey, upstreamModelID, upstreamReplyText)
	assertAuthenticatedMessages(t, baseURL, rawAPIKey, upstreamModelID, upstreamReplyText)
	assertAuthenticatedResponses(t, baseURL, rawAPIKey, upstreamModelID, upstreamReplyText)

	if len(upstream.requests) != 4 {
		t.Fatalf("upstream requests = %d, want 4", len(upstream.requests))
	}
	for _, req := range upstream.requests {
		if !req.authorizationOK {
			t.Fatalf("upstream request to %s did not receive accepted provider auth", req.path)
		}
	}
}

// TestIntegrationRequestContextDoesNotLeakPooledFasthttpContext is a regression
// guard for the data race where requestContext returned the pooled
// *fasthttp.RequestCtx directly. That pooled ctx flowed into the downstream
// net/http client; net/http's Transport wrapped it in a cancel context and, from
// a background readLoop goroutine, walked ctx.Value()/UserValue() during
// cancellation -- racing with fasthttp recycling (Server.releaseCtx -> reset)
// the same RequestCtx after the handler returned.
//
// Driving many concurrent chat completions through the real server against a real
// net/http upstream forces Transport cancellation to overlap with ctx recycling.
// Run under `go test -race`, this fails on the buggy code and is clean once
// requestContext detaches downstream work from the pooled RequestCtx.
func TestIntegrationRequestContextDoesNotLeakPooledFasthttpContext(t *testing.T) {
	const (
		apiKeySecret      = "test-secret"
		upstreamModelID   = "gpt-4o"
		upstreamReplyText = "race regression reply"
	)
	providerSecret := integrationSecret(t)

	upstream := newIntegrationFakeOpenAI(t, providerSecret, upstreamModelID, upstreamReplyText)
	defer upstream.Close()

	s := newAPITestStore(t)
	_, rawAPIKey, err := s.CreateAPIKey("integration", apiKeySecret)
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "fake-openai",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   stringPtr(providerSecret),
		IsActive: true,
		ProviderSpecificData: map[string]any{
			"region":        "local",
			"Authorization": "Bearer " + providerSecret,
		},
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.RequireAPIKey = true
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	engine := proxy.NewEngine(s)
	engine.Register(openai.New(upstream.URL))

	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "integration-test",
		Store:           s,
		UsageStore:      s,
		RequireAPIKey:   true,
		APIKeySecret:    apiKeySecret,
		APIKeyValidator: integrationStoreAPIKeyValidator{s: s},
		InferenceEngine: engine,
		ModelSource:     engine,
	})

	// Sequential round-trips: each request drives a net/http upstream round-trip
	// through the engine, so the Transport readLoop's cancellation walk overlaps
	// with fasthttp recycling the just-returned RequestCtx. Several iterations make
	// the use-after-recycle window reliably observable under -race.
	for i := 0; i < 40; i++ {
		assertAuthenticatedChatCompletion(t, baseURL, rawAPIKey, upstreamModelID, upstreamReplyText)
	}
}

func TestIntegrationManagementMutationsRoundTripThroughAuthenticatedServer(t *testing.T) {
	const (
		apiKeySecret = "test-secret"
		adminKeyName = "admin"
	)

	s := newAPITestStore(t)
	_, rawAPIKey, err := s.CreateAPIKey(adminKeyName, apiKeySecret)
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "integration-test",
		Store:           s,
		UsageStore:      s,
		RequireAPIKey:   true,
		APIKeySecret:    apiKeySecret,
		APIKeyValidator: integrationStoreAPIKeyValidator{s: s},
	})

	assertAPIKeyManagementRoundTrip(t, baseURL, rawAPIKey)
	assertAliasManagementRoundTrip(t, baseURL, rawAPIKey)
	assertComboManagementRoundTrip(t, baseURL, rawAPIKey)
	assertConnectionManagementRoundTrip(t, baseURL, rawAPIKey, s)
	assertPricingManagementRoundTrip(t, baseURL, rawAPIKey)
	assertSettingsManagementRoundTrip(t, baseURL, rawAPIKey)
}

func TestIntegrationMCPInstanceOAuthRoundTripThroughAuthenticatedServer(t *testing.T) {
	const apiKeySecret = "test-secret"

	tokenServer := newIntegrationFakeMCPOAuthTokenServer(t)
	defer tokenServer.Close()

	s := newAPITestStore(t)
	_, rawAPIKey, err := s.CreateAPIKey("admin", apiKeySecret)
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	runtime := &integrationMCPInstanceRuntime{manifest: mcp.Manifest{
		Tools: []mcp.Tool{{Name: "search", Description: "Search docs", InputSchema: json.RawMessage(`{"type":"object"}`)}},
	}}

	_, baseURL := startTestServer(t, ServerConfig{
		Port:               0,
		Version:            "integration-test",
		Store:              s,
		RequireAPIKey:      true,
		APIKeySecret:       apiKeySecret,
		APIKeyValidator:    integrationStoreAPIKeyValidator{s: s},
		MCPInstanceRuntime: runtime,
	})

	var created store.MCPInstance
	createdBody := doAuthenticatedJSON(t, http.MethodPost, baseURL+"/api/mcp/instances", rawAPIKey, `{"name":"atlassian-a","server_key":"atlassian","launch_type":"http","transport":"streamable-http","url":"https://mcp.atlassian.com/mcp","headers":{"Authorization":"Bearer secret-header"},"env":{"API_TOKEN":"secret-env"},"account_label":"work","is_active":true}`, http.StatusCreated, &created)
	if created.ID == "" || created.Name != "atlassian-a" || created.ToolManifest == nil || len(created.ToolManifest.Tools) != 1 {
		t.Fatalf("created mcp instance = %+v, want registered instance with manifest", created)
	}
	if len(runtime.registered) != 1 || runtime.registered[0] != created.ID {
		t.Fatalf("registered mcp instances = %+v, want %s", runtime.registered, created.ID)
	}
	if strings.Contains(string(createdBody), "secret-header") || strings.Contains(string(createdBody), "secret-env") {
		t.Fatalf("mcp instance create response leaked secret: %s", createdBody)
	}

	listBody := assertAuthenticatedGETStatus(t, baseURL, rawAPIKey, "/api/mcp/instances", http.StatusOK)
	if strings.Contains(string(listBody), "secret-header") || strings.Contains(string(listBody), "secret-env") {
		t.Fatalf("mcp instance list leaked secret: %s", listBody)
	}

	var started struct {
		AuthorizationURL string `json:"authorization_url"`
		ExpiresAt        string `json:"expires_at"`
	}
	doAuthenticatedJSON(t, http.MethodPost, baseURL+"/api/mcp/instances/"+created.ID+"/auth/start", rawAPIKey, `{"authorization_url":"`+tokenServer.URL+`/authorize","resource_uri":"https://mcp.atlassian.com","redirect_uri":"http://localhost:3000/api/mcp/oauth/callback"}`, http.StatusCreated, &started)
	authURL, err := url.Parse(started.AuthorizationURL)
	if err != nil {
		t.Fatalf("parse mcp authorization URL: %v", err)
	}
	state := authURL.Query().Get("state")
	if state == "" || authURL.Query().Get("code_challenge_method") != "S256" || authURL.Query().Get("code_challenge") == "" {
		t.Fatalf("authorization query = %s, want state and S256 PKCE", authURL.RawQuery)
	}

	var completed struct {
		InstanceID   string `json:"instance_id"`
		AccountLabel string `json:"account_label"`
	}
	callbackURL := "http://localhost:3000/api/mcp/oauth/callback?code=oauth-code&state=" + url.QueryEscape(state)
	doAuthenticatedJSON(t, http.MethodPost, baseURL+"/api/mcp/instances/"+created.ID+"/oauth/complete", rawAPIKey, `{"callback_url":"`+callbackURL+`"}`, http.StatusOK, &completed)
	if completed.InstanceID != created.ID || completed.AccountLabel != "work" {
		t.Fatalf("mcp oauth completion = %+v, want work account for %s", completed, created.ID)
	}
	if len(runtime.reapplied) != 1 || runtime.reapplied[0] != created.ID {
		t.Fatalf("reapplied mcp instances = %+v, want %s", runtime.reapplied, created.ID)
	}
	if tokenServer.codeVerifier == "" || tokenServer.codeVerifier == state {
		t.Fatalf("token exchange verifier = %q state = %q, want stored verifier separate from state", tokenServer.codeVerifier, state)
	}

	accountsBody := assertAuthenticatedGETStatus(t, baseURL, rawAPIKey, "/api/mcp/instances/"+created.ID+"/accounts", http.StatusOK)
	if strings.Contains(string(accountsBody), "access-token") || strings.Contains(string(accountsBody), "refresh-token") {
		t.Fatalf("mcp account response leaked tokens: %s", accountsBody)
	}
	var accounts struct {
		Data []struct {
			InstanceID   string `json:"instance_id"`
			AccountLabel string `json:"account_label"`
			Email        string `json:"email"`
			ResourceURI  string `json:"resource_uri"`
		} `json:"data"`
	}
	decodeIntegrationJSON(t, accountsBody, &accounts)
	if len(accounts.Data) != 1 || accounts.Data[0].InstanceID != created.ID || accounts.Data[0].AccountLabel != "work" || accounts.Data[0].Email != "team@example.test" || accounts.Data[0].ResourceURI != "https://mcp.atlassian.com" {
		t.Fatalf("mcp accounts = %+v, want redacted work account", accounts.Data)
	}

	doAuthenticatedJSON(t, http.MethodDelete, baseURL+"/api/mcp/instances/"+created.ID, rawAPIKey, "", http.StatusNoContent, nil)
	if len(runtime.closed) != 1 || runtime.closed[0] != created.ID {
		t.Fatalf("closed mcp instances = %+v, want %s", runtime.closed, created.ID)
	}
}

func TestIntegrationUsageQuotaLogsAndProviderOAuthThroughAuthenticatedServer(t *testing.T) {
	const apiKeySecret = "test-secret"

	s := newAPITestStore(t)
	_, rawAPIKey, err := s.CreateAPIKey("admin", apiKeySecret)
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "quota-openai",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   stringPtr("quota-secret"),
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	inputTokens := 11
	outputTokens := 7
	totalTokens := 18
	costUSD := 0.00012
	statusOK := http.StatusOK
	if err := s.LogRequest(&store.RequestLogEntry{
		RequestID:    "req-integration-usage",
		Timestamp:    time.Date(2026, 6, 4, 20, 0, 0, 0, time.UTC),
		Provider:     "openai",
		Model:        "gpt-4o",
		AuthType:     "api_key",
		InputTokens:  &inputTokens,
		OutputTokens: &outputTokens,
		TotalTokens:  &totalTokens,
		CostUSD:      &costUSD,
		StatusCode:   &statusOK,
	}); err != nil {
		t.Fatalf("LogRequest: %v", err)
	}

	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "integration-test",
		Store:           s,
		UsageStore:      s,
		RequireAPIKey:   true,
		APIKeySecret:    apiKeySecret,
		APIKeyValidator: integrationStoreAPIKeyValidator{s: s},
		OAuthFlows:      handlers.OAuthFlows{"minimax": routeOAuthFlow{}},
		QuotaFetchers:   map[providers.ModelProvider]usage.QuotaFetcher{providers.ProviderOpenAI: routeQuotaFetcher{}},
	})

	assertAuthenticatedUsageAndLogs(t, baseURL, rawAPIKey)
	assertAuthenticatedQuota(t, baseURL, rawAPIKey)
	assertAuthenticatedProviderOAuthRoutes(t, baseURL, rawAPIKey, s)
}

type integrationStoreAPIKeyValidator struct {
	s *store.Store
}

func (v integrationStoreAPIKeyValidator) ValidateAPIKey(key, secret string) (bool, error) {
	_, ok, err := v.s.ValidateAPIKey(key, secret)
	return ok, err
}

func (v integrationStoreAPIKeyValidator) ValidateAPIKeyIdentity(key, secret string) (*APIKeyIdentity, bool, error) {
	storedKey, ok, err := v.s.ValidateAPIKey(key, secret)
	if err != nil || !ok {
		return nil, ok, err
	}
	return &APIKeyIdentity{ID: storedKey.ID}, true, nil
}

type integrationUpstreamRequest struct {
	path            string
	authorizationOK bool
}

type integrationFakeOpenAI struct {
	*httptest.Server
	requests []integrationUpstreamRequest
}

type integrationFakeMCPOAuthTokenServer struct {
	*httptest.Server
	codeVerifier string
}

type integrationMCPInstanceRuntime struct {
	manifest   mcp.Manifest
	registered []string
	reapplied  []string
	closed     []string
}

func (r *integrationMCPInstanceRuntime) RegisterInstance(ctx context.Context, instance *store.MCPInstance) (mcp.Manifest, error) {
	r.registered = append(r.registered, instance.ID)
	manifest := r.manifest
	manifest.ClientID = instance.ID
	return manifest, nil
}

func (r *integrationMCPInstanceRuntime) CloseInstance(instanceID string) error {
	r.closed = append(r.closed, instanceID)
	return nil
}

func (r *integrationMCPInstanceRuntime) ReapplyInstanceCredentials(ctx context.Context, s handlers.MCPRuntimeCredentialStore, instanceID string) (mcp.Manifest, error) {
	r.reapplied = append(r.reapplied, instanceID)
	manifest := r.manifest
	manifest.ClientID = instanceID
	return manifest, nil
}

func newIntegrationFakeOpenAI(t *testing.T, expectedKey, modelID, replyText string) *integrationFakeOpenAI {
	t.Helper()

	fake := &integrationFakeOpenAI{}
	fake.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorizationOK := r.Header.Get("Authorization") == "Bearer "+expectedKey
		fake.requests = append(fake.requests, integrationUpstreamRequest{
			path:            r.URL.Path,
			authorizationOK: authorizationOK,
		})
		if !authorizationOK {
			http.Error(w, `{"error":{"message":"unauthorized"}}`, http.StatusUnauthorized)
			return
		}

		switch r.URL.Path {
		case "/v1/models":
			writeIntegrationJSON(t, w, map[string]any{
				"object": "list",
				"data": []map[string]any{{
					"id":       modelID,
					"object":   "model",
					"created":  1710000000,
					"owned_by": "openai",
				}},
			})
		case "/v1/chat/completions":
			var req struct {
				Model    string `json:"model"`
				Messages []struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"messages"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode upstream chat request: %v", err)
				http.Error(w, `{"error":{"message":"bad json"}}`, http.StatusBadRequest)
				return
			}
			if req.Model != modelID {
				t.Errorf("upstream model = %q, want %q", req.Model, modelID)
			}
			writeIntegrationJSON(t, w, map[string]any{
				"id":      "chatcmpl-integration",
				"object":  "chat.completion",
				"created": 1710000001,
				"model":   modelID,
				"choices": []map[string]any{{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": replyText,
					},
					"finish_reason": "stop",
				}},
				"usage": map[string]any{
					"prompt_tokens":     7,
					"completion_tokens": 3,
					"total_tokens":      10,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	return fake
}

func newIntegrationFakeMCPOAuthTokenServer(t *testing.T) *integrationFakeMCPOAuthTokenServer {
	t.Helper()

	fake := &integrationFakeMCPOAuthTokenServer{}
	fake.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse token form: %v", err)
			http.Error(w, `{"error":"bad form"}`, http.StatusBadRequest)
			return
		}
		if r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code") != "oauth-code" || r.Form.Get("redirect_uri") == "" || r.Form.Get("resource") != "https://mcp.atlassian.com" {
			t.Errorf("token form = %s, want authorization-code exchange", r.Form.Encode())
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
			return
		}
		fake.codeVerifier = r.Form.Get("code_verifier")
		if fake.codeVerifier == "" {
			t.Error("token exchange missing PKCE verifier")
			http.Error(w, `{"error":"missing verifier"}`, http.StatusBadRequest)
			return
		}
		writeIntegrationJSON(t, w, map[string]any{
			"access_token":  "access-token",
			"refresh_token": "refresh-token",
			"expires_in":    3600,
			"scope":         "read write",
			"account_label": "team-a",
			"email":         "team@example.test",
			"issuer":        "https://auth.example.test",
		})
	}))
	return fake
}

func writeIntegrationJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode fake upstream response: %v", err)
	}
}

func assertAuthenticatedGETStatus(t *testing.T, baseURL, rawAPIKey, path string, want int) []byte {
	t.Helper()

	req := newAuthenticatedRequest(t, http.MethodGet, baseURL+path, rawAPIKey, nil)
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read %s response: %v", path, err)
	}
	if resp.StatusCode != want {
		t.Fatalf("GET %s status = %d, want %d; body=%s", path, resp.StatusCode, want, body)
	}
	return body
}

func assertAuthenticatedConnectionsRedactSecrets(t *testing.T, baseURL, rawAPIKey string) {
	t.Helper()

	body := assertAuthenticatedGETStatus(t, baseURL, rawAPIKey, "/api/connections", http.StatusOK)
	var decoded struct {
		Data []struct {
			Provider             string         `json:"provider"`
			APIKey               *string        `json:"api_key"`
			ProviderSpecificData map[string]any `json:"provider_specific_data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode connections: %v; body=%s", err, body)
	}
	if len(decoded.Data) != 1 || decoded.Data[0].Provider != "openai" {
		t.Fatalf("connections = %+v, want one openai connection", decoded.Data)
	}
	if decoded.Data[0].APIKey != nil {
		t.Fatal("connection response exposed api key field")
	}
	if _, ok := decoded.Data[0].ProviderSpecificData["Authorization"]; ok {
		t.Fatal("connection response exposed authorization provider data")
	}
	if decoded.Data[0].ProviderSpecificData["region"] != "local" {
		t.Fatalf("redacted provider data = %+v, want non-secret region", decoded.Data[0].ProviderSpecificData)
	}
}

func assertAPIKeyManagementRoundTrip(t *testing.T, baseURL, rawAPIKey string) {
	t.Helper()

	var created apiKeyViewJSON
	createdBody := doAuthenticatedJSON(t, http.MethodPost, baseURL+"/api/keys", rawAPIKey, `{"name":"dashboard"}`, http.StatusCreated, &created)
	if created.ID == "" || created.Name != "dashboard" {
		t.Fatalf("created api key = %+v, want dashboard key with id", created)
	}
	if !strings.HasPrefix(created.FullKey, "g0r_") || created.Prefix != created.FullKey[:8] {
		t.Fatalf("created api key full_key/prefix mismatch: full_key=%q key=%+v", created.FullKey, created)
	}
	if strings.Count(string(createdBody), created.FullKey) != 1 {
		t.Fatalf("created api key response should expose full_key once; body=%s", createdBody)
	}

	listBody := assertAuthenticatedGETStatus(t, baseURL, rawAPIKey, "/api/keys", http.StatusOK)
	if strings.Contains(string(listBody), created.FullKey) {
		t.Fatalf("api key list exposed full_key; body=%s", listBody)
	}
	var listed struct {
		Data []apiKeyViewJSON `json:"data"`
	}
	decodeIntegrationJSON(t, listBody, &listed)
	if !containsAPIKeyView(listed.Data, created.ID, "dashboard", created.Prefix, true) {
		t.Fatalf("api key list = %+v, want active dashboard key", listed.Data)
	}

	// Round-trip a per-key policy via the update endpoint.
	var updated apiKeyViewJSON
	doAuthenticatedJSON(t, http.MethodPut, baseURL+"/api/keys/"+created.ID, rawAPIKey,
		`{"scopes":["gpt-*"],"rpm_limit":60,"daily_spend_cap":2.5}`, http.StatusOK, &updated)
	if len(updated.Scopes) != 1 || updated.Scopes[0] != "gpt-*" {
		t.Fatalf("updated key scopes = %+v", updated)
	}
	if updated.RateLimitRPM == nil || *updated.RateLimitRPM != 60 {
		t.Fatalf("updated key rpm = %+v", updated)
	}

	doAuthenticatedJSON(t, http.MethodDelete, baseURL+"/api/keys/"+created.ID, rawAPIKey, "", http.StatusNoContent, nil)

	listBody = assertAuthenticatedGETStatus(t, baseURL, rawAPIKey, "/api/keys", http.StatusOK)
	decodeIntegrationJSON(t, listBody, &listed)
	if !containsAPIKeyView(listed.Data, created.ID, "dashboard", created.Prefix, false) {
		t.Fatalf("api key list after delete = %+v, want inactive dashboard key", listed.Data)
	}
}

type apiKeyViewJSON struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Prefix       string   `json:"prefix"`
	FullKey      string   `json:"full_key"`
	IsActive     bool     `json:"is_active"`
	Scopes       []string `json:"scopes"`
	RateLimitRPM *int     `json:"rpm_limit"`
}

func containsAPIKeyView(keys []apiKeyViewJSON, id, name, prefix string, active bool) bool {
	for _, key := range keys {
		if key.ID == id && key.Name == name && key.Prefix == prefix && key.IsActive == active {
			return true
		}
	}
	return false
}

func assertAliasManagementRoundTrip(t *testing.T, baseURL, rawAPIKey string) {
	t.Helper()

	var created store.ModelAlias
	doAuthenticatedJSON(t, http.MethodPost, baseURL+"/api/aliases", rawAPIKey, `{"alias":"fast","provider":"openai","model":"gpt-4o-mini"}`, http.StatusCreated, &created)
	if created.Alias != "fast" || created.Provider != "openai" || created.Model != "gpt-4o-mini" {
		t.Fatalf("created alias = %+v", created)
	}

	var updated store.ModelAlias
	doAuthenticatedJSON(t, http.MethodPut, baseURL+"/api/aliases/fast", rawAPIKey, `{"provider":"anthropic","model":"claude-sonnet-4-20250514"}`, http.StatusOK, &updated)
	if updated.Alias != "fast" || updated.Provider != "anthropic" || updated.Model != "claude-sonnet-4-20250514" {
		t.Fatalf("updated alias = %+v", updated)
	}

	var listed struct {
		Data []store.ModelAlias `json:"data"`
	}
	body := assertAuthenticatedGETStatus(t, baseURL, rawAPIKey, "/api/aliases", http.StatusOK)
	decodeIntegrationJSON(t, body, &listed)
	if !containsAlias(listed.Data, updated) {
		t.Fatalf("alias list = %+v, want %+v", listed.Data, updated)
	}

	doAuthenticatedJSON(t, http.MethodDelete, baseURL+"/api/aliases/fast", rawAPIKey, "", http.StatusNoContent, nil)
	body = assertAuthenticatedGETStatus(t, baseURL, rawAPIKey, "/api/aliases", http.StatusOK)
	decodeIntegrationJSON(t, body, &listed)
	if containsAlias(listed.Data, updated) {
		t.Fatalf("alias list after delete = %+v, still contains %+v", listed.Data, updated)
	}
}

func assertComboManagementRoundTrip(t *testing.T, baseURL, rawAPIKey string) {
	t.Helper()

	var created store.Combo
	doAuthenticatedJSON(t, http.MethodPost, baseURL+"/api/combos", rawAPIKey, `{"name":"primary","is_active":true,"steps":[{"provider":"openai","model":"gpt-4o-mini"}]}`, http.StatusCreated, &created)
	if created.ID == "" || created.Name != "primary" || !created.IsActive || !sameComboSteps(created.Steps, []store.ComboStep{{Provider: "openai", Model: "gpt-4o-mini"}}) {
		t.Fatalf("created combo = %+v", created)
	}

	var updated store.Combo
	doAuthenticatedJSON(t, http.MethodPut, baseURL+"/api/combos/"+created.ID, rawAPIKey, `{"name":"primary","is_active":true,"steps":[{"provider":"anthropic","model":"claude-sonnet-4-20250514"}]}`, http.StatusOK, &updated)
	if updated.ID != created.ID || !sameComboSteps(updated.Steps, []store.ComboStep{{Provider: "anthropic", Model: "claude-sonnet-4-20250514"}}) {
		t.Fatalf("updated combo = %+v", updated)
	}

	var listed struct {
		Data []store.Combo `json:"data"`
	}
	body := assertAuthenticatedGETStatus(t, baseURL, rawAPIKey, "/api/combos", http.StatusOK)
	decodeIntegrationJSON(t, body, &listed)
	if !containsCombo(listed.Data, updated.ID, "primary") {
		t.Fatalf("combo list = %+v, want %s", listed.Data, updated.ID)
	}

	doAuthenticatedJSON(t, http.MethodDelete, baseURL+"/api/combos/"+created.ID, rawAPIKey, "", http.StatusNoContent, nil)
	body = assertAuthenticatedGETStatus(t, baseURL, rawAPIKey, "/api/combos", http.StatusOK)
	decodeIntegrationJSON(t, body, &listed)
	if containsCombo(listed.Data, updated.ID, "primary") {
		t.Fatalf("combo list after delete = %+v, still contains %s", listed.Data, updated.ID)
	}
}

func assertConnectionManagementRoundTrip(t *testing.T, baseURL, rawAPIKey string, s *store.Store) {
	t.Helper()

	createBody := `{"provider":"codex","name":"work","auth_type":"oauth","access_token":"access-secret","refresh_token":"refresh-secret","api_key":"api-secret","is_active":true,"provider_specific_data":{"region":"local","Authorization":"Bearer nested-secret","headers":{"X-API-Key":"nested-key","safe":"visible"}},"account_id":"acct-1","email":"work@example.test"}`
	var created struct {
		ID                   string         `json:"id"`
		Provider             string         `json:"provider"`
		Name                 string         `json:"name"`
		AuthType             string         `json:"auth_type"`
		ProviderSpecificData map[string]any `json:"provider_specific_data"`
		Email                *string        `json:"email"`
	}
	body := doAuthenticatedJSON(t, http.MethodPost, baseURL+"/api/connections", rawAPIKey, createBody, http.StatusCreated, &created)
	assertConnectionResponseRedacted(t, body, created.ProviderSpecificData, "access-secret", "refresh-secret", "api-secret", "nested-secret", "nested-key")
	if created.ID == "" || created.Provider != "openai" || created.Name != "work" || created.AuthType != "oauth" || created.Email == nil || *created.Email != "work@example.test" {
		t.Fatalf("created connection = %+v, want canonical openai oauth work connection", created)
	}
	if created.ProviderSpecificData["region"] != "local" {
		t.Fatalf("created provider data = %+v, want non-secret region retained", created.ProviderSpecificData)
	}
	headers, ok := created.ProviderSpecificData["headers"].(map[string]any)
	if !ok || headers["safe"] != "visible" || headers["X-API-Key"] != nil {
		t.Fatalf("created nested provider data = %+v, want redacted key and visible safe value", created.ProviderSpecificData)
	}

	stored, err := s.GetConnection(created.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if stored.AccessToken == nil || *stored.AccessToken != "access-secret" || stored.RefreshToken == nil || *stored.RefreshToken != "refresh-secret" || stored.APIKey == nil || *stored.APIKey != "api-secret" {
		t.Fatalf("stored connection secrets = access:%v refresh:%v api:%v, want original secrets persisted", stored.AccessToken, stored.RefreshToken, stored.APIKey)
	}
	if stored.ProviderSpecificData["Authorization"] != "Bearer nested-secret" {
		t.Fatalf("stored provider Authorization = %v, want original nested secret", stored.ProviderSpecificData["Authorization"])
	}
	storedHeaders, ok := stored.ProviderSpecificData["headers"].(map[string]any)
	if !ok || storedHeaders["X-API-Key"] != "nested-key" || storedHeaders["safe"] != "visible" {
		t.Fatalf("stored provider headers = %+v, want original nested key and safe value", stored.ProviderSpecificData["headers"])
	}

	tested := struct {
		OK       bool   `json:"ok"`
		Provider string `json:"provider"`
		Name     string `json:"name"`
	}{}
	doAuthenticatedJSON(t, http.MethodPost, baseURL+"/api/connections/"+created.ID+"/test", rawAPIKey, "", http.StatusOK, &tested)
	if !tested.OK || tested.Provider != "openai" || tested.Name != "work" {
		t.Fatalf("connection test response = %+v, want active canonical openai work connection", tested)
	}

	listBody := assertAuthenticatedGETStatus(t, baseURL, rawAPIKey, "/api/connections", http.StatusOK)
	assertConnectionResponseRedacted(t, listBody, nil, "access-secret", "refresh-secret", "api-secret", "nested-secret", "nested-key")
	var listed struct {
		Data []struct {
			ID       string `json:"id"`
			Provider string `json:"provider"`
			Name     string `json:"name"`
		} `json:"data"`
	}
	decodeIntegrationJSON(t, listBody, &listed)
	if !containsConnection(listed.Data, created.ID, "openai", "work") {
		t.Fatalf("connection list = %+v, want created connection", listed.Data)
	}

	updateBody := `{"provider":"anthropic","name":"work-updated","auth_type":"api_key","api_key":"updated-secret","is_active":false,"provider_specific_data":{"mode":"updated","token":"updated-nested-secret"}}`
	var updated struct {
		ID                   string         `json:"id"`
		Provider             string         `json:"provider"`
		Name                 string         `json:"name"`
		AuthType             string         `json:"auth_type"`
		IsActive             bool           `json:"is_active"`
		ProviderSpecificData map[string]any `json:"provider_specific_data"`
	}
	body = doAuthenticatedJSON(t, http.MethodPut, baseURL+"/api/connections/"+created.ID, rawAPIKey, updateBody, http.StatusOK, &updated)
	assertConnectionResponseRedacted(t, body, updated.ProviderSpecificData, "updated-secret", "updated-nested-secret")
	if updated.ID != created.ID || updated.Provider != "anthropic" || updated.Name != "work-updated" || updated.AuthType != "api_key" || updated.IsActive {
		t.Fatalf("updated connection = %+v, want inactive anthropic api-key connection", updated)
	}
	if updated.ProviderSpecificData["mode"] != "updated" || updated.ProviderSpecificData["token"] != nil {
		t.Fatalf("updated provider data = %+v, want redacted token and visible mode", updated.ProviderSpecificData)
	}
	stored, err = s.GetConnection(created.ID)
	if err != nil {
		t.Fatalf("GetConnection after update: %v", err)
	}
	if stored.Provider != "anthropic" || stored.Name != "work-updated" || stored.AuthType != store.AuthTypeAPIKey || stored.IsActive {
		t.Fatalf("stored updated connection = %+v, want inactive anthropic api-key connection", stored)
	}
	if stored.APIKey == nil || *stored.APIKey != "updated-secret" {
		t.Fatalf("stored updated API key = %v, want updated secret", stored.APIKey)
	}
	if stored.ProviderSpecificData["mode"] != "updated" || stored.ProviderSpecificData["token"] != "updated-nested-secret" {
		t.Fatalf("stored updated provider data = %+v, want original updated token and visible mode", stored.ProviderSpecificData)
	}

	doAuthenticatedJSON(t, http.MethodDelete, baseURL+"/api/connections/"+created.ID, rawAPIKey, "", http.StatusNoContent, nil)
	body = assertAuthenticatedGETStatus(t, baseURL, rawAPIKey, "/api/connections", http.StatusOK)
	decodeIntegrationJSON(t, body, &listed)
	if containsConnection(listed.Data, created.ID, "anthropic", "work-updated") {
		t.Fatalf("connection list after delete = %+v, still contains deleted connection", listed.Data)
	}
}

type integrationPricingOverride struct {
	ID         string  `json:"id"`
	Provider   string  `json:"provider"`
	Model      string  `json:"model"`
	InputCost  float64 `json:"input_cost"`
	OutputCost float64 `json:"output_cost"`
}

func assertPricingManagementRoundTrip(t *testing.T, baseURL, rawAPIKey string) {
	t.Helper()

	var created integrationPricingOverride
	doAuthenticatedJSON(t, http.MethodPost, baseURL+"/api/pricing", rawAPIKey, `{"provider":"openai","model":"unit-test-model","input_cost_per_token":0.000001,"output_cost_per_token":0.000002}`, http.StatusCreated, &created)
	if created.Provider != "openai" || created.Model != "unit-test-model" || created.InputCost != 1 || created.OutputCost != 2 {
		t.Fatalf("created pricing override = %+v", created)
	}

	var updated integrationPricingOverride
	doAuthenticatedJSON(t, http.MethodPut, baseURL+"/api/pricing/openai/unit-test-model", rawAPIKey, `{"input_cost_per_token":0.000003,"output_cost_per_token":0.000004}`, http.StatusOK, &updated)
	if updated.Provider != "openai" || updated.Model != "unit-test-model" || updated.InputCost != 3 || updated.OutputCost != 4 {
		t.Fatalf("updated pricing override = %+v", updated)
	}

	var listed struct {
		Data []integrationPricingOverride `json:"data"`
	}
	body := assertAuthenticatedGETStatus(t, baseURL, rawAPIKey, "/api/pricing", http.StatusOK)
	decodeIntegrationJSON(t, body, &listed)
	if !containsPricingOverride(listed.Data, updated) {
		t.Fatalf("pricing override list = %+v, want %+v", listed.Data, updated)
	}

	doAuthenticatedJSON(t, http.MethodDelete, baseURL+"/api/pricing/openai/unit-test-model", rawAPIKey, "", http.StatusNoContent, nil)
	body = assertAuthenticatedGETStatus(t, baseURL, rawAPIKey, "/api/pricing", http.StatusOK)
	decodeIntegrationJSON(t, body, &listed)
	if containsPricingOverride(listed.Data, updated) {
		t.Fatalf("pricing override list after delete = %+v, still contains %+v", listed.Data, updated)
	}
}

func assertConnectionResponseRedacted(t *testing.T, body []byte, providerData map[string]any, secrets ...string) {
	t.Helper()

	bodyText := string(body)
	for _, secret := range secrets {
		if strings.Contains(bodyText, secret) {
			t.Fatalf("connection response leaked secret %q: %s", secret, bodyText)
		}
	}
	if providerData == nil {
		return
	}
	for _, secretKey := range []string{"Authorization", "api_key", "X-API-Key", "token"} {
		if _, ok := providerData[secretKey]; ok {
			t.Fatalf("provider data exposed secret key %q: %+v", secretKey, providerData)
		}
	}
}

func assertSettingsManagementRoundTrip(t *testing.T, baseURL, rawAPIKey string) {
	t.Helper()

	want := store.Settings{
		RequireAPIKey:     true,
		RTKEnabled:        false,
		CavemanEnabled:    true,
		CavemanLevel:      "minimal",
		EnableRequestLogs: true,
		ProxyURL:          "http://127.0.0.1:9000",
		DataDir:           "/tmp/g0router-integration",
		AllowedSources:    []string{"local", "lan"},
	}
	var updated store.Settings
	doAuthenticatedJSON(t, http.MethodPut, baseURL+"/api/settings", rawAPIKey, `{"require_api_key":true,"rtk_enabled":false,"caveman_enabled":true,"caveman_level":"minimal","enable_request_logs":true,"proxy_url":"http://127.0.0.1:9000","data_dir":"/tmp/g0router-integration","allowed_sources":["local","lan"]}`, http.StatusOK, &updated)
	if !reflect.DeepEqual(updated, want) {
		t.Fatalf("updated settings = %+v, want %+v", updated, want)
	}

	body := assertAuthenticatedGETStatus(t, baseURL, rawAPIKey, "/api/settings", http.StatusOK)
	var got store.Settings
	decodeIntegrationJSON(t, body, &got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("settings round trip = %+v, want %+v", got, want)
	}
}

func assertAuthenticatedUsageAndLogs(t *testing.T, baseURL, rawAPIKey string) {
	t.Helper()

	for _, path := range []string{"/api/usage?provider=openai&limit=5", "/api/logs?model=gpt-4o&limit=5"} {
		body := assertAuthenticatedGETStatus(t, baseURL, rawAPIKey, path, http.StatusOK)
		var listed struct {
			Object string `json:"object"`
			Data   []struct {
				RequestID   string   `json:"request_id"`
				Provider    string   `json:"provider"`
				Model       string   `json:"model"`
				TotalTokens *int     `json:"total_tokens"`
				CostUSD     *float64 `json:"cost_usd"`
				StatusCode  *int     `json:"status_code"`
			} `json:"data"`
			Limit int `json:"limit"`
		}
		decodeIntegrationJSON(t, body, &listed)
		if listed.Object != "list" || listed.Limit != 5 || len(listed.Data) != 1 {
			t.Fatalf("%s response = %+v, want one usage/log row", path, listed)
		}
		row := listed.Data[0]
		if row.RequestID != "req-integration-usage" || row.Provider != "openai" || row.Model != "gpt-4o" || row.TotalTokens == nil || *row.TotalTokens != 18 || row.CostUSD == nil || *row.CostUSD != 0.00012 || row.StatusCode == nil || *row.StatusCode != http.StatusOK {
			t.Fatalf("%s row = %+v, want seeded usage/log row", path, row)
		}
	}

	body := assertAuthenticatedGETStatus(t, baseURL, rawAPIKey, "/api/usage/summary?provider=openai", http.StatusOK)
	var summary struct {
		RequestCount int64   `json:"request_count"`
		TotalTokens  int64   `json:"total_tokens"`
		TotalCost    float64 `json:"total_cost"`
	}
	decodeIntegrationJSON(t, body, &summary)
	if summary.RequestCount != 1 || summary.TotalTokens != 18 || summary.TotalCost != 0.00012 {
		t.Fatalf("usage summary = %+v, want seeded aggregate", summary)
	}
}

func assertAuthenticatedQuota(t *testing.T, baseURL, rawAPIKey string) {
	t.Helper()

	body := assertAuthenticatedGETStatus(t, baseURL, rawAPIKey, "/api/usage/quota/openai", http.StatusOK)
	var quota usage.Quota
	decodeIntegrationJSON(t, body, &quota)
	if quota.Provider != providers.ProviderOpenAI || quota.Limit != 100 || quota.Used != 1 || quota.Remaining != 99 {
		t.Fatalf("quota = %+v, want fake openai quota", quota)
	}
}

func assertAuthenticatedProviderOAuthRoutes(t *testing.T, baseURL, rawAPIKey string, s *store.Store) {
	t.Helper()

	var started struct {
		Provider  string `json:"provider"`
		SessionID string `json:"session_id"`
	}
	doAuthenticatedJSON(t, http.MethodPost, baseURL+"/api/oauth/minimax/authorize", rawAPIKey, `{"account_label":"integration-account"}`, http.StatusOK, &started)
	if started.Provider != "minimax" || started.SessionID != "session-1" {
		t.Fatalf("oauth start = %+v, want minimax session-1", started)
	}

	var poll struct {
		Status string `json:"status"`
	}
	doAuthenticatedJSON(t, http.MethodGet, baseURL+"/api/oauth/minimax/poll?session_id=session-1", rawAPIKey, "", http.StatusOK, &poll)
	if poll.Status != "pending" {
		t.Fatalf("oauth poll = %+v, want pending", poll)
	}

	callbackBody := doAuthenticatedJSON(t, http.MethodGet, baseURL+"/api/oauth/callback?state=session-1&code=callback-code", rawAPIKey, "", http.StatusOK, nil)
	assertOAuthConnectionResponseRedacted(t, callbackBody, "token", "callback-code")

	doAuthenticatedJSON(t, http.MethodPost, baseURL+"/api/oauth/minimax/authorize", rawAPIKey, `{"account_label":"integration-exchange"}`, http.StatusOK, &started)
	exchangeBody := doAuthenticatedJSON(t, http.MethodPost, baseURL+"/api/oauth/minimax/exchange", rawAPIKey, `{"state":"`+started.SessionID+`","code":"manual-code"}`, http.StatusOK, nil)
	assertOAuthConnectionResponseRedacted(t, exchangeBody, "token", "manual-code")

	connections, err := s.GetConnections("minimax")
	if err != nil {
		t.Fatalf("GetConnections minimax: %v", err)
	}
	if len(connections) != 2 {
		t.Fatalf("minimax connections = %d, want callback and exchange connections", len(connections))
	}
	for _, connection := range connections {
		if connection.AuthType != store.AuthTypeOAuth || connection.AccessToken == nil || *connection.AccessToken != "token" {
			t.Fatalf("stored oauth connection = %+v, want persisted token", connection)
		}
	}
}

func assertOAuthConnectionResponseRedacted(t *testing.T, body []byte, secrets ...string) {
	t.Helper()

	bodyText := string(body)
	for _, secret := range secrets {
		if strings.Contains(bodyText, secret) {
			t.Fatalf("oauth response leaked %q: %s", secret, bodyText)
		}
	}
	var decoded struct {
		ID       string `json:"id"`
		Provider string `json:"provider"`
		AuthType string `json:"auth_type"`
	}
	decodeIntegrationJSON(t, body, &decoded)
	if decoded.ID == "" || decoded.Provider != "minimax" || decoded.AuthType != string(store.AuthTypeOAuth) {
		t.Fatalf("oauth connection response = %+v, want redacted minimax oauth connection", decoded)
	}
}

func createMCPClient(t *testing.T, baseURL, rawAPIKey string) {
	t.Helper()

	body := strings.NewReader(`{"name":"docs","transport":"stdio","command":"mcp-docs","is_active":true}`)
	req := newAuthenticatedRequest(t, http.MethodPost, baseURL+"/api/mcp/clients", rawAPIKey, body)
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("POST /api/mcp/clients: %v", err)
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read /api/mcp/clients response: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/mcp/clients status = %d, want 201; body=%s", resp.StatusCode, data)
	}
}

func assertAuthenticatedModels(t *testing.T, baseURL, rawAPIKey, modelID string) {
	t.Helper()

	body := assertAuthenticatedGETStatus(t, baseURL, rawAPIKey, "/v1/models", http.StatusOK)
	var decoded struct {
		Object string            `json:"object"`
		Data   []providers.Model `json:"data"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode models: %v; body=%s", err, body)
	}
	if decoded.Object != "list" || len(decoded.Data) != 1 || decoded.Data[0].ID != modelID {
		t.Fatalf("models response = %+v, want one %s model", decoded, modelID)
	}
}

func assertAuthenticatedChatCompletion(t *testing.T, baseURL, rawAPIKey, modelID, replyText string) {
	t.Helper()

	body := strings.NewReader(`{"model":"` + modelID + `","messages":[{"role":"user","content":"hello"}]}`)
	req := newAuthenticatedRequest(t, http.MethodPost, baseURL+"/v1/chat/completions", rawAPIKey, body)
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("POST /v1/chat/completions: %v", err)
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read chat response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /v1/chat/completions status = %d, want 200; body=%s", resp.StatusCode, data)
	}
	var decoded providers.ChatResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode chat response: %v; body=%s", err, data)
	}
	if decoded.Model != modelID || len(decoded.Choices) != 1 || decoded.Choices[0].Message.Content != replyText {
		t.Fatalf("chat response = %+v, want %s reply for %s", decoded, replyText, modelID)
	}
}

func assertAuthenticatedMessages(t *testing.T, baseURL, rawAPIKey, modelID, replyText string) {
	t.Helper()

	body := strings.NewReader(`{"model":"` + modelID + `","messages":[{"role":"user","content":"hello"}]}`)
	req := newAuthenticatedRequest(t, http.MethodPost, baseURL+"/v1/messages", rawAPIKey, body)
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("POST /v1/messages: %v", err)
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read messages response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /v1/messages status = %d, want 200; body=%s", resp.StatusCode, data)
	}
	var decoded struct {
		Type    string `json:"type"`
		Role    string `json:"role"`
		Model   string `json:"model"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode messages response: %v; body=%s", err, data)
	}
	if decoded.Type != "message" || decoded.Role != "assistant" || decoded.Model != modelID || len(decoded.Content) != 1 || decoded.Content[0].Text != replyText {
		t.Fatalf("messages response = %+v, want %s reply for %s", decoded, replyText, modelID)
	}
	if decoded.Usage.InputTokens != 7 || decoded.Usage.OutputTokens != 3 {
		t.Fatalf("messages usage = %+v, want upstream usage mapped", decoded.Usage)
	}
}

func assertAuthenticatedResponses(t *testing.T, baseURL, rawAPIKey, modelID, replyText string) {
	t.Helper()

	body := strings.NewReader(`{"model":"` + modelID + `","input":[{"role":"user","content":[{"type":"input_text","text":"hello"}]}]}`)
	req := newAuthenticatedRequest(t, http.MethodPost, baseURL+"/v1/responses", rawAPIKey, body)
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("POST /v1/responses: %v", err)
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read responses response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /v1/responses status = %d, want 200; body=%s", resp.StatusCode, data)
	}
	var decoded struct {
		Object string `json:"object"`
		Status string `json:"status"`
		Model  string `json:"model"`
		Output []struct {
			Type    string `json:"type"`
			Role    string `json:"role"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode responses response: %v; body=%s", err, data)
	}
	if decoded.Object != "response" || decoded.Status != "completed" || decoded.Model != modelID || len(decoded.Output) != 1 || len(decoded.Output[0].Content) != 1 || decoded.Output[0].Content[0].Text != replyText {
		t.Fatalf("responses response = %+v, want %s reply for %s", decoded, replyText, modelID)
	}
	if decoded.Usage.InputTokens != 7 || decoded.Usage.OutputTokens != 3 || decoded.Usage.TotalTokens != 10 {
		t.Fatalf("responses usage = %+v, want upstream usage mapped", decoded.Usage)
	}
}

func newAuthenticatedRequest(t *testing.T, method, url, rawAPIKey string, body io.Reader) *http.Request {
	t.Helper()

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("new %s request %s: %v", method, url, err)
	}
	req.Header.Set("Authorization", "Bearer "+rawAPIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func doAuthenticatedJSON(t *testing.T, method, url, rawAPIKey, body string, want int, decodeInto any) []byte {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := newAuthenticatedRequest(t, method, url, rawAPIKey, reader)
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read %s %s response: %v", method, url, err)
	}
	if resp.StatusCode != want {
		t.Fatalf("%s %s status = %d, want %d; body=%s", method, url, resp.StatusCode, want, data)
	}
	if decodeInto != nil {
		decodeIntegrationJSON(t, data, decodeInto)
	}
	return data
}

func decodeIntegrationJSON(t *testing.T, data []byte, into any) {
	t.Helper()

	if err := json.Unmarshal(data, into); err != nil {
		t.Fatalf("decode integration JSON: %v; body=%s", err, data)
	}
}

func containsAlias(aliases []store.ModelAlias, want store.ModelAlias) bool {
	for _, alias := range aliases {
		if alias == want {
			return true
		}
	}
	return false
}

func containsCombo(combos []store.Combo, id, name string) bool {
	for _, combo := range combos {
		if combo.ID == id && combo.Name == name {
			return true
		}
	}
	return false
}

func containsConnection(connections []struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Name     string `json:"name"`
}, id, provider, name string) bool {
	for _, connection := range connections {
		if connection.ID == id && connection.Provider == provider && connection.Name == name {
			return true
		}
	}
	return false
}

func containsPricingOverride(overrides []integrationPricingOverride, want integrationPricingOverride) bool {
	for _, override := range overrides {
		if override == want {
			return true
		}
	}
	return false
}

func sameComboSteps(got, want []store.ComboStep) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func stringPtr(value string) *string {
	return &value
}

func integrationSecret(t *testing.T) string {
	t.Helper()

	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("generate integration secret: %v", err)
	}
	return "test-provider-" + hex.EncodeToString(buf)
}
