package anthropic

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

func TestNewForProviderWithHeadersAddsProviderHeaders(t *testing.T) {
	var gotFeature string
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotFeature = r.Header.Get("X-GitLab-Feature")
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(messageResponseJSON))
	}))
	t.Cleanup(server.Close)

	provider := NewForProviderWithHeaders(providers.ModelProvider("gitlab-duo"), server.URL, map[string]string{
		"X-GitLab-Feature": "duo",
	})
	_, err := provider.ChatCompletion(context.Background(), providers.Key{
		Value:    "direct-access-token",
		Provider: providers.ModelProvider("gitlab-duo"),
		AuthType: "oauth",
	}, testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if gotFeature != "duo" {
		t.Fatalf("X-GitLab-Feature = %q, want duo", gotFeature)
	}
	if gotAuth != "Bearer direct-access-token" {
		t.Fatalf("Authorization = %q, want direct access bearer", gotAuth)
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

func TestPreservesToolsAndToolMessages(t *testing.T) {
	var gotRequest anthropicRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(anthropicToolUseResponseJSON))
	}))
	t.Cleanup(server.Close)

	toolCallID := "call-1"
	provider := New(server.URL)
	resp, err := provider.ChatCompletion(context.Background(), testKey("api_key"), &providers.ChatRequest{
		Model:      "claude-sonnet-4-20250514",
		ToolChoice: "required",
		Tools: []providers.Tool{{
			Type: "function",
			Function: providers.ToolFunction{
				Name:        "weather",
				Description: "Get weather",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}}}`),
			},
		}},
		Messages: []providers.Message{
			{Role: "user", Content: "What is the weather?"},
			{
				Role:    "assistant",
				Content: "Calling weather.",
				ToolCalls: []providers.ToolCall{{
					ID:   toolCallID,
					Type: "function",
					Function: providers.ToolCallFunc{
						Name:      "weather",
						Arguments: `{"city":"Paris"}`,
					},
				}},
			},
			{Role: "tool", ToolCallID: &toolCallID, Content: `{"temp_c":19}`},
		},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}

	if len(gotRequest.Tools) != 1 {
		t.Fatalf("tools = %+v", gotRequest.Tools)
	}
	if gotRequest.Tools[0].Name != "weather" || gotRequest.Tools[0].Description != "Get weather" {
		t.Fatalf("tool = %+v", gotRequest.Tools[0])
	}
	if string(gotRequest.Tools[0].InputSchema) != `{"type":"object","properties":{"city":{"type":"string"}}}` {
		t.Fatalf("input schema = %s", gotRequest.Tools[0].InputSchema)
	}
	if gotRequest.ToolChoice == nil || gotRequest.ToolChoice.Type != "any" {
		t.Fatalf("tool choice = %+v", gotRequest.ToolChoice)
	}
	if len(gotRequest.Messages) != 3 {
		t.Fatalf("messages = %+v", gotRequest.Messages)
	}
	assistantBlocks := gotRequest.Messages[1].Content
	if len(assistantBlocks) != 2 {
		t.Fatalf("assistant content = %+v", assistantBlocks)
	}
	if assistantBlocks[0].Type != "text" || assistantBlocks[0].Text != "Calling weather." {
		t.Fatalf("assistant text = %+v", assistantBlocks[0])
	}
	if assistantBlocks[1].Type != "tool_use" || assistantBlocks[1].ID != toolCallID || assistantBlocks[1].Name != "weather" {
		t.Fatalf("tool_use block = %+v", assistantBlocks[1])
	}
	if string(assistantBlocks[1].Input) != `{"city":"Paris"}` {
		t.Fatalf("tool_use input = %s", assistantBlocks[1].Input)
	}
	toolBlocks := gotRequest.Messages[2].Content
	if len(toolBlocks) != 1 || toolBlocks[0].Type != "tool_result" || toolBlocks[0].ToolUseID != toolCallID {
		t.Fatalf("tool_result block = %+v", toolBlocks)
	}
	if toolBlocks[0].Content != `{"temp_c":19}` {
		t.Fatalf("tool_result content = %+v", toolBlocks[0])
	}

	if len(resp.Choices) != 1 {
		t.Fatalf("choices = %+v", resp.Choices)
	}
	toolCalls := resp.Choices[0].Message.ToolCalls
	if len(toolCalls) != 1 {
		t.Fatalf("tool calls = %+v", toolCalls)
	}
	if toolCalls[0].ID != "toolu_1" || toolCalls[0].Type != "function" {
		t.Fatalf("tool call = %+v", toolCalls[0])
	}
	if toolCalls[0].Function.Name != "weather" || toolCalls[0].Function.Arguments != `{"city":"Paris"}` {
		t.Fatalf("tool call function = %+v", toolCalls[0].Function)
	}
	if resp.Choices[0].FinishReason == nil || *resp.Choices[0].FinishReason != "tool_calls" {
		t.Fatalf("finish reason = %+v", resp.Choices[0].FinishReason)
	}
}

func TestAnthropicToolDefaultsChoicesAndResults(t *testing.T) {
	toolCallID := "call-1"
	secondToolCallID := "call-2"
	req, err := toAnthropicRequest(&providers.ChatRequest{
		Model:      "claude-sonnet-4-20250514",
		ToolChoice: map[string]any{"type": "function", "function": map[string]any{"name": "weather"}},
		Tools: []providers.Tool{{
			Type:     "function",
			Function: providers.ToolFunction{Name: "weather"},
		}},
		Messages: []providers.Message{
			{
				Role:    "assistant",
				Content: "Calling tools.",
				ToolCalls: []providers.ToolCall{
					{
						ID:   toolCallID,
						Type: "function",
						Function: providers.ToolCallFunc{
							Name:      "weather",
							Arguments: `{"city":"Paris"}`,
						},
					},
					{
						ID:   secondToolCallID,
						Type: "function",
						Function: providers.ToolCallFunc{
							Name:      "weather",
							Arguments: `{"city":"Lisbon"}`,
						},
					},
				},
			},
			{Role: "tool", ToolCallID: &toolCallID, Content: "Paris 19C"},
			{Role: "tool", ToolCallID: &secondToolCallID, Content: "Lisbon 21C"},
		},
	})
	if err != nil {
		t.Fatalf("toAnthropicRequest: %v", err)
	}

	if len(req.Tools) != 1 {
		t.Fatalf("tools = %+v", req.Tools)
	}
	if string(req.Tools[0].InputSchema) != `{"type":"object","properties":{}}` {
		t.Fatalf("default input schema = %s", req.Tools[0].InputSchema)
	}
	if req.ToolChoice == nil || req.ToolChoice.Type != "tool" || req.ToolChoice.Name != "weather" {
		t.Fatalf("tool choice = %+v", req.ToolChoice)
	}
	if len(req.Messages) != 2 {
		t.Fatalf("messages = %+v", req.Messages)
	}
	assistantBlocks := req.Messages[0].Content
	if len(assistantBlocks) != 3 {
		t.Fatalf("assistant blocks = %+v", assistantBlocks)
	}
	if assistantBlocks[1].Type != "tool_use" || assistantBlocks[2].Type != "tool_use" {
		t.Fatalf("tool use blocks = %+v", assistantBlocks)
	}
	if req.Messages[1].Role != "user" || len(req.Messages[1].Content) != 2 {
		t.Fatalf("coalesced tool result message = %+v", req.Messages[1])
	}
	if req.Messages[1].Content[0].ToolUseID != toolCallID || req.Messages[1].Content[1].ToolUseID != secondToolCallID {
		t.Fatalf("tool result IDs = %+v", req.Messages[1].Content)
	}
}

func TestAnthropicToolResultRequiresToolCallID(t *testing.T) {
	_, err := toAnthropicRequest(&providers.ChatRequest{
		Model: "claude-sonnet-4-20250514",
		Messages: []providers.Message{
			{Role: "tool", Content: "missing id"},
		},
	})
	if err == nil {
		t.Fatal("toAnthropicRequest error = nil")
	}
	if !strings.Contains(err.Error(), "tool_call_id") {
		t.Fatalf("error = %v", err)
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

func TestParseSSEStreamReportsMalformedEvent(t *testing.T) {
	server := streamServer(t, strings.Join([]string{
		"event: content_block_delta",
		`data: {"type":"content_block_delta","error":"sk-live-secret leaked upstream body"`,
		"",
		"event: content_block_delta",
		"data: " + streamContentDeltaJSON,
		"",
	}, "\n"))
	provider := New(server.URL)

	chunks, err := provider.ChatCompletionStream(context.Background(), testKey("api_key"), testChatRequest())
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

func TestParseSSEStreamMapsUpstreamErrorEvent(t *testing.T) {
	server := streamServer(t, strings.Join([]string{
		"event: error",
		`data: {"type":"error","error":{"type":"overloaded_error","message":"sk-live-secret leaked upstream body"}}`,
		"",
		"event: content_block_delta",
		"data: " + streamContentDeltaJSON,
		"",
	}, "\n"))
	provider := New(server.URL)

	chunks, err := provider.ChatCompletionStream(context.Background(), testKey("api_key"), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}

	got := collectChunks(chunks)
	if len(got) != 1 {
		t.Fatalf("chunks len = %d, want 1; chunks=%+v", len(got), got)
	}
	if got[0].Error == nil {
		t.Fatalf("chunk error = nil, want upstream stream error")
	}
	if strings.Contains(got[0].Error.Message, "sk-live-secret") || strings.Contains(got[0].Error.Message, "leaked upstream body") {
		t.Fatalf("chunk error leaked upstream body: %+v", got[0].Error)
	}
	if got[0].Error.Code != "upstream_stream_error" {
		t.Fatalf("chunk error code = %q, want upstream_stream_error", got[0].Error.Code)
	}
}

func TestParseSSEToolUseStream(t *testing.T) {
	server := streamServer(t, strings.Join([]string{
		"event: message_start",
		"data: " + streamMessageStartJSON,
		"",
		"event: content_block_start",
		"data: " + streamToolUseStartJSON,
		"",
		"event: content_block_delta",
		"data: " + streamToolUseDeltaOneJSON,
		"",
		"event: content_block_delta",
		"data: " + streamToolUseDeltaTwoJSON,
		"",
		"event: content_block_stop",
		"data: " + streamToolUseStopJSON,
		"",
		"event: message_delta",
		"data: " + streamToolUseMessageDeltaJSON,
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
		t.Fatalf("chunks len = %d: %+v", len(got), got)
	}
	toolCalls := got[1].Choices[0].Delta.ToolCalls
	if len(toolCalls) != 1 {
		t.Fatalf("tool calls = %+v", toolCalls)
	}
	if toolCalls[0].ID != "toolu_1" || toolCalls[0].Function.Name != "weather" {
		t.Fatalf("tool call = %+v", toolCalls[0])
	}
	if toolCalls[0].Function.Arguments != `{"city":"Paris"}` {
		t.Fatalf("tool args = %q", toolCalls[0].Function.Arguments)
	}
	if got[2].Choices[0].FinishReason == nil || *got[2].Choices[0].FinishReason != "tool_calls" {
		t.Fatalf("finish reason = %+v", got[2].Choices[0].FinishReason)
	}
}

func TestChatCompletionStreamReturnsBeforeUpstreamCompletes(t *testing.T) {
	release := make(chan struct{})
	server := liveStreamServer(t, release, streamContentDeltaJSON, streamMessageDeltaJSON)
	provider := New(server.URL)

	type streamResult struct {
		chunks <-chan providers.StreamChunk
		err    error
	}
	result := make(chan streamResult, 1)
	go func() {
		chunks, err := provider.ChatCompletionStream(context.Background(), testKey("api_key"), testChatRequest())
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

func liveStreamServer(t *testing.T, release <-chan struct{}, firstChunk string, secondChunk string) *httptest.Server {
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

const messageResponseJSON = `{
	"id": "msg_123",
	"type": "message",
	"role": "assistant",
	"model": "claude-sonnet-4-20250514",
	"content": [{"type": "text", "text": "hello back"}],
	"stop_reason": "end_turn",
	"usage": {"input_tokens": 5, "output_tokens": 9}
}`

const anthropicToolUseResponseJSON = `{
	"id": "msg_tool",
	"type": "message",
	"role": "assistant",
	"model": "claude-sonnet-4-20250514",
	"content": [
		{"type": "text", "text": "I need weather data."},
		{"type": "tool_use", "id": "toolu_1", "name": "weather", "input": {"city": "Paris"}}
	],
	"stop_reason": "tool_use",
	"usage": {"input_tokens": 8, "output_tokens": 4}
}`

const streamMessageStartJSON = `{"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-sonnet-4-20250514","content":[],"usage":{"input_tokens":5,"output_tokens":0}}}`

const streamContentDeltaJSON = `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello"}}`

const streamMessageDeltaJSON = `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":7}}`

const streamToolUseStartJSON = `{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_1","name":"weather","input":{}}}`

const streamToolUseDeltaOneJSON = `{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"city\""}}`

const streamToolUseDeltaTwoJSON = `{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":":\"Paris\"}"}}`

const streamToolUseStopJSON = `{"type":"content_block_stop","index":1}`

const streamToolUseMessageDeltaJSON = `{"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":7}}`

const modelsResponseJSON = `{
	"data": [
		{"id": "claude-sonnet-4-20250514", "type": "model", "created_at": "2025-05-14T00:00:00Z", "display_name": "Claude Sonnet 4"},
		{"id": "claude-3-5-haiku-20241022", "type": "model", "display_name": "Claude 3.5 Haiku"}
	]
}`
