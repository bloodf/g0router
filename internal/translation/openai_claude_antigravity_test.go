package translation

import (
	"strings"
	"testing"
)

func TestOpenAIAntigravityClaudeModelUsesClaudePath(t *testing.T) {
	reg := NewRegistry()
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "hi"},
		},
	}
	out, err := reg.TranslateRequest(FormatOpenAI, FormatAntigravity, "claude-3-5-sonnet", body, false, nil)
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	req, ok := out["request"].(map[string]any)
	if !ok {
		t.Fatal("request missing")
	}
	contents, ok := req["contents"].([]any)
	if !ok || len(contents) == 0 {
		t.Fatal("expected contents")
	}
	first := contents[0].(map[string]any)
	if first["role"] != "user" {
		t.Errorf("role = %v, want user", first["role"])
	}

	// Claude-path marker: the claude translator always sets max_tokens (the
	// Claude API requires it), so the envelope carries
	// generationConfig.maxOutputTokens even though the OpenAI body had none.
	// The gemini path only sets maxOutputTokens when the body provides
	// max_tokens, so its presence here proves the claude path was taken.
	genConfig, ok := req["generationConfig"].(map[string]any)
	if !ok {
		t.Fatal("generationConfig missing")
	}
	if genConfig["maxOutputTokens"] == nil {
		t.Error("generationConfig.maxOutputTokens missing — request did not take the claude path")
	}
	// And the gemini envelope's always-present "tools" key must be absent
	// for a tool-less claude-path request.
	if _, hasTools := req["tools"]; hasTools {
		t.Error("request.tools present — gemini envelope path was taken instead of claude path")
	}
}

func TestOpenAIClaudeAntigravityStripsClaudeCodeSystem(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "system", "content": "You are Claude Code, Anthropic's official CLI for Claude."},
			map[string]any{"role": "user", "content": "hi"},
		},
	}
	out, err := openaiToClaudeRequestForAntigravity("claude-3", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	// After stripping, the system should either be absent or not contain Claude Code
	sys, ok := out["system"]
	if ok {
		switch v := sys.(type) {
		case []any:
			for _, block := range v {
				if b, ok := block.(map[string]any); ok {
					if text, ok := b["text"].(string); ok && strings.Contains(text, "You are Claude Code") {
						t.Errorf("system still contains Claude Code prompt: %v", text)
					}
				}
			}
		case string:
			if strings.Contains(v, "You are Claude Code") {
				t.Errorf("system still contains Claude Code prompt: %v", v)
			}
		}
	}
}

func TestOpenAIClaudeAntigravityToolPrefixNoOp(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "assistant",
				"content": "",
				"tool_calls": []any{
					map[string]any{
						"id":   "tc1",
						"type": "function",
						"function": map[string]any{
							"name":      "my_tool",
							"arguments": `{}`,
						},
					},
				},
			},
		},
		"tools": []any{
			map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        "my_tool",
					"description": "does stuff",
					"parameters":  map[string]any{"type": "object"},
				},
			},
		},
	}
	out, err := openaiToClaudeRequestForAntigravity("claude-3", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	// Tools should be unchanged because prefix is empty
	tools := out["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("len(tools) = %d", len(tools))
	}
	tool := tools[0].(map[string]any)
	if tool["name"] != "my_tool" {
		t.Errorf("tool.name = %v, want my_tool", tool["name"])
	}
}
