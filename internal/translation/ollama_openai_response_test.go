package translation

import (
	"testing"
	"time"
)

func TestOllamaOpenAIInitAndContentDelta(t *testing.T) {
	state := NewStreamState()
	state.Model = "fallback-model"

	chunk := map[string]any{
		"model": "llama3",
		"message": map[string]any{
			"role":    "assistant",
			"content": "hello",
		},
		"done": false,
	}
	out, err := ollamaToOpenAIResponse(chunk, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	c := out[0]
	if c["object"] != "chat.completion.chunk" {
		t.Errorf("object = %v", c["object"])
	}
	if c["model"] != "llama3" {
		t.Errorf("model = %v, want llama3", c["model"])
	}
	choices := c["choices"].([]any)
	delta := choices[0].(map[string]any)["delta"].(map[string]any)
	if delta["content"] != "hello" {
		t.Errorf("delta.content = %v, want hello", delta["content"])
	}
	if choices[0].(map[string]any)["finish_reason"] != nil {
		t.Errorf("finish_reason should be nil on content chunk")
	}

	// Second chunk should reuse id/created/model from state
	chunk2 := map[string]any{
		"message": map[string]any{
			"content": " world",
		},
		"done": false,
	}
	out2, err := ollamaToOpenAIResponse(chunk2, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if out2[0]["id"] != c["id"] {
		t.Errorf("id changed between chunks")
	}
	if out2[0]["created"] != c["created"] {
		t.Errorf("created changed between chunks")
	}
}

func TestOllamaOpenAIThinkingDelta(t *testing.T) {
	state := NewStreamState()
	chunk := map[string]any{
		"model": "llama3",
		"message": map[string]any{
			"thinking": "deep thought",
		},
		"done": false,
	}
	out, err := ollamaToOpenAIResponse(chunk, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	choices := out[0]["choices"].([]any)
	delta := choices[0].(map[string]any)["delta"].(map[string]any)
	if delta["reasoning_content"] != "deep thought" {
		t.Errorf("reasoning_content = %v, want deep thought", delta["reasoning_content"])
	}
}

func TestOllamaOpenAIToolCallsAndDoneReason(t *testing.T) {
	// Path 1: done_reason == "tool_calls"
	state := NewStreamState()
	out, err := ollamaToOpenAIResponse(map[string]any{
		"model": "llama3",
		"message": map[string]any{
			"tool_calls": []any{
				map[string]any{
					"function": map[string]any{
						"name": "foo",
						"arguments": map[string]any{"a": 1},
					},
				},
			},
		},
		"done": false,
	}, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	choices := out[0]["choices"].([]any)
	delta := choices[0].(map[string]any)["delta"].(map[string]any)
	tcs := delta["tool_calls"].([]any)
	if len(tcs) != 1 {
		t.Fatalf("len(tool_calls) = %d, want 1", len(tcs))
	}

	// done with done_reason tool_calls
	out2, err := ollamaToOpenAIResponse(map[string]any{
		"done":        true,
		"done_reason": "tool_calls",
	}, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	fr := out2[0]["choices"].([]any)[0].(map[string]any)["finish_reason"]
	if fr != "tool_calls" {
		t.Errorf("finish_reason = %v, want tool_calls", fr)
	}

	// Path 2: prior tool_calls seen but no done_reason
	state2 := NewStreamState()
	_, err = ollamaToOpenAIResponse(map[string]any{
		"model": "llama3",
		"message": map[string]any{
			"tool_calls": []any{
				map[string]any{
					"function": map[string]any{
						"name": "foo",
						"arguments": `{"a":1}`,
					},
				},
			},
		},
		"done": false,
	}, state2)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	out3, err := ollamaToOpenAIResponse(map[string]any{
		"done": true,
	}, state2)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	fr2 := out3[0]["choices"].([]any)[0].(map[string]any)["finish_reason"]
	if fr2 != "tool_calls" {
		t.Errorf("finish_reason = %v, want tool_calls (hadToolCalls path)", fr2)
	}
}

func TestOllamaOpenAIUsageOnDone(t *testing.T) {
	state := NewStreamState()
	out, err := ollamaToOpenAIResponse(map[string]any{
		"done":               true,
		"prompt_eval_count":  100,
		"eval_count":         50,
	}, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	usage := out[0]["usage"].(map[string]any)
	if usage["prompt_tokens"] != 100 {
		t.Errorf("prompt_tokens = %v, want 100", usage["prompt_tokens"])
	}
	if usage["completion_tokens"] != 50 {
		t.Errorf("completion_tokens = %v, want 50", usage["completion_tokens"])
	}
	if usage["total_tokens"] != 150 {
		t.Errorf("total_tokens = %v, want 150", usage["total_tokens"])
	}
}

func TestOllamaBodyToOpenAI(t *testing.T) {
	body := map[string]any{
		"model": "llama3",
		"message": map[string]any{
			"content":  "hi",
			"thinking": "think",
			"tool_calls": []any{
				map[string]any{
					"id": "tc1",
					"function": map[string]any{
						"name":      "foo",
						"arguments": `{"a":1}`,
					},
				},
			},
		},
		"done_reason": "stop",
	}
	out := OllamaBodyToOpenAI(body)
	if out["object"] != "chat.completion" {
		t.Errorf("object = %v, want chat.completion", out["object"])
	}
	choices := out["choices"].([]any)
	msg := choices[0].(map[string]any)["message"].(map[string]any)
	if msg["content"] != "hi" {
		t.Errorf("content = %v, want hi", msg["content"])
	}
	if msg["reasoning_content"] != "think" {
		t.Errorf("reasoning_content = %v, want think", msg["reasoning_content"])
	}
	tcs := msg["tool_calls"].([]any)
	if len(tcs) != 1 {
		t.Fatalf("len(tool_calls) = %d, want 1", len(tcs))
	}
	fr := choices[0].(map[string]any)["finish_reason"]
	if fr != "tool_calls" {
		t.Errorf("finish_reason = %v, want tool_calls", fr)
	}

	// Empty content default
	body2 := map[string]any{
		"model":   "llama3",
		"message": map[string]any{},
	}
	out2 := OllamaBodyToOpenAI(body2)
	msg2 := out2["choices"].([]any)[0].(map[string]any)["message"].(map[string]any)
	if msg2["content"] != "" {
		t.Errorf("empty content default = %v, want empty string", msg2["content"])
	}

	// Ensure created is a reasonable unix timestamp (within last minute)
	created, ok := out["created"].(int64)
	if !ok {
		t.Fatalf("created not int64: %T", out["created"])
	}
	now := time.Now().Unix()
	if created < now-60 || created > now+60 {
		t.Errorf("created = %d, expected near %d", created, now)
	}
}
