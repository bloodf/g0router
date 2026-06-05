package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/api"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/providers/gemini"
	"github.com/bloodf/g0router/internal/providers/openai"
	"github.com/bloodf/g0router/internal/proxy"
	"github.com/valyala/fasthttp"
)

type fakeEngine struct {
	response       *providers.ChatResponse
	stream         <-chan providers.StreamChunk
	models         []providers.Model
	err            error
	streamErr      error
	received       *providers.ChatRequest
	streamReceived *providers.ChatRequest
	dispatchCtx    context.Context
	streamCtx      context.Context
	modelsCtx      context.Context
}

func (f *fakeEngine) Dispatch(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	f.dispatchCtx = ctx
	f.received = req
	return f.response, f.err
}

func (f *fakeEngine) DispatchStream(ctx context.Context, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	f.streamCtx = ctx
	f.streamReceived = req
	return f.stream, f.streamErr
}

func (f *fakeEngine) ListModels(ctx context.Context) ([]providers.Model, error) {
	f.modelsCtx = ctx
	return f.models, f.err
}

type fakeValidator struct {
	valid bool
}

func (f fakeValidator) ValidateAPIKey(key, secret string) (bool, error) {
	return f.valid && key == "g0r_valid" && secret == "test-secret", nil
}

func TestSyncInference(t *testing.T) {
	engine := &fakeEngine{response: chatResponse()}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/chat/completions", `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}],"stream":false}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}

	var decoded providers.ChatResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if decoded.ID != "chatcmpl-test" {
		t.Fatalf("ID = %q, want chatcmpl-test", decoded.ID)
	}
	if decoded.Choices[0].Message.Content != "hello back" {
		t.Fatalf("content = %#v, want hello back", decoded.Choices[0].Message.Content)
	}
	if engine.received == nil || engine.received.Model != "gpt-4o" {
		t.Fatalf("engine received = %+v", engine.received)
	}
	if _, ok := engine.dispatchCtx.(*fasthttp.RequestCtx); ok {
		t.Fatalf("dispatch context must be detached from the pooled *fasthttp.RequestCtx to avoid use-after-recycle, got %T", engine.dispatchCtx)
	}
}

func TestStreamInference(t *testing.T) {
	role := "assistant"
	content := "hello"
	finish := "stop"
	chunks := make(chan providers.StreamChunk, 3)
	chunks <- providers.StreamChunk{
		ID:      "chatcmpl-stream",
		Object:  "chat.completion.chunk",
		Created: 1710000000,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{{Index: 0, Delta: providers.StreamDelta{Role: &role}}},
	}
	chunks <- providers.StreamChunk{
		ID:      "chatcmpl-stream",
		Object:  "chat.completion.chunk",
		Created: 1710000000,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{{Index: 0, Delta: providers.StreamDelta{Content: &content}}},
	}
	chunks <- providers.StreamChunk{
		ID:      "chatcmpl-stream",
		Object:  "chat.completion.chunk",
		Created: 1710000000,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{{Index: 0, FinishReason: &finish}},
	}
	close(chunks)

	engine := &fakeEngine{stream: chunks}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/chat/completions", `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}],"stream":true}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}
	bodyText := string(body)
	if strings.Count(bodyText, "data: ") != 4 {
		t.Fatalf("SSE data count = %d, want 4; body=%s", strings.Count(bodyText, "data: "), bodyText)
	}
	if !strings.Contains(bodyText, `"content":"hello"`) {
		t.Fatalf("stream body missing content chunk: %s", bodyText)
	}
	if !strings.HasSuffix(bodyText, "data: [DONE]\n\n") {
		t.Fatalf("stream body should end with [DONE], got %q", bodyText)
	}
	if engine.streamReceived == nil || engine.streamReceived.Stream == nil || !*engine.streamReceived.Stream {
		t.Fatalf("stream request = %+v", engine.streamReceived)
	}
	if _, ok := engine.streamCtx.(*fasthttp.RequestCtx); ok {
		t.Fatalf("stream context must be detached from the pooled *fasthttp.RequestCtx to avoid use-after-recycle, got %T", engine.streamCtx)
	}
}

