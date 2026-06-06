package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/bloodf/g0router/internal/proxy"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
)

func TestHealthz(t *testing.T) {
	srv := NewServer(ServerConfig{Port: 0, Version: "test-version"})
	ln := apiTestListener(t)

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
	ln := apiTestListener(t)

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
		req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
		if err != nil {
			t.Fatalf("NewRequest %s: %v", path, err)
		}
		// /v1/* always requires a key; send the harness key so routing (not auth)
		// determines the status.
		req.Header.Set("X-API-Key", testHarnessAPIKey)
		resp, err := httpClient().Do(req)
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
		{method: http.MethodPost, path: "/api/connections/missing/test", want: http.StatusNotFound},
		{method: http.MethodGet, path: "/api/settings", want: http.StatusOK},
		{method: http.MethodGet, path: "/api/keys", want: http.StatusOK},
		{method: http.MethodGet, path: "/api/combos", want: http.StatusOK},
		{method: http.MethodGet, path: "/api/aliases", want: http.StatusOK},
		{method: http.MethodGet, path: "/api/pricing", want: http.StatusOK},
		{method: http.MethodPost, path: "/api/oauth/minimax/authorize", want: http.StatusOK},
		{method: http.MethodGet, path: "/api/usage", want: http.StatusOK},
		{method: http.MethodGet, path: "/api/usage/summary", want: http.StatusOK},
		{method: http.MethodGet, path: "/api/usage/chart?period=today", want: http.StatusOK},
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
	enableRequestLogs(t, s)

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
	if entry.AuthType != "api_key" {
		t.Fatalf("auth type = %q, want api_key (proxy always authenticates)", entry.AuthType)
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

func TestInferenceLoggingUsesPricingOverrideForCost(t *testing.T) {
	s := newAPITestStore(t)
	enableRequestLogs(t, s)
	if err := s.SetPricingOverride(store.PricingOverride{
		Provider:           "openai",
		Model:              "gpt-4o",
		InputCostPerToken:  0.00001,
		OutputCostPerToken: 0.00002,
	}); err != nil {
		t.Fatalf("SetPricingOverride: %v", err)
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
	if entries[0].CostUSD == nil || math.Abs(*entries[0].CostUSD-0.02) > 0.000000001 {
		t.Fatalf("cost USD = %v, want 0.02", entries[0].CostUSD)
	}
}

func TestInferenceLoggingUsesPublicCatalogModelForProviderQualifiedRoute(t *testing.T) {
	s := newAPITestStore(t)
	enableRequestLogs(t, s)
	response := routeChatResponseWithModel("gemini-2.5-flash", 1000, 500)
	response.Provider = providers.ProviderVertex

	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		InferenceEngine: routeInferenceEngine{response: response},
	})

	resp, body := postAPITestJSON(t, baseURL+"/v1/chat/completions", `{"model":"vertex/gemini-2.5-flash","messages":[{"role":"user","content":"hello"}]}`)
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
	if entries[0].Provider != "vertex" || entries[0].Model != "vertex/gemini-2.5-flash" {
		t.Fatalf("provider/model = %s/%s, want vertex/vertex/gemini-2.5-flash", entries[0].Provider, entries[0].Model)
	}
	if entries[0].CostUSD == nil || math.Abs(*entries[0].CostUSD-0.001496) > 0.000000001 {
		t.Fatalf("cost USD = %v, want 0.001496", entries[0].CostUSD)
	}
}

func TestInferenceLoggingUsesConfigEnableRequestLogs(t *testing.T) {
	s := newAPITestStore(t)
	config := ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		InferenceEngine: routeInferenceEngine{response: routeChatResponseWithUsage()},
	}
	config.EnableRequestLogs = true

	_, baseURL := startTestServer(t, config)

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
		t.Fatalf("usage entries = %d, want 1 when config enables request logs", len(entries))
	}
}

