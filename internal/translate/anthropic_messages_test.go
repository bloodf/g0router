package translate

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

// --- AnthropicMessageResponse ---

func TestAnthropicMessageResponseNil(t *testing.T) {
	got := AnthropicMessageResponse(nil)
	if got.Type != "message" {
		t.Fatalf("type = %q, want message", got.Type)
	}
	if got.Role != "" {
		t.Fatalf("role = %q, want empty", got.Role)
	}
}

func TestAnthropicMessageResponseEmptyChoices(t *testing.T) {
	resp := &providers.ChatResponse{
		ID:    "msg-1",
		Model: "claude-test",
	}
	got := AnthropicMessageResponse(resp)
	if got.ID != "msg-1" {
		t.Fatalf("id = %q", got.ID)
	}
	if got.Role != "assistant" {
		t.Fatalf("role = %q, want assistant", got.Role)
	}
}

func TestAnthropicMessageResponseWithContent(t *testing.T) {
	resp := &providers.ChatResponse{
		ID:    "msg-2",
		Model: "claude-test",
		Choices: []providers.Choice{{
			Message: providers.Message{
				Role:    "assistant",
				Content: "hello world",
			},
			FinishReason: strPtr("stop"),
		}},
		Usage: &providers.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
		},
	}
	got := AnthropicMessageResponse(resp)
	if len(got.Content) != 1 || got.Content[0].Type != "text" || got.Content[0].Text != "hello world" {
		t.Fatalf("content = %+v", got.Content)
	}
	if got.Usage.InputTokens != 10 || got.Usage.OutputTokens != 5 {
		t.Fatalf("usage = %+v", got.Usage)
	}
	if got.StopReason == nil || *got.StopReason != "stop" {
		t.Fatalf("stop_reason = %+v", got.StopReason)
	}
}

func TestAnthropicMessageResponseWithToolCalls(t *testing.T) {
	resp := &providers.ChatResponse{
		ID:    "msg-3",
		Model: "claude-test",
		Choices: []providers.Choice{{
			Message: providers.Message{
				Role:    "assistant",
				Content: "",
				ToolCalls: []providers.ToolCall{{
					ID:   "call-1",
					Type: "function",
					Function: providers.ToolCallFunc{
						Name:      "lookup",
						Arguments: `{"q":"x"}`,
					},
				}},
			},
			FinishReason: strPtr("tool_calls"),
		}},
	}
	got := AnthropicMessageResponse(resp)
	if len(got.Content) != 1 {
		t.Fatalf("content len = %d, want 1", len(got.Content))
	}
	if got.Content[0].Type != "tool_use" || got.Content[0].ID != "call-1" || got.Content[0].Name != "lookup" {
		t.Fatalf("tool_use content = %+v", got.Content[0])
	}
	if got.StopReason == nil || *got.StopReason != "tool_use" {
		t.Fatalf("stop_reason = %+v, want tool_use", got.StopReason)
	}
}

func TestAnthropicMessageResponseWithTextAndToolCalls(t *testing.T) {
	resp := &providers.ChatResponse{
		Choices: []providers.Choice{{
			Message: providers.Message{
				Role:    "assistant",
				Content: "let me check",
				ToolCalls: []providers.ToolCall{{
					ID:   "call-2",
					Type: "function",
					Function: providers.ToolCallFunc{
						Name:      "calc",
						Arguments: `{}`,
					},
				}},
			},
		}},
	}
	got := AnthropicMessageResponse(resp)
	if len(got.Content) != 2 {
		t.Fatalf("content len = %d, want 2", len(got.Content))
	}
	if got.Content[0].Type != "text" || got.Content[0].Text != "let me check" {
		t.Fatalf("first content = %+v", got.Content[0])
	}
	if got.Content[1].Type != "tool_use" {
		t.Fatalf("second content = %+v", got.Content[1])
	}
}

// --- AnthropicMessagesRequest ---

