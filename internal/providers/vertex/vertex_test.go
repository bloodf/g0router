package vertex

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
	provider := New("", Config{ProjectID: "test-project", Location: "us-central1"})
	if provider.client == nil {
		t.Fatal("client is nil")
	}

	var _ *fasthttp.Client = provider.client
}

func TestProviderSatisfiesInterface(t *testing.T) {
	var _ providers.Provider = New("", Config{ProjectID: "test-project", Location: "us-central1"})
}

func TestChatCompletionRequiresProjectAndLocation(t *testing.T) {
	provider := New("http://127.0.0.1:1", Config{})

	_, err := provider.ChatCompletion(context.Background(), oauthKey(), testChatRequest())
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("expected ErrUnsupported for missing config, got %v", err)
	}
	if !strings.Contains(err.Error(), "project") || !strings.Contains(err.Error(), "location") {
		t.Fatalf("error = %q, want project and location context", err.Error())
	}
}

func TestChatCompletionStreamRequiresProjectAndLocation(t *testing.T) {
	provider := New("http://127.0.0.1:1", Config{})

	_, err := provider.ChatCompletionStream(context.Background(), oauthKey(), testChatRequest())
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("expected ErrUnsupported for missing config, got %v", err)
	}
	if !strings.Contains(err.Error(), "project") || !strings.Contains(err.Error(), "location") {
		t.Fatalf("error = %q, want project and location context", err.Error())
	}
}

func TestChatCompletionBuildsVertexRequest(t *testing.T) {
	var gotPath string
	var gotQuery string
	var gotAuth string
	var gotContentType string
	var gotRequest generateContentRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(generateContentResponseJSON))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})
	temp := 0.2
	maxTokens := 32
	resp, err := provider.ChatCompletion(context.Background(), oauthKey(), &providers.ChatRequest{
		Model:       "gemini-2.5-flash",
		Temperature: &temp,
		MaxTokens:   &maxTokens,
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}

	wantPath := "/v1/projects/test-project/locations/us-central1/publishers/google/models/gemini-2.5-flash:generateContent"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotQuery != "" {
		t.Errorf("query = %q", gotQuery)
	}
	if gotAuth != "Bearer vertex-token" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q", gotContentType)
	}
	if len(gotRequest.Contents) != 1 {
		t.Fatalf("contents len = %d", len(gotRequest.Contents))
	}
	if gotRequest.Contents[0].Role != "user" {
		t.Errorf("role = %q", gotRequest.Contents[0].Role)
	}
	if gotRequest.Contents[0].Parts[0].Text != "hello" {
		t.Errorf("text = %q", gotRequest.Contents[0].Parts[0].Text)
	}
	if gotRequest.GenerationConfig == nil {
		t.Fatal("generationConfig is nil")
	}
	if gotRequest.GenerationConfig.Temperature == nil || *gotRequest.GenerationConfig.Temperature != 0.2 {
		t.Errorf("temperature = %+v", gotRequest.GenerationConfig.Temperature)
	}
	if gotRequest.GenerationConfig.MaxOutputTokens == nil || *gotRequest.GenerationConfig.MaxOutputTokens != 32 {
		t.Errorf("maxOutputTokens = %+v", gotRequest.GenerationConfig.MaxOutputTokens)
	}
	if resp.Model != "gemini-2.5-flash" {
		t.Errorf("response model = %q", resp.Model)
	}
}

