package handlers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// coverageEngine is a minimal InferenceEngine fake for exercising handler
// error and success branches that the external server tests do not reach.
type coverageEngine struct {
	resp      *providers.ChatResponse
	stream    <-chan providers.StreamChunk
	models    []providers.Model
	err       error
	streamErr error
}

func (e *coverageEngine) Dispatch(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	return e.resp, e.err
}

func (e *coverageEngine) DispatchStream(ctx context.Context, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return e.stream, e.streamErr
}

func (e *coverageEngine) ListModels(ctx context.Context) ([]providers.Model, error) {
	return e.models, e.err
}

// --- Health ---

func TestHealthReturnsStatusAndVersion(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Health(ctx, "v1.2.3")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var decoded map[string]string
	decodeJSON(t, body, &decoded)
	if decoded["status"] != "ok" || decoded["version"] != "v1.2.3" {
		t.Fatalf("health = %+v, want status ok / version v1.2.3", decoded)
	}
}

// --- Models ---

func TestModelsNilEngineUnavailable(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Models(ctx, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestModelsSuccess(t *testing.T) {
	engine := &coverageEngine{models: []providers.Model{{ID: "gpt-4o", Object: "model"}}}
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Models(ctx, engine)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var decoded modelsResponse
	decodeJSON(t, body, &decoded)
	if decoded.Object != "list" || len(decoded.Data) != 1 || decoded.Data[0].ID != "gpt-4o" {
		t.Fatalf("models = %+v, want list with gpt-4o", decoded)
	}
}

func TestModelsListErrorSanitized(t *testing.T) {
	engine := &coverageEngine{err: errors.New("sqlite: no such table models")}
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Models(ctx, engine)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoInternalDetail(t, body)
}

// --- ConnectionTest ---

func TestConnectionTestSuccess(t *testing.T) {
	s := newHandlerStore(t)
	apiKey := "sk-test"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai", Name: "primary", AuthType: store.AuthTypeAPIKey, APIKey: &apiKey, IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	conns, err := s.GetConnections("openai")
	if err != nil || len(conns) != 1 {
		t.Fatalf("GetConnections: %v len=%d", err, len(conns))
	}
	id := conns[0].ID

	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ConnectionTest(ctx, s, id)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var decoded map[string]any
	decodeJSON(t, body, &decoded)
	if decoded["ok"] != true || decoded["provider"] != "openai" || decoded["name"] != "primary" {
		t.Fatalf("test result = %+v", decoded)
	}
	assertNoCredentialFields(t, body)
}

func TestConnectionTestNilStore(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ConnectionTest(ctx, nil, "id")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestConnectionTestMethodNotAllowed(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ConnectionTest(ctx, s, "id")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", ctx.Response.StatusCode())
	}
}

func TestConnectionTestMissingID(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ConnectionTest(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestConnectionTestNotFound(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ConnectionTest(ctx, s, "missing")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
}

// --- Connections additional branches ---

func TestConnectionsNilStore(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, nil, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestConnectionsMethodNotAllowed(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPatch, "", func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", ctx.Response.StatusCode())
	}
}

func TestConnectionsPutMissingID(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"provider":"openai"}`, func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestConnectionsDeleteMissingID(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestConnectionsPutNotFound(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodPut, `{"provider":"openai","name":"x","auth_type":"api_key"}`, func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "missing")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestConnectionsPutInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{`, func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "some-id")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestConnectionsStoreClosedErrorPathsSanitized(t *testing.T) {
	s := newHandlerStore(t)
	apiKey := "sk-test"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai", Name: "primary", AuthType: store.AuthTypeAPIKey, APIKey: &apiKey, IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	conns, _ := s.GetConnections("openai")
	id := conns[0].ID
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	for _, tc := range []struct {
		name   string
		method string
		body   string
		run    func(ctx *fasthttp.RequestCtx)
	}{
		{"put", fasthttp.MethodPut, `{"provider":"openai","name":"x","auth_type":"api_key"}`, func(ctx *fasthttp.RequestCtx) { Connections(ctx, s, id) }},
		{"delete", fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) { Connections(ctx, s, id) }},
		{"test", fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) { ConnectionTest(ctx, s, id) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, body := runHandler(t, tc.method, tc.body, tc.run)
			if ctx.Response.StatusCode() < 500 && ctx.Response.StatusCode() != fasthttp.StatusNotFound {
				t.Fatalf("status = %d, want >=500 or 404; body=%s", ctx.Response.StatusCode(), body)
			}
			assertNoInternalDetail(t, body)
		})
	}
}

// --- redactProviderSpecificValue map[string]string and slice branches ---

func TestRedactProviderSpecificValueHandlesStringMapsAndSlices(t *testing.T) {
	in := map[string]any{
		"headers": map[string]string{"X-Api-Key": "secret", "X-Region": "us"},
		"list":    []any{map[string]any{"password": "p", "mode": "ro"}, "plain"},
		"scalar":  42,
	}
	out := redactProviderSpecificData(in)

	headers := out["headers"].(map[string]string)
	if _, ok := headers["X-Api-Key"]; ok {
		t.Fatalf("string-map secret not redacted: %+v", headers)
	}
	if headers["X-Region"] != "us" {
		t.Fatalf("string-map non-secret dropped: %+v", headers)
	}
	list := out["list"].([]any)
	first := list[0].(map[string]any)
	if _, ok := first["password"]; ok {
		t.Fatalf("slice nested secret not redacted: %+v", first)
	}
	if first["mode"] != "ro" || list[1] != "plain" {
		t.Fatalf("slice non-secret values altered: %+v", list)
	}
	if out["scalar"] != 42 {
		t.Fatalf("scalar altered: %+v", out["scalar"])
	}
}

func TestRedactProviderSpecificDataNilReturnsNil(t *testing.T) {
	if redactProviderSpecificData(nil) != nil {
		t.Fatal("nil input should return nil")
	}
}

// --- inference translation helpers (direct unit tests) ---

func TestToolCallInputVariants(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", `{}`},
		{"   ", `{}`},
		{`{"a":1}`, `{"a":1}`},
		{`not-json`, `{"arguments":"not-json"}`},
	}
	for _, c := range cases {
		got := string(toolCallInput(c.in))
		if got != c.want {
			t.Fatalf("toolCallInput(%q) = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestAnthropicStreamStopReasonMapping(t *testing.T) {
	if anthropicStreamStopReason(nil) != "" {
		t.Fatal("nil reason should map to empty string")
	}
	for in, want := range map[string]string{
		"stop":       "end_turn",
		"length":     "max_tokens",
		"tool_calls": "tool_use",
		"other":      "other",
	} {
		reason := in
		if got := anthropicStreamStopReason(&reason); got != want {
			t.Fatalf("anthropicStreamStopReason(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAnthropicStopReasonMapping(t *testing.T) {
	if anthropicStopReason(nil) != nil {
		t.Fatal("nil reason should map to nil")
	}
	tool := "tool_calls"
	if got := anthropicStopReason(&tool); got == nil || *got != "tool_use" {
		t.Fatalf("tool_calls should map to tool_use, got %v", got)
	}
	stop := "stop"
	if got := anthropicStopReason(&stop); got == nil || *got != "stop" {
		t.Fatalf("stop should pass through, got %v", got)
	}
}

func TestAnthropicStreamUsageNilAndPresent(t *testing.T) {
	if got := anthropicStreamUsage(nil); got["output_tokens"] != 0 {
		t.Fatalf("nil usage = %+v, want output_tokens 0", got)
	}
	got := anthropicStreamUsage(&providers.Usage{CompletionTokens: 7})
	if got["output_tokens"] != 7 {
		t.Fatalf("usage = %+v, want output_tokens 7", got)
	}
}

func TestStreamResponseUsageNilAndPresent(t *testing.T) {
	if streamResponseUsage(nil) != nil {
		t.Fatal("nil usage should map to nil")
	}
	got := streamResponseUsage(&providers.Usage{PromptTokens: 3, CompletionTokens: 4, TotalTokens: 7})
	if got == nil || got.InputTokens != 3 || got.OutputTokens != 4 || got.TotalTokens != 7 {
		t.Fatalf("usage = %+v, want 3/4/7", got)
	}
}

func TestShouldStartAnthropicMessageVariants(t *testing.T) {
	if shouldStartAnthropicMessage(providers.StreamChunk{}) {
		t.Fatal("empty chunk should not start message")
	}
	if !shouldStartAnthropicMessage(providers.StreamChunk{ID: "x"}) {
		t.Fatal("chunk with ID should start message")
	}
	if !shouldStartAnthropicMessage(providers.StreamChunk{Model: "m"}) {
		t.Fatal("chunk with Model should start message")
	}
	if !shouldStartAnthropicMessage(providers.StreamChunk{Usage: &providers.Usage{}}) {
		t.Fatal("chunk with Usage should start message")
	}
	role := "assistant"
	if !shouldStartAnthropicMessage(providers.StreamChunk{Choices: []providers.StreamChoice{{Delta: providers.StreamDelta{Role: &role}}}}) {
		t.Fatal("chunk with role delta should start message")
	}
	emptyChoice := providers.StreamChunk{Choices: []providers.StreamChoice{{}}}
	if shouldStartAnthropicMessage(emptyChoice) {
		t.Fatal("chunk with only empty choice should not start message")
	}
}

func TestMessageContentTextVariants(t *testing.T) {
	if messageContentText(nil) != "" {
		t.Fatal("nil content should be empty")
	}
	if messageContentText("hi") != "hi" {
		t.Fatal("string content passthrough failed")
	}
	if messageContentText(123) != "123" {
		t.Fatal("non-string content should be stringified")
	}
}

func TestAnthropicToolInputArgumentsVariants(t *testing.T) {
	if anthropicToolInputArguments(nil) != "{}" {
		t.Fatal("nil input should default to {}")
	}
	if anthropicToolInputArguments(json.RawMessage("null")) != "{}" {
		t.Fatal("null input should default to {}")
	}
	if anthropicToolInputArguments(json.RawMessage(`{"a":1}`)) != `{"a":1}` {
		t.Fatal("object input should pass through")
	}
}

func TestAnthropicToolResultTextVariants(t *testing.T) {
	if anthropicToolResultText(nil) != "" {
		t.Fatal("nil should be empty")
	}
	if anthropicToolResultText(json.RawMessage("null")) != "" {
		t.Fatal("null should be empty")
	}
	if anthropicToolResultText(json.RawMessage(`"plain"`)) != "plain" {
		t.Fatal("quoted string should unquote")
	}
	if anthropicToolResultText(json.RawMessage(`[{"type":"text","text":"a"},{"type":"text","text":"b"}]`)) != "ab" {
		t.Fatal("block array should concatenate text")
	}
	// Malformed quoted string falls back to raw.
	if got := anthropicToolResultText(json.RawMessage(`"unterminated`)); got != `"unterminated` {
		t.Fatalf("malformed quoted = %q, want raw passthrough", got)
	}
	// Non-array, non-quoted scalar passes through raw.
	if got := anthropicToolResultText(json.RawMessage(`123`)); got != "123" {
		t.Fatalf("scalar = %q, want 123", got)
	}
}

func TestTranslateAnthropicToolChoiceVariants(t *testing.T) {
	if got, err := translateAnthropicToolChoice(json.RawMessage(``)); err != nil || got != nil {
		t.Fatalf("empty = (%#v, %v), want (nil, nil)", got, err)
	}
	if got, err := translateAnthropicToolChoice(json.RawMessage(`null`)); err != nil || got != nil {
		t.Fatalf("null = (%#v, %v), want (nil, nil)", got, err)
	}
	if got, err := translateAnthropicToolChoice(json.RawMessage(`"auto"`)); err != nil || got != "auto" {
		t.Fatalf("bare string passthrough = (%#v, %v)", got, err)
	}
	if got, err := translateAnthropicToolChoice(json.RawMessage(`{"type":"auto"}`)); err != nil || got != "auto" {
		t.Fatalf("auto = (%#v, %v), want auto", got, err)
	}
	if got, err := translateAnthropicToolChoice(json.RawMessage(`{"type":"any"}`)); err != nil || got != "required" {
		t.Fatalf("any = (%#v, %v), want required", got, err)
	}
	got, err := translateAnthropicToolChoice(json.RawMessage(`{"type":"tool","name":"x"}`))
	if err != nil {
		t.Fatalf("tool err = %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok || m["type"] != "function" {
		t.Fatalf("tool = %#v, want function object", got)
	}
	if fn, ok := m["function"].(map[string]any); !ok || fn["name"] != "x" {
		t.Fatalf("tool function = %#v, want name x", m["function"])
	}
	if _, err := translateAnthropicToolChoice(json.RawMessage(`{"type":"web_search"}`)); !errors.Is(err, errAnthropicTranslate) {
		t.Fatalf("unknown variant err = %v, want errAnthropicTranslate", err)
	}
}

func TestTranslateAnthropicToolsVariants(t *testing.T) {
	if got, err := translateAnthropicTools(nil); err != nil || got != nil {
		t.Fatalf("nil = (%#v, %v), want (nil, nil)", got, err)
	}
	tools, err := translateAnthropicTools([]anthropicInboundTool{{
		Name:        "lookup",
		Description: "desc",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}})
	if err != nil {
		t.Fatalf("translate err: %v", err)
	}
	if len(tools) != 1 || tools[0].Type != "function" {
		t.Fatalf("tools = %+v", tools)
	}
	if tools[0].Function.Name != "lookup" || tools[0].Function.Description != "desc" {
		t.Fatalf("function = %+v", tools[0].Function)
	}
	if string(tools[0].Function.Parameters) != `{"type":"object"}` {
		t.Fatalf("parameters = %s, want input_schema verbatim", tools[0].Function.Parameters)
	}
	if _, err := translateAnthropicTools([]anthropicInboundTool{{Type: "web_search_20250305", Name: "web"}}); !errors.Is(err, errAnthropicTranslate) {
		t.Fatalf("server-side tool err = %v, want errAnthropicTranslate", err)
	}
}

func TestRejectUnsupportedAnthropicContentVariants(t *testing.T) {
	if err := rejectUnsupportedAnthropicContent(json.RawMessage(`"plain"`)); err != nil {
		t.Fatalf("string content allowed: %v", err)
	}
	if err := rejectUnsupportedAnthropicContent(json.RawMessage(`null`)); err != nil {
		t.Fatalf("null content allowed: %v", err)
	}
	if err := rejectUnsupportedAnthropicContent(json.RawMessage(`{"obj":true}`)); err == nil {
		t.Fatal("object content should be rejected")
	}
	if err := rejectUnsupportedAnthropicContent(json.RawMessage(`[{"type":"text"},{"type":"tool_use"}]`)); err != nil {
		t.Fatalf("supported block array allowed: %v", err)
	}
	if err := rejectUnsupportedAnthropicContent(json.RawMessage(`[{"type":"image"}]`)); err == nil {
		t.Fatal("image block should be rejected")
	}
	// Malformed array returns nil (lenient).
	if err := rejectUnsupportedAnthropicContent(json.RawMessage(`[broken`)); err != nil {
		t.Fatalf("malformed array should be lenient: %v", err)
	}
}

func TestRejectUnsupportedAnthropicMessageShapeInvalidJSON(t *testing.T) {
	if err := rejectUnsupportedAnthropicMessageShape([]byte(`not-json`)); err != nil {
		t.Fatalf("invalid JSON should be lenient: %v", err)
	}
}

func TestTranslateAnthropicMessagesRequestToolResultOnly(t *testing.T) {
	body := []byte(`{"model":"claude","messages":[{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":"done"}]}]}`)
	req, err := translateAnthropicMessagesRequest(body)
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	if len(req.Messages) != 1 {
		t.Fatalf("messages = %d, want 1 tool message", len(req.Messages))
	}
	msg := req.Messages[0]
	if msg.Role != "tool" || msg.ToolCallID == nil || *msg.ToolCallID != "call_1" || msg.Content != "done" {
		t.Fatalf("tool result message = %+v", msg)
	}
}

func TestTranslateAnthropicMessagesRequestInvalidJSON(t *testing.T) {
	if _, err := translateAnthropicMessagesRequest([]byte(`not-json`)); err == nil {
		t.Fatal("invalid JSON should error")
	}
}

func TestTranslateAnthropicInboundMessageInvalidBlockJSON(t *testing.T) {
	_, err := translateAnthropicInboundMessage(anthropicInboundMessage{
		Role:    "user",
		Content: json.RawMessage(`[broken`),
	})
	if err == nil {
		t.Fatal("invalid block array JSON should error")
	}
}

func TestTranslateAnthropicInboundMessageInvalidScalarJSON(t *testing.T) {
	_, err := translateAnthropicInboundMessage(anthropicInboundMessage{
		Role:    "user",
		Content: json.RawMessage(`@@@`),
	})
	if err == nil {
		t.Fatal("invalid scalar content JSON should error")
	}
}

// --- writeJSON marshal failure path ---

func TestWriteJSONMarshalFailureReturnsError(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	writeJSON(ctx, fasthttp.StatusOK, make(chan int)) // channels cannot be marshaled
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

// --- anthropicMessageResponse nil and tool-call branches ---

func TestAnthropicMessageResponseNil(t *testing.T) {
	body := anthropicMessageResponse(nil)
	if body.Type != "message" {
		t.Fatalf("nil response = %+v, want type message", body)
	}
}

func TestAnthropicMessageResponseWithToolCalls(t *testing.T) {
	stop := "tool_calls"
	resp := &providers.ChatResponse{
		ID:    "msg_1",
		Model: "claude",
		Choices: []providers.Choice{{
			FinishReason: &stop,
			Message: providers.Message{
				Role:    "assistant",
				Content: "thinking",
				ToolCalls: []providers.ToolCall{{
					ID:       "call_1",
					Function: providers.ToolCallFunc{Name: "search", Arguments: `{"q":"x"}`},
				}},
			},
		}},
		Usage: &providers.Usage{PromptTokens: 5, CompletionTokens: 6},
	}
	body := anthropicMessageResponse(resp)
	if body.StopReason == nil || *body.StopReason != "tool_use" {
		t.Fatalf("stop reason = %v, want tool_use", body.StopReason)
	}
	if len(body.Content) != 2 {
		t.Fatalf("content = %+v, want text + tool_use", body.Content)
	}
	if body.Content[0].Type != "text" || body.Content[0].Text != "thinking" {
		t.Fatalf("first block = %+v, want text", body.Content[0])
	}
	if body.Content[1].Type != "tool_use" || body.Content[1].ID != "call_1" || body.Content[1].Name != "search" {
		t.Fatalf("second block = %+v, want tool_use", body.Content[1])
	}
	if body.Usage.InputTokens != 5 || body.Usage.OutputTokens != 6 {
		t.Fatalf("usage = %+v, want 5/6", body.Usage)
	}
}

// --- streamMessages writes parallel tool-call blocks deterministically ---

func TestStreamMessagesEmitsToolUseBlocks(t *testing.T) {
	role := "assistant"
	finish := "tool_calls"
	chunks := make(chan providers.StreamChunk, 4)
	chunks <- providers.StreamChunk{ID: "m1", Model: "claude", Choices: []providers.StreamChoice{{Delta: providers.StreamDelta{Role: &role}}}}
	chunks <- providers.StreamChunk{Choices: []providers.StreamChoice{{Delta: providers.StreamDelta{ToolCalls: []providers.ToolCall{{
		ID:       "call_1",
		Function: providers.ToolCallFunc{Name: "search", Arguments: `{"q":`},
	}}}}}}
	chunks <- providers.StreamChunk{Choices: []providers.StreamChoice{{Delta: providers.StreamDelta{ToolCalls: []providers.ToolCall{{
		Function: providers.ToolCallFunc{Arguments: `"x"}`},
	}}}}}}
	chunks <- providers.StreamChunk{Choices: []providers.StreamChoice{{FinishReason: &finish}}, Usage: &providers.Usage{CompletionTokens: 2}}
	close(chunks)

	out := captureBodyStream(t, func(ctx *fasthttp.RequestCtx) {
		streamMessages(ctx, &coverageEngine{stream: chunks}, &providers.ChatRequest{Model: "claude"})
	})
	for _, want := range []string{"content_block_start", "tool_use", "input_json_delta", "content_block_stop", "message_delta", "tool_use", "message_stop"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stream output missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, `"partial_json":"{\"q\":"`) && !strings.Contains(out, "partial_json") {
		t.Fatalf("expected partial_json deltas:\n%s", out)
	}
}

func TestStreamMessagesDispatchErrorSanitized(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		streamMessages(ctx, &coverageEngine{streamErr: errors.New("sqlite boom")}, &providers.ChatRequest{Model: "claude"})
	})
	if ctx.Response.StatusCode() < 400 {
		t.Fatalf("status = %d, want >=400; body=%s", ctx.Response.StatusCode(), body)
	}
}

// captureBodyStream invokes a handler that sets a body stream writer and
// returns the rendered stream body as a string.
func captureBodyStream(t *testing.T, handler func(*fasthttp.RequestCtx)) string {
	t.Helper()
	ctx := &fasthttp.RequestCtx{}
	handler(ctx)
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	if err := ctx.Response.BodyWriteTo(w); err != nil {
		t.Fatalf("BodyWriteTo: %v", err)
	}
	_ = w.Flush()
	return buf.String()
}