func TestAnthropicMessagesRequestValid(t *testing.T) {
	body := []byte(`{
		"model": "claude-test",
		"messages": [
			{"role": "user", "content": "hi"}
		],
		"tools": [
			{"name": "lookup", "description": "looks up", "input_schema": {"type":"object"}, "type": "function"}
		],
		"tool_choice": {"type": "auto"}
	}`)
	req, err := AnthropicMessagesRequest(body)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if req.Model != "claude-test" {
		t.Fatalf("model = %q", req.Model)
	}
	if len(req.Messages) != 1 || req.Messages[0].Role != "user" {
		t.Fatalf("messages = %+v", req.Messages)
	}
	if len(req.Tools) != 1 || req.Tools[0].Function.Name != "lookup" {
		t.Fatalf("tools = %+v", req.Tools)
	}
	if req.ToolChoice != "auto" {
		t.Fatalf("tool_choice = %v", req.ToolChoice)
	}
}

func TestAnthropicMessagesRequestInvalidJSON(t *testing.T) {
	_, err := AnthropicMessagesRequest([]byte(`{bad`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestAnthropicMessagesRequestWithArrayContent(t *testing.T) {
	body := []byte(`{
		"model": "claude-test",
		"messages": [
			{"role": "user", "content": [{"type": "text", "text": "hello"}]}
		]
	}`)
	req, err := AnthropicMessagesRequest(body)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(req.Messages) != 1 {
		t.Fatalf("messages len = %d", len(req.Messages))
	}
	if req.Messages[0].Content != "hello" {
		t.Fatalf("content = %v", req.Messages[0].Content)
	}
}

func TestAnthropicMessagesRequestWithToolUseAndResult(t *testing.T) {
	body := []byte(`{
		"model": "claude-test",
		"messages": [
			{"role": "assistant", "content": [{"type": "tool_use", "id": "tu1", "name": "calc", "input": {"x": 1}}]},
			{"role": "user", "content": [{"type": "tool_result", "tool_use_id": "tu1", "content": "42"}]}
		]
	}`)
	req, err := AnthropicMessagesRequest(body)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(req.Messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(req.Messages))
	}
	if req.Messages[0].Role != "assistant" {
		t.Fatalf("first message role = %q", req.Messages[0].Role)
	}
	if len(req.Messages[0].ToolCalls) != 1 || req.Messages[0].ToolCalls[0].Function.Name != "calc" {
		t.Fatalf("first message tool_calls = %+v", req.Messages[0].ToolCalls)
	}
	if req.Messages[1].Role != "tool" {
		t.Fatalf("second message role = %q", req.Messages[1].Role)
	}
	if req.Messages[1].Content != "42" {
		t.Fatalf("second message content = %v", req.Messages[1].Content)
	}
	if req.Messages[1].ToolCallID == nil || *req.Messages[1].ToolCallID != "tu1" {
		t.Fatalf("second message tool_call_id = %+v", req.Messages[1].ToolCallID)
	}
}

func TestAnthropicMessagesRequestUnsupportedToolType(t *testing.T) {
	body := []byte(`{
		"model": "claude-test",
		"messages": [],
		"tools": [{"name": "x", "type": "web_search"}]
	}`)
	_, err := AnthropicMessagesRequest(body)
	if err == nil {
		t.Fatal("expected error for unsupported tool type")
	}
	if !errors.Is(err, ErrAnthropicTranslate) {
		t.Fatalf("error should wrap ErrAnthropicTranslate, got %v", err)
	}
}

func TestAnthropicMessagesRequestUnsupportedToolChoice(t *testing.T) {
	body := []byte(`{
		"model": "claude-test",
		"messages": [],
		"tool_choice": {"type": "unknown"}
	}`)
	_, err := AnthropicMessagesRequest(body)
	if err == nil {
		t.Fatal("expected error for unsupported tool_choice")
	}
	if !errors.Is(err, ErrAnthropicTranslate) {
		t.Fatalf("error should wrap ErrAnthropicTranslate, got %v", err)
	}
}

// --- TranslateAnthropicTools ---

func TestTranslateAnthropicToolsEmpty(t *testing.T) {
	got, err := TranslateAnthropicTools(nil)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if got != nil {
		t.Fatalf("got = %v, want nil", got)
	}
}

func TestTranslateAnthropicToolsValid(t *testing.T) {
	tools := []AnthropicInboundTool{
		{
			Name:        "lookup",
			Description: "look things up",
			InputSchema: json.RawMessage(`{"type":"object"}`),
			Type:        "custom",
		},
		{
			Name:        "calc",
			Description: "calculate",
			InputSchema: json.RawMessage(`{}`),
			Type:        "function",
		},
		{
			Name:        "empty",
			Description: "",
			InputSchema: nil,
			Type:        "",
		},
	}
	got, err := TranslateAnthropicTools(tools)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[0].Type != "function" || got[0].Function.Name != "lookup" {
		t.Fatalf("first tool = %+v", got[0])
	}
}

func TestTranslateAnthropicToolsUnsupportedType(t *testing.T) {
	tools := []AnthropicInboundTool{
		{Name: "x", Type: "web_search"},
	}
	_, err := TranslateAnthropicTools(tools)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrAnthropicTranslate) {
		t.Fatalf("error should wrap ErrAnthropicTranslate, got %v", err)
	}
}

// --- TranslateAnthropicToolChoice ---

func TestTranslateAnthropicToolChoiceNil(t *testing.T) {
	got, err := TranslateAnthropicToolChoice(nil)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if got != nil {
		t.Fatalf("got = %v, want nil", got)
	}
}

func TestTranslateAnthropicToolChoiceEmpty(t *testing.T) {
	got, err := TranslateAnthropicToolChoice([]byte("   "))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if got != nil {
		t.Fatalf("got = %v, want nil", got)
	}
}

func TestTranslateAnthropicToolChoiceNull(t *testing.T) {
	got, err := TranslateAnthropicToolChoice([]byte("null"))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if got != nil {
		t.Fatalf("got = %v, want nil", got)
	}
}

func TestTranslateAnthropicToolChoiceString(t *testing.T) {
	got, err := TranslateAnthropicToolChoice([]byte(`"auto"`))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if got != "auto" {
		t.Fatalf("got = %v, want auto", got)
	}
}

func TestTranslateAnthropicToolChoiceAuto(t *testing.T) {
	got, err := TranslateAnthropicToolChoice([]byte(`{"type":"auto"}`))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if got != "auto" {
		t.Fatalf("got = %v, want auto", got)
	}
}

func TestTranslateAnthropicToolChoiceAny(t *testing.T) {
	got, err := TranslateAnthropicToolChoice([]byte(`{"type":"any"}`))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if got != "required" {
		t.Fatalf("got = %v, want required", got)
	}
}

func TestTranslateAnthropicToolChoiceTool(t *testing.T) {
	got, err := TranslateAnthropicToolChoice([]byte(`{"type":"tool","name":"lookup"}`))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("got type = %T", got)
	}
	if m["type"] != "function" {
		t.Fatalf("type = %v", m["type"])
	}
	fn, ok := m["function"].(map[string]any)
	if !ok || fn["name"] != "lookup" {
		t.Fatalf("function = %+v", m["function"])
	}
}

