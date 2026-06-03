package api

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/api/handlers"
	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
)

func TestHealthz(t *testing.T) {
	srv := NewServer(ServerConfig{Port: 0, Version: "test-version"})
	ln := srv.listener()
	if ln == nil {
		t.Fatal("listener failed")
	}

	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Stop() })

	resp, err := httpClient().Get("http://" + localhostAddr(t, ln) + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("status: %q", result["status"])
	}
	if result["version"] != "test-version" {
		t.Errorf("version: %q", result["version"])
	}
}

func TestUnknownRoute(t *testing.T) {
	srv := NewServer(ServerConfig{Port: 0, Version: "test"})
	ln := srv.listener()
	if ln == nil {
		t.Fatal("listener failed")
	}

	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Stop() })

	resp, err := httpClient().Get("http://" + localhostAddr(t, ln) + "/api/nope")
	if err != nil {
		t.Fatalf("GET /api/nope: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestServerServesEmbeddedUI(t *testing.T) {
	_, baseURL := startTestServer(t, ServerConfig{Port: 0, Version: "test"})

	resp, err := httpClient().Get(baseURL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("GET / content-type = %q, want text/html", got)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read / body: %v", err)
	}
	content := string(body)
	if !strings.Contains(content, `<div id="root"></div>`) {
		t.Fatalf("GET / did not serve UI index: %q", content)
	}

	assetPath := firstUIAssetPath(t, content)
	assetResp, err := httpClient().Get(baseURL + assetPath)
	if err != nil {
		t.Fatalf("GET %s: %v", assetPath, err)
	}
	defer assetResp.Body.Close()

	if assetResp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200", assetPath, assetResp.StatusCode)
	}
}

func TestServerServesUIIndexForClientRoutes(t *testing.T) {
	_, baseURL := startTestServer(t, ServerConfig{Port: 0, Version: "test"})

	resp, err := httpClient().Get(baseURL + "/settings")
	if err != nil {
		t.Fatalf("GET /settings: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /settings status = %d, want 200", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read /settings body: %v", err)
	}
	if !strings.Contains(string(body), `<div id="root"></div>`) {
		t.Fatalf("GET /settings did not serve UI index: %q", string(body))
	}
}

func TestServerDoesNotServeUIForAPIRoutes(t *testing.T) {
	_, baseURL := startTestServer(t, ServerConfig{Port: 0, Version: "test"})

	tests := []string{"/api/missing", "/v1/missing"}
	for _, path := range tests {
		resp, err := httpClient().Get(baseURL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("GET %s status = %d, want 404", path, resp.StatusCode)
		}
	}
}

func TestManagementRoutesDispatchThroughServer(t *testing.T) {
	store := newAPITestStore(t)
	_, baseURL := startTestServer(t, ServerConfig{
		Port:          0,
		Version:       "test",
		Store:         store,
		APIKeySecret:  "test-secret",
		ModelSource:   routeModelSource{},
		OAuthFlows:    handlers.OAuthFlows{"minimax": routeOAuthFlow{}},
		UsageStore:    store,
		QuotaFetchers: map[providers.ModelProvider]usage.QuotaFetcher{providers.ProviderOpenAI: routeQuotaFetcher{}},
		QuotaKey:      providers.Key{Value: "sk-test", AuthType: "api_key"},
	})

	tests := []struct {
		method string
		path   string
		want   int
	}{
		{method: http.MethodGet, path: "/api/providers", want: http.StatusOK},
		{method: http.MethodGet, path: "/api/providers/openai/models", want: http.StatusOK},
		{method: http.MethodGet, path: "/api/connections", want: http.StatusOK},
		{method: http.MethodGet, path: "/api/settings", want: http.StatusOK},
		{method: http.MethodGet, path: "/api/keys", want: http.StatusOK},
		{method: http.MethodGet, path: "/api/combos", want: http.StatusOK},
		{method: http.MethodPost, path: "/api/oauth/minimax/authorize", want: http.StatusOK},
		{method: http.MethodGet, path: "/api/usage", want: http.StatusOK},
		{method: http.MethodGet, path: "/api/usage/summary", want: http.StatusOK},
		{method: http.MethodGet, path: "/api/usage/quota/openai", want: http.StatusOK},
		{method: http.MethodGet, path: "/api/logs", want: http.StatusOK},
	}

	for _, tc := range tests {
		req, err := http.NewRequest(tc.method, baseURL+tc.path, nil)
		if err != nil {
			t.Fatalf("new request %s %s: %v", tc.method, tc.path, err)
		}
		resp, err := httpClient().Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.method, tc.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != tc.want {
			t.Fatalf("%s %s status = %d, want %d", tc.method, tc.path, resp.StatusCode, tc.want)
		}
	}
}

func TestOAuthRoutesEnforceDocumentedMethods(t *testing.T) {
	_, baseURL := startTestServer(t, ServerConfig{
		Port:       0,
		Version:    "test",
		OAuthFlows: handlers.OAuthFlows{"minimax": routeOAuthFlow{}},
	})

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/oauth/minimax/authorize"},
		{method: http.MethodPost, path: "/api/oauth/minimax/poll?session_id=session-1"},
		{method: http.MethodPost, path: "/api/oauth/callback?provider=minimax&session_id=session-1&code=ok"},
	}

	for _, tc := range tests {
		req, err := http.NewRequest(tc.method, baseURL+tc.path, nil)
		if err != nil {
			t.Fatalf("new request %s %s: %v", tc.method, tc.path, err)
		}
		resp, err := httpClient().Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.method, tc.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("%s %s status = %d, want 405", tc.method, tc.path, resp.StatusCode)
		}
	}
}

