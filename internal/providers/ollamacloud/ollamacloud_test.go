package ollamacloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

func TestChatCompletionUsesNativeOllamaCloudChat(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotRequest chatRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model":"gpt-oss:120b",
			"message":{"role":"assistant","content":"hello back"},
			"done":true,
			"done_reason":"stop",
			"prompt_eval_count":7,
			"eval_count":3
		}`))
	}))
	t.Cleanup(server.Close)

	provider, err := New(server.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	resp, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if gotPath != "/api/chat" {
		t.Fatalf("path = %q, want /api/chat", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Fatalf("Authorization = %q, want bearer key", gotAuth)
	}
	if gotRequest.Model != "gpt-oss:120b" || gotRequest.Stream {
		t.Fatalf("request = %+v, want non-streaming gpt-oss:120b", gotRequest)
	}
	if len(gotRequest.Messages) != 1 || gotRequest.Messages[0].Role != "user" || gotRequest.Messages[0].Content != "hello" {
		t.Fatalf("messages = %+v, want mapped user message", gotRequest.Messages)
	}
	if resp.Provider != providers.ProviderOllamaCloud || resp.Model != "gpt-oss:120b" {
		t.Fatalf("response = %+v, want ollama-cloud gpt-oss", resp)
	}
	if len(resp.Choices) != 1 || resp.Choices[0].Message.Content != "hello back" || resp.Choices[0].FinishReason == nil || *resp.Choices[0].FinishReason != "stop" {
		t.Fatalf("choices = %+v, want mapped assistant stop", resp.Choices)
	}
	if resp.Usage == nil || resp.Usage.PromptTokens != 7 || resp.Usage.CompletionTokens != 3 || resp.Usage.TotalTokens != 10 {
		t.Fatalf("usage = %+v, want native token counts", resp.Usage)
	}
}

func TestListModelsUsesNativeTagsEndpoint(t *testing.T) {
	var gotPath string
	var gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"gpt-oss:120b"},{"model":"qwen3:32b","name":"Qwen 3 32B"}]}`))
	}))
	t.Cleanup(server.Close)

	provider, err := New(server.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	models, err := provider.ListModels(context.Background(), testKey())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if gotPath != "/api/tags" {
		t.Fatalf("path = %q, want /api/tags", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Fatalf("Authorization = %q, want bearer key", gotAuth)
	}
	if len(models) != 2 || models[0].ID != "gpt-oss:120b" || models[1].ID != "qwen3:32b" {
		t.Fatalf("models = %+v, want native model ids", models)
	}
	for _, model := range models {
		if model.Provider != providers.ProviderOllamaCloud || model.Object != "model" || model.OwnedBy != providers.ProviderOllamaCloud.String() {
			t.Fatalf("model = %+v, want ollama-cloud metadata", model)
		}
	}
}

func TestNewDefaultUsesOllamaCloudProvider(t *testing.T) {
	provider, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault: %v", err)
	}
	if provider.Name() != providers.ProviderOllamaCloud {
		t.Fatalf("Name = %q, want ollama-cloud", provider.Name())
	}
}

func testKey() providers.Key {
	return providers.Key{Value: "test-key", Provider: providers.ProviderOllamaCloud, ConnID: "conn-1", AuthType: "api_key"}
}

func testChatRequest() *providers.ChatRequest {
	return &providers.ChatRequest{
		Model: "gpt-oss:120b",
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	}
}
