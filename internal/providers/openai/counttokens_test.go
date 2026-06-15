package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

// TestCountTokensSuccess verifies the CountTokens transport posts to
// /v1/responses/input_tokens and maps the upstream {"input_tokens":N} body into
// a TokenCountResponse{Tokens:N} (PAR-BF-OAI-004).
func TestCountTokensSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses/input_tokens" {
			t.Errorf("path = %q, want /v1/responses/input_tokens", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"input_tokens":42}`)
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	resp, perr := p.CountTokens(&schemas.GatewayContext{}, schemas.Key{Value: "test-key"}, &schemas.ChatRequest{Model: "gpt-4", Messages: []schemas.Message{{Role: "user", Content: "hi"}}})
	if perr != nil {
		t.Fatalf("CountTokens error: %v", perr.Message)
	}
	if resp == nil {
		t.Fatal("response is nil")
	}
	if resp.Tokens != 42 {
		t.Errorf("tokens = %d, want 42", resp.Tokens)
	}
}

// TestCountTokensAcceptsTokensField verifies the transport also decodes the
// {"tokens":N} shape (the bare TokenCountResponse field name).
func TestCountTokensAcceptsTokensField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"tokens":7}`)
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	resp, perr := p.CountTokens(&schemas.GatewayContext{}, schemas.Key{Value: "test-key"}, &schemas.ChatRequest{Model: "gpt-4"})
	if perr != nil {
		t.Fatalf("CountTokens error: %v", perr.Message)
	}
	if resp.Tokens != 7 {
		t.Errorf("tokens = %d, want 7", resp.Tokens)
	}
}

// TestCountTokensUpstreamError verifies an upstream non-200 is converted into a
// *ProviderError carrying the upstream status code.
func TestCountTokensUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `{"error":{"message":"boom","type":"server_error"}}`)
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	resp, perr := p.CountTokens(&schemas.GatewayContext{}, schemas.Key{Value: "test-key"}, &schemas.ChatRequest{Model: "gpt-4"})
	if resp != nil {
		t.Fatal("response should be nil on error")
	}
	if perr == nil {
		t.Fatal("expected provider error, got nil")
	}
	if perr.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", perr.StatusCode)
	}
}
