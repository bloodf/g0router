package translation

import (
	"testing"
)

func TestOpenAIResponsesPassthroughWhenInput(t *testing.T) {
	body := map[string]any{
		"input": []any{map[string]any{"type": "message"}},
		"model": "gpt-4",
	}
	out, err := openaiToResponsesRequest("gpt-4", body, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["model"] != "gpt-4" {
		t.Errorf("expected model gpt-4, got %v", out["model"])
	}
	if out["stream"] != true {
		t.Errorf("expected stream true, got %v", out["stream"])
	}
	// Original input should be preserved
	input, ok := out["input"].([]any)
	if !ok || len(input) != 1 {
		t.Errorf("expected input preserved, got %v", out["input"])
	}
}

func TestOpenAIResponsesSystemToInstructions(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "system", "content": "Be helpful"},
			map[string]any{"role": "user", "content": "hi"},
		},
	}
	out, err := openaiToResponsesRequest("gpt-4", body, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["instructions"] != "Be helpful" {
		t.Errorf("expected instructions='Be helpful', got %v", out["instructions"])
	}
	input := out["input"].([]any)
	// Only user message should be in input
	if len(input) != 1 {
		t.Fatalf("expected 1 input item, got %d", len(input))
	}
	msg := input[0].(map[string]any)
	if msg["role"] != "user" {
		t.Errorf("expected user message, got %v", msg["role"])
	}
}

func TestOpenAIResponsesNoSystemEmptyInstructions(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "hi"},
		},
	}
	out, err := openaiToResponsesRequest("gpt-4", body, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["instructions"] != "" {
		t.Errorf("expected empty instructions, got %v", out["instructions"])
	}
}

func TestOpenAIResponsesContentMapping(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "hello"},
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "text", "text": "text part"},
					map[string]any{"type": "image_url", "image_url": map[string]any{"url": "http://img", "detail": "high"}},
					map[string]any{"type": "unknown_block", "foo": "bar"},
				},
			},
			map[string]any{"role": "assistant", "content": "reply"},
		},
	}
	out, err := openaiToResponsesRequest("gpt-4", body, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	input := out["input"].([]any)
	if len(input) != 3 {
		t.Fatalf("expected 3 input items, got %d", len(input))
	}

	// User string content
	item0 := input[0].(map[string]any)
	content0 := item0["content"].([]any)
	if content0[0].(map[string]any)["type"] != "input_text" || content0[0].(map[string]any)["text"] != "hello" {
		t.Errorf("unexpected item0: %v", item0)
	}

	// User array content
	item1 := input[1].(map[string]any)
	content1 := item1["content"].([]any)
	if len(content1) != 3 {
		t.Fatalf("expected 3 content blocks, got %d", len(content1))
	}
	if content1[0].(map[string]any)["type"] != "input_text" {
		t.Errorf("expected input_text, got %v", content1[0])
	}
	img := content1[1].(map[string]any)
	if img["type"] != "input_image" || img["image_url"] != "http://img" || img["detail"] != "high" {
		t.Errorf("unexpected image block: %v", img)
	}
	unknown := content1[2].(map[string]any)
	if unknown["type"] != "input_text" {
		t.Errorf("expected unknown serialized as input_text, got %v", unknown)
	}

	// Assistant string content
	item2 := input[2].(map[string]any)
	content2 := item2["content"].([]any)
	if content2[0].(map[string]any)["type"] != "output_text" || content2[0].(map[string]any)["text"] != "reply" {
		t.Errorf("unexpected item2: %v", item2)
	}
}

