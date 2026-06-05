package vertex

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

// ---- additional coverage tests ----

func TestNameReturnsVertex(t *testing.T) {
	p := New("", Config{ProjectID: "p", Location: "l"})
	if p.Name() != providers.ProviderVertex {
		t.Fatalf("Name = %q, want vertex", p.Name())
	}
}

func TestRateLimitErrorIs(t *testing.T) {
	err := &RateLimitError{Message: "too fast"}
	if !errors.Is(err, ErrRateLimit) {
		t.Fatal("expected ErrRateLimit sentinel")
	}
	blank := &RateLimitError{}
	if blank.Error() != ErrRateLimit.Error() {
		t.Fatalf("blank error = %q", blank.Error())
	}
	if err.Error() == ErrRateLimit.Error() {
		t.Fatal("message variant should differ")
	}
}

func TestListModelsRequiresConfig(t *testing.T) {
	provider := New("http://127.0.0.1:1", Config{})
	_, err := provider.ListModels(context.Background(), oauthKey())
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("expected ErrUnsupported, got %v", err)
	}
}

func TestListModelsReturnsModels(t *testing.T) {
	server := jsonServer(t, http.StatusOK, listModelsResponseJSON)
	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})

	models, err := provider.ListModels(context.Background(), oauthKey())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("models len = %d", len(models))
	}
	if models[0].ID != "gemini-2.5-flash" || models[0].Provider != providers.ProviderVertex {
		t.Errorf("model[0] = %+v", models[0])
	}
	if models[1].ID != "gemini-2.5-pro" || models[1].OwnedBy != "google" {
		t.Errorf("model[1] = %+v", models[1])
	}
}

func TestListModels401(t *testing.T) {
	server := jsonServer(t, http.StatusUnauthorized, `{"error":{"message":"bad token"}}`)
	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})

	_, err := provider.ListModels(context.Background(), oauthKey())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

func TestParseError403(t *testing.T) {
	server := jsonServer(t, http.StatusForbidden, `{"error":{"message":"forbidden"}}`)
	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})

	_, err := provider.ChatCompletion(context.Background(), oauthKey(), testChatRequest())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

func TestParseError400(t *testing.T) {
	server := jsonServer(t, http.StatusBadRequest, `{"error":{"message":"bad request"}}`)
	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})

	_, err := provider.ChatCompletion(context.Background(), oauthKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "bad request") {
		t.Fatalf("error = %q", err)
	}
}

func TestParseErrorEmptyBody(t *testing.T) {
	server := jsonServer(t, http.StatusInternalServerError, ``)
	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})

	_, err := provider.ChatCompletion(context.Background(), oauthKey(), testChatRequest())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Fatalf("error = %q, want empty response", err)
	}
}

func TestParseErrorNonJSONBody(t *testing.T) {
	server := jsonServer(t, http.StatusInternalServerError, `plain text error`)
	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})

	_, err := provider.ChatCompletion(context.Background(), oauthKey(), testChatRequest())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
}

func TestStreamHTTPError(t *testing.T) {
	server := jsonServer(t, http.StatusUnauthorized, `{"error":{"message":"bad stream token"}}`)
	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})

	_, err := provider.ChatCompletionStream(context.Background(), oauthKey(), testChatRequest())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

func TestStreamSSEDone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})
	chunks, err := provider.ChatCompletionStream(context.Background(), oauthKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}
	got := collectStreamChunks(chunks)
	if len(got) != 0 {
		t.Fatalf("chunks = %+v, want none after DONE", got)
	}
}

