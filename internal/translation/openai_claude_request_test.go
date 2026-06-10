package translation

import (
	"encoding/json"
	"testing"
)

func TestNewRegistryWiresOpenAIClaudeRequest(t *testing.T) {
	reg := NewRegistry()
	if reg.RequestTranslatorFor(FormatOpenAI, FormatClaude) == nil {
		t.Error("NewRegistry must wire openai->claude request translator")
	}
}

func TestOpenAIClaudeSystemExtraction(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "system", "content": "be nice"},
			map[string]any{"role": "user", "content": "hi"},
		},
	}
	out, err := openaiToClaudeRequest("claude-3", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	sys, ok := out["system"].([]any)
	if !ok || len(sys) != 2 {
		t.Fatalf("system = %v, want 2 blocks", out["system"])
	}
	first := sys[0].(map[string]any)
	if first["type"] != "text" || first["text"] != "You are Claude Code, Anthropic's official CLI for Claude." {
		t.Errorf("first system block = %v", first)
	}
	second := sys[1].(map[string]any)
	if second["type"] != "text" || second["text"] != "be nice" {
		t.Errorf("second system block text = %v", second["text"])
	}
	cc, ok := second["cache_control"].(map[string]any)
	if !ok {
		t.Fatalf("second block missing cache_control")
	}
	if cc["type"] != "ephemeral" || cc["ttl"] != "1h" {
		t.Errorf("cache_control = %v", cc)
	}
}

func TestOpenAIClaudeMessageMergeAndToolResultSplit(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "hello"},
			map[string]any{"role": "assistant", "content": "hi"},
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "text", "text": "q1"},
					map[string]any{"type": "tool_result", "tool_use_id": "tu1", "content": "r1"},
				},
			},
		},
	}
	out, err := openaiToClaudeRequest("claude-3", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	msgs := out["messages"].([]any)
	if len(msgs) != 4 {
		t.Fatalf("len(messages) = %d, want 4", len(msgs))
	}
	if msgs[0].(map[string]any)["role"] != "user" {
		t.Errorf("msg[0] role = %v", msgs[0].(map[string]any)["role"])
	}
	if msgs[1].(map[string]any)["role"] != "assistant" {
		t.Errorf("msg[1] role = %v", msgs[1].(map[string]any)["role"])
	}
	// tool_result split into its own message
	tr := msgs[2].(map[string]any)
	if tr["role"] != "user" {
		t.Errorf("msg[2] role = %v", tr["role"])
	}
	trContent := tr["content"].([]any)
	if len(trContent) != 1 || trContent[0].(map[string]any)["type"] != "tool_result" {
		t.Errorf("msg[2] content = %v", trContent)
	}
	// remaining user content
	if msgs[3].(map[string]any)["role"] != "user" {
		t.Errorf("msg[3] role = %v", msgs[3].(map[string]any)["role"])
	}
}

func TestOpenAIClaudeCacheControlPlacement(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "hello"},
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "text", "text": "hi"},
					map[string]any{"type": "tool_use", "id": "tu1", "name": "Read", "input": map[string]any{}},
				},
			},
		},
	}
	out, err := openaiToClaudeRequest("claude-3", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	msgs := out["messages"].([]any)
	var assistantMsg map[string]any
	for _, m := range msgs {
		mm := m.(map[string]any)
		if mm["role"] == "assistant" {
			assistantMsg = mm
			break
		}
	}
	if assistantMsg == nil {
		t.Fatal("no assistant message found")
	}
	blocks := assistantMsg["content"].([]any)
	lastBlock := blocks[len(blocks)-1].(map[string]any)
	cc, ok := lastBlock["cache_control"].(map[string]any)
	if !ok {
		t.Fatalf("last block missing cache_control")
	}
	if cc["type"] != "ephemeral" {
		t.Errorf("cache_control type = %v", cc["type"])
	}
	if _, hasTTL := cc["ttl"]; hasTTL {
		t.Error("assistant cache_control must NOT have ttl")
	}
	// thinking blocks should not get cache_control
	body2 := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "thinking", "thinking": "..."},
					map[string]any{"type": "text", "text": "hi"},
				},
			},
		},
	}
	out2, err := openaiToClaudeRequest("claude-3", body2, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	msgs2 := out2["messages"].([]any)
	blocks2 := msgs2[0].(map[string]any)["content"].([]any)
	// thinking block should have no cache_control
	thinkingBlock := blocks2[0].(map[string]any)
	if _, ok := thinkingBlock["cache_control"]; ok {
		t.Error("thinking block should not have cache_control")
	}
	// text block should have cache_control
	textBlock := blocks2[1].(map[string]any)
	if _, ok := textBlock["cache_control"]; !ok {
		t.Error("text block should have cache_control")
	}
}

