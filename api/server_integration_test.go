package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/providers/openai"
	"github.com/bloodf/g0router/internal/proxy"
	"github.com/bloodf/g0router/internal/store"
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

	if len(upstream.requests) != 2 {
		t.Fatalf("upstream requests = %d, want 2", len(upstream.requests))
	}
	for _, req := range upstream.requests {
		if !req.authorizationOK {
			t.Fatalf("upstream request to %s did not receive accepted provider auth", req.path)
		}
	}
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
			Provider             string         `json:"Provider"`
			APIKey               *string        `json:"APIKey"`
			ProviderSpecificData map[string]any `json:"ProviderSpecificData"`
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