func TestOpenAIResponsesToolCallsClamped(t *testing.T) {
	longID := ""
	for i := 0; i < 65; i++ {
		longID += "a"
	}
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "assistant",
				"content": nil,
				"tool_calls": []any{
					map[string]any{
						"id": longID,
						"type": "function",
						"function": map[string]any{"name": "toolA", "arguments": "{}"},
					},
				},
			},
		},
	}
	out, err := openaiToResponsesRequest("gpt-4", body, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	input := out["input"].([]any)
	if len(input) != 1 {
		t.Fatalf("expected 1 input item, got %d", len(input))
	}
	fc := input[0].(map[string]any)
	callID := fc["call_id"].(string)
	if len(callID) != 64 {
		t.Errorf("expected clamped call_id len 64, got %d", len(callID))
	}
}

func TestOpenAIResponsesToolResultCoercion(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "tool", "tool_call_id": "c1", "content": "plain text"},
			map[string]any{"role": "tool", "tool_call_id": "c2", "content": []any{map[string]any{"text": "part1"}, map[string]any{"text": "part2"}}},
			map[string]any{"role": "tool", "tool_call_id": "c3", "content": map[string]any{"result": 42}},
		},
	}
	out, err := openaiToResponsesRequest("gpt-4", body, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	input := out["input"].([]any)
	if len(input) != 3 {
		t.Fatalf("expected 3 input items, got %d", len(input))
	}
	if input[0].(map[string]any)["output"] != "plain text" {
		t.Errorf("expected plain text output, got %v", input[0].(map[string]any)["output"])
	}
	if input[1].(map[string]any)["output"] != "part1part2" {
		t.Errorf("expected array joined output, got %v", input[1].(map[string]any)["output"])
	}
	// JSON object should be JSON-stringified
	out2 := input[2].(map[string]any)["output"].(string)
	if out2 != `{"result":42}` {
		t.Errorf("unexpected output2: %v", out2)
	}
}

func TestOpenAIResponsesToolsFlatShape(t *testing.T) {
	body := map[string]any{
		"tools": []any{
			map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        "get_weather",
					"description": "weather",
					"parameters":  map[string]any{"type": "object"},
					"strict":      true,
				},
			},
		},
	}
	out, err := openaiToResponsesRequest("gpt-4", body, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tools, ok := out["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %v", out["tools"])
	}
	tool := tools[0].(map[string]any)
	if tool["type"] != "function" || tool["name"] != "get_weather" {
		t.Errorf("unexpected tool: %v", tool)
	}
	params, ok := tool["parameters"].(map[string]any)
	if !ok || params["type"] != "object" {
		t.Errorf("unexpected parameters: %v", tool["parameters"])
	}
	if _, ok := params["properties"]; !ok {
		t.Errorf("expected properties added to parameters")
	}
	if tool["strict"] != true {
		t.Errorf("expected strict preserved")
	}
}

func TestOpenAIResponsesParamPassthrough(t *testing.T) {
	body := map[string]any{
		"messages":    []any{map[string]any{"role": "user", "content": "hi"}},
		"temperature": 0.5,
		"max_tokens":  100,
		"top_p":       0.9,
	}
	out, err := openaiToResponsesRequest("gpt-4", body, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["temperature"] != 0.5 {
		t.Errorf("expected temperature 0.5, got %v", out["temperature"])
	}
	if out["max_tokens"] != 100 {
		t.Errorf("expected max_tokens 100, got %v", out["max_tokens"])
	}
	if out["top_p"] != 0.9 {
		t.Errorf("expected top_p 0.9, got %v", out["top_p"])
	}
}

func TestOpenAIResponsesEmptyAssistantSkipped(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role":    "assistant",
				"content": nil,
				"tool_calls": []any{
					map[string]any{"id": "c1", "type": "function", "function": map[string]any{"name": "toolA", "arguments": "{}"}},
				},
			},
		},
	}
	out, err := openaiToResponsesRequest("gpt-4", body, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	input := out["input"].([]any)
	// No message block for empty-content assistant; only function_call
	if len(input) != 1 {
		t.Fatalf("expected 1 input item (function_call only), got %d", len(input))
	}
	if input[0].(map[string]any)["type"] != "function_call" {
		t.Errorf("expected function_call, got %v", input[0].(map[string]any)["type"])
	}
}
