package api

import (
	"io"
	"net"
	"net/http"
	"regexp"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeAPIKeyValidator struct {
	validKeys map[string]bool
}

func (f fakeAPIKeyValidator) ValidateAPIKey(key, secret string) (bool, error) {
	return f.validKeys[key] && secret == "test-secret", nil
}

type fakeIdentityAPIKeyValidator struct {
	validKeys map[string]string
}

func (f fakeIdentityAPIKeyValidator) ValidateAPIKey(key, secret string) (bool, error) {
	_, ok := f.validKeys[key]
	return ok && secret == "test-secret", nil
}

func (f fakeIdentityAPIKeyValidator) ValidateAPIKeyIdentity(key, secret string) (*APIKeyIdentity, bool, error) {
	id, ok := f.validKeys[key]
	if !ok || secret != "test-secret" {
		return nil, false, nil
	}
	return &APIKeyIdentity{ID: id}, true, nil
}

func TestCORSHeaders(t *testing.T) {
	_, baseURL := startTestServer(t, ServerConfig{Port: 0, Version: "test"})

	resp, err := httpClient().Get(baseURL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got == "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want no wildcard", got)
	}
}

func TestCORSAllowsLocalOrigins(t *testing.T) {
	_, baseURL := startTestServer(t, ServerConfig{Port: 0, Version: "test"})

	req, err := http.NewRequest(http.MethodGet, baseURL+"/healthz", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Origin", "http://localhost:5173")

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("Access-Control-Allow-Origin = %q, want local origin echoed", got)
	}
}

func TestCORSRejectsNonLocalOrigins(t *testing.T) {
	_, baseURL := startTestServer(t, ServerConfig{Port: 0, Version: "test"})

	req, err := http.NewRequest(http.MethodGet, baseURL+"/healthz", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Origin", "https://evil.example")

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want empty for disallowed origin", got)
	}
}

func TestOptionsReturns204(t *testing.T) {
	_, baseURL := startTestServer(t, ServerConfig{Port: 0, Version: "test"})

	req, err := http.NewRequest(http.MethodOptions, baseURL+"/api/settings", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Origin", "http://127.0.0.1:3000")
	req.Header.Set("Access-Control-Request-Method", http.MethodPut)
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("OPTIONS /api/settings: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:3000" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want allowed origin echoed", got)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if len(body) != 0 {
		t.Fatalf("body = %q, want empty", string(body))
	}
}

func TestManagementRoutesRequireAPIKey(t *testing.T) {
	_, baseURL := startTestServer(t, ServerConfig{
		Port:          0,
		Version:       "test",
		RequireAPIKey: true,
		APIKeySecret:  "test-secret",
		APIKeyValidator: fakeAPIKeyValidator{
			validKeys: map[string]bool{"g0r_valid": true},
		},
	})

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/keys"},
		{method: http.MethodDelete, path: "/api/keys/key-1"},
		{method: http.MethodGet, path: "/api/providers"},
		{method: http.MethodGet, path: "/api/providers/openai/models"},
		{method: http.MethodGet, path: "/api/settings"},
		{method: http.MethodGet, path: "/api/connections"},
		{method: http.MethodPut, path: "/api/connections/conn-1"},
		{method: http.MethodGet, path: "/api/combos"},
		{method: http.MethodDelete, path: "/api/combos/combo-1"},
		{method: http.MethodPost, path: "/api/oauth/minimax/authorize"},
		{method: http.MethodGet, path: "/api/oauth/minimax/poll"},
		{method: http.MethodPost, path: "/api/oauth/minimax/exchange"},
		{method: http.MethodGet, path: "/api/usage"},
		{method: http.MethodGet, path: "/api/usage/summary"},
		{method: http.MethodGet, path: "/api/usage/quota/openai"},
		{method: http.MethodGet, path: "/api/logs"},
		{method: http.MethodGet, path: "/api/mcp/clients"},
		{method: http.MethodDelete, path: "/api/mcp/clients/client-1"},
		{method: http.MethodGet, path: "/api/mcp/instances"},
		{method: http.MethodDelete, path: "/api/mcp/instances/instance-1"},
		{method: http.MethodPost, path: "/api/mcp/instances/instance-1/auth/start"},
		{method: http.MethodGet, path: "/api/mcp/instances/instance-1/accounts"},
		{method: http.MethodPost, path: "/api/mcp/instances/instance-1/oauth/complete"},
		{method: http.MethodGet, path: "/api/mcp/tools"},
		{method: http.MethodPost, path: "/api/mcp/tools/search/execute"},
	}

	for _, tc := range tests {
		req, err := http.NewRequest(tc.method, baseURL+tc.path, nil)
		if err != nil {
			t.Fatalf("NewRequest %s %s: %v", tc.method, tc.path, err)
		}
		resp, err := httpClient().Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.method, tc.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s %s status = %d, want 401", tc.method, tc.path, resp.StatusCode)
		}
	}
}

