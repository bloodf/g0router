package api

import (
	"io"
	"net/http"
	"regexp"
	"testing"
)

type fakeAPIKeyValidator struct {
	validKeys map[string]bool
}

func (f fakeAPIKeyValidator) ValidateAPIKey(key, secret string) (bool, error) {
	return f.validKeys[key] && secret == "test-secret", nil
}

func TestCORSHeaders(t *testing.T) {
	_, baseURL := startTestServer(t, ServerConfig{Port: 0, Version: "test"})

	resp, err := httpClient().Get(baseURL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
	}
}

func TestOptionsReturns204(t *testing.T) {
	_, baseURL := startTestServer(t, ServerConfig{Port: 0, Version: "test"})

	req, err := http.NewRequest(http.MethodOptions, baseURL+"/v1/chat/completions", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("OPTIONS /v1/chat/completions: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if len(body) != 0 {
		t.Fatalf("body = %q, want empty", string(body))
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

func TestAuthNotRequired(t *testing.T) {
	_, baseURL := startTestServer(t, ServerConfig{
		Port:          0,
		Version:       "test",
		RequireAPIKey: false,
	})

	resp, err := httpClient().Get(baseURL + "/v1/chat/completions")
	if err != nil {
		t.Fatalf("GET /v1/chat/completions: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		t.Fatal("auth should not be required")
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 until /v1 handler exists", resp.StatusCode)
	}
}

func TestHealthzBypassesAuth(t *testing.T) {
	_, baseURL := startTestServer(t, ServerConfig{
		Port:          0,
		Version:       "test",
		RequireAPIKey: true,
		APIKeySecret:  "test-secret",
		APIKeyValidator: fakeAPIKeyValidator{
			validKeys: map[string]bool{"g0r_valid": true},
		},
	})

	resp, err := httpClient().Get(baseURL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func startTestServer(t *testing.T, config ServerConfig) (*Server, string) {
	t.Helper()

	srv := NewServer(config)
	ln := srv.listener()
	if ln == nil {
		t.Fatal("listener failed")
	}

	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Stop() })

	return srv, "http://" + localhostAddr(t, ln)
}

var _ interface {
	ValidateAPIKey(string, string) (bool, error)
} = fakeAPIKeyValidator{}