func TestMCPRoutesDispatchThroughServer(t *testing.T) {
	store := newAPITestStore(t)
	mcpClient := &routeMCPClient{tools: []mcp.Tool{
		{Name: "search", Description: "Search docs", InputSchema: json.RawMessage(`{"type":"object"}`)},
	}}
	clientManager := mcp.NewClientManager(routeMCPConnector{client: mcpClient})
	toolManager := mcp.NewToolManager()
	_, baseURL := startTestServer(t, ServerConfig{
		Port:             0,
		Version:          "test",
		Store:            store,
		MCPClientManager: clientManager,
		MCPToolManager:   toolManager,
	})

	createReq, err := http.NewRequest(http.MethodPost, baseURL+"/api/mcp/clients", strings.NewReader(`{"name":"docs","transport":"stdio","command":"mcp-docs","is_active":true}`))
	if err != nil {
		t.Fatalf("new create request: %v", err)
	}
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := httpClient().Do(createReq)
	if err != nil {
		t.Fatalf("POST /api/mcp/clients: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/mcp/clients status = %d, want 201", createResp.StatusCode)
	}

	toolsResp, err := httpClient().Get(baseURL + "/api/mcp/tools")
	if err != nil {
		t.Fatalf("GET /api/mcp/tools: %v", err)
	}
	defer toolsResp.Body.Close()
	if toolsResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/mcp/tools status = %d, want 200", toolsResp.StatusCode)
	}
	var toolsList struct {
		Data []providers.Tool `json:"data"`
	}
	if err := json.NewDecoder(toolsResp.Body).Decode(&toolsList); err != nil {
		t.Fatalf("decode tools: %v", err)
	}
	if len(toolsList.Data) != 1 {
		t.Fatalf("tools len = %d, want 1", len(toolsList.Data))
	}

	toolName := toolsList.Data[0].Function.Name
	executeReq, err := http.NewRequest(http.MethodPost, baseURL+"/api/mcp/tools/"+toolName+"/execute", strings.NewReader(`{"arguments":{"query":"mcp"}}`))
	if err != nil {
		t.Fatalf("new execute request: %v", err)
	}
	executeReq.Header.Set("Content-Type", "application/json")
	executeResp, err := httpClient().Do(executeReq)
	if err != nil {
		t.Fatalf("POST /api/mcp/tools/%s/execute: %v", toolName, err)
	}
	defer executeResp.Body.Close()
	if executeResp.StatusCode != http.StatusOK {
		t.Fatalf("execute status = %d, want 200", executeResp.StatusCode)
	}
	if len(mcpClient.calls) != 1 || string(mcpClient.calls[0].Arguments) != `{"query":"mcp"}` {
		t.Fatalf("calls = %+v, want query args", mcpClient.calls)
	}
}

func TestInferenceLoggingSkipsWhenRequestLogsDisabled(t *testing.T) {
	s := newAPITestStore(t)
	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		InferenceEngine: routeInferenceEngine{response: routeChatResponseWithUsage()},
	})

	resp, body := postAPITestJSON(t, baseURL+"/v1/chat/completions", `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	entries, err := s.GetUsage(store.UsageFilter{})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("usage entries = %+v, want none when request logs disabled", entries)
	}
}

func TestInferenceLoggingRecordsUsageAndCostWhenEnabled(t *testing.T) {
	s := newAPITestStore(t)
	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.EnableRequestLogs = true
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		InferenceEngine: routeInferenceEngine{response: routeChatResponseWithUsage()},
	})

	resp, body := postAPITestJSON(t, baseURL+"/v1/chat/completions", `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	entries, err := s.GetUsage(store.UsageFilter{})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("usage entries = %d, want 1: %+v", len(entries), entries)
	}
	entry := entries[0]
	if entry.RequestID == "" {
		t.Fatal("request ID should be logged")
	}
	if entry.Provider != "openai" || entry.Model != "gpt-4o" {
		t.Fatalf("provider/model = %s/%s, want openai/gpt-4o", entry.Provider, entry.Model)
	}
	if entry.AuthType != "noauth" {
		t.Fatalf("auth type = %q, want noauth", entry.AuthType)
	}
	if entry.InputTokens == nil || *entry.InputTokens != 1000 {
		t.Fatalf("input tokens = %v, want 1000", entry.InputTokens)
	}
	if entry.OutputTokens == nil || *entry.OutputTokens != 500 {
		t.Fatalf("output tokens = %v, want 500", entry.OutputTokens)
	}
	if entry.TotalTokens == nil || *entry.TotalTokens != 1500 {
		t.Fatalf("total tokens = %v, want 1500", entry.TotalTokens)
	}
	if entry.CacheReadTokens == nil || *entry.CacheReadTokens != 200 {
		t.Fatalf("cache read tokens = %v, want 200", entry.CacheReadTokens)
	}
	if entry.CostUSD == nil || math.Abs(*entry.CostUSD-0.00725) > 0.000000001 {
		t.Fatalf("cost USD = %v, want 0.00725", entry.CostUSD)
	}
	if entry.StatusCode == nil || *entry.StatusCode != http.StatusOK {
		t.Fatalf("status code = %v, want 200", entry.StatusCode)
	}
	if entry.LatencyMS == nil {
		t.Fatal("latency should be logged")
	}
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 2 * time.Second}
}

