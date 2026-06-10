package translation

import (
	"runtime"
	"testing"
)

func TestCommandCodeSystemExtraction(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "system", "content": "sys1"},
			map[string]any{"role": "system", "content": "sys2"},
			map[string]any{"role": "user", "content": "hi"},
		},
	}
	out, err := openaiToCommandCodeRequest("cmdcode", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	params := out["params"].(map[string]any)
	sys, ok := params["system"].(string)
	if !ok {
		t.Fatalf("system not string: %v", params["system"])
	}
	if sys != "sys1\n\nsys2" {
		t.Errorf("system = %q, want %q", sys, "sys1\n\nsys2")
	}
}

func TestCommandCodeToolResultBlocks(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role":         "tool",
				"tool_call_id": "tc1",
				"name":         "foo",
				"content":      "result",
			},
		},
	}
	out, err := openaiToCommandCodeRequest("cmdcode", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	params := out["params"].(map[string]any)
	msgs := params["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(msgs))
	}
	m := msgs[0].(map[string]any)
	if m["role"] != "tool" {
		t.Errorf("role = %v, want tool", m["role"])
	}
	blocks := m["content"].([]any)
	b := blocks[0].(map[string]any)
	if b["type"] != "tool-result" {
		t.Errorf("type = %v, want tool-result", b["type"])
	}
	if b["toolCallId"] != "tc1" {
		t.Errorf("toolCallId = %v, want tc1", b["toolCallId"])
	}
	if b["toolName"] != "foo" {
		t.Errorf("toolName = %v, want foo", b["toolName"])
	}
	outVal := b["output"].(map[string]any)["value"]
	if outVal != "result" {
		t.Errorf("output.value = %v, want result", outVal)
	}
}

func TestCommandCodeAssistantToolCallBlocks(t *testing.T) {
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
							"arguments": `{"a":1}`,
						},
					},
					map[string]any{
						"id":   "tc2",
						"type": "function",
						"function": map[string]any{
							"name":      "bar",
							"arguments": "not-json",
						},
					},
				},
			},
		},
	}
	out, err := openaiToCommandCodeRequest("cmdcode", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	params := out["params"].(map[string]any)
	msgs := params["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(msgs))
	}
	m := msgs[0].(map[string]any)
	blocks := m["content"].([]any)
	if len(blocks) != 2 {
		t.Fatalf("len(blocks) = %d, want 2", len(blocks))
	}
	b0 := blocks[0].(map[string]any)
	if b0["type"] != "tool-call" {
		t.Errorf("block[0].type = %v", b0["type"])
	}
	input0 := b0["input"].(map[string]any)
	if input0["a"] != float64(1) {
		t.Errorf("input0 = %v", input0)
	}
	b1 := blocks[1].(map[string]any)
	input1 := b1["input"].(map[string]any)
	if len(input1) != 0 {
		t.Errorf("malformed args should become empty map, got %v", input1)
	}
}

func TestCommandCodeToolsBothShapes(t *testing.T) {
	body := map[string]any{
		"tools": []any{
			// OpenAI shape
			map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        "foo",
					"description": "does foo",
					"parameters":  map[string]any{"type": "object"},
				},
			},
			// Claude-like shape
			map[string]any{
				"name":         "bar",
				"description":  "does bar",
				"input_schema": map[string]any{"type": "object"},
			},
		},
	}
	out, err := openaiToCommandCodeRequest("cmdcode", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	params := out["params"].(map[string]any)
	tools := params["tools"].([]any)
	if len(tools) != 2 {
		t.Fatalf("len(tools) = %d, want 2", len(tools))
	}
	t0 := tools[0].(map[string]any)
	if t0["name"] != "foo" || t0["description"] != "does foo" {
		t.Errorf("tool[0] = %v", t0)
	}
	t1 := tools[1].(map[string]any)
	if t1["name"] != "bar" || t1["description"] != "does bar" {
		t.Errorf("tool[1] = %v", t1)
	}
}

func TestCommandCodeEnvelopeDefaults(t *testing.T) {
	body := map[string]any{}
	out, err := openaiToCommandCodeRequest("cmdcode", body, true, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if out["threadId"] == "" {
		t.Error("threadId should be non-empty")
	}
	if out["memory"] != "" {
		t.Errorf("memory = %v, want empty", out["memory"])
	}
	config := out["config"].(map[string]any)
	if config["environment"] != runtime.GOOS {
		t.Errorf("environment = %v, want %s", config["environment"], runtime.GOOS)
	}
	params := out["params"].(map[string]any)
	if params["max_tokens"] != 64000 {
		t.Errorf("max_tokens = %v, want 64000", params["max_tokens"])
	}
	if params["temperature"] != 0.3 {
		t.Errorf("temperature = %v, want 0.3", params["temperature"])
	}
	if params["stream"] != true {
		t.Errorf("stream = %v, want true", params["stream"])
	}
	if _, ok := params["system"]; ok {
		t.Error("system should not be present when empty")
	}
}

func TestCommandCodeImageOmitted(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{
						"type": "image_url",
						"image_url": map[string]any{
							"url": "data:image/png;base64,abc",
						},
					},
				},
			},
		},
	}
	out, err := openaiToCommandCodeRequest("cmdcode", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	params := out["params"].(map[string]any)
	msgs := params["messages"].([]any)
	blocks := msgs[0].(map[string]any)["content"].([]any)
	b := blocks[0].(map[string]any)
	if b["type"] != "text" || b["text"] != "[image omitted]" {
		t.Errorf("image block = %v, want [image omitted]", b)
	}
}