func TestInferenceLoggingRecordsMessagesRouteWhenEnabled(t *testing.T) {
	s := newAPITestStore(t)
	enableRequestLogs(t, s)

	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		InferenceEngine: routeInferenceEngine{response: routeChatResponseWithModel("unknown-provider-model", 11, 7)},
	})

	resp, body := postAPITestJSON(t, baseURL+"/v1/messages", `{"model":"claude-sonnet-4","messages":[{"role":"user","content":"hello"}]}`)
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
	if entry.Provider != "unknown" || entry.Model != "unknown-provider-model" {
		t.Fatalf("provider/model = %s/%s, want unknown/unknown-provider-model", entry.Provider, entry.Model)
	}
	if entry.InputTokens == nil || *entry.InputTokens != 11 {
		t.Fatalf("input tokens = %v, want 11", entry.InputTokens)
	}
	if entry.OutputTokens == nil || *entry.OutputTokens != 7 {
		t.Fatalf("output tokens = %v, want 7", entry.OutputTokens)
	}
	if entry.TotalTokens == nil || *entry.TotalTokens != 18 {
		t.Fatalf("total tokens = %v, want 18", entry.TotalTokens)
	}
	if entry.SourceFormat == nil || *entry.SourceFormat != "anthropic" {
		t.Fatalf("source format = %v, want anthropic", entry.SourceFormat)
	}
	if entry.TargetFormat == nil || *entry.TargetFormat != "unknown" {
		t.Fatalf("target format = %v, want unknown", entry.TargetFormat)
	}
}

func TestInferenceLoggingRecordsResponsesRouteWhenEnabled(t *testing.T) {
	s := newAPITestStore(t)
	enableRequestLogs(t, s)

	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		InferenceEngine: routeInferenceEngine{response: routeChatResponseWithModel("gpt-4o", 13, 5)},
	})

	resp, body := postAPITestJSON(t, baseURL+"/v1/responses", `{"model":"gpt-4o","input":[{"role":"user","content":[{"type":"input_text","text":"hello"}]}]}`)
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
	if entry.Provider != "openai" || entry.Model != "gpt-4o" {
		t.Fatalf("provider/model = %s/%s, want openai/gpt-4o", entry.Provider, entry.Model)
	}
	if entry.InputTokens == nil || *entry.InputTokens != 13 {
		t.Fatalf("input tokens = %v, want 13", entry.InputTokens)
	}
	if entry.OutputTokens == nil || *entry.OutputTokens != 5 {
		t.Fatalf("output tokens = %v, want 5", entry.OutputTokens)
	}
	if entry.TotalTokens == nil || *entry.TotalTokens != 18 {
		t.Fatalf("total tokens = %v, want 18", entry.TotalTokens)
	}
	if entry.SourceFormat == nil || *entry.SourceFormat != "responses" {
		t.Fatalf("source format = %v, want responses", entry.SourceFormat)
	}
	if entry.TargetFormat == nil || *entry.TargetFormat != "openai" {
		t.Fatalf("target format = %v, want openai", entry.TargetFormat)
	}
}

func TestInferenceLoggingUsesResolvedProviderMetadataWhenEnabled(t *testing.T) {
	s := newAPITestStore(t)
	enableRequestLogs(t, s)
	apiKey := "groq-key"
	if err := s.CreateConnection(&store.Connection{
		Provider: "groq",
		Name:     "primary",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if err := s.SetModelAlias(store.ModelAlias{
		Alias:    "fast",
		Provider: "groq",
		Model:    "llama-3.3-70b-versatile",
	}); err != nil {
		t.Fatalf("SetModelAlias: %v", err)
	}
	engine := proxy.NewEngine(s)
	engine.Register(&routeProvider{
		name:     providers.ProviderGroq,
		response: routeChatResponseWithModel("llama-3.3-70b-versatile", 17, 9),
	})

	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		InferenceEngine: engine,
	})

	resp, body := postAPITestJSON(t, baseURL+"/v1/chat/completions", `{"model":"fast","messages":[{"role":"user","content":"hello"}]}`)
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
	if entry.Provider != "groq" || entry.Model != "llama-3.3-70b-versatile" {
		t.Fatalf("provider/model = %s/%s, want groq/llama-3.3-70b-versatile", entry.Provider, entry.Model)
	}
	if entry.ConnectionID == nil || *entry.ConnectionID == "" {
		t.Fatalf("connection ID = %v, want selected connection", entry.ConnectionID)
	}
	if entry.AuthType != string(store.AuthTypeAPIKey) {
		t.Fatalf("auth type = %q, want api_key", entry.AuthType)
	}
}

