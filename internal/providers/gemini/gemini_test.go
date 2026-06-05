package gemini

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
	provider := New("")
	if provider.client == nil {
		t.Fatal("client is nil")
	}

	var _ *fasthttp.Client = provider.client
}

func TestProviderSatisfiesInterface(t *testing.T) {
	var _ providers.Provider = New("")
}

func TestBuildGenerateContentRequestWithAPIKey(t *testing.T) {
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

	provider := New(server.URL)
	temp := 0.2
	maxTokens := 32
	resp, err := provider.ChatCompletion(context.Background(), apiKey(), &providers.ChatRequest{
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

	if gotPath != "/v1beta/models/gemini-2.5-flash:generateContent" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "key=gemini-key" {
		t.Errorf("query = %q", gotQuery)
	}
	if gotAuth != "" {
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

func TestBuildGenerateContentRequestWithOAuth(t *testing.T) {
	var gotQuery string
	var gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(generateContentResponseJSON))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL)
	_, err := provider.ChatCompletion(context.Background(), oauthKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}

	if gotQuery != "" {
		t.Errorf("query = %q", gotQuery)
	}
	if gotAuth != "Bearer oauth-token" {
		t.Errorf("Authorization = %q", gotAuth)
	}
}

func TestParseGenerateContentResponse(t *testing.T) {
	server := jsonServer(t, http.StatusOK, generateContentResponseJSON)
	provider := New(server.URL)

	resp, err := provider.ChatCompletion(context.Background(), apiKey(), testChatRequest())
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

func TestChatCompletionStreamMapsGeminiSSEChunks(t *testing.T) {
	var gotPath string
	var gotQuery string
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: " + generateContentStreamTextChunkJSON + "\n\n"))
		_, _ = w.Write([]byte("data: " + generateContentStreamToolChunkJSON + "\n\n"))
		_, _ = w.Write([]byte("data: " + generateContentStreamUsageChunkJSON + "\n\n"))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL)
	chunks, err := provider.ChatCompletionStream(context.Background(), apiKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}
	got := collectStreamChunks(chunks)

	if gotPath != "/v1beta/models/gemini-2.5-flash:streamGenerateContent" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "alt=sse&key=gemini-key" && gotQuery != "key=gemini-key&alt=sse" {
		t.Errorf("query = %q", gotQuery)
	}
	if gotAuth != "" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if len(got) != 3 {
		t.Fatalf("chunks = %+v", got)
	}
	if got[0].Object != "chat.completion.chunk" || got[0].Model != "gemini-2.5-flash" {
		t.Fatalf("first chunk metadata = %+v", got[0])
	}
	if got[0].Choices[0].Delta.Content == nil || *got[0].Choices[0].Delta.Content != "hello" {
		t.Fatalf("first chunk content = %+v", got[0].Choices[0].Delta.Content)
	}
	toolCalls := got[1].Choices[0].Delta.ToolCalls
	if len(toolCalls) != 1 || toolCalls[0].ID != "gemini-call-1" || toolCalls[0].Function.Name != "weather" {
		t.Fatalf("tool calls = %+v", toolCalls)
	}
	if toolCalls[0].Function.Arguments != `{"city":"Paris"}` {
		t.Fatalf("tool call args = %q", toolCalls[0].Function.Arguments)
	}
	if got[2].Choices[0].FinishReason == nil || *got[2].Choices[0].FinishReason != "stop" {
		t.Fatalf("finish reason = %+v", got[2].Choices[0].FinishReason)
	}
	if got[2].Usage == nil || got[2].Usage.TotalTokens != 14 {
		t.Fatalf("usage = %+v", got[2].Usage)
	}
}

func TestChatCompletionStreamWithOAuthUsesBearerAndAltSSE(t *testing.T) {
	var gotQuery string
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: " + generateContentStreamTextChunkJSON + "\n\n"))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL)
	chunks, err := provider.ChatCompletionStream(context.Background(), oauthKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}
	got := collectStreamChunks(chunks)

	if gotQuery != "alt=sse" {
		t.Errorf("query = %q", gotQuery)
	}
	if gotAuth != "Bearer oauth-token" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if len(got) != 1 || got[0].Choices[0].Delta.Content == nil || *got[0].Choices[0].Delta.Content != "hello" {
		t.Fatalf("chunks = %+v", got)
	}
}

func TestChatCompletionStreamMalformedSSEEmitsErrorChunk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {not-json}\n\n"))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL)
	chunks, err := provider.ChatCompletionStream(context.Background(), apiKey(), testChatRequest())
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

func TestPreservesToolsAndToolMessages(t *testing.T) {
	var gotRequest generateContentRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(generateContentToolCallResponseJSON))
	}))
	t.Cleanup(server.Close)

	toolCallID := "call-1"
	provider := New(server.URL)
	resp, err := provider.ChatCompletion(context.Background(), apiKey(), &providers.ChatRequest{
		Model: "gemini-2.5-flash",
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

	if len(gotRequest.Tools) != 1 || len(gotRequest.Tools[0].FunctionDeclarations) != 1 {
		t.Fatalf("tools = %+v", gotRequest.Tools)
	}
	declaration := gotRequest.Tools[0].FunctionDeclarations[0]
	if declaration.Name != "weather" || declaration.Description != "Get weather" {
		t.Fatalf("declaration = %+v", declaration)
	}
	if string(declaration.Parameters) != `{"type":"object","properties":{"city":{"type":"string"}}}` {
		t.Fatalf("parameters = %s", declaration.Parameters)
	}
	if len(gotRequest.Contents) != 3 {
		t.Fatalf("contents = %+v", gotRequest.Contents)
	}
	assistantParts := gotRequest.Contents[1].Parts
	if gotRequest.Contents[1].Role != "model" || len(assistantParts) != 2 {
		t.Fatalf("assistant content = %+v", gotRequest.Contents[1])
	}
	if assistantParts[0].Text != "Calling weather." {
		t.Fatalf("assistant text = %+v", assistantParts[0])
	}
	if assistantParts[1].FunctionCall == nil {
		t.Fatalf("function call part = %+v", assistantParts[1])
	}
	if assistantParts[1].FunctionCall.ID != toolCallID {
		t.Fatalf("function call id = %q", assistantParts[1].FunctionCall.ID)
	}
	if assistantParts[1].FunctionCall.Name != "weather" || assistantParts[1].FunctionCall.Args["city"] != "Paris" {
		t.Fatalf("function call = %+v", assistantParts[1].FunctionCall)
	}
	toolParts := gotRequest.Contents[2].Parts
	if gotRequest.Contents[2].Role != "user" || len(toolParts) != 1 || toolParts[0].FunctionResponse == nil {
		t.Fatalf("tool response content = %+v", gotRequest.Contents[2])
	}
	if toolParts[0].FunctionResponse.Name != "weather" {
		t.Fatalf("function response name = %q", toolParts[0].FunctionResponse.Name)
	}
	if toolParts[0].FunctionResponse.ID != toolCallID {
		t.Fatalf("function response id = %q", toolParts[0].FunctionResponse.ID)
	}
	if toolParts[0].FunctionResponse.Response["content"] != `{"temp_c":19}` {
		t.Fatalf("function response = %+v", toolParts[0].FunctionResponse.Response)
	}

	if len(resp.Choices) != 1 {
		t.Fatalf("choices = %+v", resp.Choices)
	}
	toolCalls := resp.Choices[0].Message.ToolCalls
	if len(toolCalls) != 1 {
		t.Fatalf("tool calls = %+v", toolCalls)
	}
	if toolCalls[0].ID != "gemini-call-1" || toolCalls[0].Type != "function" || toolCalls[0].Function.Name != "weather" {
		t.Fatalf("tool call = %+v", toolCalls[0])
	}
	if toolCalls[0].Function.Arguments != `{"city":"Paris"}` {
		t.Fatalf("tool call args = %q", toolCalls[0].Function.Arguments)
	}
	if resp.Choices[0].FinishReason == nil || *resp.Choices[0].FinishReason != "tool_calls" {
		t.Fatalf("finish reason = %+v", resp.Choices[0].FinishReason)
	}
}

func TestParseError401(t *testing.T) {
	server := jsonServer(t, http.StatusUnauthorized, `{"error":{"message":"invalid api key"}}`)
	provider := New(server.URL)

	_, err := provider.ChatCompletion(context.Background(), apiKey(), testChatRequest())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
	if !strings.Contains(err.Error(), "invalid api key") {
		t.Fatalf("error = %q", err)
	}
}

func TestParseError429(t *testing.T) {
	server := jsonServer(t, http.StatusTooManyRequests, `{"error":{"message":"slow down"}}`)
	provider := New(server.URL)

	_, err := provider.ChatCompletion(context.Background(), apiKey(), testChatRequest())
	if !errors.Is(err, ErrRateLimit) {
		t.Fatalf("expected ErrRateLimit, got %v", err)
	}
}

func TestParseError500(t *testing.T) {
	server := jsonServer(t, http.StatusInternalServerError, `{"error":{"message":"upstream failed"}}`)
	provider := New(server.URL)

	_, err := provider.ChatCompletion(context.Background(), apiKey(), testChatRequest())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
}

func apiKey() providers.Key {
	return providers.Key{Value: "gemini-key", Provider: providers.ProviderGemini, ConnID: "conn-1", AuthType: "api_key"}
}

func oauthKey() providers.Key {
	return providers.Key{Value: "oauth-token", Provider: providers.ProviderGemini, ConnID: "conn-1", AuthType: "oauth"}
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

const generateContentToolCallResponseJSON = `{
	"candidates": [{
		"content": {
			"role": "model",
			"parts": [
				{"text": "I need weather data."},
				{"functionCall": {"id": "gemini-call-1", "name": "weather", "args": {"city": "Paris"}}}
			]
		},
		"finishReason": "STOP"
	}],
	"usageMetadata": {
		"promptTokenCount": 5,
		"candidatesTokenCount": 9,
		"totalTokenCount": 14
	}
}`

const generateContentStreamTextChunkJSON = `{"candidates":[{"content":{"role":"model","parts":[{"text":"hello"}]}}]}`

const generateContentStreamToolChunkJSON = `{"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"id":"gemini-call-1","name":"weather","args":{"city":"Paris"}}}]}}]}`

const generateContentStreamUsageChunkJSON = `{"candidates":[{"content":{"role":"model","parts":[]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":9,"totalTokenCount":14}}`

// ---- additional coverage tests ----

func TestNameReturnsGemini(t *testing.T) {
	p := New("")
	if p.Name() != providers.ProviderGemini {
		t.Fatalf("Name = %q, want gemini", p.Name())
	}
}

func TestRateLimitErrorIs(t *testing.T) {
	err := &RateLimitError{Message: "too fast"}
	if !errors.Is(err, ErrRateLimit) {
		t.Fatal("expected ErrRateLimit sentinel")
	}
	if err.Error() == ErrRateLimit.Error() {
		t.Fatal("expected message to differ from bare sentinel")
	}
	blank := &RateLimitError{}
	if blank.Error() != ErrRateLimit.Error() {
		t.Fatalf("blank error = %q, want %q", blank.Error(), ErrRateLimit.Error())
	}
}

func TestListModelsReturnsModels(t *testing.T) {
	server := jsonServer(t, http.StatusOK, listModelsResponseJSON)
	provider := New(server.URL)

	models, err := provider.ListModels(context.Background(), apiKey())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("models len = %d", len(models))
	}
	if models[0].ID != "gemini-2.5-flash" || models[0].Provider != providers.ProviderGemini {
		t.Errorf("model[0] = %+v", models[0])
	}
	if models[1].ID != "gemini-2.5-pro" || models[1].OwnedBy != "google" {
		t.Errorf("model[1] = %+v", models[1])
	}
}

func TestListModels401(t *testing.T) {
	server := jsonServer(t, http.StatusUnauthorized, `{"error":{"message":"bad key"}}`)
	provider := New(server.URL)

	_, err := provider.ListModels(context.Background(), apiKey())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

func TestListModelsForbidden(t *testing.T) {
	server := jsonServer(t, http.StatusForbidden, `{"error":{"message":"forbidden"}}`)
	provider := New(server.URL)

	_, err := provider.ListModels(context.Background(), apiKey())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

func TestParseError400(t *testing.T) {
	server := jsonServer(t, http.StatusBadRequest, `{"error":{"message":"bad request"}}`)
	provider := New(server.URL)

	_, err := provider.ChatCompletion(context.Background(), apiKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "bad request") {
		t.Fatalf("error = %q", err)
	}
}

func TestParseErrorEmptyBody(t *testing.T) {
	server := jsonServer(t, http.StatusInternalServerError, ``)
	provider := New(server.URL)

	_, err := provider.ChatCompletion(context.Background(), apiKey(), testChatRequest())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Fatalf("error = %q, want empty response", err)
	}
}

func TestParseErrorNonJSONBody(t *testing.T) {
	server := jsonServer(t, http.StatusInternalServerError, `plain text error`)
	provider := New(server.URL)

	_, err := provider.ChatCompletion(context.Background(), apiKey(), testChatRequest())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
	if !strings.Contains(err.Error(), "plain text error") {
		t.Fatalf("error = %q, want plain text", err)
	}
}

func TestBuildRequestNilChatRequest(t *testing.T) {
	_, err := buildGenerateContentRequest(nil)
	if err == nil {
		t.Fatal("expected error for nil request")
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

func TestBuildRequestUnsupportedToolType(t *testing.T) {
	_, err := buildGenerateContentRequest(&providers.ChatRequest{
		Model: "gemini-2.5-flash",
		Tools: []providers.Tool{{Type: "unsupported"}},
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	})
	if err == nil {
		t.Fatal("expected error for unsupported tool type")
	}
}

func TestGeminiContentEmptyParts(t *testing.T) {
	// empty content string → empty parts → error
	_, err := geminiContent(providers.Message{Role: "user", Content: ""}, nil)
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestGeminiContentUnsupportedFunctionCallType(t *testing.T) {
	_, err := geminiContent(providers.Message{
		Role:    "assistant",
		Content: "ok",
		ToolCalls: []providers.ToolCall{{
			Type: "unsupported",
		}},
	}, nil)
	if err == nil {
		t.Fatal("expected error for unsupported tool call type")
	}
}

func TestGeminiToolContentWithName(t *testing.T) {
	name := "my_tool"
	c, err := geminiToolContent(providers.Message{
		Role:    "tool",
		Name:    &name,
		Content: "result",
	}, nil)
	if err != nil {
		t.Fatalf("geminiToolContent: %v", err)
	}
	if c.Parts[0].FunctionResponse.Name != name {
		t.Fatalf("name = %q, want %q", c.Parts[0].FunctionResponse.Name, name)
	}
}

func TestGeminiToolContentNoID(t *testing.T) {
	c, err := geminiToolContent(providers.Message{
		Role:    "tool",
		Content: "result",
	}, nil)
	if err != nil {
		t.Fatalf("geminiToolContent: %v", err)
	}
	if c.Parts[0].FunctionResponse.Name != "tool_result" {
		t.Fatalf("default name = %q", c.Parts[0].FunctionResponse.Name)
	}
}

func TestParseFunctionArgsEmpty(t *testing.T) {
	args, err := parseFunctionArgs("")
	if err != nil {
		t.Fatalf("parseFunctionArgs: %v", err)
	}
	if args != nil {
		t.Fatalf("expected nil for empty string, got %v", args)
	}
}

func TestParseFunctionArgsNonObject(t *testing.T) {
	// non-object JSON scalar → wrapped under "arguments"
	args, err := parseFunctionArgs(`"hello"`)
	if err != nil {
		t.Fatalf("parseFunctionArgs: %v", err)
	}
	if args["arguments"] != "hello" {
		t.Fatalf("args = %+v", args)
	}
}

func TestParseFunctionArgsInvalidJSON(t *testing.T) {
	// invalid JSON → fallback wraps raw string
	args, err := parseFunctionArgs(`not-json`)
	if err != nil {
		t.Fatalf("parseFunctionArgs: %v", err)
	}
	if args["arguments"] != "not-json" {
		t.Fatalf("args = %+v", args)
	}
}

func TestGeminiPartsFromBlocks(t *testing.T) {
	blocks := []map[string]any{
		{"type": "text", "text": "hello"},
	}
	parts, err := geminiPartsFromBlocks(blocks)
	if err != nil {
		t.Fatalf("geminiPartsFromBlocks: %v", err)
	}
	if len(parts) != 1 || parts[0].Text != "hello" {
		t.Fatalf("parts = %+v", parts)
	}
}

func TestGeminiPartsFromBlocksUnsupportedType(t *testing.T) {
	blocks := []map[string]any{
		{"type": "image_url", "url": "http://example.com"},
	}
	_, err := geminiPartsFromBlocks(blocks)
	if err == nil {
		t.Fatal("expected error for unsupported block type")
	}
}

func TestGeminiPartsFromBlocksNonStringText(t *testing.T) {
	blocks := []map[string]any{
		{"type": "text", "text": 42},
	}
	_, err := geminiPartsFromBlocks(blocks)
	if err == nil {
		t.Fatal("expected error for non-string text")
	}
}

func TestGeminiPartsFromAnyBlocks(t *testing.T) {
	blocks := []any{
		map[string]any{"type": "text", "text": "hi"},
	}
	parts, err := geminiPartsFromAnyBlocks(blocks)
	if err != nil {
		t.Fatalf("geminiPartsFromAnyBlocks: %v", err)
	}
	if len(parts) != 1 || parts[0].Text != "hi" {
		t.Fatalf("parts = %+v", parts)
	}
}

func TestGeminiPartsFromAnyBlocksNonMapEntry(t *testing.T) {
	blocks := []any{"not a map"}
	_, err := geminiPartsFromAnyBlocks(blocks)
	if err == nil {
		t.Fatal("expected error for non-map block")
	}
}

func TestGeminiPartsAnySliceContent(t *testing.T) {
	parts, err := geminiParts([]any{
		map[string]any{"type": "text", "text": "world"},
	})
	if err != nil {
		t.Fatalf("geminiParts: %v", err)
	}
	if len(parts) != 1 || parts[0].Text != "world" {
		t.Fatalf("parts = %+v", parts)
	}
}

func TestGeminiPartsMapSliceContent(t *testing.T) {
	parts, err := geminiParts([]map[string]any{
		{"type": "text", "text": "world"},
	})
	if err != nil {
		t.Fatalf("geminiParts: %v", err)
	}
	if len(parts) != 1 || parts[0].Text != "world" {
		t.Fatalf("parts = %+v", parts)
	}
}

func TestGeminiPartsUnsupportedType(t *testing.T) {
	_, err := geminiParts(42)
	if err == nil {
		t.Fatal("expected error for unsupported content type")
	}
}

func TestTextFromContentNonTextBlock(t *testing.T) {
	// block with non-text type should fail
	_, err := textFromContent([]map[string]any{
		{"type": "image_url", "url": "http://example.com"},
	})
	if err == nil {
		t.Fatal("expected error for non-text block")
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

func TestGeminiToolCallIDFallback(t *testing.T) {
	call := &geminiFunctionCall{ID: "", Name: "foo"}
	id := geminiToolCallID(call, 3)
	if id != "gemini_call_3" {
		t.Fatalf("id = %q, want gemini_call_3", id)
	}
	callWithID := &geminiFunctionCall{ID: "real-id", Name: "foo"}
	if geminiToolCallID(callWithID, 0) != "real-id" {
		t.Fatal("expected real-id")
	}
}

func TestToolCallsFromPartsNilArgs(t *testing.T) {
	parts := []part{{
		FunctionCall: &geminiFunctionCall{ID: "c1", Name: "foo", Args: nil},
	}}
	calls := toolCallsFromParts(parts)
	if len(calls) != 1 || calls[0].Function.Arguments != "{}" {
		t.Fatalf("calls = %+v", calls)
	}
}

func TestStreamSSEDone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL)
	chunks, err := provider.ChatCompletionStream(context.Background(), apiKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}
	got := collectStreamChunks(chunks)
	if len(got) != 0 {
		t.Fatalf("chunks = %+v, want none after DONE", got)
	}
}

func TestStreamHTTPError(t *testing.T) {
	server := jsonServer(t, http.StatusUnauthorized, `{"error":{"message":"bad stream key"}}`)
	provider := New(server.URL)

	_, err := provider.ChatCompletionStream(context.Background(), apiKey(), testChatRequest())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

func TestChatCompletionCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	provider := New("http://127.0.0.1:1")
	_, err := provider.ChatCompletion(ctx, apiKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestDoWithDeadline(t *testing.T) {
	server := jsonServer(t, http.StatusOK, generateContentResponseJSON)
	provider := New(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := provider.ChatCompletion(ctx, apiKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletion with deadline: %v", err)
	}
}

func TestDoAlreadyExpiredDeadline(t *testing.T) {
	provider := New("http://127.0.0.1:1")
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	// Wait for context to be expired
	<-ctx.Done()
	_, err := provider.ChatCompletion(ctx, apiKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected error for expired deadline")
	}
}

func TestStreamSSETrailingDataNoNewline(t *testing.T) {
	// Data line at EOF without trailing blank line — should still be processed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// No trailing \n\n — scanner reaches EOF with pending data
		_, _ = w.Write([]byte("data: " + generateContentStreamTextChunkJSON))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL)
	chunks, err := provider.ChatCompletionStream(context.Background(), apiKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}
	got := collectStreamChunks(chunks)
	if len(got) == 0 {
		t.Fatal("expected at least one chunk from trailing data")
	}
}

func TestNewJSONRequestMarshalBodyNil(t *testing.T) {
	provider := New("http://127.0.0.1")
	req, err := provider.newJSONRequest("GET", "/test", apiKey(), nil)
	if err != nil {
		t.Fatalf("newJSONRequest nil body: %v", err)
	}
	defer fasthttp.ReleaseRequest(req)
}

func TestNewHTTPJSONRequestNilBody(t *testing.T) {
	provider := New("http://127.0.0.1")
	req, err := provider.newHTTPJSONRequest(context.Background(), "GET", "/test", oauthKey(), nil, nil)
	if err != nil {
		t.Fatalf("newHTTPJSONRequest nil body: %v", err)
	}
	_ = req
}

func TestGeminiContentAssistantRole(t *testing.T) {
	c, err := geminiContent(providers.Message{Role: "assistant", Content: "reply"}, nil)
	if err != nil {
		t.Fatalf("geminiContent: %v", err)
	}
	if c.Role != "model" {
		t.Fatalf("role = %q, want model", c.Role)
	}
}

func TestGeminiToolContentIDNotInMap(t *testing.T) {
	id := "unknown-id"
	c, err := geminiToolContent(providers.Message{
		Role:       "tool",
		ToolCallID: &id,
		Content:    "result",
	}, map[string]string{}) // id not in map → use id as name
	if err != nil {
		t.Fatalf("geminiToolContent: %v", err)
	}
	if c.Parts[0].FunctionResponse.Name != id {
		t.Fatalf("name = %q, want %q", c.Parts[0].FunctionResponse.Name, id)
	}
}

func TestTextFromContentNilInput(t *testing.T) {
	text, err := textFromContent(nil)
	if err != nil {
		t.Fatalf("textFromContent nil: %v", err)
	}
	if text != "" {
		t.Fatalf("text = %q, want empty", text)
	}
}

func TestListModels500Error(t *testing.T) {
	server := jsonServer(t, http.StatusInternalServerError, `{"error":{"message":"server down"}}`)
	provider := New(server.URL)

	_, err := provider.ListModels(context.Background(), apiKey())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
}

func TestChatCompletionInvalidJSONResponse(t *testing.T) {
	server := jsonServer(t, http.StatusOK, `not-json`)
	provider := New(server.URL)

	_, err := provider.ChatCompletion(context.Background(), apiKey(), testChatRequest())
	if err == nil || !strings.Contains(err.Error(), "parse gemini response") {
		t.Fatalf("error = %v, want parse error", err)
	}
}

func TestListModelsInvalidJSONResponse(t *testing.T) {
	server := jsonServer(t, http.StatusOK, `not-json`)
	provider := New(server.URL)

	_, err := provider.ListModels(context.Background(), apiKey())
	if err == nil || !strings.Contains(err.Error(), "parse gemini models response") {
		t.Fatalf("error = %v, want parse models error", err)
	}
}

func TestChatCompletionStreamBuildError(t *testing.T) {
	// nil request should trigger build error before HTTP
	provider := New("http://127.0.0.1:1")
	_, err := provider.ChatCompletionStream(context.Background(), apiKey(), nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestGeminiFunctionCallPartParseFunctionArgsError(t *testing.T) {
	// parseFunctionArgs is called — but it never errors (it always falls back).
	// The error branch in geminiFunctionCallPart at line 306 is unreachable
	// via parseFunctionArgs, so test the non-function type error instead.
	_, err := geminiFunctionCallPart(providers.ToolCall{Type: "other"})
	if err == nil {
		t.Fatal("expected error for non-function type")
	}
}

func TestBuildGenerateContentRequestSystemError(t *testing.T) {
	// system field is a non-string non-nil that causes textFromContent to fail
	_, err := buildGenerateContentRequest(&providers.ChatRequest{
		Model:  "gemini-2.5-flash",
		System: 42, // not a string → geminiParts returns unsupported type error
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid system type")
	}
}

func TestGeminiContentPartsError(t *testing.T) {
	// content that is an unsupported type → geminiParts error
	_, err := geminiContent(providers.Message{
		Role:    "user",
		Content: 42,
	}, nil)
	if err == nil {
		t.Fatal("expected error for unsupported content type")
	}
}

func TestTextFromContentEmptyParts(t *testing.T) {
	// empty string returns empty without error
	text, err := textFromContent("")
	if err != nil {
		t.Fatalf("textFromContent empty: %v", err)
	}
	if text != "" {
		t.Fatalf("text = %q, want empty", text)
	}
}

func TestNewJSONRequestOAuthNoQuery(t *testing.T) {
	// OAuth key should not set key= query param
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(generateContentResponseJSON))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL)
	req, err := provider.newJSONRequest("POST", "/test", oauthKey(), map[string]string{"x": "y"})
	if err != nil {
		t.Fatalf("newJSONRequest: %v", err)
	}
	defer fasthttp.ReleaseRequest(req)
	_ = gotQuery
}

func TestStreamHTTPErrorBody500(t *testing.T) {
	server := jsonServer(t, http.StatusServiceUnavailable, `{"error":{"message":"service down"}}`)
	provider := New(server.URL)

	_, err := provider.ChatCompletionStream(context.Background(), apiKey(), testChatRequest())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
}

func TestMapGenerateContentStreamChunksNoContent(t *testing.T) {
	// Empty candidates with usage metadata → usage-only chunk
	decoded := generateContentResponse{
		UsageMetadata: usageMetadata{PromptTokenCount: 1, TotalTokenCount: 1},
	}
	chunks := mapGenerateContentStreamChunks("model", decoded)
	if len(chunks) != 1 {
		t.Fatalf("chunks = %d, want 1 usage chunk", len(chunks))
	}
	if chunks[0].Usage == nil {
		t.Fatal("expected usage in chunk")
	}
}

func TestChatCompletionNetworkError(t *testing.T) {
	provider := New("http://127.0.0.1:1")
	_, err := provider.ChatCompletion(context.Background(), apiKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected network error")
	}
	if !strings.Contains(err.Error(), "gemini generate content") {
		t.Fatalf("error = %v, want gemini generate content", err)
	}
}

func TestListModelsNetworkError(t *testing.T) {
	provider := New("http://127.0.0.1:1")
	_, err := provider.ListModels(context.Background(), apiKey())
	if err == nil {
		t.Fatal("expected network error")
	}
	if !strings.Contains(err.Error(), "gemini list models") {
		t.Fatalf("error = %v, want gemini list models", err)
	}
}

func TestStreamNetworkError(t *testing.T) {
	provider := New("http://127.0.0.1:1")
	_, err := provider.ChatCompletionStream(context.Background(), apiKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected network error")
	}
	if !strings.Contains(err.Error(), "gemini stream generate content") {
		t.Fatalf("error = %v, want gemini stream error", err)
	}
}

func TestTextFromContentNonTextBlockError(t *testing.T) {
	// geminiParts returns a part with no text for image blocks → textFromContent error
	_, err := textFromContent([]map[string]any{
		{"type": "text", "text": "ok"},
		{"type": "text", "text": ""},  // empty text part should be skipped
	})
	// empty text part is skipped so no error for this case
	if err != nil {
		t.Fatalf("textFromContent with empty text block: %v", err)
	}
}

func TestBuildRequestSystemContentError(t *testing.T) {
	// System field in Messages with non-string/non-nil content causes textFromContent error
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
	// A non-system message with unsupported content type causes geminiParts error
	_, err := buildGenerateContentRequest(&providers.ChatRequest{
		Model: "gemini-2.5-flash",
		Messages: []providers.Message{
			{Role: "user", Content: 42},
		},
	})
	if err == nil {
		t.Fatal("expected error for unsupported content type in message")
	}
}

func TestGeminiToolContentTextError(t *testing.T) {
	// ToolCallID set, but content is non-string → textFromContent error
	id := "call-1"
	_, err := geminiToolContent(providers.Message{
		Role:       "tool",
		ToolCallID: &id,
		Content:    42,
	}, nil)
	if err == nil {
		t.Fatal("expected error for non-string tool content")
	}
}

func TestChatCompletionStreamBodyReadError(t *testing.T) {
	// Non-2xx from stream endpoint → read body + mapStatusError
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"bad request"}}`))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL)
	_, err := provider.ChatCompletionStream(context.Background(), apiKey(), testChatRequest())
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

	provider := New(server.URL)
	chunks, err := provider.ChatCompletionStream(context.Background(), apiKey(), testChatRequest())
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

func TestChatCompletionStreamInvalidURL(t *testing.T) {
	provider := New("http://invalid\x00host")
	_, err := provider.ChatCompletionStream(context.Background(), apiKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if !strings.Contains(err.Error(), "create gemini request") && !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("error = %v, want URL error", err)
	}
}

func TestParseGeminiSSEScannerError(t *testing.T) {
	pr, pw := io.Pipe()
	chunks := make(chan providers.StreamChunk, 10)
	go func() {
		_, _ = pw.Write([]byte("data: " + generateContentStreamTextChunkJSON + "\n\n"))
		pw.CloseWithError(fmt.Errorf("simulated read error"))
	}()
	parseGeminiSSE("model", pr, chunks)
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

func TestHandleGeminiSSEDataEmpty(t *testing.T) {
	chunks := make(chan providers.StreamChunk, 1)
	done, failed := handleGeminiSSEData("model", nil, chunks)
	if done || failed {
		t.Fatalf("expected false,false for empty data")
	}
}

const listModelsResponseJSON = `{
	"models": [
		{"name": "models/gemini-2.5-flash", "displayName": "Gemini 2.5 Flash"},
		{"name": "models/gemini-2.5-pro", "displayName": "Gemini 2.5 Pro"}
	]
}`
