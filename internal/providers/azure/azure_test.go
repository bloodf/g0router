package azure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestChatCompletionStreamReportsMalformedEvent(t *testing.T) {
	server := streamServer(t, strings.Join([]string{
		`data: {"error":"sk-live-secret leaked upstream body"`,
		"",
		"data: " + streamChunkContentJSON,
		"",
	}, "\n"))
	provider := New(server.URL, "2024-02-15-preview")

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
	provider := New(server.URL, "2024-02-15-preview")

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

func liveStreamServer(t *testing.T, release <-chan struct{}, firstChunk string, secondChunk string) *httptest.Server {
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

// ---- additional coverage tests ----

func TestNameReturnsAzure(t *testing.T) {
	p := New("", "")
	if p.Name() != providers.ProviderAzure {
		t.Fatalf("Name = %q, want azure", p.Name())
	}
}

func TestNewDefaultsAPIVersion(t *testing.T) {
	p := New("", "")
	if p.apiVersion != defaultAPIVersion {
		t.Fatalf("apiVersion = %q, want default", p.apiVersion)
	}
}

func TestRateLimitErrorWithRetryAfter(t *testing.T) {
	err := &RateLimitError{Message: "slow down", RetryAfter: 5}
	if !errors.Is(err, ErrRateLimit) {
		t.Fatal("expected ErrRateLimit sentinel")
	}
	if !strings.Contains(err.Error(), "retry after 5s") {
		t.Fatalf("error = %q, want retry after in message", err.Error())
	}
}

func TestRateLimitErrorNoRetryAfter(t *testing.T) {
	err := &RateLimitError{Message: "slow down"}
	if !strings.Contains(err.Error(), "slow down") {
		t.Fatalf("error = %q, want message", err.Error())
	}
}

func TestRateLimitErrorEmpty(t *testing.T) {
	err := &RateLimitError{}
	if err.Error() != ErrRateLimit.Error() {
		t.Fatalf("blank error = %q, want %q", err.Error(), ErrRateLimit.Error())
	}
}

func TestParseError403(t *testing.T) {
	server := jsonServer(t, http.StatusForbidden, `{"error":{"message":"forbidden"}}`, nil)
	provider := New(server.URL, "2024-02-15-preview")

	_, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

func TestParseError400(t *testing.T) {
	server := jsonServer(t, http.StatusBadRequest, `{"error":{"message":"bad request"}}`, nil)
	provider := New(server.URL, "2024-02-15-preview")

	_, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "bad request") {
		t.Fatalf("error = %q", err)
	}
}

func TestParseErrorEmptyBody(t *testing.T) {
	server := jsonServer(t, http.StatusInternalServerError, ``, nil)
	provider := New(server.URL, "2024-02-15-preview")

	_, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Fatalf("error = %q, want empty response", err.Error())
	}
}

func TestParseErrorNonJSONBody(t *testing.T) {
	server := jsonServer(t, http.StatusInternalServerError, `plain text`, nil)
	provider := New(server.URL, "2024-02-15-preview")

	_, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
	if !strings.Contains(err.Error(), "plain text") {
		t.Fatalf("error = %q, want plain text", err.Error())
	}
}

func TestListModels401(t *testing.T) {
	server := jsonServer(t, http.StatusUnauthorized, `{"error":{"message":"bad key"}}`, nil)
	provider := New(server.URL, "2024-02-15-preview")

	_, err := provider.ListModels(context.Background(), testKey())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

func TestListModels500(t *testing.T) {
	server := jsonServer(t, http.StatusInternalServerError, `{"error":{"message":"server error"}}`, nil)
	provider := New(server.URL, "2024-02-15-preview")

	_, err := provider.ListModels(context.Background(), testKey())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
}

func TestStreamHTTPError(t *testing.T) {
	server := jsonServer(t, http.StatusUnauthorized, `{"error":{"message":"bad stream key"}}`, nil)
	provider := New(server.URL, "2024-02-15-preview")

	_, err := provider.ChatCompletionStream(context.Background(), testKey(), testChatRequest())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

func TestStreamHTTPErrorWithRetryAfter(t *testing.T) {
	server := jsonServer(t, http.StatusTooManyRequests, `{"error":{"message":"rate limited"}}`, map[string]string{"Retry-After": "3"})
	provider := New(server.URL, "2024-02-15-preview")

	_, err := provider.ChatCompletionStream(context.Background(), testKey(), testChatRequest())
	if !errors.Is(err, ErrRateLimit) {
		t.Fatalf("expected ErrRateLimit, got %v", err)
	}
	var rl *RateLimitError
	if !errors.As(err, &rl) || rl.RetryAfter != 3 {
		t.Fatalf("retry after = %+v", rl)
	}
}

func TestChatCompletionCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	provider := New("http://127.0.0.1:1", "2024-02-15-preview")
	_, err := provider.ChatCompletion(ctx, testKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestStreamSSEDone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL, "2024-02-15-preview")
	chunks, err := provider.ChatCompletionStream(context.Background(), testKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}
	got := collectChunks(chunks)
	if len(got) != 0 {
		t.Fatalf("chunks = %+v, want none after DONE", got)
	}
}

func TestRetryAfterSecondsInvalidValue(t *testing.T) {
	result := retryAfterSeconds("not-a-number")
	if result != 0 {
		t.Fatalf("retryAfterSeconds = %d, want 0", result)
	}
}

func TestRetryAfterSecondsEmpty(t *testing.T) {
	result := retryAfterSeconds("")
	if result != 0 {
		t.Fatalf("retryAfterSeconds = %d, want 0", result)
	}
}

func TestDoWithDeadline(t *testing.T) {
	server := jsonServer(t, http.StatusOK, chatResponseJSON, nil)
	provider := New(server.URL, "2024-02-15-preview")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := provider.ChatCompletion(ctx, testKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletion with deadline: %v", err)
	}
}

func TestDoAlreadyExpiredDeadline(t *testing.T) {
	provider := New("http://127.0.0.1:1", "2024-02-15-preview")
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	<-ctx.Done()
	_, err := provider.ChatCompletion(ctx, testKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected error for expired deadline")
	}
}

func TestStreamSSETrailingData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// No trailing blank line — pending data processed at EOF
		_, _ = w.Write([]byte("data: " + streamChunkContentJSON))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL, "2024-02-15-preview")
	chunks, err := provider.ChatCompletionStream(context.Background(), testKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}
	got := collectChunks(chunks)
	if len(got) == 0 {
		t.Fatal("expected at least one chunk from trailing data")
	}
}