func TestInferenceLoggingRecordsFailedRequestWhenEnabled(t *testing.T) {
	s := newAPITestStore(t)
	enableRequestLogs(t, s)

	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		InferenceEngine: routeInferenceEngine{err: errors.New("chat completion: Authorization: Bearer sk-live-secret")},
	})

	resp, body := postAPITestJSON(t, baseURL+"/v1/chat/completions", `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502; body=%s", resp.StatusCode, body)
	}

	entries, err := s.GetUsage(store.UsageFilter{})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("usage entries = %d, want failed request log: %+v", len(entries), entries)
	}
	entry := entries[0]
	if entry.Provider != "openai" || entry.Model != "gpt-4o" {
		t.Fatalf("provider/model = %s/%s, want openai/gpt-4o", entry.Provider, entry.Model)
	}
	if entry.StatusCode == nil || *entry.StatusCode != http.StatusBadGateway {
		t.Fatalf("status code = %v, want 502", entry.StatusCode)
	}
	if entry.Error == nil || *entry.Error != "upstream_error: upstream provider error" {
		t.Fatalf("error = %v, want sanitized upstream error", entry.Error)
	}
	if entry.InputTokens != nil || entry.OutputTokens != nil || entry.TotalTokens != nil || entry.CostUSD != nil {
		t.Fatalf("failed request should not invent usage or cost: %+v", entry)
	}
	if strings.Contains(*entry.Error, "sk-live-secret") || strings.Contains(*entry.Error, "Authorization") {
		t.Fatalf("logged error leaked upstream secret detail: %q", *entry.Error)
	}
}

func TestInferenceLoggingRecordsStreamingUsageWhenEnabled(t *testing.T) {
	s := newAPITestStore(t)
	enableRequestLogs(t, s)

	chunks := make(chan providers.StreamChunk, 2)
	chunks <- providers.StreamChunk{
		ID:      "chatcmpl-stream",
		Object:  "chat.completion.chunk",
		Created: 1710000000,
		Model:   "gpt-4o",
		Usage:   &providers.Usage{PromptTokens: 21, CompletionTokens: 13, TotalTokens: 34},
	}
	close(chunks)
	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		InferenceEngine: routeInferenceEngine{stream: chunks},
	})

	resp, body := postAPITestJSON(t, baseURL+"/v1/chat/completions", `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}],"stream":true}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	entries, err := s.GetUsage(store.UsageFilter{})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("usage entries = %d, want 1 stream request log: %+v", len(entries), entries)
	}
	entry := entries[0]
	if entry.Provider != "openai" || entry.Model != "gpt-4o" {
		t.Fatalf("provider/model = %s/%s, want openai/gpt-4o", entry.Provider, entry.Model)
	}
	if entry.InputTokens == nil || *entry.InputTokens != 21 {
		t.Fatalf("input tokens = %v, want 21", entry.InputTokens)
	}
	if entry.OutputTokens == nil || *entry.OutputTokens != 13 {
		t.Fatalf("output tokens = %v, want 13", entry.OutputTokens)
	}
	if entry.TotalTokens == nil || *entry.TotalTokens != 34 {
		t.Fatalf("total tokens = %v, want 34", entry.TotalTokens)
	}
	if entry.StatusCode == nil || *entry.StatusCode != http.StatusOK {
		t.Fatalf("status code = %v, want 200", entry.StatusCode)
	}
}

