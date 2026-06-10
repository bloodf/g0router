package translation

import (
	"testing"
)

func TestNewRegistryWiresClaudeOpenAIResponse(t *testing.T) {
	reg := NewRegistry()
	if reg.ResponseTranslatorFor(FormatClaude, FormatOpenAI) == nil {
		t.Error("NewRegistry must wire claude->openai response translator")
	}
}

func TestClaudeOpenAIMessageStart(t *testing.T) {
	s := NewStreamState()
	chunk := map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":    "msg_abc123",
			"model": "claude-3-opus",
		},
	}
	out, err := claudeToOpenAIResponse(chunk, s)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	ch := out[0]
	if ch["id"] != "chatcmpl-msg_abc123" {
		t.Errorf("id = %q", ch["id"])
	}
	choices := ch["choices"].([]any)
	delta := choices[0].(map[string]any)["delta"].(map[string]any)
	if delta["role"] != "assistant" {
		t.Errorf("delta.role = %v", delta["role"])
	}
}

func TestClaudeOpenAITextDelta(t *testing.T) {
	s := NewStreamState()
	// message_start
	claudeToOpenAIResponse(map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":    "msg_1",
			"model": "claude-3",
		},
	}, s)
	// text delta
	out, err := claudeToOpenAIResponse(map[string]any{
		"type": "content_block_delta",
		"index": 0,
		"delta": map[string]any{"type": "text_delta", "text": "hello"},
	}, s)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	choices := out[0]["choices"].([]any)
	delta := choices[0].(map[string]any)["delta"].(map[string]any)
	if delta["content"] != "hello" {
		t.Errorf("content = %v", delta["content"])
	}
}

func TestClaudeOpenAIThinkingNoMarkers(t *testing.T) {
	s := NewStreamState()
	// message_start
	claudeToOpenAIResponse(map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":    "msg_1",
			"model": "claude-3",
		},
	}, s)
	// thinking block start
	out, err := claudeToOpenAIResponse(map[string]any{
		"type":           "content_block_start",
		"index":          0,
		"content_block":  map[string]any{"type": "thinking"},
	}, s)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("thinking start emitted %d chunks; want 0", len(out))
	}
	// thinking delta
	out, err = claudeToOpenAIResponse(map[string]any{
		"type":  "content_block_delta",
		"index": 0,
		"delta": map[string]any{"type": "thinking_delta", "thinking": "reasoning"},
	}, s)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	choices := out[0]["choices"].([]any)
	delta := choices[0].(map[string]any)["delta"].(map[string]any)
	if delta["reasoning_content"] != "reasoning" {
		t.Errorf("reasoning_content = %v", delta["reasoning_content"])
	}
	// thinking block stop
	out, err = claudeToOpenAIResponse(map[string]any{
		"type":  "content_block_stop",
		"index": 0,
	}, s)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("thinking stop emitted %d chunks; want 0", len(out))
	}
}

func TestClaudeOpenAIToolUseStartAndArgs(t *testing.T) {
	s := NewStreamState()
	// message_start
	claudeToOpenAIResponse(map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":    "msg_1",
			"model": "claude-3",
		},
	}, s)
	// tool_use start
	out, err := claudeToOpenAIResponse(map[string]any{
		"type":          "content_block_start",
		"index":         1,
		"content_block": map[string]any{"type": "tool_use", "id": "tu1", "name": "Read"},
	}, s)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	choices := out[0]["choices"].([]any)
	delta := choices[0].(map[string]any)["delta"].(map[string]any)
	toolCalls := delta["tool_calls"].([]any)
	if len(toolCalls) != 1 {
		t.Fatalf("len(tool_calls) = %d, want 1", len(toolCalls))
	}
	tc := toolCalls[0].(map[string]any)
	if tc["index"] != 0 {
		t.Errorf("tool_call index = %v, want 0", tc["index"])
	}
	if tc["id"] != "tu1" {
		t.Errorf("tool_call id = %v", tc["id"])
	}
	fn := tc["function"].(map[string]any)
	if fn["name"] != "Read" {
		t.Errorf("function.name = %v", fn["name"])
	}
	// input_json_delta
	out, err = claudeToOpenAIResponse(map[string]any{
		"type":  "content_block_delta",
		"index": 1,
		"delta": map[string]any{"type": "input_json_delta", "partial_json": `{"a":1`},
	}, s)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	choices = out[0]["choices"].([]any)
	delta = choices[0].(map[string]any)["delta"].(map[string]any)
	toolCalls = delta["tool_calls"].([]any)
	tc = toolCalls[0].(map[string]any)
	fn = tc["function"].(map[string]any)
	if fn["arguments"] != `{"a":1` {
		t.Errorf("arguments = %v", fn["arguments"])
	}
}

