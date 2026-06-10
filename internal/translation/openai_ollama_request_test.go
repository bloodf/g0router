package translation

import (
	"testing"
)

func TestOpenAIOllamaOptionsMapping(t *testing.T) {
	body := map[string]any{
		"temperature": 0.7,
		"max_tokens":  512,
		"top_p":       0.9,
	}
	out, err := openaiToOllamaRequest("llama3", body, true, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if out["model"] != "llama3" {
		t.Errorf("model = %v, want llama3", out["model"])
	}
	if out["stream"] != true {
		t.Errorf("stream = %v, want true", out["stream"])
	}
	opts, ok := out["options"].(map[string]any)
	if !ok {
		t.Fatalf("options missing")
	}
	if opts["temperature"] != 0.7 {
		t.Errorf("temperature = %v, want 0.7", opts["temperature"])
	}
	if opts["num_predict"] != 512 {
		t.Errorf("num_predict = %v, want 512", opts["num_predict"])
	}
	if opts["top_p"] != 0.9 {
		t.Errorf("top_p = %v, want 0.9", opts["top_p"])
	}
}

func TestOpenAIOllamaToolResultNaming(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "assistant",
				"tool_calls": []any{
					map[string]any{
						"id": "tc1",
						"function": map[string]any{
							"name": "lookup",
						},
					},
				},
			},
			map[string]any{
				"role":         "tool",
				"tool_call_id": "tc1",
				"content":      "result1",
			},
			map[string]any{
				"role":         "tool",
				"tool_call_id": "tc_missing",
				"name":         "fallback_name",
				"content":      "result2",
			},
			map[string]any{
				"role":         "tool",
				"tool_call_id": "tc_missing2",
				"content":      "result3",
			},
		},
	}
	out, err := openaiToOllamaRequest("llama3", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	msgs := out["messages"].([]any)
	if len(msgs) != 4 {
		t.Fatalf("len(messages) = %d, want 4", len(msgs))
	}
	// first message is assistant with tool_calls
	if msgs[0].(map[string]any)["role"] != "assistant" {
		t.Errorf("msg[0].role = %v, want assistant", msgs[0].(map[string]any)["role"])
	}
	m1 := msgs[1].(map[string]any)
	if m1["tool_name"] != "lookup" {
		t.Errorf("msg[1].tool_name = %v, want lookup", m1["tool_name"])
	}
	m2 := msgs[2].(map[string]any)
	if m2["tool_name"] != "fallback_name" {
		t.Errorf("msg[2].tool_name = %v, want fallback_name", m2["tool_name"])
	}
	m3 := msgs[3].(map[string]any)
	if m3["tool_name"] != "unknown_tool" {
		t.Errorf("msg[3].tool_name = %v, want unknown_tool", m3["tool_name"])
	}
}

func TestOpenAIOllamaAssistantToolCallsParsedArgs(t *testing.T) {
	// Object arguments pass through
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "assistant",
				"tool_calls": []any{
					map[string]any{
						"id":   "tc1",
						"type": "function",
						"function": map[string]any{
							"name":      "foo",
							"arguments": map[string]any{"a": 1},
						},
					},
				},
			},
		},
	}
	out, err := openaiToOllamaRequest("llama3", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	msgs := out["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(msgs))
	}
	tcs := msgs[0].(map[string]any)["tool_calls"].([]any)
	fn := tcs[0].(map[string]any)["function"].(map[string]any)
	args := fn["arguments"].(map[string]any)
	var aVal float64
	switch v := args["a"].(type) {
	case float64:
		aVal = v
	case int:
		aVal = float64(v)
	default:
		t.Fatalf("unexpected type for a: %T", args["a"])
	}
	if aVal != 1 {
		t.Errorf("arguments[a] = %v, want 1", args["a"])
	}

	// Malformed string arguments → error
	body2 := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "assistant",
				"tool_calls": []any{
					map[string]any{
						"id":   "tc1",
						"type": "function",
						"function": map[string]any{
							"name":      "foo",
							"arguments": "not-json",
						},
					},
				},
			},
		},
	}
	_, err = openaiToOllamaRequest("llama3", body2, false, nil)
	if err == nil {
		t.Fatal("expected error for malformed JSON arguments")
	}
}

func TestOpenAIOllamaContentFlattenAndImages(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "text", "text": "hello"},
					map[string]any{"type": "text", "text": "world"},
					map[string]any{
						"type": "image_url",
						"image_url": map[string]any{
							"url": "data:image/png;base64,abc123",
						},
					},
					map[string]any{
						"type": "image_url",
						"image_url": map[string]any{
							"url": "https://example.com/img.png",
						},
					},
				},
			},
		},
	}
	out, err := openaiToOllamaRequest("llama3", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	msgs := out["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(msgs))
	}
	m := msgs[0].(map[string]any)
	if m["content"] != "hello\nworld" {
		t.Errorf("content = %q, want %q", m["content"], "hello\nworld")
	}
	imgs := m["images"].([]any)
	if len(imgs) != 1 {
		t.Fatalf("len(images) = %d, want 1", len(imgs))
	}
	if imgs[0] != "abc123" {
		t.Errorf("image = %v, want abc123", imgs[0])
	}
}

func TestOpenAIOllamaSkipsEmptyNonAssistant(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": ""},
			map[string]any{"role": "assistant", "content": ""},
			map[string]any{"role": "user", "content": "hi"},
		},
	}
	out, err := openaiToOllamaRequest("llama3", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	msgs := out["messages"].([]any)
	if len(msgs) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(msgs))
	}
	if msgs[0].(map[string]any)["role"] != "assistant" {
		t.Errorf("msg[0].role = %v, want assistant", msgs[0].(map[string]any)["role"])
	}
	if msgs[1].(map[string]any)["role"] != "user" {
		t.Errorf("msg[1].role = %v, want user", msgs[1].(map[string]any)["role"])
	}
}