func TestOpenAIClaudeResponseFormatJSONSchema(t *testing.T) {
	body := map[string]any{
		"response_format": map[string]any{
			"type":        "json_schema",
			"json_schema": map[string]any{"schema": map[string]any{"type": "object"}},
		},
	}
	out, err := openaiToClaudeRequest("claude-3", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	sys := out["system"].([]any)
	last := sys[len(sys)-1].(map[string]any)
	text := last["text"].(string)
	// Exact text per the ref (openai-to-claude.js:112-117): prompt header,
	// fenced 2-space pretty-printed schema, verbatim closing instruction.
	want := "You must respond with valid JSON that strictly follows this JSON schema:\n" +
		"```json\n" +
		"{\n  \"type\": \"object\"\n}\n" +
		"```\n" +
		"Respond ONLY with the JSON object, no other text."
	if text != want {
		t.Errorf("system text = %q, want %q", text, want)
	}
}

func TestOpenAIClaudeResponseFormatJSONObject(t *testing.T) {
	body := map[string]any{
		"response_format": map[string]any{"type": "json_object"},
	}
	out, err := openaiToClaudeRequest("claude-3", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	sys := out["system"].([]any)
	last := sys[len(sys)-1].(map[string]any)
	text := last["text"].(string)
	want := "You must respond with valid JSON. Respond ONLY with a JSON object, no other text."
	if text != want {
		t.Errorf("system text = %q, want %q", text, want)
	}
}

func TestOpenAIClaudeToolsConversion(t *testing.T) {
	body := map[string]any{
		"tools": []any{
			// built-in tool passes through
			map[string]any{"type": "web_search_20250305", "name": "search"},
			// function tool gets rewritten
			map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        "Read",
					"description": "reads files",
					"parameters":  map[string]any{"type": "object"},
				},
			},
			// function without description gets empty default
			map[string]any{
				"type":     "function",
				"function": map[string]any{"name": "Write"},
			},
		},
	}
	out, err := openaiToClaudeRequest("claude-3", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	tools := out["tools"].([]any)
	if len(tools) != 3 {
		t.Fatalf("len(tools) = %d, want 3", len(tools))
	}
	// built-in passes through untouched
	if tools[0].(map[string]any)["type"] != "web_search_20250305" {
		t.Errorf("tool[0] = %v", tools[0])
	}
	// function rewritten
	t1 := tools[1].(map[string]any)
	if t1["name"] != "Read" {
		t.Errorf("tool[1].name = %v", t1["name"])
	}
	if t1["description"] != "reads files" {
		t.Errorf("tool[1].description = %v", t1["description"])
	}
	is := t1["input_schema"].(map[string]any)
	if is["type"] != "object" {
		t.Errorf("tool[1].input_schema = %v", is)
	}
	// empty description default
	t2 := tools[2].(map[string]any)
	if t2["description"] != "" {
		t.Errorf("tool[2].description = %v", t2["description"])
	}
	// default empty schema when no parameters
	t2is := t2["input_schema"].(map[string]any)
	if t2is["type"] != "object" {
		t.Errorf("tool[2].input_schema.type = %v", t2is["type"])
	}
	// last tool has cache_control with ttl
	lastTool := tools[len(tools)-1].(map[string]any)
	cc, ok := lastTool["cache_control"].(map[string]any)
	if !ok {
		t.Fatal("last tool missing cache_control")
	}
	if cc["type"] != "ephemeral" || cc["ttl"] != "1h" {
		t.Errorf("last tool cache_control = %v", cc)
	}
}

func TestOpenAIClaudeToolChoice(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want any
	}{
		{"string required", "required", map[string]any{"type": "any"}},
		{"string auto", "auto", map[string]any{"type": "auto"}},
		{"string none", "none", map[string]any{"type": "auto"}},
		{"object function", map[string]any{"type": "function", "function": map[string]any{"name": "Read"}}, map[string]any{"type": "tool", "name": "Read"}},
		{"object auto", map[string]any{"type": "auto"}, map[string]any{"type": "auto"}},
		{"object any", map[string]any{"type": "any"}, map[string]any{"type": "any"}},
		{"object tool", map[string]any{"type": "tool", "name": "Read"}, map[string]any{"type": "tool", "name": "Read"}},
		{"object none", map[string]any{"type": "none"}, map[string]any{"type": "none"}},
		{"malformed function no name", map[string]any{"type": "function"}, map[string]any{"type": "auto"}},
		{"unknown type", map[string]any{"type": "garbage"}, map[string]any{"type": "auto"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]any{"tool_choice": tc.in}
			out, err := openaiToClaudeRequest("claude-3", body, false, nil)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			got := out["tool_choice"]
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(tc.want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("tool_choice = %s, want %s", gotJSON, wantJSON)
			}
		})
	}
}