func TestClaudeOpenAIServerToolSkipped(t *testing.T) {
	s := NewStreamState()
	// message_start
	claudeToOpenAIResponse(map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":    "msg_1",
			"model": "claude-3",
		},
	}, s)
	// server_tool_use start
	out, err := claudeToOpenAIResponse(map[string]any{
		"type":          "content_block_start",
		"index":         0,
		"content_block": map[string]any{"type": "server_tool_use"},
	}, s)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("server_tool start emitted %d chunks; want 0", len(out))
	}
	// delta for server tool should be skipped
	out, err = claudeToOpenAIResponse(map[string]any{
		"type":  "content_block_delta",
		"index": 0,
		"delta": map[string]any{"type": "text_delta", "text": "ignored"},
	}, s)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("server_tool delta emitted %d chunks; want 0", len(out))
	}
	// stop for server tool should be skipped
	out, err = claudeToOpenAIResponse(map[string]any{
		"type":  "content_block_stop",
		"index": 0,
	}, s)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("server_tool stop emitted %d chunks; want 0", len(out))
	}
}

func TestClaudeOpenAIStopReasonMapping(t *testing.T) {
	cases := []struct {
		stopReason string
		want       string
	}{
		{"end_turn", "stop"},
		{"max_tokens", "length"},
		{"tool_use", "tool_calls"},
		{"stop_sequence", "stop"},
		{"unknown", "stop"},
	}
	for _, tc := range cases {
		t.Run(tc.stopReason, func(t *testing.T) {
			s := NewStreamState()
			claudeToOpenAIResponse(map[string]any{
				"type": "message_start",
				"message": map[string]any{
					"id":    "msg_1",
					"model": "claude-3",
				},
			}, s)
			out, err := claudeToOpenAIResponse(map[string]any{
				"type": "message_delta",
				"delta": map[string]any{
					"stop_reason": tc.stopReason,
				},
				"usage": map[string]any{
					"input_tokens":  10,
					"output_tokens": 5,
				},
			}, s)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			var finishChunk map[string]any
			for _, ch := range out {
				choices := ch["choices"].([]any)
				if choices[0].(map[string]any)["finish_reason"] != nil {
					finishChunk = ch
					break
				}
			}
			if finishChunk == nil {
				t.Fatal("no finish chunk found")
			}
			choices := finishChunk["choices"].([]any)
			if choices[0].(map[string]any)["finish_reason"] != tc.want {
				t.Errorf("finish_reason = %v, want %v", choices[0].(map[string]any)["finish_reason"], tc.want)
			}
		})
	}
}

func TestClaudeOpenAIUsageCacheTokens(t *testing.T) {
	s := NewStreamState()
	claudeToOpenAIResponse(map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":    "msg_1",
			"model": "claude-3",
		},
	}, s)
	out, err := claudeToOpenAIResponse(map[string]any{
		"type": "message_delta",
		"delta": map[string]any{"stop_reason": "end_turn"},
		"usage": map[string]any{
			"input_tokens":                  10,
			"output_tokens":                 5,
			"cache_read_input_tokens":       3,
			"cache_creation_input_tokens":   2,
		},
	}, s)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	var finishChunk map[string]any
	for _, ch := range out {
		choices := ch["choices"].([]any)
		if choices[0].(map[string]any)["finish_reason"] != nil {
			finishChunk = ch
			break
		}
	}
	if finishChunk == nil {
		t.Fatal("no finish chunk found")
	}
	usage := finishChunk["usage"].(map[string]any)
	// prompt_tokens = input_tokens + cache_read + cache_creation = 10 + 3 + 2 = 15
	if usage["prompt_tokens"] != 15 {
		t.Errorf("prompt_tokens = %v, want 15", usage["prompt_tokens"])
	}
	if usage["completion_tokens"] != 5 {
		t.Errorf("completion_tokens = %v, want 5", usage["completion_tokens"])
	}
	if usage["total_tokens"] != 20 {
		t.Errorf("total_tokens = %v, want 20", usage["total_tokens"])
	}
	details := usage["prompt_tokens_details"].(map[string]any)
	if details["cached_tokens"] != 3 {
		t.Errorf("cached_tokens = %v, want 3", details["cached_tokens"])
	}
	if details["cache_creation_tokens"] != 2 {
		t.Errorf("cache_creation_tokens = %v, want 2", details["cache_creation_tokens"])
	}
}

func TestClaudeOpenAIMessageStopFallback(t *testing.T) {
	s := NewStreamState()
	claudeToOpenAIResponse(map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":    "msg_1",
			"model": "claude-3",
		},
	}, s)
	// message_stop without prior finish_reason
	out, err := claudeToOpenAIResponse(map[string]any{
		"type": "message_stop",
	}, s)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	choices := out[0]["choices"].([]any)
	if choices[0].(map[string]any)["finish_reason"] != "stop" {
		t.Errorf("finish_reason = %v, want stop", choices[0].(map[string]any)["finish_reason"])
	}
}