func TestStreamInferenceWritesSanitizedStreamError(t *testing.T) {
	chunks := make(chan providers.StreamChunk, 1)
	chunks <- providers.StreamChunk{
		Error: &providers.StreamError{
			Message: "upstream stream failed with sk-live-secret",
			Type:    "server_error",
			Code:    "upstream_stream_malformed",
		},
	}
	close(chunks)
	engine := &fakeEngine{stream: chunks}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/chat/completions", `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}],"stream":true}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	bodyText := string(body)
	if !strings.Contains(bodyText, `"code":"upstream_stream_malformed"`) {
		t.Fatalf("stream body missing stream error code: %s", bodyText)
	}
	if strings.Contains(bodyText, "sk-live-secret") {
		t.Fatalf("stream body leaked provider detail: %s", bodyText)
	}
	if strings.Contains(bodyText, "data: [DONE]") {
		t.Fatalf("stream body should stop after error without [DONE]: %s", bodyText)
	}
}

func TestResponsesStreamingTranslatesChatStream(t *testing.T) {
	role := "assistant"
	content := "hello"
	finish := "stop"
	chunks := make(chan providers.StreamChunk, 4)
	chunks <- providers.StreamChunk{
		ID:      "resp-stream",
		Object:  "chat.completion.chunk",
		Created: 1710000000,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{{Index: 0, Delta: providers.StreamDelta{Role: &role}}},
	}
	chunks <- providers.StreamChunk{
		ID:      "resp-stream",
		Object:  "chat.completion.chunk",
		Created: 1710000000,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{{Index: 0, Delta: providers.StreamDelta{Content: &content}}},
	}
	chunks <- providers.StreamChunk{
		ID:      "resp-stream",
		Object:  "chat.completion.chunk",
		Created: 1710000000,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{{Index: 0, FinishReason: &finish}},
		Usage:   &providers.Usage{PromptTokens: 3, CompletionTokens: 2, TotalTokens: 5},
	}
	close(chunks)
	engine := &fakeEngine{stream: chunks}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/responses", `{"model":"gpt-4o","input":[{"role":"user","content":[{"type":"input_text","text":"hello"}]}],"stream":true}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}
	bodyText := string(body)
	if !strings.Contains(bodyText, `event: response.output_text.delta`) || !strings.Contains(bodyText, `"delta":"hello"`) {
		t.Fatalf("stream body missing response text delta: %s", bodyText)
	}
	if !strings.Contains(bodyText, `event: response.output_text.done`) || !strings.Contains(bodyText, `"text":"hello"`) {
		t.Fatalf("stream body missing response text done: %s", bodyText)
	}
	if !strings.Contains(bodyText, `event: response.completed`) || !strings.Contains(bodyText, `"output_text":"hello"`) {
		t.Fatalf("stream body missing completed response: %s", bodyText)
	}
	if !strings.HasSuffix(bodyText, "data: [DONE]\n\n") {
		t.Fatalf("stream body should end with [DONE], got %q", bodyText)
	}
	if engine.streamReceived == nil || engine.streamReceived.Stream == nil || !*engine.streamReceived.Stream {
		t.Fatalf("stream request = %+v", engine.streamReceived)
	}
}

