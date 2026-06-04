package openai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestBuildRequest(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotContentType string
	var gotRequest providers.ChatRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(chatResponseJSON))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL)
	temp := 0.2
	resp, err := provider.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{
		Model:       "gpt-4o-mini",
		Temperature: &temp,
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}

	if gotPath != "/v1/chat/completions" {
		t.Errorf("path = %q", gotPath)
	}
	if gotAuth != "Bearer sk-test" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q", gotContentType)
	}
	if gotRequest.Model != "gpt-4o-mini" {
		t.Errorf("model = %q", gotRequest.Model)
	}
	if gotRequest.Temperature == nil || *gotRequest.Temperature != 0.2 {
		t.Errorf("temperature = %+v", gotRequest.Temperature)
	}
	if resp.ID != "chatcmpl-123" {
		t.Errorf("response id = %q", resp.ID)
	}
}

func TestParseResponse(t *testing.T) {
	server := jsonServer(t, http.StatusOK, chatResponseJSON, nil)
	provider := New(server.URL)

	resp, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}

	if resp.ID != "chatcmpl-123" {
		t.Errorf("ID = %q", resp.ID)
	}
	if resp.Model != "gpt-4o-mini" {
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

func TestParseSSEStream(t *testing.T) {
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
	provider := New(server.URL)

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

func TestParseSSEWithUsage(t *testing.T) {
	server := streamServer(t, strings.Join([]string{
		"data: " + streamChunkContentJSON,
		"",
		"data: " + streamChunkUsageJSON,
		"",
		"data: [DONE]",
		"",
	}, "\n"))
	provider := New(server.URL)

	chunks, err := provider.ChatCompletionStream(context.Background(), testKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}

	got := collectChunks(chunks)
	if len(got) != 2 {
		t.Fatalf("chunks len = %d", len(got))
	}
	if got[1].Usage == nil || got[1].Usage.TotalTokens != 12 {
		t.Errorf("usage = %+v", got[1].Usage)
	}
}

func TestParseSSEStreamReportsMalformedEvent(t *testing.T) {
	server := streamServer(t, strings.Join([]string{
		`data: {"error":"sk-live-secret leaked upstream body"`,
		"",
		"data: " + streamChunkContentJSON,
		"",
	}, "\n"))
	provider := New(server.URL)

	chunks, err := provider.ChatCompletionStream(context.Background(), testKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}

	got := collectChunks(chunks)
	if len(got) != 1 {
		t.Fatalf("chunks len = %d, want 1; chunks=%+v", len(got), got)
	}
	if got[0].Error == nil {
		t.Fatalf("chunk error = nil, want malformed stream error")
	}
	if strings.Contains(got[0].Error.Message, "sk-live-secret") || strings.Contains(got[0].Error.Message, "leaked upstream body") {
		t.Fatalf("chunk error leaked upstream body: %+v", got[0].Error)
	}
	if got[0].Error.Code != "upstream_stream_malformed" {
		t.Fatalf("chunk error code = %q, want upstream_stream_malformed", got[0].Error.Code)
	}
}

func TestChatCompletionStreamReturnsBeforeUpstreamCompletes(t *testing.T) {
	release := make(chan struct{})
	server := liveStreamServer(t, release, streamChunkContentJSON, streamChunkFinalJSON)
	provider := New(server.URL)

	type streamResult struct {
		chunks <-chan providers.StreamChunk
		err    error
	}
	result := make(chan streamResult, 1)
	go func() {
		chunks, err := provider.ChatCompletionStream(context.Background(), testKey(), testChatRequest())
		result <- streamResult{chunks: chunks, err: err}
	}()

	var chunks <-chan providers.StreamChunk
	select {
	case got := <-result:
		if got.err != nil {
			t.Fatalf("ChatCompletionStream: %v", got.err)
		}
		chunks = got.chunks
	case <-time.After(200 * time.Millisecond):
		close(release)
		t.Fatal("ChatCompletionStream blocked until the upstream response completed")
	}

	select {
	case chunk := <-chunks:
		if chunk.Choices[0].Delta.Content == nil || *chunk.Choices[0].Delta.Content != "hello" {
			t.Fatalf("first chunk = %+v", chunk)
		}
	case <-time.After(200 * time.Millisecond):
		close(release)
		t.Fatal("stream did not deliver the first chunk before upstream completion")
	}

	close(release)
	rest := collectChunks(chunks)
	if len(rest) != 1 || rest[0].Choices[0].FinishReason == nil || *rest[0].Choices[0].FinishReason != "stop" {
		t.Fatalf("remaining chunks = %+v", rest)
	}
}

func TestParseError401(t *testing.T) {
	server := jsonServer(t, http.StatusUnauthorized, `{"error":{"message":"invalid api key"}}`, nil)
	provider := New(server.URL)

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
	provider := New(server.URL)

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
	provider := New(server.URL)

	_, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
}

func TestListModels(t *testing.T) {
	server := jsonServer(t, http.StatusOK, modelsResponseJSON, nil)
	provider := New(server.URL)

	models, err := provider.ListModels(context.Background(), testKey())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}

	if len(models) != 2 {
		t.Fatalf("models len = %d", len(models))
	}
	if models[0].ID != "gpt-4o" || models[0].Provider != providers.ProviderOpenAI {
		t.Errorf("model[0] = %+v", models[0])
	}
	if models[1].ID != "gpt-4o-mini" || models[1].OwnedBy != "system" {
		t.Errorf("model[1] = %+v", models[1])
	}
}

func testKey() providers.Key {
	return providers.Key{Value: "sk-test", Provider: providers.ProviderOpenAI, ConnID: "conn-1", AuthType: "api_key"}
}

func testChatRequest() *providers.ChatRequest {
	return &providers.ChatRequest{
		Model: "gpt-4o-mini",
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
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
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

func liveStreamServer(t *testing.T, release <-chan struct{}, firstChunk string, secondChunk string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		var got providers.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode stream request: %v", err)
		}
		if got.Stream == nil || !*got.Stream {
			t.Errorf("stream = %+v", got.Stream)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: " + firstChunk + "\n\n"))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		<-release
		_, _ = w.Write([]byte("data: " + secondChunk + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
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
	"id": "chatcmpl-123",
	"object": "chat.completion",
	"created": 1710000000,
	"model": "gpt-4o-mini",
	"choices": [{
		"index": 0,
		"message": {"role": "assistant", "content": "hello back"},
		"finish_reason": "stop"
	}],
	"usage": {"prompt_tokens": 5, "completion_tokens": 9, "total_tokens": 14}
}`

const streamChunkRoleJSON = `{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1710000000,"model":"gpt-4o-mini","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`

const streamChunkContentJSON = `{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1710000000,"model":"gpt-4o-mini","choices":[{"index":0,"delta":{"content":"hello"},"finish_reason":null}]}`

const streamChunkFinalJSON = `{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1710000000,"model":"gpt-4o-mini","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`

const streamChunkUsageJSON = `{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1710000000,"model":"gpt-4o-mini","choices":[],"usage":{"prompt_tokens":5,"completion_tokens":7,"total_tokens":12}}`

const modelsResponseJSON = `{
	"object": "list",
	"data": [
		{"id": "gpt-4o", "object": "model", "created": 1710000001, "owned_by": "openai"},
		{"id": "gpt-4o-mini", "object": "model", "created": 1710000002, "owned_by": "system"}
	]
}`
