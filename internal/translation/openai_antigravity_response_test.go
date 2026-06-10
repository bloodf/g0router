package translation

import (
	"testing"
)

func TestNewRegistryWiresOpenAIAntigravityResponse(t *testing.T) {
	reg := NewRegistry()
	if reg.ResponseTranslatorFor(FormatOpenAI, FormatAntigravity) == nil {
		t.Error("NewRegistry must wire openai->antigravity response translator")
	}
}

func TestOpenAIAntigravityResponseAccumulatesToolCalls(t *testing.T) {
	state := NewStreamState()
	fn := openaiToAntigravityResponse

	// First chunk: partial tool call
	chunks, err := fn(map[string]any{
		"choices": []any{map[string]any{
			"delta": map[string]any{
				"tool_calls": []any{map[string]any{
					"index": 0,
					"id":    "call_1",
					"function": map[string]any{
						"name":      "my_tool",
						"arguments": `{"a":`,
					},
				}},
			},
		}},
	}, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(chunks) != 0 {
		t.Fatalf("expected no emit on partial tool call, got %d chunks", len(chunks))
	}

	// Second chunk: finish reason
	chunks, err = fn(map[string]any{
		"choices": []any{map[string]any{
			"delta":         map[string]any{},
			"finish_reason": "tool_calls",
		}},
	}, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	resp := chunks[0]["response"].(map[string]any)
	candidates := resp["candidates"].([]any)
	candidate := candidates[0].(map[string]any)
	content := candidate["content"].(map[string]any)
	parts := content["parts"].([]any)
	if len(parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(parts))
	}
	fc := parts[0].(map[string]any)["functionCall"].(map[string]any)
	if fc["name"] != "my_tool" {
		t.Errorf("name = %v, want my_tool", fc["name"])
	}
}

func TestOpenAIAntigravityResponseFinishReasonMapping(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"stop", "STOP"},
		{"length", "MAX_TOKENS"},
		{"tool_calls", "STOP"},
		{"content_filter", "SAFETY"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			state := NewStreamState()
			chunks, err := openaiToAntigravityResponse(map[string]any{
				"choices": []any{map[string]any{
					"delta":         map[string]any{"content": ""},
					"finish_reason": tc.in,
				}},
			}, state)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			resp := chunks[0]["response"].(map[string]any)
			candidate := resp["candidates"].([]any)[0].(map[string]any)
			if candidate["finishReason"] != tc.want {
				t.Errorf("finishReason = %v, want %s", candidate["finishReason"], tc.want)
			}
		})
	}
}

func TestOpenAIAntigravityResponseUsageMetadata(t *testing.T) {
	state := NewStreamState()
	chunks, err := openaiToAntigravityResponse(map[string]any{
		"choices": []any{map[string]any{
			"delta":         map[string]any{},
			"finish_reason": "stop",
		}},
		"usage": map[string]any{
			"prompt_tokens":            float64(10),
			"completion_tokens":        float64(5),
			"total_tokens":             float64(15),
			"prompt_tokens_details":    map[string]any{"cached_tokens": float64(3)},
			"completion_tokens_details": map[string]any{"reasoning_tokens": float64(2)},
		},
	}, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	resp := chunks[0]["response"].(map[string]any)
	um := resp["usageMetadata"].(map[string]any)
	if um["promptTokenCount"] != float64(10) {
		t.Errorf("promptTokenCount = %v", um["promptTokenCount"])
	}
	if um["candidatesTokenCount"] != float64(5) {
		t.Errorf("candidatesTokenCount = %v", um["candidatesTokenCount"])
	}
	if um["totalTokenCount"] != float64(15) {
		t.Errorf("totalTokenCount = %v", um["totalTokenCount"])
	}
	if um["cachedContentTokenCount"] != float64(3) {
		t.Errorf("cachedContentTokenCount = %v", um["cachedContentTokenCount"])
	}
	if um["thoughtsTokenCount"] != float64(2) {
		t.Errorf("thoughtsTokenCount = %v", um["thoughtsTokenCount"])
	}
}

func TestOpenAIAntigravityToolCallsOrderedAndBadJSONFallsBack(t *testing.T) {
	state := NewStreamState()
	mkChunk := func(idx int, id, name, args string) map[string]any {
		return map[string]any{
			"choices": []any{map[string]any{"delta": map[string]any{
				"tool_calls": []any{map[string]any{
					"index": float64(idx),
					"id":    id,
					"function": map[string]any{
						"name":      name,
						"arguments": args,
					},
				}},
			}}},
		}
	}
	// Arrive out of order; index 1 carries malformed JSON arguments.
	if _, err := openaiToAntigravityResponse(mkChunk(1, "b", "second", `{"x":`), state); err != nil {
		t.Fatalf("accumulate idx1: %v", err)
	}
	if _, err := openaiToAntigravityResponse(mkChunk(0, "a", "first", `{"ok":true}`), state); err != nil {
		t.Fatalf("accumulate idx0: %v", err)
	}
	chunks, err := openaiToAntigravityResponse(map[string]any{
		"choices": []any{map[string]any{
			"delta":         map[string]any{},
			"finish_reason": "tool_calls",
		}},
	}, state)
	if err != nil {
		t.Fatalf("finish: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("len(chunks) = %d, want 1", len(chunks))
	}
	response, ok := chunks[0]["response"].(map[string]any)
	if !ok {
		t.Fatalf("response envelope missing: %v", chunks[0])
	}
	candidates := response["candidates"].([]any)
	content := candidates[0].(map[string]any)["content"].(map[string]any)
	parts := content["parts"].([]any)
	if len(parts) != 2 {
		t.Fatalf("len(parts) = %d, want 2 functionCall parts", len(parts))
	}
	fc0 := parts[0].(map[string]any)["functionCall"].(map[string]any)
	fc1 := parts[1].(map[string]any)["functionCall"].(map[string]any)
	if fc0["name"] != "first" || fc1["name"] != "second" {
		t.Errorf("tool call order = %v, %v — want first, second (ascending index)", fc0["name"], fc1["name"])
	}
	args1, ok := fc1["args"].(map[string]any)
	if !ok || len(args1) != 0 {
		t.Errorf("malformed JSON args must fall back to empty object, got %v", fc1["args"])
	}
}