func TestManagementRoutesAcceptValidAPIKey(t *testing.T) {
	store := newAPITestStore(t)
	_, baseURL := startTestServer(t, ServerConfig{
		Port:          0,
		Version:       "test",
		Store:         store,
		RequireAPIKey: true,
		APIKeySecret:  "test-secret",
		APIKeyValidator: fakeAPIKeyValidator{
			validKeys: map[string]bool{"g0r_valid": true},
		},
	})

	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/settings", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("X-API-Key", "g0r_valid")

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("GET /api/settings: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestOAuthCallbacksBypassManagementAuth(t *testing.T) {
	_, baseURL := startTestServer(t, ServerConfig{
		Port:          0,
		Version:       "test",
		RequireAPIKey: true,
		APIKeySecret:  "test-secret",
		APIKeyValidator: fakeAPIKeyValidator{
			validKeys: map[string]bool{"g0r_valid": true},
		},
	})

	for _, path := range []string{
		"/api/oauth/callback",
		"/api/mcp/oauth/callback",
	} {
		resp, err := httpClient().Get(baseURL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusUnauthorized {
			t.Fatalf("GET %s status = 401, want callback to bypass management auth", path)
		}
	}
}

func TestRequestIDPresent(t *testing.T) {
	_, baseURL := startTestServer(t, ServerConfig{Port: 0, Version: "test"})

	resp, err := httpClient().Get(baseURL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	requestID := resp.Header.Get("X-Request-ID")
	if requestID == "" {
		t.Fatal("X-Request-ID should be set")
	}
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !uuidPattern.MatchString(requestID) {
		t.Fatalf("X-Request-ID = %q, want UUID v4", requestID)
	}
}

func TestRequestIDUnique(t *testing.T) {
	_, baseURL := startTestServer(t, ServerConfig{Port: 0, Version: "test"})

	first, err := httpClient().Get(baseURL + "/healthz")
	if err != nil {
		t.Fatalf("first GET /healthz: %v", err)
	}
	defer first.Body.Close()

	second, err := httpClient().Get(baseURL + "/healthz")
	if err != nil {
		t.Fatalf("second GET /healthz: %v", err)
	}
	defer second.Body.Close()

	firstID := first.Header.Get("X-Request-ID")
	secondID := second.Header.Get("X-Request-ID")
	if firstID == "" || secondID == "" {
		t.Fatalf("request IDs should be set, got %q and %q", firstID, secondID)
	}
	if firstID == secondID {
		t.Fatalf("request IDs should be unique, both were %q", firstID)
	}
}

func TestAuthRequiredMissingKey(t *testing.T) {
	_, baseURL := startTestServer(t, ServerConfig{
		Port:          0,
		Version:       "test",
		RequireAPIKey: true,
		APIKeySecret:  "test-secret",
		APIKeyValidator: fakeAPIKeyValidator{
			validKeys: map[string]bool{"g0r_valid": true},
		},
	})

	resp, err := httpClient().Get(baseURL + "/v1/chat/completions")
	if err != nil {
		t.Fatalf("GET /v1/chat/completions: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestAuthRequiredValidKey(t *testing.T) {
	_, baseURL := startTestServer(t, ServerConfig{
		Port:          0,
		Version:       "test",
		RequireAPIKey: true,
		APIKeySecret:  "test-secret",
		APIKeyValidator: fakeAPIKeyValidator{
			validKeys: map[string]bool{"g0r_valid": true},
		},
	})

	req, err := http.NewRequest(http.MethodGet, baseURL+"/v1/chat/completions", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer g0r_valid")
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("GET /v1/chat/completions: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		t.Fatal("valid API key should pass auth")
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 until /v1 handler exists", resp.StatusCode)
	}
}

func TestProxyAlwaysRequiresAPIKey(t *testing.T) {
	_, baseURL := startTestServer(t, ServerConfig{
		Port:          0,
		Version:       "test",
		RequireAPIKey: false,
		APIKeySecret:  "test-secret",
		APIKeyValidator: fakeAPIKeyValidator{
			validKeys: map[string]bool{"g0r_valid": true},
		},
	})

	// Without a key the proxy is closed even though RequireAPIKey is false.
	resp, err := httpClient().Get(baseURL + "/v1/chat/completions")
	if err != nil {
		t.Fatalf("GET /v1/chat/completions: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (proxy always requires a key)", resp.StatusCode)
	}

	// A valid key passes auth.
	req, err := http.NewRequest(http.MethodGet, baseURL+"/v1/chat/completions", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer g0r_valid")
	keyed, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("GET /v1/chat/completions with key: %v", err)
	}
	keyed.Body.Close()
	if keyed.StatusCode == http.StatusUnauthorized {
		t.Fatal("valid API key should pass proxy auth")
	}
}

func TestManagementOpenWhenRequireAPIKeyFalse(t *testing.T) {
	store := newAPITestStore(t)
	_, baseURL := startTestServer(t, ServerConfig{
		Port:          0,
		Version:       "test",
		Store:         store,
		RequireAPIKey: false,
	})

	resp, err := httpClient().Get(baseURL + "/api/connections")
	if err != nil {
		t.Fatalf("GET /api/connections: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		t.Fatal("management plane should be open when RequireAPIKey is false")
	}
}

func TestPublicRoutesBypassAuth(t *testing.T) {
	_, baseURL := startTestServer(t, ServerConfig{
		Port:          0,
		Version:       "test",
		RequireAPIKey: true,
		APIKeySecret:  "test-secret",
		APIKeyValidator: fakeAPIKeyValidator{
			validKeys: map[string]bool{"g0r_valid": true},
		},
	})

	for _, path := range []string{"/healthz", "/"} {
		resp, err := httpClient().Get(baseURL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200", path, resp.StatusCode)
		}
	}
}

func TestClassifySourceIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want string
	}{
		{"loopback v4", "127.0.0.1", "local"},
		{"loopback v6", "::1", "local"},
		{"private 10", "10.1.2.3", "lan"},
		{"private 192.168", "192.168.1.5", "lan"},
		{"private 172.16", "172.16.4.4", "lan"},
		{"link-local v4", "169.254.1.1", "lan"},
		{"ula v6 fc00", "fc00::1", "lan"},
		{"link-local v6 fe80", "fe80::1", "lan"},
		{"tailscale low", "100.64.0.1", "tailscale"},
		{"tailscale high", "100.127.255.254", "tailscale"},
		{"public v4", "8.8.8.8", "public"},
		{"public v6", "2606:4700:4700::1111", "public"},
		{"public 100.128 above cgnat", "100.128.0.1", "public"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			if ip == nil {
				t.Fatalf("ParseIP(%q) = nil", tc.ip)
			}
			if got := classifySourceIP(ip); got != tc.want {
				t.Fatalf("classifySourceIP(%s) = %q, want %q", tc.ip, got, tc.want)
			}
		})
	}
}

func TestSourceAllowedPolicy(t *testing.T) {
	tests := []struct {
		name    string
		allowed []string
		remote  string
		path    string
		want    bool
	}{
		{"public blocked when only local", []string{"local"}, "8.8.8.8:1234", "/v1/chat/completions", false},
		{"public allowed when public present", []string{"local", "public"}, "8.8.8.8:1234", "/v1/chat/completions", true},
		{"loopback allowed when local present", []string{"local"}, "127.0.0.1:1234", "/v1/chat/completions", true},
		{"tailscale allowed only when tailscale present", []string{"tailscale"}, "100.64.0.1:1234", "/v1/chat/completions", true},
		{"tailscale blocked without tailscale", []string{"local"}, "100.64.0.1:1234", "/v1/chat/completions", false},
		{"api path enforced", []string{"local"}, "8.8.8.8:1234", "/api/settings", false},
		{"healthz bypasses policy", []string{"local"}, "8.8.8.8:1234", "/healthz", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := &Server{settingsCache: &store.Settings{AllowedSources: tc.allowed}}
			ctx := &fasthttp.RequestCtx{}
			ctx.Request.SetRequestURI(tc.path)
			addr, err := net.ResolveTCPAddr("tcp", tc.remote)
			if err != nil {
				t.Fatalf("ResolveTCPAddr(%q): %v", tc.remote, err)
			}
			ctx.SetRemoteAddr(addr)
			if got := srv.sourceAllowed(ctx); got != tc.want {
				t.Fatalf("sourceAllowed(remote=%s path=%s) = %v, want %v", tc.remote, tc.path, got, tc.want)
			}
		})
	}
}

// testHarnessAPIKey is accepted by the default validator injected into test
// servers that do not provide their own. The shared POST helpers send it so the
// always-on /v1 proxy auth passes without each test wiring its own validator.
const testHarnessAPIKey = "g0r_test"

func startTestServer(t *testing.T, config ServerConfig) (*Server, string) {
	t.Helper()

	// The inference proxy now always requires a valid API key. Tests that do not
	// exercise auth directly leave the validator unset; give them one that
	// accepts testHarnessAPIKey so /v1 requests can pass auth.
	if config.APIKeyValidator == nil {
		config.APIKeyValidator = fakeAPIKeyValidator{validKeys: map[string]bool{testHarnessAPIKey: true}}
		if config.APIKeySecret == "" {
			config.APIKeySecret = "test-secret"
		}
	}

	srv := NewServer(config)
	ln := apiTestListener(t)

	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Stop() })

	return srv, "http://" + localhostAddr(t, ln)
}

var _ interface {
	ValidateAPIKey(string, string) (bool, error)
} = fakeAPIKeyValidator{}
