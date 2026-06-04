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

func TestChatCompletionUsesAccountScopedCloudflareOpenAIEndpoint(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotRequest providers.ChatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl-cf","object":"chat.completion","created":1,"model":"openai/gpt-4.1","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL)
	resp, err := provider.ChatCompletion(context.Background(), providers.Key{
		Value:     "cf-token",
		Provider:  providers.ProviderCloudflare,
		AccountID: "account-123",
	}, &providers.ChatRequest{
		Model: "openai/gpt-4.1",
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if gotPath != "/accounts/account-123/ai/v1/chat/completions" {
		t.Fatalf("path = %q, want account-scoped OpenAI-compatible endpoint", gotPath)
	}
	if gotAuth != "Bearer cf-token" {
		t.Fatalf("Authorization = %q, want bearer token", gotAuth)
	}
	if gotRequest.Model != "openai/gpt-4.1" {
		t.Fatalf("model = %q, want openai/gpt-4.1", gotRequest.Model)
	}
	if resp.ID != "chatcmpl-cf" {
		t.Fatalf("response id = %q", resp.ID)
	}
}

func TestChatCompletionRequiresAccountID(t *testing.T) {
	provider := New("http://127.0.0.1:1")

	_, err := provider.ChatCompletion(context.Background(), providers.Key{
		Value:    "cf-token",
		Provider: providers.ProviderCloudflare,
	}, &providers.ChatRequest{Model: "openai/gpt-4.1"})
	if err == nil {
		t.Fatal("ChatCompletion error is nil")
	}
	if !strings.Contains(err.Error(), "account id") {
		t.Fatalf("error = %q, want account id context", err.Error())
	}
}