func TestTranslateAnthropicToolChoiceInvalidStringJSON(t *testing.T) {
	_, err := TranslateAnthropicToolChoice([]byte(`"unterminated`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTranslateAnthropicToolChoiceInvalidObjectJSON(t *testing.T) {
	_, err := TranslateAnthropicToolChoice([]byte(`{`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTranslateAnthropicToolChoiceUnknownType(t *testing.T) {
	_, err := TranslateAnthropicToolChoice([]byte(`{"type":"bad"}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrAnthropicTranslate) {
		t.Fatalf("error should wrap ErrAnthropicTranslate, got %v", err)
	}
}

// --- TranslateAnthropicInboundMessage ---

func TestTranslateAnthropicInboundMessageStringContent(t *testing.T) {
	inbound := AnthropicInboundMessage{
		Role:    "user",
		Content: json.RawMessage(`"hello"`),
	}
	got, err := TranslateAnthropicInboundMessage(inbound)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(got) != 1 || got[0].Role != "user" || got[0].Content != "hello" {
		t.Fatalf("got = %+v", got)
	}
}

func TestTranslateAnthropicInboundMessageNullContent(t *testing.T) {
	inbound := AnthropicInboundMessage{
		Role:    "user",
		Content: json.RawMessage(`null`),
	}
	got, err := TranslateAnthropicInboundMessage(inbound)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(got) != 1 || got[0].Content != nil {
		t.Fatalf("got = %+v", got)
	}
}

func TestTranslateAnthropicInboundMessageEmptyContent(t *testing.T) {
	inbound := AnthropicInboundMessage{
		Role:    "user",
		Content: json.RawMessage(`   `),
	}
	got, err := TranslateAnthropicInboundMessage(inbound)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(got) != 1 || got[0].Content != nil {
		t.Fatalf("got = %+v", got)
	}
}

func TestTranslateAnthropicInboundMessageNumberContent(t *testing.T) {
	inbound := AnthropicInboundMessage{
		Role:    "user",
		Content: json.RawMessage(`42`),
	}
	got, err := TranslateAnthropicInboundMessage(inbound)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(got) != 1 || got[0].Content != float64(42) {
		t.Fatalf("got = %+v", got)
	}
}

func TestTranslateAnthropicInboundMessageInvalidNonArrayContent(t *testing.T) {
	inbound := AnthropicInboundMessage{
		Role:    "user",
		Content: json.RawMessage(`{bad`),
	}
	_, err := TranslateAnthropicInboundMessage(inbound)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTranslateAnthropicInboundMessageTextBlocks(t *testing.T) {
	inbound := AnthropicInboundMessage{
		Role: "user",
		Content: json.RawMessage(`[
			{"type": "text", "text": "hello "},
			{"type": "text", "text": "world"}
		]`),
	}
	got, err := TranslateAnthropicInboundMessage(inbound)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(got) != 1 || got[0].Content != "hello world" {
		t.Fatalf("got = %+v", got)
	}
}

func TestTranslateAnthropicInboundMessageEmptyTextBlocks(t *testing.T) {
	inbound := AnthropicInboundMessage{
		Role: "user",
		Content: json.RawMessage(`[
			{"type": "text", "text": ""},
			{"type": "text", "text": ""}
		]`),
	}
	got, err := TranslateAnthropicInboundMessage(inbound)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	// empty text parts produce empty joined string; still one message because results==0
	if len(got) != 1 || got[0].Content != "" {
		t.Fatalf("got = %+v", got)
	}
}

func TestTranslateAnthropicInboundMessageToolUseBlocks(t *testing.T) {
	inbound := AnthropicInboundMessage{
		Role: "assistant",
		Content: json.RawMessage(`[
			{"type": "tool_use", "id": "tu1", "name": "calc", "input": {"x": 1}}
		]`),
	}
	got, err := TranslateAnthropicInboundMessage(inbound)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(got) != 1 || got[0].Role != "assistant" {
		t.Fatalf("got = %+v", got)
	}
	if len(got[0].ToolCalls) != 1 || got[0].ToolCalls[0].ID != "tu1" {
		t.Fatalf("tool_calls = %+v", got[0].ToolCalls)
	}
}

func TestTranslateAnthropicInboundMessageToolResultBlocks(t *testing.T) {
	inbound := AnthropicInboundMessage{
		Role: "user",
		Content: json.RawMessage(`[
			{"type": "tool_result", "tool_use_id": "tu1", "content": "42"}
		]`),
	}
	got, err := TranslateAnthropicInboundMessage(inbound)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(got) != 1 || got[0].Role != "tool" {
		t.Fatalf("got = %+v", got)
	}
	if got[0].Content != "42" {
		t.Fatalf("content = %v", got[0].Content)
	}
	if got[0].ToolCallID == nil || *got[0].ToolCallID != "tu1" {
		t.Fatalf("tool_call_id = %+v", got[0].ToolCallID)
	}
}

func TestTranslateAnthropicInboundMessageMixedBlocks(t *testing.T) {
	inbound := AnthropicInboundMessage{
		Role: "assistant",
		Content: json.RawMessage(`[
			{"type": "text", "text": "ok"},
			{"type": "tool_use", "id": "tu1", "name": "calc", "input": {}},
			{"type": "tool_result", "tool_use_id": "tu0", "content": "prev"}
		]`),
	}
	got, err := TranslateAnthropicInboundMessage(inbound)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	// first message has text + tool_calls, then tool_result messages appended
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Role != "assistant" || got[0].Content != "ok" || len(got[0].ToolCalls) != 1 {
		t.Fatalf("first = %+v", got[0])
	}
	if got[1].Role != "tool" || got[1].Content != "prev" {
		t.Fatalf("second = %+v", got[1])
	}
}

func TestTranslateAnthropicInboundMessageInvalidBlocksJSON(t *testing.T) {
	inbound := AnthropicInboundMessage{
		Role:    "user",
		Content: json.RawMessage(`[bad]`),
	}
	_, err := TranslateAnthropicInboundMessage(inbound)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTranslateAnthropicInboundMessageOnlyToolResults(t *testing.T) {
	// When there are only tool_results, the first message (with role) is skipped
	// because len(textParts)==0 && len(toolCalls)==0 && len(results)>0
	inbound := AnthropicInboundMessage{
		Role: "user",
		Content: json.RawMessage(`[
			{"type": "tool_result", "tool_use_id": "tu1", "content": "42"}
		]`),
	}
	got, err := TranslateAnthropicInboundMessage(inbound)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(got) != 1 || got[0].Role != "tool" {
		t.Fatalf("got = %+v", got)
	}
}

// --- AnthropicToolInputArguments ---

func TestAnthropicToolInputArgumentsNil(t *testing.T) {
	got := AnthropicToolInputArguments(nil)
	if got != "{}" {
		t.Fatalf("got = %q, want {}", got)
	}
}

func TestAnthropicToolInputArgumentsNull(t *testing.T) {
	got := AnthropicToolInputArguments([]byte("null"))
	if got != "{}" {
		t.Fatalf("got = %q, want {}", got)
	}
}

func TestAnthropicToolInputArgumentsEmpty(t *testing.T) {
	got := AnthropicToolInputArguments([]byte("   "))
	if got != "{}" {
		t.Fatalf("got = %q, want {}", got)
	}
}

func TestAnthropicToolInputArgumentsValid(t *testing.T) {
	got := AnthropicToolInputArguments([]byte(`{"x":1}`))
	if got != `{"x":1}` {
		t.Fatalf("got = %q", got)
	}
}

// --- AnthropicToolResultText ---

func TestAnthropicToolResultTextNil(t *testing.T) {
	got := AnthropicToolResultText(nil)
	if got != "" {
		t.Fatalf("got = %q, want empty", got)
	}
}

func TestAnthropicToolResultTextNull(t *testing.T) {
	got := AnthropicToolResultText([]byte("null"))
	if got != "" {
		t.Fatalf("got = %q, want empty", got)
	}
}

func TestAnthropicToolResultTextEmpty(t *testing.T) {
	got := AnthropicToolResultText([]byte("   "))
	if got != "" {
		t.Fatalf("got = %q, want empty", got)
	}
}

func TestAnthropicToolResultTextString(t *testing.T) {
	got := AnthropicToolResultText([]byte(`"hello world"`))
	if got != "hello world" {
		t.Fatalf("got = %q", got)
	}
}

func TestAnthropicToolResultTextRawString(t *testing.T) {
	// Invalid JSON string (unterminated) falls back to raw bytes
	got := AnthropicToolResultText([]byte(`"unterminated`))
	if got != `"unterminated` {
		t.Fatalf("got = %q", got)
	}
}

func TestAnthropicToolResultTextArrayBlocks(t *testing.T) {
	got := AnthropicToolResultText([]byte(`[
		{"type": "text", "text": "part1"},
		{"type": "text", "text": "part2"}
	]`))
	if got != "part1part2" {
		t.Fatalf("got = %q", got)
	}
}

func TestAnthropicToolResultTextInvalidArray(t *testing.T) {
	// Array that fails unmarshal falls back to raw string
	got := AnthropicToolResultText([]byte(`[bad]`))
	if got != "[bad]" {
		t.Fatalf("got = %q", got)
	}
}

func TestAnthropicToolResultTextObject(t *testing.T) {
	got := AnthropicToolResultText([]byte(`{"result":42}`))
	if got != `{"result":42}` {
		t.Fatalf("got = %q", got)
	}
}

// --- RejectUnsupportedAnthropicMessageShape ---

func TestRejectUnsupportedAnthropicMessageShapeInvalidJSON(t *testing.T) {
	// Invalid JSON returns nil error (unmarshal fails silently)
	err := RejectUnsupportedAnthropicMessageShape([]byte(`{bad`))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
}

func TestRejectUnsupportedAnthropicMessageShapeValid(t *testing.T) {
	body := []byte(`{"messages": [{"content": "hello"}, {"content": [{"type": "text"}]}]}`)
	err := RejectUnsupportedAnthropicMessageShape(body)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
}

func TestRejectUnsupportedAnthropicMessageShapeUnsupportedBlock(t *testing.T) {
	body := []byte(`{"messages": [{"content": [{"type": "image"}]}]}`)
	err := RejectUnsupportedAnthropicMessageShape(body)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- RejectUnsupportedAnthropicContent ---

func TestRejectUnsupportedAnthropicContentNil(t *testing.T) {
	err := RejectUnsupportedAnthropicContent(nil)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
}

func TestRejectUnsupportedAnthropicContentEmpty(t *testing.T) {
	err := RejectUnsupportedAnthropicContent([]byte("   "))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
}

func TestRejectUnsupportedAnthropicContentString(t *testing.T) {
	err := RejectUnsupportedAnthropicContent([]byte(`"hello"`))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
}

func TestRejectUnsupportedAnthropicContentNull(t *testing.T) {
	err := RejectUnsupportedAnthropicContent([]byte("null"))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
}

func TestRejectUnsupportedAnthropicContentValidArray(t *testing.T) {
	err := RejectUnsupportedAnthropicContent([]byte(`[
		{"type": "text"},
		{"type": "tool_use"},
		{"type": "tool_result"}
	]`))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
}

func TestRejectUnsupportedAnthropicContentEmptyType(t *testing.T) {
	// Empty type is treated as text (falls through)
	err := RejectUnsupportedAnthropicContent([]byte(`[{"type":""}]`))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
}

func TestRejectUnsupportedAnthropicContentUnsupportedShape(t *testing.T) {
	err := RejectUnsupportedAnthropicContent([]byte(`{"not":"array"}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRejectUnsupportedAnthropicContentInvalidArrayJSON(t *testing.T) {
	// Unmarshal error returns nil
	err := RejectUnsupportedAnthropicContent([]byte(`[bad]`))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
}

func TestRejectUnsupportedAnthropicContentUnsupportedBlockType(t *testing.T) {
	err := RejectUnsupportedAnthropicContent([]byte(`[{"type":"image"}]`))
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- BytesTrimSpace ---

func TestBytesTrimSpaceNil(t *testing.T) {
	got := BytesTrimSpace(nil)
	if len(got) != 0 {
		t.Fatalf("got = %q", got)
	}
}

func TestBytesTrimSpaceWhitespace(t *testing.T) {
	got := BytesTrimSpace([]byte("  hello  "))
	if string(got) != "hello" {
		t.Fatalf("got = %q", got)
	}
}

func TestBytesTrimSpaceNoWhitespace(t *testing.T) {
	got := BytesTrimSpace([]byte("hello"))
	if string(got) != "hello" {
		t.Fatalf("got = %q", got)
	}
}

// --- AnthropicStopReason ---

func TestAnthropicStopReasonNil(t *testing.T) {
	got := AnthropicStopReason(nil)
	if got != nil {
		t.Fatalf("got = %v", got)
	}
}

func TestAnthropicStopReasonToolCalls(t *testing.T) {
	reason := "tool_calls"
	got := AnthropicStopReason(&reason)
	if got == nil || *got != "tool_use" {
		t.Fatalf("got = %v", got)
	}
}

func TestAnthropicStopReasonOther(t *testing.T) {
	reason := "stop"
	got := AnthropicStopReason(&reason)
	if got == nil || *got != "stop" {
		t.Fatalf("got = %v", got)
	}
}

// --- AnthropicStreamStopReason ---

func TestAnthropicStreamStopReasonNil(t *testing.T) {
	got := AnthropicStreamStopReason(nil)
	if got != "" {
		t.Fatalf("got = %q", got)
	}
}

func TestAnthropicStreamStopReasonStop(t *testing.T) {
	reason := "stop"
	got := AnthropicStreamStopReason(&reason)
	if got != "end_turn" {
		t.Fatalf("got = %q", got)
	}
}

func TestAnthropicStreamStopReasonLength(t *testing.T) {
	reason := "length"
	got := AnthropicStreamStopReason(&reason)
	if got != "max_tokens" {
		t.Fatalf("got = %q", got)
	}
}

func TestAnthropicStreamStopReasonToolCalls(t *testing.T) {
	reason := "tool_calls"
	got := AnthropicStreamStopReason(&reason)
	if got != "tool_use" {
		t.Fatalf("got = %q", got)
	}
}

func TestAnthropicStreamStopReasonDefault(t *testing.T) {
	reason := "custom_reason"
	got := AnthropicStreamStopReason(&reason)
	if got != "custom_reason" {
		t.Fatalf("got = %q", got)
	}
}

// --- ToolCallInput ---

func TestToolCallInputEmpty(t *testing.T) {
	got := ToolCallInput("")
	if string(got) != `{}` {
		t.Fatalf("got = %s", got)
	}
}

func TestToolCallInputWhitespace(t *testing.T) {
	got := ToolCallInput("   ")
	if string(got) != `{}` {
		t.Fatalf("got = %s", got)
	}
}

func TestToolCallInputValidObject(t *testing.T) {
	got := ToolCallInput(`{"x":1}`)
	if string(got) != `{"x":1}` {
		t.Fatalf("got = %s", got)
	}
}

func TestToolCallInputInvalidJSON(t *testing.T) {
	got := ToolCallInput(`not-json`)
	var wrapped map[string]string
	if err := json.Unmarshal(got, &wrapped); err != nil {
		t.Fatalf("unmarshal wrapped: %v", err)
	}
	if wrapped["arguments"] != `not-json` {
		t.Fatalf("wrapped = %+v", wrapped)
	}
}

func TestToolCallInputScalarJSON(t *testing.T) {
	got := ToolCallInput(`"hello"`)
	var wrapped map[string]string
	if err := json.Unmarshal(got, &wrapped); err != nil {
		t.Fatalf("unmarshal wrapped: %v", err)
	}
	if wrapped["arguments"] != `"hello"` {
		t.Fatalf("wrapped = %+v", wrapped)
	}
}

func TestToolCallInputArrayJSON(t *testing.T) {
	got := ToolCallInput(`[1,2,3]`)
	var wrapped map[string]string
	if err := json.Unmarshal(got, &wrapped); err != nil {
		t.Fatalf("unmarshal wrapped: %v", err)
	}
	if wrapped["arguments"] != `[1,2,3]` {
		t.Fatalf("wrapped = %+v", wrapped)
	}
}

// --- MessageContentText ---

func TestMessageContentTextNil(t *testing.T) {
	got := MessageContentText(nil)
	if got != "" {
		t.Fatalf("got = %q", got)
	}
}

func TestMessageContentTextString(t *testing.T) {
	got := MessageContentText("hello")
	if got != "hello" {
		t.Fatalf("got = %q", got)
	}
}

func TestMessageContentTextInt(t *testing.T) {
	got := MessageContentText(42)
	if got != "42" {
		t.Fatalf("got = %q", got)
	}
}

func TestMessageContentTextStruct(t *testing.T) {
	got := MessageContentText(struct{ A int }{A: 1})
	if got == "" {
		t.Fatal("got empty for struct")
	}
}

// helpers

func strPtr(s string) *string {
	return &s
}

// Ensure test coverage also hits the helper implicitly
func TestAnthropicMessageResponseUsageNil(t *testing.T) {
	resp := &providers.ChatResponse{
		Choices: []providers.Choice{{
			Message: providers.Message{Role: "assistant", Content: "hi"},
		}},
	}
	got := AnthropicMessageResponse(resp)
	if got.Usage.InputTokens != 0 || got.Usage.OutputTokens != 0 {
		t.Fatalf("usage = %+v", got.Usage)
	}
}

func TestAnthropicMessagesRequestToolChoiceString(t *testing.T) {
	body := []byte(`{
		"model": "claude-test",
		"messages": [],
		"tool_choice": "required"
	}`)
	req, err := AnthropicMessagesRequest(body)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if req.ToolChoice != "required" {
		t.Fatalf("tool_choice = %v", req.ToolChoice)
	}
}

func TestAnthropicMessagesRequestToolChoiceTool(t *testing.T) {
	body := []byte(`{
		"model": "claude-test",
		"messages": [],
		"tool_choice": {"type": "tool", "name": "lookup"}
	}`)
	req, err := AnthropicMessagesRequest(body)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	m, ok := req.ToolChoice.(map[string]any)
	if !ok {
		t.Fatalf("tool_choice type = %T", req.ToolChoice)
	}
	if !reflect.DeepEqual(m, map[string]any{
		"type":     "function",
		"function": map[string]any{"name": "lookup"},
	}) {
		t.Fatalf("tool_choice = %+v", m)
	}
}

func TestAnthropicMessagesRequestTranslateMessageError(t *testing.T) {
	body := []byte(`{
		"model": "claude-test",
		"messages": [
			{"role": "user", "content": [1]}
		]
	}`)
	_, err := AnthropicMessagesRequest(body)
	if err == nil {
		t.Fatal("expected error")
	}
}
