package translation

import (
	"encoding/json"
	"testing"
)

func TestKiroOpenAIPassthroughChunk(t *testing.T) {
	chunk := map[string]any{
		"id":      "chatcmpl-existing",
		"object":  "chat.completion.chunk",
		"created": 1234567890,
		"model":   "gpt-4",
		"choices": []any{
			map[string]any{
				"index": 0,
				"delta": map[string]any{"content": "hello"},
			},
		},
	}

	state := NewStreamState()
	out, err := kiroToOpenAIResponse(chunk, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(out))
	}
	if out[0]["id"] != "chatcmpl-existing" {
		t.Errorf("id = %v", out[0]["id"])
	}
}

func TestKiroOpenAIContentAndRoleInjection(t *testing.T) {
	t.Run("event_type_form", func(t *testing.T) {
		state := NewStreamState()
		chunk := map[string]any{
			"_eventType": "assistantResponseEvent",
			"textDelta":  "hello ",
		}
		out, err := kiroToOpenAIResponse(chunk, state)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if len(out) != 1 {
			t.Fatalf("expected 1 chunk, got %d", len(out))
		}
		choice := out[0]["choices"].([]any)[0].(map[string]any)
		delta := choice["delta"].(map[string]any)
		if delta["content"] != "hello " {
			t.Errorf("content = %v", delta["content"])
		}
		if delta["role"] != "assistant" {
			t.Errorf("role = %v", delta["role"])
		}

		// Second chunk should NOT inject role again.
		chunk2 := map[string]any{
			"_eventType": "assistantResponseEvent",
			"textDelta":  "world",
		}
		out2, err := kiroToOpenAIResponse(chunk2, state)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		delta2 := out2[0]["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)
		if _, ok := delta2["role"]; ok {
			t.Error("second chunk should not have role")
		}
	})

	t.Run("wrapped_key_form", func(t *testing.T) {
		state := NewStreamState()
		chunk := map[string]any{
			"assistantResponseEvent": map[string]any{
				"textDelta": "wrapped ",
			},
		}
		out, err := kiroToOpenAIResponse(chunk, state)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if len(out) != 1 {
			t.Fatalf("expected 1 chunk, got %d", len(out))
		}
		delta := out[0]["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)
		if delta["content"] != "wrapped " {
			t.Errorf("content = %v", delta["content"])
		}
		if delta["role"] != "assistant" {
			t.Errorf("role = %v", delta["role"])
		}
	})

	t.Run("empty_content_nil", func(t *testing.T) {
		state := NewStreamState()
		chunk := map[string]any{
			"_eventType": "assistantResponseEvent",
			"textDelta":  "",
		}
		out, err := kiroToOpenAIResponse(chunk, state)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if out != nil {
			t.Errorf("expected nil for empty content, got %v", out)
		}
	})
}

func TestKiroOpenAIReasoningDelta(t *testing.T) {
	t.Run("text_field", func(t *testing.T) {
		state := NewStreamState()
		chunk := map[string]any{
			"_eventType": "reasoningContentEvent",
			"text":       "thinking...",
		}
		out, err := kiroToOpenAIResponse(chunk, state)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if len(out) != 1 {
			t.Fatalf("expected 1 chunk, got %d", len(out))
		}
		delta := out[0]["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)
		if delta["reasoning_content"] != "thinking..." {
			t.Errorf("reasoning_content = %v", delta["reasoning_content"])
		}
	})

	t.Run("content_field_fallback", func(t *testing.T) {
		state := NewStreamState()
		chunk := map[string]any{
			"_eventType": "reasoningContentEvent",
			"content":    "reasoning here",
		}
		out, err := kiroToOpenAIResponse(chunk, state)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if len(out) != 1 {
			t.Fatalf("expected 1 chunk, got %d", len(out))
		}
		delta := out[0]["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)
		if delta["reasoning_content"] != "reasoning here" {
			t.Errorf("reasoning_content = %v", delta["reasoning_content"])
		}
	})

	t.Run("string_field_fallback", func(t *testing.T) {
		state := NewStreamState()
		chunk := map[string]any{
			"_eventType": "reasoningContentEvent",
			"string":     "string reasoning",
		}
		out, err := kiroToOpenAIResponse(chunk, state)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if len(out) != 1 {
			t.Fatalf("expected 1 chunk, got %d", len(out))
		}
		delta := out[0]["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)
		if delta["reasoning_content"] != "string reasoning" {
			t.Errorf("reasoning_content = %v", delta["reasoning_content"])
		}
	})
}

func TestKiroOpenAIToolUseEvent(t *testing.T) {
	state := NewStreamState()
	chunk := map[string]any{
		"_eventType": "toolUseEvent",
		"toolUseId":  "toolu_123",
		"name":       "read_file",
		"input": map[string]any{
			"path": "/tmp/test",
		},
	}

	out, err := kiroToOpenAIResponse(chunk, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(out))
	}

	choice := out[0]["choices"].([]any)[0].(map[string]any)
	delta := choice["delta"].(map[string]any)
	tcs := delta["tool_calls"].([]any)
	if len(tcs) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(tcs))
	}

	tc := tcs[0].(map[string]any)
	if tc["index"] != 0 {
		t.Errorf("index = %v", tc["index"])
	}
	if tc["id"] != "toolu_123" {
		t.Errorf("id = %v", tc["id"])
	}
	if tc["type"] != "function" {
		t.Errorf("type = %v", tc["type"])
	}

	fn := tc["function"].(map[string]any)
	if fn["name"] != "read_file" {
		t.Errorf("name = %v", fn["name"])
	}

	var args map[string]any
	if err := json.Unmarshal([]byte(fn["arguments"].(string)), &args); err != nil {
		t.Fatalf("arguments not valid JSON: %v", err)
	}
	if args["path"] != "/tmp/test" {
		t.Errorf("arguments path = %v", args["path"])
	}
}

func TestKiroOpenAIUsageThenStop(t *testing.T) {
	state := NewStreamState()

	// Usage event buffers usage.
	usageChunk := map[string]any{
		"_eventType":   "usageEvent",
		"inputTokens":  42,
		"outputTokens": 17,
	}
	out, err := kiroToOpenAIResponse(usageChunk, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if out != nil {
		t.Errorf("expected nil for usage event, got %v", out)
	}
	if state.KiroUsage == nil {
		t.Fatal("expected usage to be buffered in state")
	}

	// Stop event should emit finish chunk with buffered usage.
	stopChunk := map[string]any{
		"_eventType": "messageStopEvent",
	}
	out, err = kiroToOpenAIResponse(stopChunk, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(out))
	}

	choice := out[0]["choices"].([]any)[0].(map[string]any)
	if choice["finish_reason"] != "stop" {
		t.Errorf("finish_reason = %v", choice["finish_reason"])
	}
	if out[0]["usage"] == nil {
		t.Fatal("expected usage on stop chunk")
	}
	usage := out[0]["usage"].(map[string]any)
	if usage["prompt_tokens"] != 42 {
		t.Errorf("prompt_tokens = %v", usage["prompt_tokens"])
	}
	if usage["completion_tokens"] != 17 {
		t.Errorf("completion_tokens = %v", usage["completion_tokens"])
	}
	if usage["total_tokens"] != 59 {
		t.Errorf("total_tokens = %v", usage["total_tokens"])
	}
}

func TestKiroOpenAIUnknownEventNil(t *testing.T) {
	state := NewStreamState()
	chunk := map[string]any{
		"_eventType": "someRandomEvent",
		"data":       "whatever",
	}
	out, err := kiroToOpenAIResponse(chunk, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if out != nil {
		t.Errorf("expected nil for unknown event, got %v", out)
	}
}