func TestParseGenerateContentResponse(t *testing.T) {
	server := jsonServer(t, http.StatusOK, generateContentResponseJSON)
	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})

	resp, err := provider.ChatCompletion(context.Background(), oauthKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}

	if resp.Object != "chat.completion" {
		t.Errorf("Object = %q", resp.Object)
	}
	if resp.Model != "gemini-2.5-flash" {
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

func TestChatCompletionStreamMapsVertexSSEChunks(t *testing.T) {
	var gotPath string
	var gotQuery string
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: " + generateContentStreamTextChunkJSON + "\n\n"))
		_, _ = w.Write([]byte("data: " + generateContentStreamUsageChunkJSON + "\n\n"))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})
	chunks, err := provider.ChatCompletionStream(context.Background(), oauthKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}
	got := collectStreamChunks(chunks)

	wantPath := "/v1/projects/test-project/locations/us-central1/publishers/google/models/gemini-2.5-flash:streamGenerateContent"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotQuery != "alt=sse" {
		t.Errorf("query = %q", gotQuery)
	}
	if gotAuth != "Bearer vertex-token" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if len(got) != 2 {
		t.Fatalf("chunks = %+v", got)
	}
	if got[0].Object != "chat.completion.chunk" || got[0].Model != "gemini-2.5-flash" {
		t.Fatalf("first chunk metadata = %+v", got[0])
	}
	if got[0].Choices[0].Delta.Content == nil || *got[0].Choices[0].Delta.Content != "hello" {
		t.Fatalf("first chunk content = %+v", got[0].Choices[0].Delta.Content)
	}
	if got[1].Choices[0].FinishReason == nil || *got[1].Choices[0].FinishReason != "stop" {
		t.Fatalf("finish reason = %+v", got[1].Choices[0].FinishReason)
	}
	if got[1].Usage == nil || got[1].Usage.TotalTokens != 14 {
		t.Fatalf("usage = %+v", got[1].Usage)
	}
}

func TestChatCompletionStreamMalformedSSEEmitsErrorChunk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {not-json}\n\n"))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})
	chunks, err := provider.ChatCompletionStream(context.Background(), oauthKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}
	got := collectStreamChunks(chunks)

	if len(got) != 1 || got[0].Error == nil {
		t.Fatalf("chunks = %+v, want one error chunk", got)
	}
	if got[0].Error.Code != "upstream_stream_malformed" {
		t.Fatalf("error code = %q", got[0].Error.Code)
	}
}

func TestParseError401(t *testing.T) {
	server := jsonServer(t, http.StatusUnauthorized, `{"error":{"message":"invalid token"}}`)
	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})

	_, err := provider.ChatCompletion(context.Background(), oauthKey(), testChatRequest())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
	if !strings.Contains(err.Error(), "invalid token") {
		t.Fatalf("error = %q", err)
	}
}

func TestParseError429(t *testing.T) {
	server := jsonServer(t, http.StatusTooManyRequests, `{"error":{"message":"slow down"}}`)
	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})

	_, err := provider.ChatCompletion(context.Background(), oauthKey(), testChatRequest())
	if !errors.Is(err, ErrRateLimit) {
		t.Fatalf("expected ErrRateLimit, got %v", err)
	}
}

func TestParseError500(t *testing.T) {
	server := jsonServer(t, http.StatusInternalServerError, `{"error":{"message":"upstream failed"}}`)
	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})

	_, err := provider.ChatCompletion(context.Background(), oauthKey(), testChatRequest())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
}

func oauthKey() providers.Key {
	return providers.Key{Value: "vertex-token", Provider: providers.ProviderVertex, ConnID: "conn-1", AuthType: "oauth"}
}

func testChatRequest() *providers.ChatRequest {
	return &providers.ChatRequest{
		Model: "gemini-2.5-flash",
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	}
}

func jsonServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func collectStreamChunks(chunks <-chan providers.StreamChunk) []providers.StreamChunk {
	var got []providers.StreamChunk
	for chunk := range chunks {
		got = append(got, chunk)
	}
	return got
}

const generateContentResponseJSON = `{
	"candidates": [{
		"content": {"role": "model", "parts": [{"text": "hello back"}]},
		"finishReason": "STOP"
	}],
	"usageMetadata": {
		"promptTokenCount": 5,
		"candidatesTokenCount": 9,
		"totalTokenCount": 14
	}
}`

const generateContentStreamTextChunkJSON = `{"candidates":[{"content":{"role":"model","parts":[{"text":"hello"}]}}]}`

const generateContentStreamUsageChunkJSON = `{"candidates":[{"content":{"role":"model","parts":[]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":9,"totalTokenCount":14}}`
