package translation

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCursorExtractContent(t *testing.T) {
	// string passthrough
	if got := extractContent("hello"); got != "hello" {
		t.Errorf("extractContent(string) = %q, want %q", got, "hello")
	}
	// array: keep only text blocks, joined with NO separator
	arr := []any{
		map[string]any{"type": "text", "text": "a"},
		map[string]any{"type": "image_url", "url": "http://x"},
		map[string]any{"type": "text", "text": "b"},
	}
	if got := extractContent(arr); got != "ab" {
		t.Errorf("extractContent(array) = %q, want %q", got, "ab")
	}
	// non-string, non-array returns empty
	if got := extractContent(42); got != "" {
		t.Errorf("extractContent(int) = %q, want empty", got)
	}
	// nil returns empty
	if got := extractContent(nil); got != "" {
		t.Errorf("extractContent(nil) = %q, want empty", got)
	}
}

func TestCursorToolResultBlockXML(t *testing.T) {
	// Exact 5-line shape with escaping and control-char stripping.
	block := buildToolResultBlock("My<Tool>", "id&1", "result\x00with\x07chars\x80")
	lines := strings.Split(block, "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d: %q", len(lines), block)
	}
	want := []string{
		"<tool_result>",
		"<tool_name>My&lt;Tool&gt;</tool_name>",
		"<tool_call_id>id&amp;1</tool_call_id>",
		"<result>resultwithchars\x80</result>",
		"</tool_result>",
	}
	for i, w := range want {
		if lines[i] != w {
			t.Errorf("line %d = %q, want %q", i, lines[i], w)
		}
	}
	// default tool name
	block2 := buildToolResultBlock("", "x", "y")
	if !strings.Contains(block2, "<tool_name>tool</tool_name>") {
		t.Errorf("default tool name not applied: %q", block2)
	}
}

func TestCursorSystemToUserInstructions(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "system", "content": "Be nice."},
			map[string]any{"role": "user", "content": "hi"},
		},
	}
	out, err := buildCursorRequest("m", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	msgs := out["messages"].([]any)
	if len(msgs) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(msgs))
	}
	m0 := msgs[0].(map[string]any)
	if m0["role"] != "user" {
		t.Errorf("msg[0].role = %v, want user", m0["role"])
	}
	wantContent := "[System Instructions]\nBe nice."
	if m0["content"] != wantContent {
		t.Errorf("msg[0].content = %q, want %q", m0["content"], wantContent)
	}
}

func TestCursorToolRoleNamePrecedence(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "assistant",
				"tool_calls": []any{
					map[string]any{
						"id":       "tc1",
						"function": map[string]any{"name": "from_meta"},
					},
					map[string]any{
						"id":       "tcNorm\nextra",
						"function": map[string]any{"name": "from_meta_norm"},
					},
					map[string]any{
						"id":       "tcRaw",
						"function": map[string]any{"name": "from_meta"},
					},
				},
			},
			map[string]any{
				"role":         "tool",
				"tool_call_id": "tc1",
				"name":         "from_msg",
				"content":      "r1",
			},
			map[string]any{
				"role":         "tool",
				"tool_call_id": "tc2",
				"content":      "r2",
			},
			map[string]any{
				"role":         "tool",
				"tool_call_id": "tcNorm",
				"content":      "r3",
			},
			map[string]any{
				"role":         "tool",
				"tool_call_id": "tcRaw",
				"content":      "r4",
			},
		},
	}
	out, err := buildCursorRequest("m", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	msgs := out["messages"].([]any)
	if len(msgs) != 5 {
		t.Fatalf("len(messages) = %d, want 5", len(msgs))
	}
	// msg[0] is assistant with tool_calls
	if msgs[0].(map[string]any)["role"] != "assistant" {
		t.Errorf("msg[0].role = %v, want assistant", msgs[0].(map[string]any)["role"])
	}
	m1 := msgs[1].(map[string]any)
	content1 := m1["content"].(string)
	if !strings.Contains(content1, "<tool_name>from_msg</tool_name>") {
		t.Errorf("msg[1] should use msg.name: %q", content1)
	}
	m2 := msgs[2].(map[string]any)
	content2 := m2["content"].(string)
	if !strings.Contains(content2, "<tool_name>tool</tool_name>") {
		t.Errorf("msg[2] should fallback to 'tool': %q", content2)
	}
	m3 := msgs[3].(map[string]any)
	content3 := m3["content"].(string)
	if !strings.Contains(content3, "<tool_name>from_meta_norm</tool_name>") {
		t.Errorf("msg[3] should resolve via normalized key: %q", content3)
	}
	m4 := msgs[4].(map[string]any)
	content4 := m4["content"].(string)
	if !strings.Contains(content4, "<tool_name>from_meta</tool_name>") {
		t.Errorf("msg[4] should resolve via raw meta-map fallback: %q", content4)
	}
}

