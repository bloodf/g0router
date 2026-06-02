package mistral

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

func TestNewBuildsMistralOpenAICompatibleProvider(t *testing.T) {
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
		_, _ = w.Write([]byte(chatResponseJSON))
	}))
	t.Cleanup(server.Close)

	provider, err := New(server.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if provider.Name() != providers.ProviderMistral {
		t.Fatalf("Name = %q", provider.Name())
	}

	resp, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if gotPath != "/v1/chat/completions" {
		t.Errorf("path = %q", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotRequest.Model != "test-model" {
		t.Errorf("model = %q", gotRequest.Model)
	}
	if resp.ID != "chatcmpl-test" {
		t.Errorf("response id = %q", resp.ID)
	}
}

func TestNewDefault(t *testing.T) {
	provider, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault: %v", err)
	}
	if provider.Name() != providers.ProviderMistral {
		t.Fatalf("Name = %q", provider.Name())
	}
}

func testKey() providers.Key {
	return providers.Key{Value: "test-key", Provider: providers.ProviderMistral, ConnID: "conn-1", AuthType: "api_key"}
}

func testChatRequest() *providers.ChatRequest {
	return &providers.ChatRequest{
		Model: "test-model",
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	}
}

const chatResponseJSON = `{
	"id": "chatcmpl-test",
	"object": "chat.completion",
	"created": 1710000000,
	"model": "test-model",
	"choices": [{"index": 0, "message": {"role": "assistant", "content": "hello back"}, "finish_reason": "stop"}]
}`
