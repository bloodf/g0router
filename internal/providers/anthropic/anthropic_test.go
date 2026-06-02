package anthropic

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
	provider := New("")
	if provider.client == nil {
		t.Fatal("client is nil")
	}

	var _ *fasthttp.Client = provider.client
}

func TestBuildMessagesRequest(t *testing.T) {
	var gotPath string
	var gotAPIKey string
	var gotVersion string
	var gotContentType string
	var gotRequest anthropicRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAPIKey = r.Header.Get("x-api-key")
		gotVersion = r.Header.Get("anthropic-version")
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(messageResponseJSON))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL)
	temp := 0.2
	maxTokens := 256
	system := "You are concise."
	resp, err := provider.ChatCompletion(context.Background(), testKey("api_key"), &providers.ChatRequest{
		Model:       "claude-sonnet-4-20250514",
		System:      system,
		Temperature: &temp,
		MaxTokens:   &maxTokens,
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}

	if gotPath != "/v1/messages" {
		t.Errorf("path = %q", gotPath)
	}
	if gotAPIKey != "sk-ant-test" {
		t.Errorf("x-api-key = %q", gotAPIKey)
	}
	if gotVersion != anthropicVersion {
		t.Errorf("anthropic-version = %q", gotVersion)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q", gotContentType)
	}
	if gotRequest.Model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %q", gotRequest.Model)
	}
	if gotRequest.System != "You are concise." {
		t.Errorf("system = %#v", gotRequest.System)
	}
	if gotRequest.MaxTokens != 256 {
		t.Errorf("max_tokens = %d", gotRequest.MaxTokens)
	}
	if gotRequest.Temperature == nil || *gotRequest.Temperature != 0.2 {
		t.Errorf("temperature = %+v", gotRequest.Temperature)
	}
	if len(gotRequest.Messages) != 1 || gotRequest.Messages[0].Role != "user" {
		t.Fatalf("messages = %+v", gotRequest.Messages)
	}
	if len(gotRequest.Messages[0].Content) != 1 || gotRequest.Messages[0].Content[0].Text != "hello" {
		t.Fatalf("content = %+v", gotRequest.Messages[0].Content)
	}
	if resp.ID != "msg_123" {
		t.Errorf("response id = %q", resp.ID)
	}
}

func TestBuildOAuthRequest(t *testing.T) {
	var gotAuth string
	var gotAPIKey string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAPIKey = r.Header.Get("x-api-key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(messageResponseJSON))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL)
	_, err := provider.ChatCompletion(context.Background(), testKey("oauth"), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}

	if gotAuth != "Bearer sk-ant-test" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotAPIKey != "" {
		t.Errorf("x-api-key = %q", gotAPIKey)
	}
}

func TestParseMessageResponse(t *testing.T) {
	server := jsonServer(t, http.StatusOK, messageResponseJSON, nil)
	provider := New(server.URL)

	resp, err := provider.ChatCompletion(context.Background(), testKey("api_key"), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}

	if resp.ID != "msg_123" {
		t.Errorf("ID = %q", resp.ID)
	}
	if resp.Object != "chat.completion" {
		t.Errorf("Object = %q", resp.Object)
	}
	if resp.Model != "claude-sonnet-4-20250514" {
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
	if resp.Choices[0].FinishReason == nil || *resp.Choices[0].FinishReason != "stop" {
		t.Errorf("finish reason = %+v", resp.Choices[0].FinishReason)
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 14 {
		t.Errorf("usage = %+v", resp.Usage)
	}
}

func TestParseSSEStream(t *testing.T) {
	server := streamServer(t, strings.Join([]string{
		"event: message_start",
		"data: " + streamMessageStartJSON,
		"",
		"event: content_block_delta",
		"data: " + streamContentDeltaJSON,
		"",
		"event: message_delta",
		"data: " + streamMessageDeltaJSON,
		"",
		"event: message_stop",
		"data: {}",
		"",
	}, "\n"))
	provider := New(server.URL)

	chunks, err := provider.ChatCompletionStream(context.Background(), testKey("api_key"), testChatRequest())
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
	if got[2].Usage == nil || got[2].Usage.TotalTokens != 12 {
		t.Errorf("usage = %+v", got[2].Usage)
	}
}

func TestParseError401(t *testing.T) {
	server := jsonServer(t, http.StatusUnauthorized, `{"error":{"message":"invalid api key"}}`, nil)
	provider := New(server.URL)

	_, err := provider.ChatCompletion(context.Background(), testKey("api_key"), testChatRequest())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
	if !strings.Contains(err.Error(), "invalid api key") {
		t.Fatalf("error = %q", err)
	}
}

func TestParseError429(t *testing.T) {
	server := jsonServer(t, http.StatusTooManyRequests, `{"error":{"message":"slow down"}}`, map[string]string{"Retry-After": "7"})
	provider := New(server.URL)

	_, err := provider.ChatCompletion(context.Background(), testKey("api_key"), testChatRequest())
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
	provider := New(server.URL)

	_, err := provider.ChatCompletion(context.Background(), testKey("api_key"), testChatRequest())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
}

func TestListModels(t *testing.T) {
	server := jsonServer(t, http.StatusOK, modelsResponseJSON, nil)
	provider := New(server.URL)

	models, err := provider.ListModels(context.Background(), testKey("api_key"))
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}

	if len(models) != 2 {
		t.Fatalf("models len = %d", len(models))
	}
	if models[0].ID != "claude-sonnet-4-20250514" || models[0].Provider != providers.ProviderAnthropic {
		t.Errorf("model[0] = %+v", models[0])
	}
	if models[1].ID != "claude-3-5-haiku-20241022" || models[1].OwnedBy != "anthropic" {
		t.Errorf("model[1] = %+v", models[1])
	}
}

func testKey(authType string) providers.Key {
	return providers.Key{Value: "sk-ant-test", Provider: providers.ProviderAnthropic, ConnID: "conn-1", AuthType: authType}
}

func testChatRequest() *providers.ChatRequest {
	maxTokens := 256
	return &providers.ChatRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: &maxTokens,
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
		if r.URL.Path != "/v1/messages" {
			t.Errorf("path = %q", r.URL.Path)
		}
		var got anthropicRequest
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

const messageResponseJSON = `{
	"id": "msg_123",
	"type": "message",
	"role": "assistant",
	"model": "claude-sonnet-4-20250514",
	"content": [{"type": "text", "text": "hello back"}],
	"stop_reason": "end_turn",
	"usage": {"input_tokens": 5, "output_tokens": 9}
}`

const streamMessageStartJSON = `{"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-sonnet-4-20250514","content":[],"usage":{"input_tokens":5,"output_tokens":0}}}`

const streamContentDeltaJSON = `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello"}}`

const streamMessageDeltaJSON = `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":7}}`

const modelsResponseJSON = `{
	"data": [
		{"id": "claude-sonnet-4-20250514", "type": "model", "created_at": "2025-05-14T00:00:00Z", "display_name": "Claude Sonnet 4"},
		{"id": "claude-3-5-haiku-20241022", "type": "model", "display_name": "Claude 3.5 Haiku"}
	]
}`