func TestNewJSONRequestNilBody(t *testing.T) {
	provider := New("http://127.0.0.1", "2024-02-15-preview")
	req, err := provider.newJSONRequest("GET", "/test", testKey(), nil)
	if err != nil {
		t.Fatalf("newJSONRequest nil body: %v", err)
	}
	defer fasthttp.ReleaseRequest(req)
}

func TestChatCompletionStreamInvalidURL(t *testing.T) {
	provider := New("http://invalid\x00host", "2024-02-15-preview")
	_, err := provider.ChatCompletionStream(context.Background(), testKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if !strings.Contains(err.Error(), "create azure request") && !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("error = %v, want URL error", err)
	}
}

func TestNewHTTPJSONRequestNilBody(t *testing.T) {
	provider := New("http://127.0.0.1", "2024-02-15-preview")
	req, err := provider.newHTTPJSONRequest(context.Background(), "GET", "/test", testKey(), nil)
	if err != nil {
		t.Fatalf("newHTTPJSONRequest nil body: %v", err)
	}
	_ = req
}

func TestChatCompletionInvalidJSONResponse(t *testing.T) {
	server := jsonServer(t, http.StatusOK, `not-json`, nil)
	provider := New(server.URL, "2024-02-15-preview")

	_, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if err == nil || !strings.Contains(err.Error(), "parse azure chat response") {
		t.Fatalf("error = %v, want parse error", err)
	}
}

func TestListModelsInvalidJSONResponse(t *testing.T) {
	server := jsonServer(t, http.StatusOK, `not-json`, nil)
	provider := New(server.URL, "2024-02-15-preview")

	_, err := provider.ListModels(context.Background(), testKey())
	if err == nil || !strings.Contains(err.Error(), "parse azure deployments response") {
		t.Fatalf("error = %v, want parse deployments error", err)
	}
}