func TestMessagesStreamingTranslatesChatStream(t *testing.T) {
	role := "assistant"
	content := "hello"
	finish := "stop"
	chunks := make(chan providers.StreamChunk, 3)
	chunks <- providers.StreamChunk{
		ID:      "msg-stream",
		Object:  "chat.completion.chunk",
		Created: 1710000000,
		Model:   "claude-sonnet-4",
		Choices: []providers.StreamChoice{{Index: 0, Delta: providers.StreamDelta{Role: &role}}},
	}
	chunks <- providers.StreamChunk{
		ID:      "msg-stream",
		Object:  "chat.completion.chunk",
		Created: 1710000000,
		Model:   "claude-sonnet-4",
		Choices: []providers.StreamChoice{{Index: 0, Delta: providers.StreamDelta{Content: &content}}},
	}
	chunks <- providers.StreamChunk{
		ID:      "msg-stream",
		Object:  "chat.completion.chunk",
		Created: 1710000000,
		Model:   "claude-sonnet-4",
		Choices: []providers.StreamChoice{{Index: 0, FinishReason: &finish}},
		Usage:   &providers.Usage{PromptTokens: 3, CompletionTokens: 2, TotalTokens: 5},
	}
	close(chunks)
	engine := &fakeEngine{stream: chunks}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/messages", `{"model":"claude-sonnet-4","messages":[{"role":"user","content":"hello"}],"stream":true}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}
	bodyText := string(body)
	if !strings.Contains(bodyText, `event: message_start`) || !strings.Contains(bodyText, `"type":"message_start"`) {
		t.Fatalf("stream body missing message_start: %s", bodyText)
	}
	if !strings.Contains(bodyText, `event: content_block_delta`) || !strings.Contains(bodyText, `"text":"hello"`) {
		t.Fatalf("stream body missing text delta: %s", bodyText)
	}
	if !strings.Contains(bodyText, `event: message_delta`) || !strings.Contains(bodyText, `"stop_reason":"end_turn"`) {
		t.Fatalf("stream body missing message delta: %s", bodyText)
	}
	if !strings.Contains(bodyText, `event: message_stop`) {
		t.Fatalf("stream body missing message_stop: %s", bodyText)
	}
	if strings.Contains(bodyText, "data: [DONE]") {
		t.Fatalf("messages stream should not use OpenAI [DONE] sentinel: %s", bodyText)
	}
	if !strings.HasSuffix(bodyText, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n") {
		t.Fatalf("messages stream should end with message_stop, got %q", bodyText)
	}
	if engine.streamReceived == nil || engine.streamReceived.Stream == nil || !*engine.streamReceived.Stream {
		t.Fatalf("stream request = %+v", engine.streamReceived)
	}
}

func TestMessagesStreamingTranslatesToolCalls(t *testing.T) {
	role := "assistant"
	content := "let me check"
	finish := "tool_calls"
	chunks := make(chan providers.StreamChunk, 5)
	chunks <- providers.StreamChunk{
		ID:      "msg-tool",
		Model:   "claude-sonnet-4",
		Choices: []providers.StreamChoice{{Index: 0, Delta: providers.StreamDelta{Role: &role}}},
	}
	chunks <- providers.StreamChunk{
		ID:      "msg-tool",
		Model:   "claude-sonnet-4",
		Choices: []providers.StreamChoice{{Index: 0, Delta: providers.StreamDelta{Content: &content}}},
	}
	chunks <- providers.StreamChunk{
		ID:    "msg-tool",
		Model: "claude-sonnet-4",
		Choices: []providers.StreamChoice{{Index: 0, Delta: providers.StreamDelta{ToolCalls: []providers.ToolCall{{
			ID:       "call_1",
			Type:     "function",
			Function: providers.ToolCallFunc{Name: "get_weather", Arguments: `{"loc`},
		}}}}},
	}
	chunks <- providers.StreamChunk{
		ID:    "msg-tool",
		Model: "claude-sonnet-4",
		Choices: []providers.StreamChoice{{Index: 0, Delta: providers.StreamDelta{ToolCalls: []providers.ToolCall{{
			Function: providers.ToolCallFunc{Arguments: `ation":"SF"}`},
		}}}}},
	}
	chunks <- providers.StreamChunk{
		ID:      "msg-tool",
		Model:   "claude-sonnet-4",
		Choices: []providers.StreamChoice{{Index: 0, FinishReason: &finish}},
		Usage:   &providers.Usage{PromptTokens: 5, CompletionTokens: 4, TotalTokens: 9},
	}
	close(chunks)
	engine := &fakeEngine{stream: chunks}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/messages", `{"model":"claude-sonnet-4","messages":[{"role":"user","content":"hi"}],"stream":true}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	bodyText := string(body)

	order := []string{
		`event: message_start`,
		`"type":"text"`,
		`"type":"text_delta"`,
		`event: content_block_stop`,
		`"id":"call_1"`,
		`"name":"get_weather"`,
		`"type":"tool_use"`,
		`"partial_json":"{\"loc"`,
		`"type":"input_json_delta"`,
		`"partial_json":"ation\":\"SF\"}"`,
		`event: content_block_stop`,
		`"stop_reason":"tool_use"`,
		`event: message_stop`,
	}
	prev := 0
	for _, frag := range order {
		idx := strings.Index(bodyText[prev:], frag)
		if idx < 0 {
			t.Fatalf("stream body missing %q in order; body=%s", frag, bodyText)
		}
		prev += idx + len(frag)
	}

	if strings.Contains(bodyText, "data: [DONE]") {
		t.Fatalf("messages stream should not use OpenAI [DONE] sentinel: %s", bodyText)
	}
	if !strings.HasSuffix(bodyText, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n") {
		t.Fatalf("messages stream should end with message_stop, got %q", bodyText)
	}
}

