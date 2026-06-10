package translation

import (
	"strings"
	"testing"
)

func TestNewRegistryWiresOpenAIAntigravityRequest(t *testing.T) {
	reg := NewRegistry()
	if reg.RequestTranslatorFor(FormatOpenAI, FormatAntigravity) == nil {
		t.Error("NewRegistry must wire openai->antigravity request translator")
	}
}

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
}

func TestOpenAIAntigravityGeminiModelUsesGeminiCLIEnvelope(t *testing.T) {
	reg := NewRegistry()
	body := map[string]any{
		"messages": []any{map[string]any{"role": "user", "content": "hi"}},
	}
	out, err := reg.TranslateRequest(FormatOpenAI, FormatAntigravity, "gemini-2.0-flash", body, false, nil)
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	if out["userAgent"] != "antigravity" {
		t.Errorf("userAgent = %v, want antigravity", out["userAgent"])
	}
	if out["requestType"] != "agent" {
		t.Errorf("requestType = %v, want agent", out["requestType"])
	}
	req, ok := out["request"].(map[string]any)
	if !ok {
		t.Fatal("request missing")
	}
	if req["contents"] == nil {
		t.Error("expected contents in request")
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
