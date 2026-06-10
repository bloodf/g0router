package translation

import (
	"encoding/json"
	"strings"
	"testing"
)

func state() *StreamState {
	return NewStreamState()
}

func TestClaudeResponseMessageStart(t *testing.T) {
	chunk := map[string]any{
		"id":      "chatcmpl-abc123xyz",
		"model":   "gpt-4",
		"choices": []any{map[string]any{"delta": map[string]any{}, "index": 0}},
	}
	out, err := openaiToClaudeResponse(chunk, state())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0]["type"] != "message_start" {
		t.Errorf("type = %q", out[0]["type"])
	}
	msg := out[0]["message"].(map[string]any)
	if msg["id"] != "abc123xyz" {
		t.Errorf("id = %q", msg["id"])
	}
	if msg["role"] != "assistant" {
		t.Errorf("role = %q", msg["role"])
	}
	if msg["model"] != "gpt-4" {
		t.Errorf("model = %q", msg["model"])
	}
}

func TestClaudeResponseTextDelta(t *testing.T) {
	s := state()
	// Prime with message_start.
	openaiToClaudeResponse(map[string]any{
		"id":      "chatcmpl-x",
		"model":   "gpt-4",
		"choices": []any{map[string]any{"delta": map[string]any{}, "index": 0}},
	}, s)
	out, err := openaiToClaudeResponse(map[string]any{
		"choices": []any{map[string]any{"delta": map[string]any{"content": "hello"}, "index": 0}},
	}, s)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2 (content_block_start + delta)", len(out))
	}
	if out[0]["type"] != "content_block_start" {
		t.Errorf("first event = %q", out[0]["type"])
	}
	if out[1]["type"] != "content_block_delta" {
		t.Errorf("second event = %q", out[1]["type"])
	}
	delta := out[1]["delta"].(map[string]any)
	if delta["type"] != "text_delta" || delta["text"] != "hello" {
		t.Errorf("delta = %v", delta)
	}
}

func TestClaudeResponseThinkingDelta(t *testing.T) {
	s := state()
	openaiToClaudeResponse(map[string]any{
		"id":      "chatcmpl-x",
		"model":   "gpt-4",
		"choices": []any{map[string]any{"delta": map[string]any{}, "index": 0}},
	}, s)
	out, err := openaiToClaudeResponse(map[string]any{
		"choices": []any{map[string]any{"delta": map[string]any{"reasoning_content": "think"}, "index": 0}},
	}, s)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	if out[0]["type"] != "content_block_start" {
		t.Errorf("first event = %q", out[0]["type"])
	}
	cb := out[0]["content_block"].(map[string]any)
	if cb["type"] != "thinking" {
		t.Errorf("block type = %q", cb["type"])
	}
	delta := out[1]["delta"].(map[string]any)
	if delta["type"] != "thinking_delta" || delta["thinking"] != "think" {
		t.Errorf("delta = %v", delta)
	}
}

func TestClaudeResponseToolCallBuffering(t *testing.T) {
	s := state()
	openaiToClaudeResponse(map[string]any{
		"id":      "chatcmpl-x",
		"model":   "gpt-4",
		"choices": []any{map[string]any{"delta": map[string]any{}, "index": 0}},
	}, s)

	// First tool call chunk with id starts block.
	_, err := openaiToClaudeResponse(map[string]any{
		"choices": []any{map[string]any{"delta": map[string]any{"tool_calls": []any{map[string]any{"index": 0, "id": "tc1", "function": map[string]any{"name": "Read"}}}}, "index": 0}},
	}, s)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	// Argument deltas must be buffered, not emitted.
	out, err := openaiToClaudeResponse(map[string]any{
		"choices": []any{map[string]any{"delta": map[string]any{"tool_calls": []any{map[string]any{"index": 0, "function": map[string]any{"arguments": `{"fil`}}}}, "index": 0}},
	}, s)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("arg delta emitted %d chunks before finish; want 0", len(out))
	}
}

func TestClaudeResponseStripsProxyPrefix(t *testing.T) {
	s := state()
	openaiToClaudeResponse(map[string]any{
		"id":      "chatcmpl-x",
		"model":   "gpt-4",
		"choices": []any{map[string]any{"delta": map[string]any{}, "index": 0}},
	}, s)
	out, err := openaiToClaudeResponse(map[string]any{
		"choices": []any{map[string]any{"delta": map[string]any{"tool_calls": []any{map[string]any{"index": 0, "id": "tc1", "function": map[string]any{"name": "proxy_Read"}}}}, "index": 0}},
	}, s)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	cb := out[0]["content_block"].(map[string]any)
	if cb["name"] != "Read" {
		t.Errorf("name = %q, want Read", cb["name"])
	}
}