func TestInferenceInvalidJSON(t *testing.T) {
	engine := &fakeEngine{response: chatResponse()}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/chat/completions", `{"model":`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", resp.StatusCode, body)
	}
}

func TestInferenceUnknownModel(t *testing.T) {
	engine := &fakeEngine{err: proxy.ErrProviderNotFound}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/chat/completions", `{"model":"nonexistent","messages":[{"role":"user","content":"hello"}]}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", resp.StatusCode, body)
	}
}

func TestInferenceQuotaExhausted(t *testing.T) {
	engine := &fakeEngine{err: proxy.ErrQuotaExhausted}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/chat/completions", `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429; body=%s", resp.StatusCode, body)
	}
}

func TestInferenceDispatchErrorIsSanitizedOpenAIError(t *testing.T) {
	engine := &fakeEngine{err: errors.New("chat completion: upstream said Authorization: Bearer sk-live-secret")}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/chat/completions", `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502; body=%s", resp.StatusCode, body)
	}
	assertOpenAIError(t, body, "upstream provider error", "server_error", "upstream_error")
	if strings.Contains(string(body), "sk-live-secret") || strings.Contains(string(body), "Authorization") || strings.Contains(string(body), "chat completion") {
		t.Fatalf("response leaked upstream error detail: %s", body)
	}
}

func TestStreamInferenceDispatchErrorIsSanitizedOpenAIError(t *testing.T) {
	engine := &fakeEngine{streamErr: errors.New("stream failed with api_key=sk-live-secret")}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/chat/completions", `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}],"stream":true}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502; body=%s", resp.StatusCode, body)
	}
	assertOpenAIError(t, body, "upstream provider error", "server_error", "upstream_error")
	if strings.Contains(string(body), "sk-live-secret") || strings.Contains(string(body), "stream failed") {
		t.Fatalf("response leaked upstream stream error detail: %s", body)
	}
	if engine.streamReceived == nil || engine.streamReceived.Stream == nil || !*engine.streamReceived.Stream {
		t.Fatalf("stream request = %+v", engine.streamReceived)
	}
}

func TestStreamInferenceUnsupportedProviderUsesStableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "gemini sentinel", err: gemini.ErrUnsupported},
		{name: "shared provider sentinel", err: providers.ErrStreamingUnsupported},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			engine := &fakeEngine{streamErr: fmt.Errorf("chat completion stream: %w", tc.err)}
			_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

			resp, body := postJSON(t, baseURL+"/v1/chat/completions", `{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"hello"}],"stream":true}`, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNotImplemented {
				t.Fatalf("status = %d, want 501; body=%s", resp.StatusCode, body)
			}
			assertOpenAIError(t, body, "streaming unsupported for provider", "invalid_request_error", "streaming_unsupported")
			if strings.Contains(string(body), "gemini unsupported operation") || strings.Contains(string(body), "chat completion stream") {
				t.Fatalf("response leaked provider unsupported detail: %s", body)
			}
		})
	}
}

func TestInferenceClassifiedUpstreamAuthErrorKeepsStatusAndIsSanitized(t *testing.T) {
	engine := &fakeEngine{err: fmt.Errorf("chat completion: %w: Authorization: Bearer sk-live-secret", openai.ErrAuth)}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/chat/completions", `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", resp.StatusCode, body)
	}
	assertOpenAIError(t, body, "upstream provider authentication failed", "invalid_request_error", "upstream_auth_error")
	if strings.Contains(string(body), "sk-live-secret") || strings.Contains(string(body), "Authorization") {
		t.Fatalf("response leaked upstream auth detail: %s", body)
	}
}

func TestStreamInferenceClassifiedUpstreamRateLimitKeepsStatusAndIsSanitized(t *testing.T) {
	engine := &fakeEngine{streamErr: fmt.Errorf("chat completion stream: %w", &openai.RateLimitError{Message: "retry later with api_key=sk-live-secret"})}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/chat/completions", `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}],"stream":true}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429; body=%s", resp.StatusCode, body)
	}
	assertOpenAIError(t, body, "upstream provider rate limit", "rate_limit_error", "upstream_rate_limit")
	if strings.Contains(string(body), "sk-live-secret") || strings.Contains(string(body), "retry later") {
		t.Fatalf("response leaked upstream rate-limit detail: %s", body)
	}
}