func TestStreamHTTPError500(t *testing.T) {
	server := jsonServer(t, http.StatusServiceUnavailable, `{"error":{"message":"down"}}`, nil)
	provider := New(server.URL, "2024-02-15-preview")

	_, err := provider.ChatCompletionStream(context.Background(), testKey(), testChatRequest())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
}

func TestParseSSEScannerError(t *testing.T) {
	pr, pw := io.Pipe()
	chunks := make(chan providers.StreamChunk, 10)
	go func() {
		_, _ = pw.Write([]byte("data: " + streamChunkContentJSON + "\n\n"))
		pw.CloseWithError(fmt.Errorf("simulated read error"))
	}()
	parseSSE(pr, chunks)
	close(chunks)
	var got []providers.StreamChunk
	for c := range chunks {
		got = append(got, c)
	}
	hasError := false
	for _, c := range got {
		if c.Error != nil && c.Error.Code == "upstream_stream_error" {
			hasError = true
		}
	}
	if !hasError {
		t.Fatalf("chunks = %+v, want upstream_stream_error", got)
	}
}

func TestHandleSSEDataEmpty(t *testing.T) {
	ch := make(chan providers.StreamChunk, 1)
	done, failed := handleSSEData(nil, ch)
	if done || failed {
		t.Fatalf("expected false,false for nil data")
	}
}

func TestNewJSONRequestWithBody(t *testing.T) {
	provider := New("http://127.0.0.1", "2024-02-15-preview")
	req, err := provider.newJSONRequest("POST", "/test", testKey(), &providers.ChatRequest{Model: "x"})
	if err != nil {
		t.Fatalf("newJSONRequest with body: %v", err)
	}
	defer fasthttp.ReleaseRequest(req)
}

func TestChatCompletionNetworkError(t *testing.T) {
	provider := New("http://127.0.0.1:1", "2024-02-15-preview")
	_, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected network error")
	}
	if !strings.Contains(err.Error(), "azure chat completion") {
		t.Fatalf("error = %v, want azure chat completion", err)
	}
}

func TestListModelsNetworkError(t *testing.T) {
	provider := New("http://127.0.0.1:1", "2024-02-15-preview")
	_, err := provider.ListModels(context.Background(), testKey())
	if err == nil {
		t.Fatal("expected network error")
	}
	if !strings.Contains(err.Error(), "azure list models") {
		t.Fatalf("error = %v, want azure list models", err)
	}
}

func TestStreamNetworkError(t *testing.T) {
	provider := New("http://127.0.0.1:1", "2024-02-15-preview")
	_, err := provider.ChatCompletionStream(context.Background(), testKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected network error")
	}
	if !strings.Contains(err.Error(), "azure chat completion stream") {
		t.Fatalf("error = %v, want azure chat completion stream", err)
	}
}

func TestChatCompletionStreamBodyReadError(t *testing.T) {
	// Stream endpoint returns non-2xx with body — covers read azure error response path
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"bad req"}}`))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL, "2024-02-15-preview")
	_, err := provider.ChatCompletionStream(context.Background(), testKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected error for stream 400")
	}
	if !strings.Contains(err.Error(), "bad req") && !strings.Contains(err.Error(), "400") {
		t.Fatalf("error = %v", err)
	}
}

func TestStreamSSETrailingMalformedData(t *testing.T) {
	// Malformed JSON in trailing data (no blank line at end) hits the failed return
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {not-json}"))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL, "2024-02-15-preview")
	chunks, err := provider.ChatCompletionStream(context.Background(), testKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}
	got := collectChunks(chunks)
	if len(got) != 1 || got[0].Error == nil {
		t.Fatalf("chunks = %+v, want one error chunk", got)
	}
	if got[0].Error.Code != "upstream_stream_malformed" {
		t.Fatalf("error code = %q", got[0].Error.Code)
	}
}

func TestListModelsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	provider := New("http://127.0.0.1:1", "2024-02-15-preview")
	_, err := provider.ListModels(ctx, testKey())
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}
