package azure

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/valyala/fasthttp"
)

func TestNewUsesFastHTTPClient(t *testing.T) {
	provider := New("", "2024-02-15-preview")
	if provider.client == nil {
		t.Fatal("client is nil")
	}

	var _ *fasthttp.Client = provider.client
}

func TestChatCompletionBuildsAzureRequest(t *testing.T) {
	var gotPath string
	var gotAPIKey string
	var gotAPIVersion string
	var gotContentType string
	var gotRequest providers.ChatRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAPIKey = r.Header.Get("api-key")
		gotAPIVersion = r.URL.Query().Get("api-version")
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(chatResponseJSON))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL, "2024-02-15-preview")
	temp := 0.2
	resp, err := provider.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{
		Model:       "gpt-4o-prod",
		Temperature: &temp,
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}

	if gotPath != "/openai/deployments/gpt-4o-prod/chat/completions" {
		t.Errorf("path = %q", gotPath)
	}
	if gotAPIKey != "azure-key" {
		t.Errorf("api-key = %q", gotAPIKey)
	}
	if gotAPIVersion != "2024-02-15-preview" {
		t.Errorf("api-version = %q", gotAPIVersion)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q", gotContentType)
	}
	if gotRequest.Model != "gpt-4o-prod" {
		t.Errorf("model = %q", gotRequest.Model)
	}
	if gotRequest.Temperature == nil || *gotRequest.Temperature != 0.2 {
		t.Errorf("temperature = %+v", gotRequest.Temperature)
	}
	if resp.ID != "chatcmpl-azure-123" {
		t.Errorf("response id = %q", resp.ID)
	}
}

func TestChatCompletionParsesResponse(t *testing.T) {
	server := jsonServer(t, http.StatusOK, chatResponseJSON, nil)
	provider := New(server.URL, "2024-02-15-preview")

	resp, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}

	if resp.ID != "chatcmpl-azure-123" {
		t.Errorf("ID = %q", resp.ID)
	}
	if resp.Model != "gpt-4o-prod" {
		t.Errorf("Model = %q", resp.Model)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("choices len = %d", len(resp.Choices))
	}
	if resp.Choices[0].Message.Role != "assistant" {
		t.Errorf("role = %q", resp.Choices[0].Message.Role)
	}
	if resp.Choices[0].Message.Content != "hello back" {
		t.Errorf("content = %#v", resp.Choices[0].Message.Content)
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 14 {
		t.Errorf("usage = %+v", resp.Usage)
	}
}

func TestChatCompletionStreamParsesSSE(t *testing.T) {
	server := streamServer(t, strings.Join([]string{
		"data: " + streamChunkRoleJSON,
		"",
		"data: " + streamChunkContentJSON,
		"",
		"data: " + streamChunkFinalJSON,
		"",
		"data: [DONE]",
		"",
	}, "\n"))
	provider := New(server.URL, "2024-02-15-preview")

	chunks, err := provider.ChatCompletionStream(context.Background(), testKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}

	got := collectChunks(chunks)
	if len(got) != 3 {
		t.Fatalf("chunks len = %d", len(got))
	}
	if got[0].Choices[0].Delta.Role == nil || *got[0].Choices[0].Delta.Role != "assistant" {
		t.Errorf("first role = %+v", got[0].Choices[0].Delta.Role)
	}
	if got[1].Choices[0].Delta.Content == nil || *got[1].Choices[0].Delta.Content != "hello" {
		t.Errorf("second content = %+v", got[1].Choices[0].Delta.Content)
	}
	if got[2].Choices[0].FinishReason == nil || *got[2].Choices[0].FinishReason != "stop" {
		t.Errorf("finish reason = %+v", got[2].Choices[0].FinishReason)
	}
}

func TestListModelsParsesDeployments(t *testing.T) {
	var gotPath string
	var gotAPIVersion string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAPIVersion = r.URL.Query().Get("api-version")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(deploymentsResponseJSON))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL, "2024-02-15-preview")
	models, err := provider.ListModels(context.Background(), testKey())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}

	if gotPath != "/openai/deployments" {
		t.Errorf("path = %q", gotPath)
	}
	if gotAPIVersion != "2024-02-15-preview" {
		t.Errorf("api-version = %q", gotAPIVersion)
	}
	if len(models) != 2 {
		t.Fatalf("models len = %d", len(models))
	}
	if models[0].ID != "gpt-4o-prod" || models[0].Provider != providers.ProviderAzure {
		t.Errorf("model[0] = %+v", models[0])
	}
	if models[1].ID != "embedding-prod" || models[1].OwnedBy != "azure" {
		t.Errorf("model[1] = %+v", models[1])
	}
}