func TestInferenceKnownDispatchErrorsUseStableOpenAIErrorCodes(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		status  int
		message string
		typ     string
		code    string
	}{
		{
			name:    "provider not found",
			err:     proxy.ErrProviderNotFound,
			status:  http.StatusNotFound,
			message: "provider not found",
			typ:     "invalid_request_error",
			code:    "provider_not_found",
		},
		{
			name:    "no connections",
			err:     proxy.ErrNoConnections,
			status:  http.StatusServiceUnavailable,
			message: "no active provider connections",
			typ:     "server_error",
			code:    "no_active_connections",
		},
		{
			name:    "quota exhausted",
			err:     proxy.ErrQuotaExhausted,
			status:  http.StatusTooManyRequests,
			message: "quota exhausted",
			typ:     "rate_limit_error",
			code:    "quota_exhausted",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			engine := &fakeEngine{err: tc.err}
			_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

			resp, body := postJSON(t, baseURL+"/v1/chat/completions", `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`, nil)
			defer resp.Body.Close()

			if resp.StatusCode != tc.status {
				t.Fatalf("status = %d, want %d; body=%s", resp.StatusCode, tc.status, body)
			}
			assertOpenAIError(t, body, tc.message, tc.typ, tc.code)
		})
	}
}

func TestInferenceNoAuth(t *testing.T) {
	engine := &fakeEngine{response: chatResponse()}
	_, baseURL := startInferenceServer(t, api.ServerConfig{
		Version:         "test",
		RequireAPIKey:   true,
		APIKeySecret:    "test-secret",
		APIKeyValidator: fakeValidator{valid: true},
		InferenceEngine: engine,
	})

	resp, body := postJSON(t, baseURL+"/v1/chat/completions", `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", resp.StatusCode, body)
	}
	if engine.received != nil {
		t.Fatal("engine should not be called when auth fails")
	}
}

func TestGetModels(t *testing.T) {
	engine := &fakeEngine{models: []providers.Model{
		{ID: "gpt-4o", Object: "model", Created: 1710000000, OwnedBy: "openai", Provider: providers.ProviderOpenAI},
		{ID: "claude-3-5-sonnet", Object: "model", Created: 1710000001, OwnedBy: "anthropic", Provider: providers.ProviderAnthropic},
	}}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, err := httpClient().Get(baseURL + "/v1/models")
	if err != nil {
		t.Fatalf("GET /v1/models: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	var decoded struct {
		Object string            `json:"object"`
		Data   []providers.Model `json:"data"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal models: %v", err)
	}
	if decoded.Object != "list" {
		t.Fatalf("object = %q, want list", decoded.Object)
	}
	if len(decoded.Data) != 2 {
		t.Fatalf("models len = %d, want 2", len(decoded.Data))
	}
	if decoded.Data[0].ID != "gpt-4o" || decoded.Data[1].ID != "claude-3-5-sonnet" {
		t.Fatalf("models = %+v", decoded.Data)
	}
	if _, ok := engine.modelsCtx.(*fasthttp.RequestCtx); ok {
		t.Fatalf("models context must be detached from the pooled *fasthttp.RequestCtx to avoid use-after-recycle, got %T", engine.modelsCtx)
	}
}

func TestMessagesRejectsAnthropicNativeTools(t *testing.T) {
	engine := &fakeEngine{response: chatResponse()}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/messages", `{"model":"claude-sonnet-4","tools":[{"name":"lookup","input_schema":{"type":"object"}}],"messages":[{"role":"user","content":"hello"}]}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501; body=%s", resp.StatusCode, body)
	}
	if engine.received != nil {
		t.Fatalf("engine request = %+v, want no dispatch for unsupported native tools", engine.received)
	}
}

func TestMessagesRejectsAnthropicNativeToolChoice(t *testing.T) {
	engine := &fakeEngine{response: chatResponse()}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/messages", `{"model":"claude-sonnet-4","tool_choice":{"type":"tool","name":"lookup"},"messages":[{"role":"user","content":"hello"}]}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501; body=%s", resp.StatusCode, body)
	}
	if engine.received != nil {
		t.Fatalf("engine request = %+v, want no dispatch for unsupported native tool choice", engine.received)
	}
}