func TestBuildRequestSystemMessage(t *testing.T) {
	req, err := buildGenerateContentRequest(&providers.ChatRequest{
		Model:  "gemini-2.5-flash",
		System: "you are a bot",
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("buildGenerateContentRequest: %v", err)
	}
	if req.SystemInstruction == nil || req.SystemInstruction.Parts[0].Text != "you are a bot" {
		t.Fatalf("system instruction = %+v", req.SystemInstruction)
	}
}

func TestBuildRequestSystemMessageInMessages(t *testing.T) {
	req, err := buildGenerateContentRequest(&providers.ChatRequest{
		Model: "gemini-2.5-flash",
		Messages: []providers.Message{
			{Role: "system", Content: "you are a bot"},
			{Role: "user", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("buildGenerateContentRequest: %v", err)
	}
	if req.SystemInstruction == nil || req.SystemInstruction.Parts[0].Text != "you are a bot" {
		t.Fatalf("system instruction = %+v", req.SystemInstruction)
	}
	if len(req.Contents) != 1 {
		t.Fatalf("contents = %d, want 1 (no system in contents)", len(req.Contents))
	}
}

func TestBuildRequestMaxCompletionTokens(t *testing.T) {
	maxComp := 128
	req, err := buildGenerateContentRequest(&providers.ChatRequest{
		Model:               "gemini-2.5-flash",
		MaxCompletionTokens: &maxComp,
		Messages:            []providers.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("buildGenerateContentRequest: %v", err)
	}
	if req.GenerationConfig == nil || *req.GenerationConfig.MaxOutputTokens != 128 {
		t.Fatalf("maxOutputTokens = %+v", req.GenerationConfig)
	}
}

func TestBuildRequestUnsupportedContentType(t *testing.T) {
	// vertex textContent only accepts strings; non-string should fail
	_, err := buildGenerateContentRequest(&providers.ChatRequest{
		Model: "gemini-2.5-flash",
		Messages: []providers.Message{
			{Role: "user", Content: []string{"not", "a", "string"}},
		},
	})
	if err == nil {
		t.Fatal("expected error for non-string content")
	}
}

func TestVertexRoleMapping(t *testing.T) {
	if vertexRole("assistant") != "model" {
		t.Fatal("assistant should map to model")
	}
	if vertexRole("user") != "user" {
		t.Fatal("user should map to user")
	}
}

func TestMapFinishReasonAllBranches(t *testing.T) {
	cases := []struct{ input, want string }{
		{"STOP", "stop"},
		{"MAX_TOKENS", "length"},
		{"SAFETY", "content_filter"},
		{"RECITATION", "content_filter"},
		{"OTHER", "other"},
	}
	for _, c := range cases {
		got := mapFinishReason(c.input)
		if got != c.want {
			t.Errorf("mapFinishReason(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestChatCompletionCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	provider := New("http://127.0.0.1:1", Config{ProjectID: "p", Location: "l"})
	_, err := provider.ChatCompletion(ctx, oauthKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestDoWithDeadline(t *testing.T) {
	server := jsonServer(t, http.StatusOK, generateContentResponseJSON)
	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := provider.ChatCompletion(ctx, oauthKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletion with deadline: %v", err)
	}
}

func TestDoAlreadyExpiredDeadline(t *testing.T) {
	provider := New("http://127.0.0.1:1", Config{ProjectID: "p", Location: "l"})
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	<-ctx.Done()
	_, err := provider.ChatCompletion(ctx, oauthKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected error for expired deadline")
	}
}

func TestStreamSSETrailingData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// No trailing blank line — pending data at EOF
		_, _ = w.Write([]byte("data: " + generateContentStreamTextChunkJSON))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})
	chunks, err := provider.ChatCompletionStream(context.Background(), oauthKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}
	got := collectStreamChunks(chunks)
	if len(got) == 0 {
		t.Fatal("expected at least one chunk from trailing data")
	}
}

func TestNewJSONRequestNilBody(t *testing.T) {
	provider := New("http://127.0.0.1", Config{ProjectID: "p", Location: "l"})
	req, err := provider.newJSONRequest("GET", "/test", oauthKey(), nil)
	if err != nil {
		t.Fatalf("newJSONRequest nil body: %v", err)
	}
	defer fasthttp.ReleaseRequest(req)
}

func TestChatCompletionStreamInvalidURL(t *testing.T) {
	// URL with control character causes http.NewRequestWithContext to fail in newHTTPJSONRequest
	provider := New("http://invalid\x00host", Config{ProjectID: "p", Location: "l"})
	_, err := provider.ChatCompletionStream(context.Background(), oauthKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestNewHTTPJSONRequestNilBody(t *testing.T) {
	provider := New("http://127.0.0.1", Config{ProjectID: "p", Location: "l"})
	req, err := provider.newHTTPJSONRequest(context.Background(), "GET", "/test", oauthKey(), nil, nil)
	if err != nil {
		t.Fatalf("newHTTPJSONRequest nil body: %v", err)
	}
	_ = req
}

func TestListModels500Error(t *testing.T) {
	server := jsonServer(t, http.StatusInternalServerError, `{"error":{"message":"server down"}}`)
	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})

	_, err := provider.ListModels(context.Background(), oauthKey())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
}

func TestChatCompletionInvalidJSONResponse(t *testing.T) {
	server := jsonServer(t, http.StatusOK, `not-json`)
	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})

	_, err := provider.ChatCompletion(context.Background(), oauthKey(), testChatRequest())
	if err == nil || !strings.Contains(err.Error(), "parse vertex response") {
		t.Fatalf("error = %v, want parse error", err)
	}
}

func TestListModelsInvalidJSONResponse(t *testing.T) {
	server := jsonServer(t, http.StatusOK, `not-json`)
	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})

	_, err := provider.ListModels(context.Background(), oauthKey())
	if err == nil || !strings.Contains(err.Error(), "parse vertex models response") {
		t.Fatalf("error = %v, want parse models error", err)
	}
}

func TestChatCompletionStreamBuildError(t *testing.T) {
	provider := New("http://127.0.0.1:1", Config{ProjectID: "p", Location: "l"})
	_, err := provider.ChatCompletionStream(context.Background(), oauthKey(), &providers.ChatRequest{
		Model:    "gemini-2.5-flash",
		Messages: []providers.Message{{Role: "user", Content: []int{1, 2, 3}}},
	})
	if err == nil {
		t.Fatal("expected build error")
	}
}

func TestStreamHTTPError500(t *testing.T) {
	server := jsonServer(t, http.StatusServiceUnavailable, `{"error":{"message":"service down"}}`)
	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})

	_, err := provider.ChatCompletionStream(context.Background(), oauthKey(), testChatRequest())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
}

func TestMapGenerateContentStreamChunksUsageOnly(t *testing.T) {
	decoded := generateContentResponse{
		UsageMetadata: usageMetadata{PromptTokenCount: 1, TotalTokenCount: 1},
	}
	chunks := mapGenerateContentStreamChunks("model", decoded)
	if len(chunks) != 1 {
		t.Fatalf("chunks = %d, want 1 usage-only chunk", len(chunks))
	}
	if chunks[0].Usage == nil {
		t.Fatal("expected usage in chunk")
	}
}

func TestChatCompletionNetworkError(t *testing.T) {
	provider := New("http://127.0.0.1:1", Config{ProjectID: "p", Location: "l"})
	_, err := provider.ChatCompletion(context.Background(), oauthKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected network error")
	}
	if !strings.Contains(err.Error(), "vertex generate content") {
		t.Fatalf("error = %v, want vertex generate content", err)
	}
}

func TestListModelsNetworkError(t *testing.T) {
	provider := New("http://127.0.0.1:1", Config{ProjectID: "p", Location: "l"})
	_, err := provider.ListModels(context.Background(), oauthKey())
	if err == nil {
		t.Fatal("expected network error")
	}
	if !strings.Contains(err.Error(), "vertex list models") {
		t.Fatalf("error = %v, want vertex list models", err)
	}
}

func TestStreamNetworkError(t *testing.T) {
	provider := New("http://127.0.0.1:1", Config{ProjectID: "p", Location: "l"})
	_, err := provider.ChatCompletionStream(context.Background(), oauthKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected network error")
	}
	if !strings.Contains(err.Error(), "vertex stream generate content") {
		t.Fatalf("error = %v, want vertex stream error", err)
	}
}

func TestBuildRequestSystemContentError(t *testing.T) {
	// System field in Messages with non-string content causes textContent error
	_, err := buildGenerateContentRequest(&providers.ChatRequest{
		Model: "gemini-2.5-flash",
		Messages: []providers.Message{
			{Role: "system", Content: 42},
		},
	})
	if err == nil {
		t.Fatal("expected error for non-string system content in message")
	}
}

func TestBuildRequestContentError(t *testing.T) {
	_, err := buildGenerateContentRequest(&providers.ChatRequest{
		Model: "gemini-2.5-flash",
		Messages: []providers.Message{
			{Role: "user", Content: 42},
		},
	})
	if err == nil {
		t.Fatal("expected error for non-string content in message")
	}
}

func TestChatCompletionStreamBodyReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"bad request"}}`))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL, Config{ProjectID: "test-project", Location: "us-central1"})
	_, err := provider.ChatCompletionStream(context.Background(), oauthKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected error for stream 400")
	}
}

func TestStreamSSETrailingMalformedData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {not-json}"))
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

func TestParseVertexSSEScannerError(t *testing.T) {
	// io.Reader that returns an error mid-stream triggers scanner.Err branch
	pr, pw := io.Pipe()
	chunks := make(chan providers.StreamChunk, 10)
	go func() {
		// Write some valid data then close with error
		_, _ = pw.Write([]byte("data: " + generateContentStreamTextChunkJSON + "\n\n"))
		pw.CloseWithError(fmt.Errorf("simulated read error"))
	}()
	parseVertexSSE("model", pr, chunks)
	close(chunks)
	var got []providers.StreamChunk
	for c := range chunks {
		got = append(got, c)
	}
	// Should get at least a text chunk and then a scanner error chunk
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

func TestHandleVertexSSEDataEmpty(t *testing.T) {
	ch := make(chan providers.StreamChunk, 1)
	done, failed := handleVertexSSEData("model", nil, ch)
	if done || failed {
		t.Fatalf("expected false,false for nil data")
	}
}

func TestNewJSONRequestWithBody(t *testing.T) {
	provider := New("http://127.0.0.1", Config{ProjectID: "p", Location: "l"})
	req, err := provider.newJSONRequest("POST", "/test", oauthKey(), map[string]string{"x": "y"})
	if err != nil {
		t.Fatalf("newJSONRequest with body: %v", err)
	}
	defer fasthttp.ReleaseRequest(req)
}

const listModelsResponseJSON = `{
	"models": [
		{"name": "projects/test-project/locations/us-central1/publishers/google/models/gemini-2.5-flash", "displayName": "Gemini 2.5 Flash"},
		{"name": "projects/test-project/locations/us-central1/publishers/google/models/gemini-2.5-pro", "displayName": "Gemini 2.5 Pro"}
	]
}`