func TestParseError401(t *testing.T) {
	server := jsonServer(t, http.StatusUnauthorized, `{"error":{"message":"invalid api key"}}`, nil)
	provider := New(server.URL, "2024-02-15-preview")

	_, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
	if !strings.Contains(err.Error(), "invalid api key") {
		t.Fatalf("error = %q", err)
	}
}

func TestParseError429(t *testing.T) {
	server := jsonServer(t, http.StatusTooManyRequests, `{"error":{"message":"slow down"}}`, map[string]string{"Retry-After": "7"})
	provider := New(server.URL, "2024-02-15-preview")

	_, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if !errors.Is(err, ErrRateLimit) {
		t.Fatalf("expected ErrRateLimit, got %v", err)
	}
	var rateLimitErr *RateLimitError
	if !errors.As(err, &rateLimitErr) {
		t.Fatalf("expected RateLimitError, got %T", err)
	}
	if rateLimitErr.RetryAfter != 7 {
		t.Errorf("RetryAfter = %d", rateLimitErr.RetryAfter)
	}
}

func TestParseError500(t *testing.T) {
	server := jsonServer(t, http.StatusInternalServerError, `{"error":{"message":"upstream failed"}}`, nil)
	provider := New(server.URL, "2024-02-15-preview")

	_, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
}

func testKey() providers.Key {
	return providers.Key{Value: "azure-key", Provider: providers.ProviderAzure, ConnID: "conn-1", AuthType: "api_key"}
}

func testChatRequest() *providers.ChatRequest {
	return &providers.ChatRequest{
		Model: "gpt-4o-prod",
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	}
}

func jsonServer(t *testing.T, status int, body string, headers map[string]string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for key, value := range headers {
			w.Header().Set(key, value)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func streamServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/openai/deployments/gpt-4o-prod/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("api-version") != "2024-02-15-preview" {
			t.Errorf("api-version = %q", r.URL.Query().Get("api-version"))
		}
		if r.Header.Get("api-key") != "azure-key" {
			t.Errorf("api-key = %q", r.Header.Get("api-key"))
		}
		var got providers.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode stream request: %v", err)
		}
		if got.Stream == nil || !*got.Stream {
			t.Errorf("stream = %+v", got.Stream)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func collectChunks(chunks <-chan providers.StreamChunk) []providers.StreamChunk {
	var got []providers.StreamChunk
	for chunk := range chunks {
		got = append(got, chunk)
	}
	return got
}

const chatResponseJSON = `{
	"id": "chatcmpl-azure-123",
	"object": "chat.completion",
	"created": 1710000000,
	"model": "gpt-4o-prod",
	"choices": [{
		"index": 0,
		"message": {"role": "assistant", "content": "hello back"},
		"finish_reason": "stop"
	}],
	"usage": {"prompt_tokens": 5, "completion_tokens": 9, "total_tokens": 14}
}`

const streamChunkRoleJSON = `{"id":"chatcmpl-azure-123","object":"chat.completion.chunk","created":1710000000,"model":"gpt-4o-prod","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`

const streamChunkContentJSON = `{"id":"chatcmpl-azure-123","object":"chat.completion.chunk","created":1710000000,"model":"gpt-4o-prod","choices":[{"index":0,"delta":{"content":"hello"},"finish_reason":null}]}`

const streamChunkFinalJSON = `{"id":"chatcmpl-azure-123","object":"chat.completion.chunk","created":1710000000,"model":"gpt-4o-prod","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`

const deploymentsResponseJSON = `{
	"data": [
		{"id": "gpt-4o-prod", "object": "deployment", "created_at": 1710000001, "model": "gpt-4o"},
		{"id": "embedding-prod", "object": "deployment", "created_at": 1710000002, "model": "text-embedding-3-large"}
	]
}`