func TestInferenceLoggingWriteFailureDoesNotFailInference(t *testing.T) {
	s := newAPITestStore(t)
	enableRequestLogs(t, s)
	failingLogStore := failingRequestLogStore{err: errors.New("database busy")}

	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      failingLogStore,
		InferenceEngine: routeInferenceEngine{response: routeChatResponseWithUsage()},
	})

	resp, body := postAPITestJSON(t, baseURL+"/v1/chat/completions", `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want inference to succeed despite log write failure; body=%s", resp.StatusCode, body)
	}
}

func TestInferenceLoggingRecordsAPIKeyIdentityWhenValidatorProvidesIt(t *testing.T) {
	s := newAPITestStore(t)
	storedKey, rawKey, err := s.CreateAPIKey("identity test", "test-secret")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	_, baseURL := startTestServer(t, ServerConfig{
		Port:              0,
		Version:           "test",
		Store:             s,
		UsageStore:        s,
		RequireAPIKey:     true,
		APIKeySecret:      "test-secret",
		EnableRequestLogs: true,
		APIKeyValidator: fakeIdentityAPIKeyValidator{
			validKeys: map[string]string{rawKey: storedKey.ID},
		},
		InferenceEngine: routeInferenceEngine{response: routeChatResponseWithUsage()},
	})

	req, err := http.NewRequest(http.MethodPost, baseURL+"/v1/chat/completions", strings.NewReader(`{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rawKey)
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("POST /v1/chat/completions: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
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
	if entry.AuthType != "api_key" {
		t.Fatalf("auth type = %q, want api_key", entry.AuthType)
	}
	if entry.APIKeyID == nil || *entry.APIKeyID != storedKey.ID {
		t.Fatalf("api key id = %v, want %s", entry.APIKeyID, storedKey.ID)
	}
	if entry.APIKeyID != nil && *entry.APIKeyID == rawKey {
		t.Fatal("api key id must not contain the raw API key")
	}
}

func TestInferenceAppliesRTKAndCavemanSettingsBeforeDispatch(t *testing.T) {
	s := newAPITestStore(t)
	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.RTKEnabled = true
	settings.CavemanEnabled = true
	settings.CavemanLevel = "lite"
	settings.EnableRequestLogs = true
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	var received []providers.ChatRequest
	engine := routeInferenceEngine{response: routeChatResponseWithUsage(), received: &received}
	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		InferenceEngine: engine,
	})

	body := `{"model":"gpt-4o","messages":[{"role":"tool","content":"On branch codex/test\nChanges not staged for commit:\n  modified:   internal/rtk/rtk.go\nUntracked files:\n  api/server_test.go\n"},{"role":"user","content":"summarize"}]}`
	resp, respBody := postAPITestJSON(t, baseURL+"/v1/chat/completions", body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, respBody)
	}
	if len(received) != 1 {
		t.Fatalf("engine requests = %d, want 1", len(received))
	}
	req := received[0]
	if len(req.Messages) != 3 {
		t.Fatalf("messages len = %d, want caveman system plus original messages: %+v", len(req.Messages), req.Messages)
	}
	if req.Messages[0].Role != "system" || !strings.Contains(fmt.Sprint(req.Messages[0].Content), "Respond tersely") {
		t.Fatalf("first message = %+v, want caveman system prompt", req.Messages[0])
	}
	toolContent, ok := req.Messages[1].Content.(string)
	if !ok {
		t.Fatalf("tool content type = %T, want string", req.Messages[1].Content)
	}
	if strings.Contains(toolContent, "Changes not staged for commit") || !strings.Contains(toolContent, "M internal/rtk/rtk.go") {
		t.Fatalf("tool content was not RTK-compressed: %q", toolContent)
	}

	entries, err := s.GetUsage(store.UsageFilter{})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("usage entries = %d, want 1", len(entries))
	}
	entry := entries[0]
	if entry.RTKEnabled == nil || !*entry.RTKEnabled {
		t.Fatalf("rtk enabled = %v, want true", entry.RTKEnabled)
	}
	if entry.CavemanEnabled == nil || !*entry.CavemanEnabled {
		t.Fatalf("caveman enabled = %v, want true", entry.CavemanEnabled)
	}
	if entry.SourceFormat == nil || *entry.SourceFormat != "openai" {
		t.Fatalf("source format = %v, want openai", entry.SourceFormat)
	}
	if entry.TargetFormat == nil || *entry.TargetFormat != "openai" {
		t.Fatalf("target format = %v, want openai", entry.TargetFormat)
	}
}