func TestOpenAIClaudeReasoningEffort(t *testing.T) {
	cases := []struct {
		effort string
		want   any // nil means no thinking block
	}{
		{"none", nil},
		{"low", map[string]any{"type": "enabled", "budget_tokens": 4096}},
		{"medium", map[string]any{"type": "enabled", "budget_tokens": 8192}},
		{"high", map[string]any{"type": "enabled", "budget_tokens": 16384}},
		{"xhigh", map[string]any{"type": "enabled", "budget_tokens": 32768}},
	}
	for _, tc := range cases {
		t.Run(tc.effort, func(t *testing.T) {
			body := map[string]any{"reasoning_effort": tc.effort}
			out, err := openaiToClaudeRequest("claude-3", body, false, nil)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			thinking, ok := out["thinking"]
			if tc.want == nil {
				if ok {
					t.Errorf("thinking = %v, want nil", thinking)
				}
				return
			}
			if !ok {
				t.Fatalf("thinking missing")
			}
			gotJSON, _ := json.Marshal(thinking)
			wantJSON, _ := json.Marshal(tc.want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("thinking = %s, want %s", gotJSON, wantJSON)
			}
		})
	}
}

func TestOpenAIClaudeThinkingPassthrough(t *testing.T) {
	body := map[string]any{
		"thinking": map[string]any{
			"type":          "enabled",
			"budget_tokens": 1024,
			"max_tokens":    2048,
		},
	}
	out, err := openaiToClaudeRequest("claude-3", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	thinking := out["thinking"].(map[string]any)
	if thinking["type"] != "enabled" {
		t.Errorf("type = %v", thinking["type"])
	}
	if thinking["budget_tokens"] != 1024 {
		t.Errorf("budget_tokens = %v", thinking["budget_tokens"])
	}
	if thinking["max_tokens"] != 2048 {
		t.Errorf("max_tokens = %v", thinking["max_tokens"])
	}
}

func TestOpenAIClaudeThinkingDropsTemperature(t *testing.T) {
	body := map[string]any{
		"temperature": 0.5,
		"thinking": map[string]any{
			"type":          "enabled",
			"budget_tokens": 1024,
		},
	}
	out, err := openaiToClaudeRequest("claude-3", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if _, ok := out["temperature"]; ok {
		t.Error("temperature should be dropped when thinking is enabled")
	}

	// PAR-PR-1264 applies only to enabled thinking: a disabled thinking
	// config must keep temperature.
	body2 := map[string]any{
		"temperature": 0.5,
		"thinking":    map[string]any{"type": "disabled"},
	}
	out2, err := openaiToClaudeRequest("claude-3", body2, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if out2["temperature"] != 0.5 {
		t.Errorf("temperature = %v, want 0.5 kept when thinking is not enabled", out2["temperature"])
	}
}

func TestOpenAIClaudeImageBlocks(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want map[string]any
	}{
		{
			"data uri",
			"data:image/png;base64,abc123",
			map[string]any{"type": "image", "source": map[string]any{"type": "base64", "media_type": "image/png", "data": "abc123"}},
		},
		{
			"http url",
			"https://example.com/img.png",
			map[string]any{"type": "image", "source": map[string]any{"type": "url", "url": "https://example.com/img.png"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]any{
				"messages": []any{
					map[string]any{
						"role": "user",
						"content": []any{
							map[string]any{"type": "image_url", "image_url": map[string]any{"url": tc.url}},
						},
					},
				},
			}
			out, err := openaiToClaudeRequest("claude-3", body, false, nil)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			msgs := out["messages"].([]any)
			blocks := msgs[0].(map[string]any)["content"].([]any)
			if len(blocks) != 1 {
				t.Fatalf("len(blocks) = %d, want 1", len(blocks))
			}
			gotJSON, _ := json.Marshal(blocks[0])
			wantJSON, _ := json.Marshal(tc.want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("block = %s, want %s", gotJSON, wantJSON)
			}
		})
	}

	t.Run("raw image passthrough", func(t *testing.T) {
		body := map[string]any{
			"messages": []any{
				map[string]any{
					"role": "user",
					"content": []any{
						map[string]any{"type": "image", "source": map[string]any{"type": "base64", "media_type": "image/png", "data": "abc"}},
					},
				},
			},
		}
		out, err := openaiToClaudeRequest("claude-3", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		msgs := out["messages"].([]any)
		blocks := msgs[0].(map[string]any)["content"].([]any)
		if len(blocks) != 1 {
			t.Fatalf("len(blocks) = %d, want 1", len(blocks))
		}
		want := map[string]any{"type": "image", "source": map[string]any{"type": "base64", "media_type": "image/png", "data": "abc"}}
		gotJSON, _ := json.Marshal(blocks[0])
		wantJSON, _ := json.Marshal(want)
		if string(gotJSON) != string(wantJSON) {
			t.Errorf("block = %s, want %s", gotJSON, wantJSON)
		}
	})
}

func TestOpenAIClaudeToolCallArgumentsParse(t *testing.T) {
	cases := []struct {
		name string
		args string
		want any
	}{
		{"valid json", `{"a":1}`, map[string]any{"a": float64(1)}},
		{"invalid json", `not json`, "not json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]any{
				"messages": []any{
					map[string]any{
						"role": "assistant",
						"tool_calls": []any{
							map[string]any{
								"id":   "tc1",
								"type": "function",
								"function": map[string]any{
									"name":      "Read",
									"arguments": tc.args,
								},
							},
						},
					},
				},
			}
			out, err := openaiToClaudeRequest("claude-3", body, false, nil)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			msgs := out["messages"].([]any)
			blocks := msgs[0].(map[string]any)["content"].([]any)
			if len(blocks) != 1 {
				t.Fatalf("len(blocks) = %d, want 1", len(blocks))
			}
			tu := blocks[0].(map[string]any)
			got := tu["input"]
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(tc.want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("input = %s, want %s", gotJSON, wantJSON)
			}
		})
	}
}

func TestOpenAIClaudeRequestMissingRoleDoesNotPanic(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"content": "no role on this message"},
			map[string]any{"role": 42, "content": "non-string role"},
			map[string]any{"role": "user", "content": "hi"},
		},
	}
	out, err := openaiToClaudeRequest("claude-3-opus", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	// Messages with missing/non-string roles contribute no content blocks and
	// are dropped (9router: undefined role matches no conversion branch);
	// valid messages survive.
	msgs, ok := out["messages"].([]any)
	if !ok || len(msgs) != 1 {
		t.Fatalf("expected exactly the valid user message, got %v", out["messages"])
	}
	first, ok := msgs[0].(map[string]any)
	if !ok {
		t.Fatalf("message shape: %v", msgs[0])
	}
	if first["role"] != "user" {
		t.Errorf("role = %v, want user", first["role"])
	}
}