func TestMessagesRejectsAnthropicToolUseBlocks(t *testing.T) {
	engine := &fakeEngine{response: chatResponse()}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/messages", `{"model":"claude-sonnet-4","messages":[{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"lookup","input":{"query":"docs"}}]}]}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501; body=%s", resp.StatusCode, body)
	}
	if engine.received != nil {
		t.Fatalf("engine request = %+v, want no dispatch for unsupported native tool use", engine.received)
	}
}

func TestMessagesRejectsAnthropicToolResultBlocks(t *testing.T) {
	engine := &fakeEngine{response: chatResponse()}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/messages", `{"model":"claude-sonnet-4","messages":[{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"value"}]}]}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501; body=%s", resp.StatusCode, body)
	}
	if engine.received != nil {
		t.Fatalf("engine request = %+v, want no dispatch for unsupported native tool results", engine.received)
	}
}

func TestMessagesResponsePreservesToolUseBlocks(t *testing.T) {
	finish := "tool_calls"
	engine := &fakeEngine{response: &providers.ChatResponse{
		ID:      "chatcmpl-tools",
		Object:  "chat.completion",
		Created: 1710000000,
		Model:   "claude-sonnet-4",
		Choices: []providers.Choice{{
			Message: providers.Message{
				Role: "assistant",
				ToolCalls: []providers.ToolCall{{
					ID:   "toolu_lookup",
					Type: "function",
					Function: providers.ToolCallFunc{
						Name:      "lookup",
						Arguments: `{"query":"docs"}`,
					},
				}},
			},
			FinishReason: &finish,
		}},
		Usage: &providers.Usage{PromptTokens: 3, CompletionTokens: 2, TotalTokens: 5},
	}}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/messages", `{"model":"claude-sonnet-4","messages":[{"role":"user","content":"hello"}]}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	var decoded struct {
		Content []struct {
			Type  string          `json:"type"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
		StopReason *string `json:"stop_reason"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal response: %v; body=%s", err, body)
	}
	if len(decoded.Content) != 1 {
		t.Fatalf("content len = %d, want 1: %+v", len(decoded.Content), decoded.Content)
	}
	if decoded.Content[0].Type != "tool_use" || decoded.Content[0].ID != "toolu_lookup" || decoded.Content[0].Name != "lookup" {
		t.Fatalf("tool use content = %+v", decoded.Content[0])
	}
	if string(decoded.Content[0].Input) != `{"query":"docs"}` {
		t.Fatalf("tool input = %s, want query JSON", decoded.Content[0].Input)
	}
	if decoded.StopReason == nil || *decoded.StopReason != "tool_use" {
		t.Fatalf("stop reason = %+v, want tool_use", decoded.StopReason)
	}
}

func chatResponse() *providers.ChatResponse {
	finish := "stop"
	return &providers.ChatResponse{
		ID:      "chatcmpl-test",
		Object:  "chat.completion",
		Created: 1710000000,
		Model:   "gpt-4o",
		Choices: []providers.Choice{{
			Index: 0,
			Message: providers.Message{
				Role:    "assistant",
				Content: "hello back",
			},
			FinishReason: &finish,
		}},
		Usage: &providers.Usage{PromptTokens: 3, CompletionTokens: 2, TotalTokens: 5},
	}
}

func assertOpenAIError(t *testing.T, body []byte, message string, typ string, code string) {
	t.Helper()

	var decoded struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal error response: %v; body=%s", err, body)
	}
	if decoded.Error.Message != message || decoded.Error.Type != typ || decoded.Error.Code != code {
		t.Fatalf("error = %+v, want message=%q type=%q code=%q", decoded.Error, message, typ, code)
	}
}

func postJSON(t *testing.T, url string, body string, headers map[string]string) (*http.Response, []byte) {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		t.Fatalf("read body: %v", err)
	}
	resp.Body = io.NopCloser(bytes.NewReader(data))
	return resp, data
}

func startInferenceServer(t *testing.T, config api.ServerConfig) (*api.Server, string) {
	t.Helper()

	srv := api.NewServer(config)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("Serve: %v", err)
		}
	}()
	t.Cleanup(func() { _ = srv.Stop() })

	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener addr is %T, want *net.TCPAddr", ln.Addr())
	}
	return srv, "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(tcpAddr.Port))
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 2 * time.Second}
}