func TestInferenceAddsRegisteredMCPToolsBeforeDispatch(t *testing.T) {
	toolManager := mcp.NewToolManager()
	if err := toolManager.RegisterManifest(mcp.Manifest{
		ClientID: "docs",
		Tools: []mcp.Tool{{
			Name:        "search",
			Description: "Search docs",
			InputSchema: json.RawMessage(`{
				"type":"object",
				"properties":{"query":{"type":"string"}},
				"required":["query"]
			}`),
		}},
	}); err != nil {
		t.Fatalf("RegisterManifest: %v", err)
	}

	var received []providers.ChatRequest
	engine := routeInferenceEngine{response: routeChatResponseWithUsage(), received: &received}
	_, baseURL := startTestServer(t, ServerConfig{
		Port:             0,
		Version:          "test",
		InferenceEngine:  engine,
		MCPToolManager:   toolManager,
		MCPClientManager: mcp.NewClientManager(routeMCPConnector{client: &routeMCPClient{}}),
	})

	cases := []struct {
		path string
		body string
	}{
		{path: "/v1/chat/completions", body: `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`},
		{path: "/v1/messages", body: `{"model":"claude-sonnet-4","messages":[{"role":"user","content":"hello"}]}`},
		{path: "/v1/responses", body: `{"model":"gpt-4o","input":[{"role":"user","content":[{"type":"input_text","text":"hello"}]}]}`},
	}

	for _, tc := range cases {
		resp, body := postAPITestJSON(t, baseURL+tc.path, tc.body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s status = %d, want 200; body=%s", tc.path, resp.StatusCode, body)
		}
	}

	if len(received) != len(cases) {
		t.Fatalf("engine requests = %d, want %d", len(received), len(cases))
	}
	for i, req := range received {
		if len(req.Tools) != 1 {
			t.Fatalf("request %d tools = %+v, want one MCP tool", i, req.Tools)
		}
		if req.Tools[0].Function.Name != "docs__search" {
			t.Fatalf("request %d tool name = %q, want docs__search", i, req.Tools[0].Function.Name)
		}
	}

	resp, body := postAPITestJSON(t, baseURL+"/v1/chat/completions", `{
		"model":"gpt-4o",
		"tools":[{"type":"function","function":{"name":"caller_lookup","description":"Caller tool"}}],
		"messages":[{"role":"user","content":"hello"}]
	}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("caller tool status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	req := received[len(received)-1]
	if len(req.Tools) != 1 || req.Tools[0].Function.Name != "caller_lookup" {
		t.Fatalf("caller tools = %+v, want only caller_lookup", req.Tools)
	}
}

func TestDocumentedV1MessagesRouteDispatches(t *testing.T) {
	var received []providers.ChatRequest
	engine := routeInferenceEngine{response: routeChatResponseWithUsage(), received: &received}
	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		InferenceEngine: engine,
	})

	resp, body := postAPITestJSON(t, baseURL+"/v1/messages", `{"model":"claude-sonnet-4","system":"answer tersely","max_tokens":128,"messages":[{"role":"user","content":"hello"}]}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	if len(received) != 1 {
		t.Fatalf("engine requests = %d, want 1", len(received))
	}
	req := received[0]
	if req.Model != "claude-sonnet-4" || req.System != "answer tersely" {
		t.Fatalf("engine request model/system = %q/%#v", req.Model, req.System)
	}
	if req.MaxTokens == nil || *req.MaxTokens != 128 {
		t.Fatalf("max tokens = %+v, want 128", req.MaxTokens)
	}
	if len(req.Messages) != 1 || req.Messages[0].Role != "user" || req.Messages[0].Content != "hello" {
		t.Fatalf("messages = %+v", req.Messages)
	}

	var decoded struct {
		Type    string `json:"type"`
		Role    string `json:"role"`
		Model   string `json:"model"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal messages response: %v; body=%s", err, body)
	}
	if decoded.Type != "message" || decoded.Role != "assistant" || decoded.Model != "gpt-4o" {
		t.Fatalf("messages response metadata = %+v", decoded)
	}
	if len(decoded.Content) != 1 || decoded.Content[0].Type != "text" || decoded.Content[0].Text != "hello back" {
		t.Fatalf("messages content = %+v", decoded.Content)
	}
}

func TestDocumentedV1ResponsesRouteDispatches(t *testing.T) {
	var received []providers.ChatRequest
	engine := routeInferenceEngine{response: routeChatResponseWithUsage(), received: &received}
	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		InferenceEngine: engine,
	})

	resp, body := postAPITestJSON(t, baseURL+"/v1/responses", `{"model":"gpt-4o","instructions":"be brief","max_output_tokens":64,"input":[{"role":"user","content":[{"type":"input_text","text":"hello"}]}]}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	if len(received) != 1 {
		t.Fatalf("engine requests = %d, want 1", len(received))
	}
	req := received[0]
	if req.Model != "gpt-4o" || req.System != "be brief" {
		t.Fatalf("engine request model/system = %q/%#v", req.Model, req.System)
	}
	if req.MaxCompletionTokens == nil || *req.MaxCompletionTokens != 64 {
		t.Fatalf("max completion tokens = %+v, want 64", req.MaxCompletionTokens)
	}
	if len(req.Messages) != 1 || req.Messages[0].Role != "user" || req.Messages[0].Content != "hello" {
		t.Fatalf("messages = %+v", req.Messages)
	}

	var decoded struct {
		Object     string `json:"object"`
		Status     string `json:"status"`
		OutputText string `json:"output_text"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal responses response: %v; body=%s", err, body)
	}
	if decoded.Object != "response" || decoded.Status != "completed" || decoded.OutputText != "hello back" {
		t.Fatalf("responses body = %+v", decoded)
	}
}

func httpClient() *http.Client {
	return &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}
}

func TestHTTPClientUsesIsolatedTransport(t *testing.T) {
	first := httpClient()
	second := httpClient()

	if first.Transport == nil || second.Transport == nil {
		t.Fatal("httpClient should not use the shared default transport")
	}
	if first.Transport == second.Transport {
		t.Fatal("httpClient should isolate transports between ephemeral test servers")
	}
}

func TestAPITestListenerBindsIPv4Loopback(t *testing.T) {
	ln := apiTestListener(t)
	defer ln.Close()

	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener addr is %T, want *net.TCPAddr", ln.Addr())
	}
	if !tcpAddr.IP.Equal(net.IPv4(127, 0, 0, 1)) {
		t.Fatalf("listener address = %s, want IPv4 loopback", tcpAddr.IP)
	}
}

func localhostAddr(t *testing.T, ln net.Listener) string {
	t.Helper()

	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener addr is %T, want *net.TCPAddr", ln.Addr())
	}
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(tcpAddr.Port))
}

func apiTestListener(t *testing.T) net.Listener {
	t.Helper()

	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on IPv4 loopback: %v", err)
	}
	return ln
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
	stream   <-chan providers.StreamChunk
	err      error
	received *[]providers.ChatRequest
}

func (e routeInferenceEngine) Dispatch(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	if e.received != nil {
		*e.received = append(*e.received, *req)
	}
	if e.err != nil {
		return nil, e.err
	}
	return e.response, nil
}

func (e routeInferenceEngine) DispatchStream(ctx context.Context, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	if e.err != nil {
		return nil, e.err
	}
	return e.stream, nil
}

func (e routeInferenceEngine) ListModels(ctx context.Context) ([]providers.Model, error) {
	return nil, nil
}

type routeProvider struct {
	name     providers.ModelProvider
	response *providers.ChatResponse
}

func (p *routeProvider) Name() providers.ModelProvider {
	return p.name
}

func (p *routeProvider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	return p.response, nil
}

func (p *routeProvider) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return nil, nil
}

func (p *routeProvider) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	return nil, nil
}

func routeChatResponseWithUsage() *providers.ChatResponse {
	return routeChatResponseWithModel("gpt-4o", 1000, 500)
}

func routeChatResponseWithModel(model string, inputTokens int, outputTokens int) *providers.ChatResponse {
	return &providers.ChatResponse{
		ID:      "chatcmpl-usage",
		Object:  "chat.completion",
		Created: 1710000000,
		Model:   model,
		Choices: []providers.Choice{
			{Index: 0, Message: providers.Message{Role: "assistant", Content: "hello back"}},
		},
		Usage: &providers.Usage{
			PromptTokens:     inputTokens,
			CompletionTokens: outputTokens,
			TotalTokens:      inputTokens + outputTokens,
			PromptTokensDetails: &providers.PromptTokensDetails{
				CachedTokens: 200,
			},
		},
	}
}

func enableRequestLogs(t *testing.T, s *store.Store) {
	t.Helper()

	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.EnableRequestLogs = true
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
}

func postAPITestJSON(t *testing.T, url string, body string) (*http.Response, []byte) {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testHarnessAPIKey)
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

type failingRequestLogStore struct {
	err error
}

func (f failingRequestLogStore) GetUsage(filter store.UsageFilter) ([]store.RequestLogEntry, error) {
	return nil, nil
}

func (f failingRequestLogStore) GetUsageSummary(filter store.UsageFilter) (*store.UsageSummary, error) {
	return &store.UsageSummary{}, nil
}

func (f failingRequestLogStore) CountUsage(filter store.UsageFilter) (int, error) {
	return 0, nil
}
func (f failingRequestLogStore) GetUsageChart(period, granularity string, now time.Time) (*store.UsageChart, error) {
	return nil, f.err
}

func (f failingRequestLogStore) LogRequest(entry *store.RequestLogEntry) error {
	return f.err
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