func TestCursorAssistantToolCallsIndexStripped(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "assistant",
				"content": "",
				"tool_calls": []any{
					map[string]any{
						"id":       "tc1",
						"type":     "function",
						"index":    0,
						"function": map[string]any{"name": "foo"},
					},
				},
			},
		},
	}
	out, err := buildCursorRequest("m", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	msgs := out["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(msgs))
	}
	tcs := msgs[0].(map[string]any)["tool_calls"].([]any)
	if len(tcs) != 1 {
		t.Fatalf("len(tool_calls) = %d, want 1", len(tcs))
	}
	tc := tcs[0].(map[string]any)
	if _, hasIndex := tc["index"]; hasIndex {
		t.Errorf("tool_call should not have index key: %v", tc)
	}
	if tc["id"] != "tc1" {
		t.Errorf("tc.id = %v, want tc1", tc["id"])
	}
}

func TestCursorToolUseExtraction(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "text", "text": "ok"},
					map[string]any{"type": "tool_use", "id": "tu1", "name": "n1", "input": map[string]any{"a": 1}},
					map[string]any{"type": "tool_use", "id": "", "name": "n2", "input": map[string]any{}},
				},
			},
		},
	}
	out, err := buildCursorRequest("m", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	msgs := out["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(msgs))
	}
	tcs := msgs[0].(map[string]any)["tool_calls"].([]any)
	if len(tcs) != 1 {
		t.Fatalf("len(tool_calls) = %d, want 1", len(tcs))
	}
	tc := tcs[0].(map[string]any)
	if tc["id"] != "tu1" {
		t.Errorf("tc.id = %v, want tu1", tc["id"])
	}
	fn := tc["function"].(map[string]any)
	if fn["name"] != "n1" {
		t.Errorf("tc.function.name = %v, want n1", fn["name"])
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(fn["arguments"].(string)), &args); err != nil {
		t.Fatalf("arguments not valid JSON: %v", err)
	}
	if args["a"] != float64(1) {
		t.Errorf("arguments[a] = %v, want 1", args["a"])
	}
}

func TestCursorFieldStrippingAndMaxTokens(t *testing.T) {
	body := map[string]any{
		"model":          "gpt-4",
		"messages":       []any{map[string]any{"role": "user", "content": "hi"}},
		"user":           "u1",
		"metadata":       map[string]any{"x": 1},
		"tool_choice":    "auto",
		"stream_options": map[string]any{"y": 2},
		"system":         "sys",
		"temperature":    0.7,
	}
	out, err := buildCursorRequest("m", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	for _, key := range []string{"user", "metadata", "tool_choice", "stream_options", "system"} {
		if _, ok := out[key]; ok {
			t.Errorf("%s should be stripped", key)
		}
	}
	if out["temperature"] != 0.7 {
		t.Errorf("temperature = %v, want 0.7", out["temperature"])
	}
	if out["model"] != "gpt-4" {
		t.Errorf("model = %v, want gpt-4", out["model"])
	}
	if out["max_tokens"] != 32000 {
		t.Errorf("max_tokens = %v, want 32000", out["max_tokens"])
	}
}
