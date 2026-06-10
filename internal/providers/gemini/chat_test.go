package gemini

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

func TestChatCompletionStreamUsesStreamingEndpoint(t *testing.T) {
	var capturedURI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURI = r.URL.Path
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"Hello\"}]}}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	ch, perr := p.ChatCompletionStream(&schemas.GatewayContext{}, nil, schemas.Key{Value: "test-key"}, &schemas.ChatRequest{
		Model: "gemini-1.5-pro",
		Messages: []schemas.Message{
			{Role: "user", Content: "Hi"},
		},
	})
	if perr != nil {
		t.Fatalf("ChatCompletionStream error: %v", perr.Message)
	}

	for range ch {
	}

	if !strings.Contains(capturedURI, ":streamGenerateContent") {
		t.Errorf("URI = %q, expected to contain ':streamGenerateContent'", capturedURI)
	}
}

func TestChatCompletionSanitizesModelInURI(t *testing.T) {
	var capturedURI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURI = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"candidates":[{"content":{"parts":[{"text":"Hello"}]},"finishReason":"STOP"}]}`)
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	_, perr := p.ChatCompletion(&schemas.GatewayContext{}, schemas.Key{Value: "test-key"}, &schemas.ChatRequest{
		Model: "gemini/gemini-1.5-pro",
		Messages: []schemas.Message{
			{Role: "user", Content: "Hi"},
		},
	})
	if perr != nil {
		t.Fatalf("ChatCompletion error: %v", perr.Message)
	}

	if !strings.Contains(capturedURI, "/models/gemini-1.5-pro:") {
		t.Errorf("URI = %q, expected to contain '/models/gemini-1.5-pro:'", capturedURI)
	}
	if strings.Contains(capturedURI, "gemini/gemini-1.5-pro") {
		t.Errorf("URI = %q, should not contain prefixed model name", capturedURI)
	}
}
