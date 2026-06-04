package mistral

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestNewSupportsStreaming(t *testing.T) {
	var gotRequest providers.ChatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"id":"chunk-1","object":"chat.completion.chunk","created":1710000000,"model":"test-model","choices":[{"index":0,"delta":{"content":"hello"},"finish_reason":null}]}`,
			"",
			"data: [DONE]",
			"",
		}, "\n")))
	}))
	t.Cleanup(server.Close)

	provider, err := New(server.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	chunks, err := provider.ChatCompletionStream(context.Background(), testKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}

	chunk, ok := <-chunks
	if !ok {
		t.Fatal("stream closed before first chunk")
	}
	if gotRequest.Stream == nil || !*gotRequest.Stream {
		t.Fatalf("request stream = %v, want true", gotRequest.Stream)
	}
	if chunk.ID != "chunk-1" {
		t.Fatalf("chunk ID = %q, want chunk-1", chunk.ID)
	}
	if chunk.Choices[0].Delta.Content == nil || *chunk.Choices[0].Delta.Content != "hello" {
		t.Fatalf("chunk content = %+v", chunk.Choices[0].Delta.Content)
	}
}

func TestNewSupportsListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"mistral-small-latest","object":"model","created":1710000000,"owned_by":"mistral"}]}`))
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
	if len(models) != 1 || models[0].ID != "mistral-small-latest" {
		t.Fatalf("models = %+v", models)
	}
	if models[0].Provider != providers.ProviderMistral {
		t.Fatalf("model provider = %q, want mistral", models[0].Provider)
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
