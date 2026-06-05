package xiaomi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

func TestNewDefaultUsesXiaomiProvider(t *testing.T) {
	p := NewDefault()
	if p.Name() != providers.ProviderXiaomi {
		t.Fatalf("Name = %q, want xiaomi", p.Name())
	}
	if p.standard == nil {
		t.Fatal("standard provider is nil")
	}
	if p.tokenPlan == nil {
		t.Fatal("tokenPlan provider is nil")
	}
}

func TestNewCustomURLs(t *testing.T) {
	p := New("http://std.example.com", "http://tp.example.com")
	if p.standard == nil || p.tokenPlan == nil {
		t.Fatal("providers should not be nil with custom URLs")
	}
}

func TestNewEmptyURLsFallsBackToDefaults(t *testing.T) {
	p := New("", "")
	if p.Name() != providers.ProviderXiaomi {
		t.Fatalf("Name = %q, want xiaomi", p.Name())
	}
}

func TestChatCompletionStreamStandardKey(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: " + streamChunkJSON + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	t.Cleanup(server.Close)

	p := New(server.URL, "http://tp.invalid/anthropic")
	ch, err := p.ChatCompletionStream(context.Background(), providers.Key{
		Provider: providers.ProviderXiaomi,
		Value:    "sk-standard",
		AuthType: "api_key",
	}, &providers.ChatRequest{
		Model:    "claude-sonnet-4",
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}

	var chunks []providers.StreamChunk
	for c := range ch {
		chunks = append(chunks, c)
	}
	if gotPath != "/v1/messages" {
		t.Fatalf("path = %q, want /v1/messages", gotPath)
	}
	if len(chunks) == 0 {
		t.Fatal("expected at least one stream chunk")
	}
}

func TestChatCompletionStreamTokenPlanKey(t *testing.T) {
	standard := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("standard should not be called for tp- key")
	}))
	t.Cleanup(standard.Close)

	var gotPath string
	tokenPlan := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: " + streamChunkJSON + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	t.Cleanup(tokenPlan.Close)

	p := New(standard.URL, tokenPlan.URL)
	ch, err := p.ChatCompletionStream(context.Background(), providers.Key{
		Provider: providers.ProviderXiaomi,
		Value:    "tp-key",
		AuthType: "api_key",
	}, &providers.ChatRequest{
		Model:    "claude-sonnet-4",
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}

	for range ch {
	}
	if gotPath != "/v1/messages" {
		t.Fatalf("path = %q, want /v1/messages (token plan)", gotPath)
	}
}

func TestListModelsStandardKey(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		// Anthropic models endpoint response
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	t.Cleanup(server.Close)

	p := New(server.URL, "http://tp.invalid/anthropic")
	_, err := p.ListModels(context.Background(), providers.Key{
		Provider: providers.ProviderXiaomi,
		Value:    "sk-standard",
		AuthType: "api_key",
	})
	// Error is acceptable (model list format mismatch), but path must be hit
	_ = err
	if gotPath != "/v1/models" {
		t.Fatalf("path = %q, want /v1/models", gotPath)
	}
}

func TestListModelsTokenPlanKey(t *testing.T) {
	standard := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("standard should not be called for tp- key")
	}))
	t.Cleanup(standard.Close)

	var gotPath string
	tokenPlan := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	t.Cleanup(tokenPlan.Close)

	p := New(standard.URL, tokenPlan.URL)
	_, err := p.ListModels(context.Background(), providers.Key{
		Provider: providers.ProviderXiaomi,
		Value:    "tp-key",
		AuthType: "api_key",
	})
	_ = err
	if gotPath != "/v1/models" {
		t.Fatalf("path = %q, want /v1/models (token plan)", gotPath)
	}
}

// streamChunkJSON is a minimal Anthropic SSE event that the anthropic provider can parse
const streamChunkJSON = `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hi"}}`