func localhostAddr(t *testing.T, ln net.Listener) string {
	t.Helper()

	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener addr is %T, want *net.TCPAddr", ln.Addr())
	}
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(tcpAddr.Port))
}

func firstUIAssetPath(t *testing.T, content string) string {
	t.Helper()

	match := regexp.MustCompile(`/(assets/[^"]+)`).FindStringSubmatch(content)
	if len(match) != 2 {
		t.Fatalf("index.html does not reference an asset: %q", content)
	}
	return "/" + match[1]
}

func newAPITestStore(t *testing.T) *store.Store {
	t.Helper()

	s, err := store.NewStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	return s
}

type routeModelSource struct{}

func (routeModelSource) ListModels(ctx context.Context) ([]providers.Model, error) {
	return []providers.Model{
		{ID: "gpt-4o", Object: "model", Provider: providers.ProviderOpenAI},
	}, nil
}

type routeOAuthFlow struct{}

func (routeOAuthFlow) ProviderID() oauth.ProviderID {
	return "minimax"
}

func (routeOAuthFlow) Start(ctx context.Context) (oauth.AuthSession, error) {
	return oauth.AuthSession{Provider: "minimax", SessionID: "session-1"}, nil
}

func (routeOAuthFlow) Exchange(ctx context.Context, session oauth.AuthSession, code string) (oauth.TokenResult, error) {
	return oauth.TokenResult{Provider: "minimax", AccessToken: "token"}, nil
}

func (routeOAuthFlow) Poll(ctx context.Context, session oauth.AuthSession) (oauth.PollResult, error) {
	return oauth.PollResult{Status: oauth.PollStatusPending}, nil
}

type routeQuotaFetcher struct{}

func (routeQuotaFetcher) FetchQuota(ctx context.Context, key providers.Key) (usage.Quota, error) {
	return usage.Quota{Provider: key.Provider, Limit: 100, Used: 1, Remaining: 99}, nil
}

type routeInferenceEngine struct {
	response *providers.ChatResponse
}

func (e routeInferenceEngine) Dispatch(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	return e.response, nil
}

func (e routeInferenceEngine) DispatchStream(ctx context.Context, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return nil, nil
}

func (e routeInferenceEngine) ListModels(ctx context.Context) ([]providers.Model, error) {
	return nil, nil
}

func routeChatResponseWithUsage() *providers.ChatResponse {
	return &providers.ChatResponse{
		ID:      "chatcmpl-usage",
		Object:  "chat.completion",
		Created: 1710000000,
		Model:   "gpt-4o",
		Choices: []providers.Choice{
			{Index: 0, Message: providers.Message{Role: "assistant", Content: "hello back"}},
		},
		Usage: &providers.Usage{
			PromptTokens:     1000,
			CompletionTokens: 500,
			TotalTokens:      1500,
			PromptTokensDetails: &providers.PromptTokensDetails{
				CachedTokens: 200,
			},
		},
	}
}

func postAPITestJSON(t *testing.T, url string, body string) (*http.Response, []byte) {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		t.Fatalf("read response: %v", err)
	}
	return resp, data
}

type routeMCPConnector struct {
	client *routeMCPClient
}

func (c routeMCPConnector) Connect(ctx context.Context, cfg mcp.ClientConfig) (mcp.Client, error) {
	return c.client, nil
}

type routeMCPClient struct {
	tools []mcp.Tool
	calls []mcp.CallRequest
}

func (c *routeMCPClient) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	return c.tools, nil
}

func (c *routeMCPClient) CallTool(ctx context.Context, req mcp.CallRequest) (mcp.CallResult, error) {
	c.calls = append(c.calls, req)
	return mcp.CallResult{Content: map[string]bool{"ok": true}}, nil
}

func (c *routeMCPClient) Close() error {
	return nil
}
