package cloudflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

func TestNameReturnsCloudflare(t *testing.T) {
	p := New("")
	if p.Name() != providers.ProviderCloudflare {
		t.Fatalf("Name = %q, want cloudflare", p.Name())
	}
}

func TestNewWithDefaultBaseURL(t *testing.T) {
	p := New("")
	if p.baseURL != defaultBaseURL {
		t.Fatalf("baseURL = %q, want %q", p.baseURL, defaultBaseURL)
	}
}

func TestNewWithCustomBaseURL(t *testing.T) {
	p := New("http://custom.example.com/")
	if p.baseURL != "http://custom.example.com" {
		t.Fatalf("baseURL = %q, expected trailing slash stripped", p.baseURL)
	}
}

func TestListModelsUsesAccountScopedEndpoint(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"@cf/meta/llama-3","object":"model","created":1,"owned_by":"cloudflare"}]}`))
	}))
	t.Cleanup(server.Close)

	p := New(server.URL)
	models, err := p.ListModels(context.Background(), providers.Key{
		Value:     "cf-token",
		Provider:  providers.ProviderCloudflare,
		AccountID: "acct-xyz",
	})
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if gotPath != "/accounts/acct-xyz/ai/v1/models" {
		t.Fatalf("path = %q, want account-scoped models endpoint", gotPath)
	}
	if len(models) != 1 || models[0].ID != "@cf/meta/llama-3" {
		t.Fatalf("models = %+v", models)
	}
}

func TestListModelsRequiresAccountID(t *testing.T) {
	p := New("http://127.0.0.1:1")
	_, err := p.ListModels(context.Background(), providers.Key{
		Value:    "cf-token",
		Provider: providers.ProviderCloudflare,
	})
	if err == nil {
		t.Fatal("expected error for missing account id")
	}
	if !strings.Contains(err.Error(), "account id") {
		t.Fatalf("error = %q, want account id context", err.Error())
	}
}

func TestChatCompletionStreamUsesAccountScopedEndpoint(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		var req providers.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: " + streamChunkJSON + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	t.Cleanup(server.Close)

	p := New(server.URL)
	ch, err := p.ChatCompletionStream(context.Background(), providers.Key{
		Value:     "cf-token",
		Provider:  providers.ProviderCloudflare,
		AccountID: "acct-abc",
	}, &providers.ChatRequest{
		Model:    "openai/gpt-4.1",
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}

	var chunks []providers.StreamChunk
	for c := range ch {
		chunks = append(chunks, c)
	}

	if gotPath != "/accounts/acct-abc/ai/v1/chat/completions" {
		t.Fatalf("path = %q, want account-scoped chat endpoint", gotPath)
	}
	if len(chunks) != 1 {
		t.Fatalf("chunks len = %d, want 1", len(chunks))
	}
}

func TestChatCompletionStreamRequiresAccountID(t *testing.T) {
	p := New("http://127.0.0.1:1")
	_, err := p.ChatCompletionStream(context.Background(), providers.Key{
		Value:    "cf-token",
		Provider: providers.ProviderCloudflare,
	}, &providers.ChatRequest{
		Model:    "openai/gpt-4.1",
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err == nil {
		t.Fatal("expected error for missing account id")
	}
	if !strings.Contains(err.Error(), "account id") {
		t.Fatalf("error = %q, want account id context", err.Error())
	}
}

// accountID with special chars is included in the path
func TestChatCompletionAccountIDInPath(t *testing.T) {
	var gotRawPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// r.RequestURI preserves the raw (escaped) path as sent
		gotRawPath = r.RequestURI
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"x","object":"chat.completion","created":1,"model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
	}))
	t.Cleanup(server.Close)

	p := New(server.URL)
	_, err := p.ChatCompletion(context.Background(), providers.Key{
		Value:     "cf-token",
		Provider:  providers.ProviderCloudflare,
		AccountID: "acct-plain",
	}, &providers.ChatRequest{
		Model:    "openai/gpt-4.1",
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if !strings.Contains(gotRawPath, "acct-plain") {
		t.Fatalf("raw path = %q, want account id in path", gotRawPath)
	}
	if !strings.Contains(gotRawPath, "/ai/v1/chat/completions") {
		t.Fatalf("raw path = %q, want ai/v1 endpoint", gotRawPath)
	}
}

const streamChunkJSON = `{"id":"chatcmpl-cf","object":"chat.completion.chunk","created":1,"model":"openai/gpt-4.1","choices":[{"index":0,"delta":{"content":"hello"},"finish_reason":null}]}`
