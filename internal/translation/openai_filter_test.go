package translation

import (
	"testing"
)

func TestFilterDeveloperRole(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "developer", "content": "be nice"},
		},
	}
	out := FilterToOpenAIFormat(body)
	msgs := out["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d", len(msgs))
	}
	if msgs[0].(map[string]any)["role"] != "system" {
		t.Errorf("role = %v", msgs[0].(map[string]any)["role"])
	}
}

func TestFilterThinkingBlocks(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "thinking", "thinking": "..."},
					map[string]any{"type": "redacted_thinking", "data": "..."},
					map[string]any{"type": "text", "text": "hi", "signature": "sig", "cache_control": map[string]any{"type": "ephemeral"}},
					map[string]any{"type": "tool_use", "id": "tu1"},
					map[string]any{"type": "tool_result", "tool_use_id": "tu1", "content": "ok", "signature": "sig"},
				},
			},
		},
	}
	out := FilterToOpenAIFormat(body)
	msgs := out["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d", len(msgs))
	}
	content := msgs[0].(map[string]any)["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("len(content) = %d, want 2", len(content))
	}
	// text kept, signature/cache_control stripped
	textBlock := content[0].(map[string]any)
	if textBlock["type"] != "text" || textBlock["text"] != "hi" {
		t.Errorf("text block = %v", textBlock)
	}
	if _, ok := textBlock["signature"]; ok {
		t.Error("signature should be stripped")
	}
	if _, ok := textBlock["cache_control"]; ok {
		t.Error("cache_control should be stripped")
	}
	// tool_result kept cleaned
	trBlock := content[1].(map[string]any)
	if trBlock["type"] != "tool_result" {
		t.Errorf("tool_result block = %v", trBlock)
	}
	if _, ok := trBlock["signature"]; ok {
		t.Error("signature should be stripped from tool_result")
	}
}

func TestFilterEmptyMessagesKeepsToolMessages(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "   "},
			map[string]any{"role": "tool", "tool_call_id": "tc1", "content": "result"},
			map[string]any{"role": "assistant", "content": "", "tool_calls": []any{map[string]any{"id": "tc1"}}},
			map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "   "}}},
		},
	}
	out := FilterToOpenAIFormat(body)
	msgs := out["messages"].([]any)
	if len(msgs) != 2 {
		t.Fatalf("len(msgs) = %d, want 2", len(msgs))
	}
	if msgs[0].(map[string]any)["role"] != "tool" {
		t.Errorf("msg[0] role = %v", msgs[0].(map[string]any)["role"])
	}
	if msgs[1].(map[string]any)["role"] != "assistant" {
		t.Errorf("msg[1] role = %v", msgs[1].(map[string]any)["role"])
	}
}

func TestFilterToolsNormalization(t *testing.T) {
	body := map[string]any{
		"tools": []any{
			// Claude format
			map[string]any{"name": "Read", "description": "reads", "input_schema": map[string]any{"type": "object"}},
			// Gemini format
			map[string]any{
				"functionDeclarations": []any{
					map[string]any{"name": "Write", "description": "writes", "parameters": map[string]any{"type": "object"}},
				},
			},
			// Already OpenAI format
			map[string]any{"type": "function", "function": map[string]any{"name": "Run", "description": "runs", "parameters": map[string]any{"type": "object"}}},
		},
	}
	out := FilterToOpenAIFormat(body)
	tools := out["tools"].([]any)
	if len(tools) != 3 {
		t.Fatalf("len(tools) = %d, want 3", len(tools))
	}
	// Claude normalized
	t0 := tools[0].(map[string]any)
	if t0["type"] != "function" {
		t.Errorf("tool[0].type = %v", t0["type"])
	}
	if t0["function"].(map[string]any)["name"] != "Read" {
		t.Errorf("tool[0].name = %v", t0["function"].(map[string]any)["name"])
	}
	// Gemini normalized
	t1 := tools[1].(map[string]any)
	if t1["type"] != "function" {
		t.Errorf("tool[1].type = %v", t1["type"])
	}
	if t1["function"].(map[string]any)["name"] != "Write" {
		t.Errorf("tool[1].name = %v", t1["function"].(map[string]any)["name"])
	}
	// OpenAI passes through
	t2 := tools[2].(map[string]any)
	if t2["type"] != "function" {
		t.Errorf("tool[2].type = %v", t2["type"])
	}
	if t2["function"].(map[string]any)["name"] != "Run" {
		t.Errorf("tool[2].name = %v", t2["function"].(map[string]any)["name"])
	}
}

func TestFilterToolsEmptyArrayDeleted(t *testing.T) {
	body := map[string]any{
		"tools": []any{},
	}
	out := FilterToOpenAIFormat(body)
	if _, ok := out["tools"]; ok {
		t.Error("empty tools array should be deleted")
	}
}

func TestFilterToolChoiceNormalization(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want any
	}{
		{"auto object", map[string]any{"type": "auto"}, "auto"},
		{"any object", map[string]any{"type": "any"}, "required"},
		{"tool object", map[string]any{"type": "tool", "name": "Read"}, map[string]any{"type": "function", "function": map[string]any{"name": "Read"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]any{"tool_choice": tc.in}
			out := FilterToOpenAIFormat(body)
			got := out["tool_choice"]
			gotMap, gotIsMap := got.(map[string]any)
			wantMap, wantIsMap := tc.want.(map[string]any)
			if gotIsMap && wantIsMap {
				if gotMap["type"] != wantMap["type"] {
					t.Errorf("type = %v, want %v", gotMap["type"], wantMap["type"])
				}
				gotFn := gotMap["function"].(map[string]any)
				wantFn := wantMap["function"].(map[string]any)
				if gotFn["name"] != wantFn["name"] {
					t.Errorf("name = %v, want %v", gotFn["name"], wantFn["name"])
				}
			} else if got != tc.want {
				t.Errorf("tool_choice = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFilterKeepsAssistantWithEmptyToolCalls(t *testing.T) {
	// 9router checks truthiness of msg.tool_calls — an empty array is truthy
	// in JS, so the assistant message is kept untouched (openaiHelper.js:20,64).
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "assistant", "content": "", "tool_calls": []any{}},
		},
	}
	out := FilterToOpenAIFormat(body)
	msgs, ok := out["messages"].([]any)
	if !ok || len(msgs) != 1 {
		t.Fatalf("expected assistant message with empty tool_calls kept, got %v", out["messages"])
	}
}

func TestFilterClaudeToolWithOnlyNamePassesThrough(t *testing.T) {
	// 9router converts Claude-shaped tools only when name AND
	// (input_schema OR description) are present (openaiHelper.js:88);
	// a bare {name} tool falls through unchanged.
	body := map[string]any{
		"messages": []any{map[string]any{"role": "user", "content": "hi"}},
		"tools":    []any{map[string]any{"name": "bare"}},
	}
	out := FilterToOpenAIFormat(body)
	tools := out["tools"].([]any)
	tool0, ok := tools[0].(map[string]any)
	if !ok {
		t.Fatalf("tool shape: %v", tools[0])
	}
	if _, converted := tool0["function"]; converted {
		t.Errorf("bare-name tool must pass through unconverted, got %v", tool0)
	}
	if tool0["name"] != "bare" {
		t.Errorf("tool = %v, want original {name:bare}", tool0)
	}
}
