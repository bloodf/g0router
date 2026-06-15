package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

// TestTextCompletionSuccess verifies the non-streaming legacy completions
// transport maps an upstream 200 body into a TextCompletionResponse with the
// choices[].text content populated.
func TestTextCompletionSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/completions" {
			t.Errorf("path = %q, want /v1/completions", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"cmpl-1","object":"text_completion","model":"gpt-3.5-turbo-instruct","choices":[{"text":"hello world","index":0,"finish_reason":"stop"}]}`)
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	resp, perr := p.TextCompletion(&schemas.GatewayContext{}, schemas.Key{Value: "test-key"}, &schemas.TextCompletionRequest{Model: "gpt-3.5-turbo-instruct", Prompt: "hi"})
	if perr != nil {
		t.Fatalf("TextCompletion error: %v", perr.Message)
	}
	if resp == nil {
		t.Fatal("response is nil")
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("choices = %d, want 1", len(resp.Choices))
	}
	if resp.Choices[0].Text != "hello world" {
		t.Errorf("text = %q, want %q", resp.Choices[0].Text, "hello world")
	}
	if resp.Object != "text_completion" {
		t.Errorf("object = %q, want text_completion", resp.Object)
	}
}

// TestTextCompletionUpstreamError verifies an upstream non-200 is converted
// into a *ProviderError carrying the upstream status code.
func TestTextCompletionUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `{"error":{"message":"boom","type":"server_error"}}`)
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	resp, perr := p.TextCompletion(&schemas.GatewayContext{}, schemas.Key{Value: "test-key"}, &schemas.TextCompletionRequest{Model: "gpt-3.5-turbo-instruct", Prompt: "hi"})
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

// TestTextCompletionStreamYieldsChunks verifies the streaming transport drains
// SSE chunks and terminates cleanly on [DONE].
func TestTextCompletionStreamYieldsChunks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"id\":\"cmpl-1\",\"object\":\"text_completion\",\"choices\":[{\"index\":0,\"text\":\"a\"}]}\n\n")
		io.WriteString(w, "data: {\"id\":\"cmpl-1\",\"object\":\"text_completion\",\"choices\":[{\"index\":0,\"text\":\"b\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	ch, perr := p.TextCompletionStream(&schemas.GatewayContext{}, nil, schemas.Key{Value: "test-key"}, &schemas.TextCompletionRequest{Model: "gpt-3.5-turbo-instruct", Prompt: "hi"})
	if perr != nil {
		t.Fatalf("TextCompletionStream error: %v", perr.Message)
	}

	var content, errChunks int
	for chunk := range ch {
		if chunk.Error != nil {
			errChunks++
			continue
		}
		content++
	}
	if content != 2 {
		t.Errorf("content chunks = %d, want 2", content)
	}
	if errChunks != 0 {
		t.Errorf("error chunks = %d, want 0", errChunks)
	}
}

// TestTextCompletionStreamAbortsOnMalformedChunk verifies AUD-045 parity: a
// malformed chunk aborts the stream with one in-band error chunk; the valid
// chunk after it must never arrive.
func TestTextCompletionStreamAbortsOnMalformedChunk(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"id\":\"cmpl-1\",\"choices\":[{\"index\":0,\"text\":\"a\"}]}\n\n")
		io.WriteString(w, "data: not-json{\n\n")
		io.WriteString(w, "data: {\"id\":\"cmpl-2\",\"choices\":[{\"index\":0,\"text\":\"b\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	ch, perr := p.TextCompletionStream(&schemas.GatewayContext{}, nil, schemas.Key{Value: "test-key"}, &schemas.TextCompletionRequest{Model: "gpt-3.5-turbo-instruct", Prompt: "hi"})
	if perr != nil {
		t.Fatalf("TextCompletionStream error: %v", perr.Message)
	}

	var content, errChunks int
	for chunk := range ch {
		if chunk.Error != nil {
			errChunks++
			continue
		}
		content++
	}
	if content != 1 {
		t.Errorf("content chunks = %d, want 1 (stream must abort at malformed chunk)", content)
	}
	if errChunks != 1 {
		t.Errorf("error chunks = %d, want 1 (abort must be distinguishable from clean EOF)", errChunks)
	}
}
