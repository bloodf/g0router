package anthropic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

// TestNewForProviderClaudeFormat verifies the additive constructor used by the
// w7-prov-special-a claude-format providers (glm/kimi/minimax/minimax-cn). The
// provider must POST to the catalog base URL VERBATIM (the ref baseUrl is the
// full .../v1/messages URL) with the ?beta=true suffix, the x-api-key auth
// header, and the Anthropic-Beta header (CLAUDE_API_HEADERS). The canned
// Anthropic-Messages response must be converted to an OpenAI ChatResponse.
func TestNewForProviderClaudeFormat(t *testing.T) {
	var gotPath, gotQuery, gotAPIKey, gotVersion, gotBeta string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAPIKey = r.Header.Get("x-api-key")
		gotVersion = r.Header.Get("anthropic-version")
		gotBeta = r.Header.Get("anthropic-beta")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"glm-5.1","content":[{"type":"text","text":"hello"}],"stop_reason":"end_turn","usage":{"input_tokens":3,"output_tokens":2}}`))
	}))
	defer srv.Close()

	// The catalog base URL is the full messages endpoint (ref baseUrl).
	p := NewForProvider("glm", srv.URL+"/api/anthropic/v1/messages")

	if p.GetProvider() != schemas.ModelProvider("glm") {
		t.Errorf("GetProvider() = %q, want glm", p.GetProvider())
	}

	resp, perr := p.ChatCompletion(&schemas.GatewayContext{}, schemas.Key{Value: "secret-key"}, &schemas.ChatRequest{Model: "glm-5.1"})
	if perr != nil {
		t.Fatalf("ChatCompletion error: %v", perr.Message)
	}

	if gotPath != "/api/anthropic/v1/messages" {
		t.Errorf("request path = %q, want /api/anthropic/v1/messages (base URL verbatim, no extra /v1/messages)", gotPath)
	}
	if gotQuery != "beta=true" {
		t.Errorf("request query = %q, want beta=true", gotQuery)
	}
	if gotAPIKey != "secret-key" {
		t.Errorf("x-api-key = %q, want secret-key", gotAPIKey)
	}
	if gotVersion != "2023-06-01" {
		t.Errorf("anthropic-version = %q, want 2023-06-01", gotVersion)
	}
	if gotBeta != "claude-code-20250219,interleaved-thinking-2025-05-14" {
		t.Errorf("anthropic-beta = %q, want claude-code-20250219,interleaved-thinking-2025-05-14", gotBeta)
	}
	if resp == nil || len(resp.Choices) == 0 {
		t.Fatalf("ChatCompletion response empty: %+v", resp)
	}
	if got := resp.Choices[0].Message.Content; got != "hello" {
		t.Errorf("response content = %q, want hello", got)
	}
}

// TestNewProviderUnchanged verifies the existing NewProvider() path is
// untouched: it still targets the hardcoded anthropic base + /v1/messages with
// no beta suffix.
func TestNewProviderUnchanged(t *testing.T) {
	var gotPath, gotQuery, gotBeta string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotBeta = r.Header.Get("anthropic-beta")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	_, perr := p.ChatCompletion(&schemas.GatewayContext{}, schemas.Key{Value: "k"}, &schemas.ChatRequest{Model: "claude-3-opus"})
	if perr != nil {
		t.Fatalf("ChatCompletion error: %v", perr.Message)
	}
	if gotPath != "/v1/messages" {
		t.Errorf("NewProvider path = %q, want /v1/messages (unchanged)", gotPath)
	}
	if gotQuery != "" {
		t.Errorf("NewProvider query = %q, want empty (no beta suffix)", gotQuery)
	}
	if gotBeta != "" {
		t.Errorf("NewProvider anthropic-beta = %q, want empty (unchanged)", gotBeta)
	}
}
