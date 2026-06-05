package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/api"
	"github.com/bloodf/g0router/internal/mcp"
	providermatrix "github.com/bloodf/g0router/internal/provider"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/proxy"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
)

func TestRootCommandPrintsVersion(t *testing.T) {
	cmd := NewRootCommand("0.1.0-test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if got := strings.TrimSpace(out.String()); got != "g0router 0.1.0-test" {
		t.Fatalf("output = %q, want version", got)
	}
}

func TestRootCommandIncludesExpectedSubcommands(t *testing.T) {
	cmd := NewRootCommand("0.1.0-test")
	names := commandNames(cmd.Commands())

	for _, want := range []string{"auth", "healthcheck", "install", "keys", "login", "logout", "providers", "serve", "status", "uninstall", "version"} {
		if !names[want] {
			t.Fatalf("missing subcommand %q in %v", want, names)
		}
	}
}

func TestServeCommandStartsServerRunner(t *testing.T) {
	var got serveConfig
	called := false
	t.Setenv("API_KEY_SECRET", "test-api-key-secret")
	cmd := newRootCommand(rootConfig{
		Version: "0.1.0-test",
		Serve: func(ctx context.Context, config serveConfig) error {
			called = true
			got = config
			return nil
		},
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"serve", "--port", "20128", "--data-dir", t.TempDir()})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if !called {
		t.Fatal("serve runner was not called")
	}
	if got.Port != 20128 {
		t.Fatalf("port = %d, want 20128", got.Port)
	}
	if got.BindAddress != "127.0.0.1" {
		t.Fatalf("bind address = %q, want 127.0.0.1", got.BindAddress)
	}
	if got.Version != "0.1.0-test" {
		t.Fatalf("version = %q, want 0.1.0-test", got.Version)
	}
	if !got.RequireAPIKey {
		t.Fatal("RequireAPIKey = false, want true")
	}
	if got.APIKeySecret != "test-api-key-secret" {
		t.Fatalf("APIKeySecret = %q, want test-api-key-secret", got.APIKeySecret)
	}
	if got.DataDir == "" {
		t.Fatal("data dir should be passed to serve runner")
	}
}