func TestClaudeResponseToolArgSanitization(t *testing.T) {
	s := state()
	openaiToClaudeResponse(map[string]any{
		"id":      "chatcmpl-x",
		"model":   "gpt-4",
		"choices": []any{map[string]any{"delta": map[string]any{}, "index": 0}},
	}, s)
	openaiToClaudeResponse(map[string]any{
		"choices": []any{map[string]any{"delta": map[string]any{"tool_calls": []any{map[string]any{"index": 0, "id": "tc1", "function": map[string]any{"name": "Read"}}}}, "index": 0}},
	}, s)
	openaiToClaudeResponse(map[string]any{
		"choices": []any{map[string]any{"delta": map[string]any{"tool_calls": []any{map[string]any{"index": 0, "function": map[string]any{"arguments": `{"limit":"5000","pages":"1-5"}`}}}}, "index": 0}},
	}, s)

	// Finish flushes buffered args with sanitization.
	out, err := openaiToClaudeResponse(map[string]any{
		"choices": []any{map[string]any{"delta": map[string]any{}, "finish_reason": "tool_calls", "index": 0}},
	}, s)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	var inputDelta map[string]any
	for _, ev := range out {
		if ev["type"] == "content_block_delta" {
			delta := ev["delta"].(map[string]any)
			if delta["type"] == "input_json_delta" {
				inputDelta = delta
				break
			}
		}
	}
	if inputDelta == nil {
		t.Fatal("no input_json_delta found at finish")
	}
	payload := inputDelta["partial_json"].(string)
	var args map[string]any
	if err := json.Unmarshal([]byte(payload), &args); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	// limit string "5000" → number clamped to 2000; pages removed because file_path missing.
	if args["limit"] != float64(2000) {
		t.Errorf("limit = %v, want 2000", args["limit"])
	}
	if _, ok := args["pages"]; ok {
		t.Errorf("pages should be sanitized away, got %v", args["pages"])
	}
}

func TestClaudeResponseFinish(t *testing.T) {
	cases := []struct {
		finish   string
		expected string
	}{
		{"stop", "end_turn"},
		{"length", "max_tokens"},
		{"tool_calls", "tool_use"},
		{"unknown", "end_turn"},
	}
	for _, tc := range cases {
		t.Run(tc.finish, func(t *testing.T) {
			s := state()
			openaiToClaudeResponse(map[string]any{
				"id":      "chatcmpl-x",
				"model":   "gpt-4",
				"choices": []any{map[string]any{"delta": map[string]any{}, "index": 0}},
			}, s)
			out, err := openaiToClaudeResponse(map[string]any{
				"choices": []any{map[string]any{"delta": map[string]any{}, "finish_reason": tc.finish, "index": 0}},
			}, s)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			var delta map[string]any
			for _, ev := range out {
				if ev["type"] == "message_delta" {
					delta = ev["delta"].(map[string]any)
					break
				}
			}
			if delta == nil {
				t.Fatal("no message_delta found")
			}
			if delta["stop_reason"] != tc.expected {
				t.Errorf("stop_reason = %q, want %q", delta["stop_reason"], tc.expected)
			}
		})
	}
}

func TestClaudeResponseUsageCacheTokens(t *testing.T) {
	s := state()
	openaiToClaudeResponse(map[string]any{
		"id":      "chatcmpl-x",
		"model":   "gpt-4",
		"choices": []any{map[string]any{"delta": map[string]any{}, "index": 0}},
	}, s)
	out, err := openaiToClaudeResponse(map[string]any{
		"usage": map[string]any{
			"prompt_tokens":            100,
			"completion_tokens":        20,
			"prompt_tokens_details":    map[string]any{"cached_tokens": 30, "cache_creation_tokens": 10},
		},
		"choices": []any{map[string]any{"delta": map[string]any{}, "finish_reason": "stop", "index": 0}},
	}, s)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	var msgDelta map[string]any
	for _, ev := range out {
		if ev["type"] == "message_delta" {
			msgDelta = ev
			break
		}
	}
	if msgDelta == nil {
		t.Fatal("no message_delta found")
	}
	usage := msgDelta["usage"].(map[string]any)
	// input_tokens = prompt_tokens - cached_tokens - cache_creation_tokens
	if usage["input_tokens"] != float64(60) {
		t.Errorf("input_tokens = %v, want 60", usage["input_tokens"])
	}
	if usage["output_tokens"] != float64(20) {
		t.Errorf("output_tokens = %v, want 20", usage["output_tokens"])
	}
	if usage["cache_read_input_tokens"] != float64(30) {
		t.Errorf("cache_read_input_tokens = %v, want 30", usage["cache_read_input_tokens"])
	}
	if usage["cache_creation_input_tokens"] != float64(10) {
		t.Errorf("cache_creation_input_tokens = %v, want 10", usage["cache_creation_input_tokens"])
	}
}

func TestClaudeResponseMessageStartFallbackID(t *testing.T) {
	chunk := map[string]any{
		"id":      "chat",
		"model":   "gpt-4",
		"choices": []any{map[string]any{"delta": map[string]any{}, "index": 0}},
	}
	out, err := openaiToClaudeResponse(chunk, state())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	msg := out[0]["message"].(map[string]any)
	id := msg["id"].(string)
	if id == "chat" || len(id) < 8 || !strings.HasPrefix(id, "msg_") {
		t.Errorf("fallback id = %q", id)
	}
}
