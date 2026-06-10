package translation

import (
	"encoding/json"
	"testing"
)

func TestClaudeRequestSystemString(t *testing.T) {
	body := map[string]any{
		"model":       "claude-3-5-sonnet",
		"max_tokens":  1024,
		"system":      "you are helpful",
		"temperature": 0.5,
	}
	out, err := claudeToOpenAIRequest("claude-3-5-sonnet", body, false)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	msgs := out["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(msgs))
	}
	m := msgs[0].(map[string]any)
	if m["role"] != "system" || m["content"] != "you are helpful" {
		t.Errorf("system message = %v", m)
	}
	if out["max_tokens"] != 1024 {
		t.Errorf("max_tokens = %v", out["max_tokens"])
	}
	if out["temperature"] != 0.5 {
		t.Errorf("temperature = %v", out["temperature"])
	}
}

func TestClaudeRequestSystemArray(t *testing.T) {
	body := map[string]any{
		"system": []any{
			map[string]any{"type": "text", "text": "line1"},
			map[string]any{"type": "text", "text": "line2"},
		},
	}
	out, err := claudeToOpenAIRequest("claude-3", body, false)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	msgs := out["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(msgs))
	}
	m := msgs[0].(map[string]any)
	if m["content"] != "line1\nline2" {
		t.Errorf("system content = %q", m["content"])
	}
}

func TestClaudeRequestImageBlock(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{
						"type": "image",
						"source": map[string]any{
							"type":      "base64",
							"media_type": "image/png",
							"data":      "abc123",
						},
					},
				},
			},
		},
	}
	out, err := claudeToOpenAIRequest("claude-3", body, false)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	msgs := out["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(msgs))
	}
	parts := msgs[0].(map[string]any)["content"].([]any)
	if len(parts) != 1 {
		t.Fatalf("len(parts) = %d, want 1", len(parts))
	}
	p := parts[0].(map[string]any)
	if p["type"] != "image_url" {
		t.Errorf("type = %q", p["type"])
	}
	url := p["image_url"].(map[string]any)["url"].(string)
	if url != "data:image/png;base64,abc123" {
		t.Errorf("url = %q", url)
	}
}

func TestClaudeRequestToolUse(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{
						"type":  "tool_use",
						"id":    "tu_1",
						"name":  "Read",
						"input": map[string]any{"file_path": "/tmp/a"},
					},
				},
			},
		},
	}
	out, err := claudeToOpenAIRequest("claude-3", body, false)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	msgs := out["messages"].([]any)
	// fixMissingToolResponses inserts a placeholder after unresponded tool_calls.
	if len(msgs) != 2 {
		t.Fatalf("len(messages) = %d, want 2 (assistant + missing tool placeholder)", len(msgs))
	}
	m := msgs[0].(map[string]any)
	tcs := m["tool_calls"].([]any)
	if len(tcs) != 1 {
		t.Fatalf("len(tool_calls) = %d, want 1", len(tcs))
	}
	tc := tcs[0].(map[string]any)
	if tc["id"] != "tu_1" || tc["type"] != "function" {
		t.Errorf("tool_call = %v", tc)
	}
	fn := tc["function"].(map[string]any)
	if fn["name"] != "Read" {
		t.Errorf("name = %q", fn["name"])
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(fn["arguments"].(string)), &args); err != nil {
		t.Fatalf("arguments not JSON: %v", err)
	}
	if args["file_path"] != "/tmp/a" {
		t.Errorf("args = %v", args)
	}
}

func TestClaudeRequestToolResult(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{
						"type":        "tool_result",
						"tool_use_id": "tu_1",
						"content":     "result text",
					},
				},
			},
		},
	}
	out, err := claudeToOpenAIRequest("claude-3", body, false)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	msgs := out["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(msgs))
	}
	m := msgs[0].(map[string]any)
	if m["role"] != "tool" || m["tool_call_id"] != "tu_1" || m["content"] != "result text" {
		t.Errorf("tool message = %v", m)
	}
}

func TestClaudeRequestToolResultTextBlocks(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{
						"type":        "tool_result",
						"tool_use_id": "tu_1",
						"content": []any{
							map[string]any{"type": "text", "text": "a"},
							map[string]any{"type": "text", "text": "b"},
						},
					},
				},
			},
		},
	}
	out, err := claudeToOpenAIRequest("claude-3", body, false)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	m := out["messages"].([]any)[0].(map[string]any)
	if m["content"] != "a\nb" {
		t.Errorf("content = %q", m["content"])
	}
}

func TestClaudeRequestMissingToolResponses(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{
						"type": "tool_use",
						"id":   "tu_1",
						"name": "Read",
						"input": map[string]any{},
					},
				},
			},
			map[string]any{"role": "user", "content": "next"},
		},
	}
	out, err := claudeToOpenAIRequest("claude-3", body, false)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	msgs := out["messages"].([]any)
	if len(msgs) != 3 {
		t.Fatalf("len(messages) = %d, want 3", len(msgs))
	}
	m := msgs[1].(map[string]any)
	if m["role"] != "tool" || m["tool_call_id"] != "tu_1" || m["content"] != "[No response received]" {
		t.Errorf("missing tool response = %v", m)
	}
}

func TestClaudeRequestTools(t *testing.T) {
	body := map[string]any{
		"tools": []any{
			map[string]any{
				"name":        "Read",
				"description": "reads files",
				"input_schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"file_path": map[string]any{"type": "string"},
					},
				},
			},
		},
	}
	out, err := claudeToOpenAIRequest("claude-3", body, false)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	tools := out["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("len(tools) = %d, want 1", len(tools))
	}
	tool := tools[0].(map[string]any)
	if tool["type"] != "function" {
		t.Errorf("type = %q", tool["type"])
	}
	fn := tool["function"].(map[string]any)
	if fn["name"] != "Read" || fn["description"] != "reads files" {
		t.Errorf("function = %v", fn)
	}
	params := fn["parameters"].(map[string]any)
	if params["type"] != "object" {
		t.Errorf("parameters = %v", params)
	}
}

func TestClaudeRequestToolChoice(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want any
	}{
		{"string auto", "auto", "auto"},
		{"string any", "any", "required"},
		{"object auto", map[string]any{"type": "auto"}, "auto"},
		{"object any", map[string]any{"type": "any"}, "required"},
		{"object tool", map[string]any{"type": "tool", "name": "Read"}, map[string]any{"type": "function", "function": map[string]any{"name": "Read"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]any{"tool_choice": tc.in}
			out, err := claudeToOpenAIRequest("claude-3", body, false)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			got := out["tool_choice"]
			if tc.want == "auto" || tc.want == "required" {
				if got != tc.want {
					t.Errorf("tool_choice = %v, want %v", got, tc.want)
				}
				return
			}
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(tc.want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("tool_choice = %s, want %s", gotJSON, wantJSON)
			}
		})
	}
}