func TestServeCommandUsesEnvironmentDefaults(t *testing.T) {
	var got serveConfig
	dataDir := filepath.Join(t.TempDir(), "env-data")
	t.Setenv("PORT", "22345")
	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("BIND_ADDRESS", "0.0.0.0")
	t.Setenv("API_KEY_SECRET", "test-api-key-secret")
	cmd := newRootCommand(rootConfig{
		Version: "0.1.0-test",
		Serve: func(ctx context.Context, config serveConfig) error {
			got = config
			return nil
		},
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"serve"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if got.Port != 22345 {
		t.Fatalf("port = %d, want env port", got.Port)
	}
	if got.BindAddress != "0.0.0.0" {
		t.Fatalf("bind address = %q, want env bind address", got.BindAddress)
	}
	if got.DataDir != dataDir {
		t.Fatalf("data dir = %q, want env data dir %q", got.DataDir, dataDir)
	}
}

func TestServeCommandPassesEnableRequestLogsFromEnvironment(t *testing.T) {
	var got serveConfig
	dataDir := filepath.Join(t.TempDir(), "env-data")
	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("API_KEY_SECRET", "test-api-key-secret")
	t.Setenv("ENABLE_REQUEST_LOGS", "true")
	cmd := newRootCommand(rootConfig{
		Version: "0.1.0-test",
		Serve: func(ctx context.Context, config serveConfig) error {
			got = config
			return nil
		},
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"serve"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if !got.EnableRequestLogs {
		t.Fatal("EnableRequestLogs = false, want true from environment")
	}
}

func TestServeCommandFlagsOverrideEnvironmentDefaults(t *testing.T) {
	var got serveConfig
	flagDataDir := filepath.Join(t.TempDir(), "flag-data")
	t.Setenv("PORT", "22345")
	t.Setenv("DATA_DIR", filepath.Join(t.TempDir(), "env-data"))
	t.Setenv("API_KEY_SECRET", "test-api-key-secret")
	cmd := newRootCommand(rootConfig{
		Version: "0.1.0-test",
		Serve: func(ctx context.Context, config serveConfig) error {
			got = config
			return nil
		},
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--data-dir", flagDataDir, "serve", "--port", "20129"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if got.Port != 20129 {
		t.Fatalf("port = %d, want flag port", got.Port)
	}
	if got.DataDir != flagDataDir {
		t.Fatalf("data dir = %q, want flag data dir %q", got.DataDir, flagDataDir)
	}
}

func TestServeCommandFailsInvalidPortEnv(t *testing.T) {
	t.Setenv("PORT", "99999")
	t.Setenv("API_KEY_SECRET", "test-api-key-secret")
	cmd := newRootCommand(rootConfig{
		Version: "0.1.0-test",
		Serve: func(ctx context.Context, config serveConfig) error {
			t.Fatal("serve runner should not be called")
			return nil
		},
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"serve"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("execute should fail")
	}
	if !strings.Contains(err.Error(), "port must be 1-65535") {
		t.Fatalf("error = %q", err)
	}
}

func TestServeCommandFailsInvalidBooleanEnv(t *testing.T) {
	t.Setenv("RTK_ENABLED", "maybe")
	t.Setenv("API_KEY_SECRET", "test-api-key-secret")
	cmd := newRootCommand(rootConfig{
		Version: "0.1.0-test",
		Serve: func(ctx context.Context, config serveConfig) error {
			t.Fatal("serve runner should not be called")
			return nil
		},
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"serve"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("execute should fail")
	}
	if !strings.Contains(err.Error(), "RTK_ENABLED must be a boolean") {
		t.Fatalf("error = %q", err)
	}
}

func TestServeCommandRequiresAPIKeySecretByDefault(t *testing.T) {
	cmd := newRootCommand(rootConfig{
		Version: "0.1.0-test",
		Serve: func(ctx context.Context, config serveConfig) error {
			t.Fatal("serve runner should not be called")
			return nil
		},
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"serve", "--data-dir", t.TempDir()})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("execute should fail")
	}
	if !strings.Contains(err.Error(), "API_KEY_SECRET required when REQUIRE_API_KEY=true") {
		t.Fatalf("error = %q", err)
	}
}

func TestExpandServeDataDirExpandsHome(t *testing.T) {
	got, err := expandServeDataDir("~/.g0router")
	if err != nil {
		t.Fatalf("expandServeDataDir: %v", err)
	}

	if strings.Contains(got, "~") {
		t.Fatalf("expanded path = %q, should not contain ~", got)
	}
	if !strings.HasSuffix(got, ".g0router") {
		t.Fatalf("expanded path = %q, want .g0router suffix", got)
	}
}

func TestVersionCommandPrintsVersion(t *testing.T) {
	cmd := NewRootCommand("0.1.0-test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if got := strings.TrimSpace(out.String()); got != "g0router 0.1.0-test" {
		t.Fatalf("output = %q, want version", got)
	}
}

func TestStatusCommandUsesDataDir(t *testing.T) {
	cmd := NewRootCommand("0.1.0-test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--data-dir", t.TempDir(), "status"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if got := out.String(); !strings.Contains(got, "store: ok") {
		t.Fatalf("output = %q, want store status", got)
	}
}

func TestHealthcheckCommandFailsWhenServerUnavailable(t *testing.T) {
	cmd := NewRootCommand("test")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"healthcheck", "--url", "http://127.0.0.1:1/healthz"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("execute error is nil")
	}
}

func TestHealthcheckCommandChecksServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			t.Fatalf("path = %q, want /healthz", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cmd := NewRootCommand("test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"healthcheck", "--url", server.URL + "/healthz"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if got := out.String(); !strings.Contains(got, "healthcheck: ok") {
		t.Fatalf("output = %q, want healthcheck status", got)
	}
}

func TestDefaultServerConfigWiresWave4ADependencies(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	cfg := newServerConfig(context.Background(), serveConfig{Port: 20128, Version: "test"}, s)
	if cfg.OAuthFlows["minimax"] == nil {
		t.Fatal("minimax oauth flow should be wired")
	}

	models, err := cfg.ModelSource.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	foundOpenAI := false
	for _, model := range models {
		if model.Provider == providers.ProviderOpenAI {
			foundOpenAI = true
		}
	}
	if !foundOpenAI {
		t.Fatalf("models = %+v, want openai model", models)
	}

	if cfg.QuotaFetchers[providers.ProviderOpenAI] == nil {
		t.Fatal("openai quota fetcher should be wired")
	}
}

func TestDefaultServerConfigWrapsDefaultQuotaFetchersWithCache(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	cfg := newServerConfig(context.Background(), serveConfig{Port: 20128, Version: "test"}, s)
	fetcher := cfg.QuotaFetchers[providers.ProviderOpenAI]
	if _, ok := fetcher.(*usage.CachingQuotaFetcher); !ok {
		t.Fatalf("openai quota fetcher is %T, want cached fetcher", fetcher)
	}

	_, err := fetcher.FetchQuota(context.Background(), providers.Key{Provider: providers.ProviderOpenAI})
	if !errors.Is(err, usage.ErrQuotaUnsupported) {
		t.Fatalf("error = %v, want ErrQuotaUnsupported", err)
	}
}

func TestDefaultServerConfigRegistersOpenRouterQuotaFetcher(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	cfg := newServerConfig(context.Background(), serveConfig{Port: 20128, Version: "test"}, s)
	fetcher := cfg.QuotaFetchers[providers.ProviderOpenRouter]
	if fetcher == nil {
		t.Fatal("openrouter quota fetcher should be wired")
	}
	if !usage.IsOpenRouterQuotaFetcher(fetcher) {
		t.Fatalf("openrouter quota fetcher is %T, want OpenRouter quota fetcher", fetcher)
	}
}

func TestDefaultQuotaFetchersReturnUnsupportedForQuotaFalseProviders(t *testing.T) {
	fetchers := defaultQuotaFetchers()
	matrix := providermatrix.ProviderMatrix()

	for _, entry := range matrix.Entries() {
		if !entry.PublicInference {
			continue
		}
		provider := providers.ModelProvider(entry.G0RouterID)
		fetcher := fetchers[provider]
		if fetcher == nil {
			t.Fatalf("%s default quota fetcher missing", provider)
		}
		if entry.Quota {
			continue
		}
		_, err := fetcher.FetchQuota(context.Background(), providers.Key{Provider: provider})
		if !errors.Is(err, usage.ErrQuotaUnsupported) {
			t.Fatalf("%s quota error = %v, want ErrQuotaUnsupported", provider, err)
		}
	}
}

func TestProvidersListShowsSupportedInferenceProvidersOnly(t *testing.T) {
	cmd := NewRootCommand("test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"providers", "list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	output := out.String()
	for _, want := range []string{"openai", "alibaba", "anthropic", "azure", "bedrock", "cerebras", "cloudflare-ai-gateway", "cohere", "deepseek", "fireworks", "gemini", "github-copilot", "groq", "huggingface", "kilo", "kimi", "litellm", "lm-studio", "mistral", "minimax", "nebius", "nvidia", "ollama", "openrouter", "perplexity", "qianfan", "qwen", "together", "vercel-ai-gateway", "vertex", "vllm", "xai", "xiaomi", "zhipu"} {
		if !strings.Contains(output, want+"\n") {
			t.Fatalf("output = %q, want supported provider %q", output, want)
		}
	}
	for _, unsupported := range []string{"cursor"} {
		if strings.Contains(output, unsupported+"\n") {
			t.Fatalf("output = %q, should not advertise %q as inference provider", output, unsupported)
		}
	}
}

func TestDefaultServerConfigWiresWave7BRuntime(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	cfg := newServerConfig(context.Background(), serveConfig{
		Port:          20128,
		Version:       "test",
		RequireAPIKey: true,
		APIKeySecret:  "env-secret",
	}, s)
	if cfg.InferenceEngine == nil {
		t.Fatal("InferenceEngine is nil")
	}
	if cfg.MCPClientManager == nil {
		t.Fatal("MCPClientManager is nil")
	}
	if cfg.MCPToolManager == nil {
		t.Fatal("MCPToolManager is nil")
	}
	if cfg.MCPInstanceRuntime == nil {
		t.Fatal("MCPInstanceRuntime is nil")
	}
	engine, ok := cfg.InferenceEngine.(*proxy.Engine)
	if !ok {
		t.Fatalf("InferenceEngine type = %T, want *proxy.Engine", cfg.InferenceEngine)
	}
	if engine.MCPToolManager() != cfg.MCPToolManager {
		t.Fatal("InferenceEngine is not wired to the server MCP tool manager")
	}

	models, err := cfg.InferenceEngine.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("ListModels returned no models")
	}

	_, err = cfg.InferenceEngine.Dispatch(context.Background(), &providers.ChatRequest{
		Model:    "gpt-4o",
		Messages: []providers.Message{{Role: "user", Content: "ping"}},
	})
	if !errors.Is(err, proxy.ErrNoConnections) {
		t.Fatalf("Dispatch error = %v, want ErrNoConnections", err)
	}
}

func TestDefaultInferenceEngineRegistersImplementedVertexProvider(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	engine := newDefaultInferenceEngine(s)

	if !containsModelProvider(engine.RegisteredProviders(), providers.ProviderVertex) {
		t.Fatalf("registered providers = %v, want vertex", engine.RegisteredProviders())
	}
}

func TestDefaultInferenceEngineRegistersKiloProvider(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	engine := newDefaultInferenceEngine(s)

	if !containsModelProvider(engine.RegisteredProviders(), providers.ProviderKilo) {
		t.Fatalf("registered providers = %v, want kilo", engine.RegisteredProviders())
	}
}

func TestDefaultInferenceEngineRegistersOllamaCloudProvider(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	engine := newDefaultInferenceEngine(s)

	if !containsModelProvider(engine.RegisteredProviders(), providers.ProviderOllamaCloud) {
		t.Fatalf("registered providers = %v, want ollama-cloud", engine.RegisteredProviders())
	}
}

func TestDefaultInferenceEngineRegistersReplicateProvider(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	engine := newDefaultInferenceEngine(s)

	if !containsModelProvider(engine.RegisteredProviders(), providers.ProviderReplicate) {
		t.Fatalf("registered providers = %v, want replicate", engine.RegisteredProviders())
	}
}

func TestDefaultServerConfigServesGatewayAndMCPRuntime(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	cfg := newServerConfig(context.Background(), serveConfig{Port: 20128, Version: "test"}, s)
	_, baseURL := startCLITestServer(t, cfg)

	// The proxy always requires a valid API key; mint one and send it.
	_, proxyKey, err := s.CreateAPIKey("gateway-test", "")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	authHeader := map[string]string{"Authorization": "Bearer " + proxyKey}

	resp, body := getCLITestWithHeaders(t, baseURL+"/v1/models", authHeader)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /v1/models status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	if strings.Contains(string(body), "engine unavailable") {
		t.Fatalf("/v1/models returned unavailable body: %s", body)
	}

	resp, body = postCLITestWithHeaders(t, baseURL+"/v1/chat/completions", `{"model":"gpt-4o","messages":[{"role":"user","content":"ping"}]}`, authHeader)
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("POST /v1/chat/completions status = %d, want 503; body=%s", resp.StatusCode, body)
	}
	var chatError struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &chatError); err != nil {
		t.Fatalf("unmarshal chat error: %v; body=%s", err, body)
	}
	if chatError.Error.Message != "no active provider connections" || chatError.Error.Code != "no_active_connections" {
		t.Fatalf("chat error = %+v, want no_active_connections", chatError.Error)
	}
	if strings.Contains(string(body), "engine unavailable") {
		t.Fatalf("chat response returned unavailable body: %s", body)
	}

	mcpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("mcp method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("MCP-Protocol-Version"); got == "" {
			t.Fatal("mcp initialize request missing protocol version")
		}
		w.Header().Set("Mcp-Session-Id", "session-1")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  map[string]any{},
		})
	}))
	defer mcpServer.Close()

	resp, body = postCLITest(t, baseURL+"/api/mcp/clients", `{"name":"docs","transport":"streamable-http","url":"`+mcpServer.URL+`","is_active":true}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/mcp/clients status = %d, want 201; body=%s", resp.StatusCode, body)
	}
	if strings.Contains(string(body), "runtime unavailable") {
		t.Fatalf("mcp registration returned runtime unavailable body: %s", body)
	}
}

func TestDefaultServerConfigRehydratesActiveMCPInstances(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	mcpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer startup-secret" {
			http.Error(w, "missing startup auth", http.StatusUnauthorized)
			return
		}
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "initialize":
			w.Header().Set("Mcp-Session-Id", "startup-session")
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": map[string]any{}})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": map[string]any{
					"tools": []map[string]any{{
						"name":        "search",
						"description": "Search startup docs",
						"inputSchema": map[string]any{"type": "object"},
					}},
				},
			})
		default:
			http.Error(w, "unexpected method "+req.Method, http.StatusBadRequest)
		}
	}))
	defer mcpServer.Close()

	urlValue := mcpServer.URL
	instance := &store.MCPInstance{
		Name:       "startup-docs",
		ServerKey:  "startup-docs",
		LaunchType: mcp.LaunchHTTP,
		Transport:  mcp.TransportStreamableHTTP,
		URL:        &urlValue,
		Headers:    map[string]string{"Authorization": "Bearer startup-secret"},
		IsActive:   true,
	}
	if err := s.CreateMCPInstance(instance); err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}

	cfg := newServerConfig(context.Background(), serveConfig{Port: 20128, Version: "test"}, s)

	tools := cfg.MCPToolManager.CompactTools()
	if len(tools) != 1 || tools[0].Function.Name != instance.ID+"__search" {
		t.Fatalf("tools = %+v, want rehydrated startup tool", tools)
	}
	got, err := s.GetMCPInstance(instance.ID)
	if err != nil {
		t.Fatalf("GetMCPInstance: %v", err)
	}
	if got.ToolManifest == nil || len(got.ToolManifest.Tools) != 1 {
		t.Fatalf("manifest = %+v, want cached startup manifest", got.ToolManifest)
	}
	if got.HealthStatus != "healthy" {
		t.Fatalf("health = %q, want healthy", got.HealthStatus)
	}
	if got.LastHealthCheck == nil || *got.LastHealthCheck == "" {
		t.Fatalf("last health check = %v, want timestamp", got.LastHealthCheck)
	}
}

func TestDefaultServerConfigRegistersBuiltInSearchTools(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	kagiKey := "kagi-key"
	tavilyKey := "tavily-key"
	for _, conn := range []*store.Connection{
		{Provider: "kagi", Name: "kagi", AuthType: store.AuthTypeAPIKey, APIKey: &kagiKey, IsActive: true},
		{Provider: "tavily", Name: "tavily", AuthType: store.AuthTypeAPIKey, APIKey: &tavilyKey, IsActive: true},
	} {
		if err := s.CreateConnection(conn); err != nil {
			t.Fatalf("CreateConnection %s: %v", conn.Provider, err)
		}
	}

	cfg := newServerConfig(context.Background(), serveConfig{Port: 20128, Version: "test"}, s)

	tools := cfg.MCPToolManager.CompactTools()
	got := make(map[string]bool, len(tools))
	for _, tool := range tools {
		got[tool.Function.Name] = true
	}
	for _, want := range []string{"kagi__search", "tavily__search"} {
		if !got[want] {
			t.Fatalf("tools = %+v, missing %s", tools, want)
		}
	}
}

func TestDefaultServerConfigRehydratesMCPInstanceWithPersistedOAuth(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad token request", http.StatusBadRequest)
			return
		}
		if r.PostForm.Get("grant_type") != "refresh_token" || r.PostForm.Get("refresh_token") != "refresh-token" {
			http.Error(w, "unexpected refresh form", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "refreshed-token",
			"refresh_token": "next-refresh-token",
			"expires_in":    3600,
		})
	}))
	defer tokenServer.Close()

	mcpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer refreshed-token" {
			http.Error(w, "missing oauth bearer", http.StatusUnauthorized)
			return
		}
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "initialize":
			w.Header().Set("Mcp-Session-Id", "oauth-session")
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": map[string]any{}})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": map[string]any{
					"tools": []map[string]any{{"name": "search", "description": "Search"}},
				},
			})
		default:
			http.Error(w, "unexpected method "+req.Method, http.StatusBadRequest)
		}
	}))
	defer mcpServer.Close()

	urlValue := mcpServer.URL
	accountLabel := "work"
	instance := &store.MCPInstance{
		Name:         "oauth-docs",
		ServerKey:    "oauth-docs",
		LaunchType:   mcp.LaunchHTTP,
		Transport:    mcp.TransportStreamableHTTP,
		URL:          &urlValue,
		AccountLabel: &accountLabel,
		IsActive:     true,
	}
	if err := s.CreateMCPInstance(instance); err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}
	if err := s.UpsertMCPOAuthAccount(&store.MCPOAuthAccount{
		InstanceID:   instance.ID,
		AccountLabel: accountLabel,
		ResourceURI:  mcpServer.URL,
		AccessToken:  "expired-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(-time.Minute),
		AuthMetadata: map[string]string{"token_endpoint": tokenServer.URL},
	}); err != nil {
		t.Fatalf("UpsertMCPOAuthAccount: %v", err)
	}

	cfg := newServerConfig(context.Background(), serveConfig{Port: 20128, Version: "test"}, s)

	tools := cfg.MCPToolManager.CompactTools()
	if len(tools) != 1 || tools[0].Function.Name != instance.ID+"__search" {
		t.Fatalf("tools = %+v, want oauth-backed startup tool", tools)
	}
	accounts, err := s.ListMCPOAuthAccounts(instance.ID)
	if err != nil {
		t.Fatalf("ListMCPOAuthAccounts: %v", err)
	}
	if len(accounts) != 1 || accounts[0].AccessToken != "refreshed-token" || accounts[0].RefreshToken != "next-refresh-token" {
		t.Fatalf("accounts = %+v, want refreshed persisted account", accounts)
	}
}

func TestDefaultServerConfigUsesAuthEnvironment(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	cfg := newServerConfig(context.Background(), serveConfig{
		Port:          20128,
		Version:       "test",
		RequireAPIKey: true,
		APIKeySecret:  "env-secret",
	}, s)
	if !cfg.RequireAPIKey {
		t.Fatal("RequireAPIKey = false, want true")
	}
	if cfg.APIKeySecret != "env-secret" {
		t.Fatalf("APIKeySecret = %q, want env-secret", cfg.APIKeySecret)
	}
	if cfg.APIKeyValidator == nil {
		t.Fatal("APIKeyValidator is nil")
	}

	_, raw, err := s.CreateAPIKey("default", "env-secret")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	ok, err := cfg.APIKeyValidator.ValidateAPIKey(raw, cfg.APIKeySecret)
	if err != nil {
		t.Fatalf("ValidateAPIKey: %v", err)
	}
	if !ok {
		t.Fatal("ValidateAPIKey = false, want true")
	}
}

func TestDefaultServerConfigPassesRequestLoggingToggle(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	cfg := newServerConfig(context.Background(), serveConfig{
		Port:              20128,
		Version:           "test",
		EnableRequestLogs: true,
	}, s)

	if !cfg.EnableRequestLogs {
		t.Fatal("EnableRequestLogs = false, want true")
	}
}

func startCLITestServer(t *testing.T, config api.ServerConfig) (*api.Server, string) {
	t.Helper()

	srv := api.NewServer(config)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("Serve: %v", err)
		}
	}()
	t.Cleanup(func() { _ = srv.Stop() })

	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener addr is %T, want *net.TCPAddr", ln.Addr())
	}
	return srv, "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(addr.Port))
}

func getCLITest(t *testing.T, url string) (*http.Response, []byte) {
	t.Helper()

	resp, err := cliHTTPClient().Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	return resp, readCLIResponseBody(t, resp)
}

func postCLITest(t *testing.T, url string, body string) (*http.Response, []byte) {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := cliHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp, readCLIResponseBody(t, resp)
}

func getCLITestWithHeaders(t *testing.T, url string, headers map[string]string) (*http.Response, []byte) {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := cliHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	return resp, readCLIResponseBody(t, resp)
}

func postCLITestWithHeaders(t *testing.T, url string, body string, headers map[string]string) (*http.Response, []byte) {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := cliHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp, readCLIResponseBody(t, resp)
}

func readCLIResponseBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		t.Fatalf("read response body: %v", err)
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return body
}

func cliHTTPClient() *http.Client {
	return &http.Client{Timeout: 2 * time.Second}
}

func containsModelProvider(values []providers.ModelProvider, want providers.ModelProvider) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
