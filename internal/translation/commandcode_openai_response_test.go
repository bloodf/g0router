package translation

import (
	"testing"
)

func TestCommandCodeOpenAIPassthroughChunk(t *testing.T) {
	state := NewStreamState()
	chunk := map[string]any{
		"object": "chat.completion.chunk",
		"id":     "existing",
		"choices": []any{
			map[string]any{"index": 0, "delta": map[string]any{"content": "hi"}},
		},
	}
	out, err := commandcodeToOpenAIResponse(chunk, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0]["id"] != "existing" {
		t.Errorf("id = %v, want existing", out[0]["id"])
	}
}

func TestCommandCodeOpenAITextAndRoleInjection(t *testing.T) {
	state := NewStreamState()
	out, err := commandcodeToOpenAIResponse(map[string]any{
		"type": "text-delta",
		"text": "hello",
	}, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	delta := out[0]["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)
	if delta["role"] != "assistant" {
		t.Errorf("first chunk should inject role, got %v", delta["role"])
	}
	if delta["content"] != "hello" {
		t.Errorf("content = %v, want hello", delta["content"])
	}

	// Second chunk should not inject role
	out2, err := commandcodeToOpenAIResponse(map[string]any{
		"type": "text-delta",
		"text": " world",
	}, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	delta2 := out2[0]["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)
	if _, ok := delta2["role"]; ok {
		t.Error("second chunk should not inject role")
	}
	if delta2["content"] != " world" {
		t.Errorf("content = %v, want ' world'", delta2["content"])
	}
}

func TestCommandCodeOpenAIReasoningDelta(t *testing.T) {
	state := NewStreamState()
	out, err := commandcodeToOpenAIResponse(map[string]any{
		"type": "reasoning-delta",
		"text": "think",
	}, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	delta := out[0]["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)
	if delta["reasoning_content"] != "think" {
		t.Errorf("reasoning_content = %v, want think", delta["reasoning_content"])
	}
}

func TestCommandCodeOpenAIToolInputLifecycle(t *testing.T) {
	state := NewStreamState()
	// tool-input-start
	out, err := commandcodeToOpenAIResponse(map[string]any{
		"type":     "tool-input-start",
		"id":       "tc1",
		"toolName": "foo",
	}, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	tcs := out[0]["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)["tool_calls"].([]any)
	if len(tcs) != 1 {
		t.Fatalf("len(tool_calls) = %d, want 1", len(tcs))
	}
	if tcs[0].(map[string]any)["id"] != "tc1" {
		t.Errorf("id = %v", tcs[0].(map[string]any)["id"])
	}

	// tool-input-delta
	out2, err := commandcodeToOpenAIResponse(map[string]any{
		"type":  "tool-input-delta",
		"id":    "tc1",
		"delta": `{"a":`,
	}, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	tcs2 := out2[0]["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)["tool_calls"].([]any)
	fn := tcs2[0].(map[string]any)["function"].(map[string]any)
	if fn["arguments"] != `{"a":` {
		t.Errorf("arguments = %v", fn["arguments"])
	}

	// unknown-id should be skipped
	out3, err := commandcodeToOpenAIResponse(map[string]any{
		"type":  "tool-input-delta",
		"id":    "unknown",
		"delta": "x",
	}, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if out3 != nil {
		t.Errorf("unknown id should be skipped, got %v", out3)
	}
}

func TestCommandCodeOpenAIConsolidatedToolCallDedup(t *testing.T) {
	state := NewStreamState()
	// First, tool-input-start for tc1
	_, err := commandcodeToOpenAIResponse(map[string]any{
		"type":     "tool-input-start",
		"id":       "tc1",
		"toolName": "foo",
	}, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	// Then tool-call for same tc1 should be skipped (already seen)
	out, err := commandcodeToOpenAIResponse(map[string]any{
		"type":       "tool-call",
		"toolCallId": "tc1",
		"toolName":   "foo",
		"input":      map[string]any{"a": 1},
	}, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if out != nil {
		t.Errorf("tool-call after tool-input-start for same id should be skipped, got %v", out)
	}
}

func TestCommandCodeOpenAIFinishUsage(t *testing.T) {
	state := NewStreamState()
	// finish-step with usage
	_, err := commandcodeToOpenAIResponse(map[string]any{
		"type":         "finish-step",
		"finishReason": "stop",
		"usage": map[string]any{
			"inputTokens":  10,
			"outputTokens": 5,
		},
	}, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	// finish with totalUsage overriding state usage
	out, err := commandcodeToOpenAIResponse(map[string]any{
		"type": "finish",
		"totalUsage": map[string]any{
			"inputTokens":  20,
			"outputTokens": 10,
			"totalTokens":  30,
		},
	}, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	usage := out[0]["usage"].(map[string]any)
	if usage["prompt_tokens"] != 20 {
		t.Errorf("prompt_tokens = %v, want 20", usage["prompt_tokens"])
	}
	if usage["completion_tokens"] != 10 {
		t.Errorf("completion_tokens = %v, want 10", usage["completion_tokens"])
	}
	if usage["total_tokens"] != 30 {
		t.Errorf("total_tokens = %v, want 30", usage["total_tokens"])
	}

	// finish without totalUsage should fall back to state.usage
	state2 := NewStreamState()
	_, err = commandcodeToOpenAIResponse(map[string]any{
		"type":         "finish-step",
		"finishReason": "stop",
		"usage": map[string]any{
			"inputTokens":  7,
			"outputTokens": 3,
		},
	}, state2)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	out2, err := commandcodeToOpenAIResponse(map[string]any{
		"type": "finish",
	}, state2)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	usage2 := out2[0]["usage"].(map[string]any)
	if usage2["prompt_tokens"] != 7 {
		t.Errorf("fallback prompt_tokens = %v, want 7", usage2["prompt_tokens"])
	}
	if usage2["total_tokens"] != 10 {
		t.Errorf("fallback total_tokens = %v, want 10", usage2["total_tokens"])
	}
}

func TestCommandCodeOpenAIErrorTwoChunks(t *testing.T) {
	state := NewStreamState()
	out, err := commandcodeToOpenAIResponse(map[string]any{
		"type":  "error",
		"error": "boom",
	}, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	content := out[0]["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)["content"]
	if content != "\n\n[CommandCode error: boom]" {
		t.Errorf("error content = %v", content)
	}
	fr := out[1]["choices"].([]any)[0].(map[string]any)["finish_reason"]
	if fr != "stop" {
		t.Errorf("error finish_reason = %v, want stop", fr)
	}
}

func TestCommandCodeOpenAIFinishReasonMapping(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"stop", "stop"},
		{"length", "length"},
		{"tool-calls", "tool_calls"},
		{"tool_use", "tool_calls"},
		{"content-filter", "content_filter"},
		{"error", "stop"},
		{"", "stop"},
		{"unknown", "unknown"},
	}
	for _, tc := range cases {
		state := NewStreamState()
		_, err := commandcodeToOpenAIResponse(map[string]any{
			"type":         "finish-step",
			"finishReason": tc.in,
		}, state)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		out, err := commandcodeToOpenAIResponse(map[string]any{
			"type": "finish",
		}, state)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		got := out[0]["choices"].([]any)[0].(map[string]any)["finish_reason"]
		if got != tc.want {
			t.Errorf("finishReason(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
