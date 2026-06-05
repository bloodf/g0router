package api

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/store"
)

func postAPITestJSONWithHeaders(t *testing.T, url, body string, headers map[string]string) (*http.Response, []byte) {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if _, ok := headers["X-API-Key"]; !ok {
		if _, ok := headers["Authorization"]; !ok {
			req.Header.Set("X-API-Key", testHarnessAPIKey)
		}
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
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

func TestInferenceLoggingRecordsClientToolFromHeader(t *testing.T) {
	s := newAPITestStore(t)
	enableRequestLogs(t, s)

	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		InferenceEngine: routeInferenceEngine{response: routeChatResponseWithUsage()},
	})

	resp, body := postAPITestJSONWithHeaders(t, baseURL+"/v1/chat/completions",
		`{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`,
		map[string]string{"X-Client-Tool": "codex"})
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	entries, err := s.GetUsage(store.UsageFilter{})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("usage entries = %d, want 1", len(entries))
	}
	if entries[0].ClientTool == nil || *entries[0].ClientTool != "codex" {
		t.Fatalf("client tool = %v, want codex", entries[0].ClientTool)
	}
}

func TestInferenceLoggingFallsBackToUserAgentForClientTool(t *testing.T) {
	s := newAPITestStore(t)
	enableRequestLogs(t, s)

	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		InferenceEngine: routeInferenceEngine{response: routeChatResponseWithUsage()},
	})

	resp, body := postAPITestJSONWithHeaders(t, baseURL+"/v1/chat/completions",
		`{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`,
		map[string]string{"User-Agent": "my-agent/1.0"})
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	entries, err := s.GetUsage(store.UsageFilter{})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 1 || entries[0].ClientTool == nil || *entries[0].ClientTool != "my-agent/1.0" {
		t.Fatalf("client tool = %+v, want my-agent/1.0", entries)
	}
}

func TestInferenceLoggingRecordsRTKBytesSaved(t *testing.T) {
	s := newAPITestStore(t)
	enableRequestLogs(t, s)
	// RTK is enabled by default in store settings.

	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		InferenceEngine: routeInferenceEngine{response: routeChatResponseWithUsage()},
	})

	bulky := strings.Repeat("compressible log line\n", 600)
	body := `{"model":"gpt-4o","messages":[{"role":"tool","tool_call_id":"call-1","content":` + jsonString(bulky) + `}]}`

	resp, respBody := postAPITestJSON(t, baseURL+"/v1/chat/completions", body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, respBody)
	}

	entries, err := s.GetUsage(store.UsageFilter{})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("usage entries = %d, want 1", len(entries))
	}
	if entries[0].RTKBytesSaved == nil || *entries[0].RTKBytesSaved <= 0 {
		t.Fatalf("rtk bytes saved = %v, want > 0", entries[0].RTKBytesSaved)
	}
}

func TestInferenceLoggingRecordsComboName(t *testing.T) {
	s := newAPITestStore(t)
	enableRequestLogs(t, s)
	combo := &store.Combo{
		Name:     "my-combo",
		Steps:    []store.ComboStep{{Provider: "openai", Model: "gpt-4o"}},
		IsActive: true,
	}
	if err := s.CreateCombo(combo); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		InferenceEngine: routeInferenceEngine{response: routeChatResponseWithUsage()},
	})

	resp, body := postAPITestJSON(t, baseURL+"/v1/chat/completions",
		`{"model":"my-combo","messages":[{"role":"user","content":"hello"}]}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	entries, err := s.GetUsage(store.UsageFilter{})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("usage entries = %d, want 1", len(entries))
	}
	if entries[0].ComboName == nil || *entries[0].ComboName != "my-combo" {
		t.Fatalf("combo name = %v, want my-combo", entries[0].ComboName)
	}
}

func jsonString(value string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range value {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
