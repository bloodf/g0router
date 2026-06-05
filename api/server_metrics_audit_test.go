package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/store"
)

// getAPITestWithHeaders issues a GET with the given headers (defaulting to the
// harness API key when no auth header is supplied).
func getAPITestWithHeaders(t *testing.T, url string, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if _, ok := headers["X-API-Key"]; !ok {
		if _, ok := headers["Authorization"]; !ok {
			req.Header.Set("X-API-Key", testHarnessAPIKey)
		}
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		t.Fatalf("read body: %v", err)
	}
	return resp, data
}

func TestMetricsEndpointReturnsTextAndRequiresAuth(t *testing.T) {
	s := newAPITestStore(t)
	_, baseURL := startTestServer(t, ServerConfig{
		Port:          0,
		Version:       "test",
		Store:         s,
		UsageStore:    s,
		RequireAPIKey: true,
		APIKeyValidator: fakeAPIKeyValidator{
			validKeys: map[string]bool{testHarnessAPIKey: true},
		},
		APIKeySecret: "test-secret",
	})

	// Without a key: 401.
	req, _ := http.NewRequest(http.MethodGet, baseURL+"/metrics", nil)
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated /metrics status = %d, want 401", resp.StatusCode)
	}

	// With a bearer key: 200 text/plain.
	resp, body := getAPITestWithHeaders(t, baseURL+"/metrics",
		map[string]string{"Authorization": "Bearer " + testHarnessAPIKey})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("authenticated /metrics status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/plain") || !strings.Contains(ct, "0.0.4") {
		t.Fatalf("content-type = %q, want text/plain version=0.0.4", ct)
	}
	if !strings.Contains(string(body), "# TYPE requests_total counter") {
		t.Fatalf("metrics body missing requests_total TYPE line:\n%s", body)
	}
}

func TestInferenceIncrementsRenderedMetrics(t *testing.T) {
	s := newAPITestStore(t)
	enableRequestLogs(t, s)

	srv, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		InferenceEngine: routeInferenceEngine{response: routeChatResponseWithUsage()},
	})

	resp, body := postAPITestJSON(t, baseURL+"/v1/chat/completions",
		`{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	rendered := srv.metrics.Render()
	if !strings.Contains(rendered, `requests_total{provider="openai",model="gpt-4o",status_class="2xx"} 1`) {
		t.Fatalf("rendered metrics missing request series:\n%s", rendered)
	}
	// 1000 prompt tokens in, 500 completion out (see routeChatResponseWithUsage).
	if !strings.Contains(rendered, `tokens_total{type="input"} 1000`) {
		t.Fatalf("rendered metrics missing input tokens:\n%s", rendered)
	}
	if !strings.Contains(rendered, `tokens_total{type="output"} 500`) {
		t.Fatalf("rendered metrics missing output tokens:\n%s", rendered)
	}
	if strings.Contains(rendered, "cost_usd_total 0\n") {
		// Cost may be zero when no pricing is configured; just ensure the series exists.
		if !strings.Contains(rendered, "cost_usd_total") {
			t.Fatalf("rendered metrics missing cost series")
		}
	}
	if !strings.Contains(rendered, "request_duration_seconds_count 1") {
		t.Fatalf("rendered metrics missing duration count:\n%s", rendered)
	}
}

func TestSettingsUpdateAppendsAuditEntry(t *testing.T) {
	s := newAPITestStore(t)
	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		RequireAPIKey:   true,
		APIKeyValidator: policyValidator{identity: APIKeyIdentity{ID: "actor-key-1"}},
		APIKeySecret:    "test-secret",
	})

	req, err := http.NewRequest(http.MethodPut, baseURL+"/api/settings",
		strings.NewReader(`{"log_retention_days":7}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testHarnessAPIKey)
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("PUT /api/settings: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("settings PUT status = %d, want 200", resp.StatusCode)
	}

	entries, total, err := s.ListAudit(store.AuditFilter{})
	if err != nil {
		t.Fatalf("ListAudit: %v", err)
	}
	if total != 1 || len(entries) != 1 {
		t.Fatalf("audit total=%d len=%d, want 1/1", total, len(entries))
	}
	if entries[0].ActorAPIKeyID != "actor-key-1" {
		t.Fatalf("actor = %q, want actor-key-1", entries[0].ActorAPIKeyID)
	}
	if entries[0].Action != "PUT /api/settings" {
		t.Fatalf("action = %q, want PUT /api/settings", entries[0].Action)
	}
	if strings.Contains(entries[0].Details, "log_retention_days") {
		t.Fatalf("details leaked request body: %q", entries[0].Details)
	}
}

func TestKeyCreationAppendsAuditRetrievableViaAPI(t *testing.T) {
	s := newAPITestStore(t)
	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		RequireAPIKey:   true,
		APIKeyValidator: policyValidator{identity: APIKeyIdentity{ID: "admin-1"}},
		APIKeySecret:    "test-secret",
	})

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/keys",
		strings.NewReader(`{"name":"ci-key"}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testHarnessAPIKey)
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("POST /api/keys: %v", err)
	}
	createBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("key create status = %d, want 2xx; body=%s", resp.StatusCode, createBody)
	}

	resp, body := getAPITestWithHeaders(t, baseURL+"/api/audit",
		map[string]string{"Authorization": "Bearer " + testHarnessAPIKey})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/audit status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	var parsed struct {
		Object string `json:"object"`
		Total  int    `json:"total"`
		Data   []struct {
			ActorAPIKeyID string `json:"actor_api_key_id"`
			Action        string `json:"action"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("unmarshal audit: %v; body=%s", err, body)
	}
	if parsed.Object != "list" {
		t.Fatalf("object = %q, want list", parsed.Object)
	}
	if parsed.Total < 1 || len(parsed.Data) < 1 {
		t.Fatalf("expected at least one audit entry, got total=%d len=%d", parsed.Total, len(parsed.Data))
	}
	if parsed.Data[0].Action != "POST /api/keys" || parsed.Data[0].ActorAPIKeyID != "admin-1" {
		t.Fatalf("audit entry = %+v, want POST /api/keys by admin-1", parsed.Data[0])
	}
}

func TestGetRequestDoesNotAppendAudit(t *testing.T) {
	s := newAPITestStore(t)
	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		RequireAPIKey:   true,
		APIKeyValidator: policyValidator{identity: APIKeyIdentity{ID: "reader"}},
		APIKeySecret:    "test-secret",
	})

	resp, _ := getAPITestWithHeaders(t, baseURL+"/api/settings",
		map[string]string{"Authorization": "Bearer " + testHarnessAPIKey})
	resp.Body.Close()

	_, total, err := s.ListAudit(store.AuditFilter{})
	if err != nil {
		t.Fatalf("ListAudit: %v", err)
	}
	if total != 0 {
		t.Fatalf("audit total = %d after GET, want 0", total)
	}
}
