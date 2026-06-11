package translation

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBypassNonClaudeCliNil(t *testing.T) {
	body := map[string]any{
		"messages": []any{map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "text", "text": "{"}}}},
	}
	resp, bypassed := HandleBypassRequest(body, "claude-3", "some-other-agent", false)
	if bypassed {
		t.Error("expected no bypass for non-claude-cli userAgent")
	}
	if resp != nil {
		t.Errorf("expected nil response, got %v", resp)
	}
}

func TestBypassTitleExtraction(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "text", "text": "{"}}}},
	}
	resp, bypassed := HandleBypassRequest(body, "claude-3", "claude-cli/1.0", false)
	if !bypassed {
		t.Fatal("expected bypass")
	}
	if len(resp) == 0 {
		t.Fatal("expected non-empty response")
	}
	content := extractBypassContent(resp)
	if content != defaultBypassText {
		t.Errorf("content = %q, want %q", content, defaultBypassText)
	}
}

func TestBypassWarmup(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "Warmup"}}},
		},
	}
	resp, bypassed := HandleBypassRequest(body, "claude-3", "claude-cli/1.0", false)
	if !bypassed {
		t.Fatal("expected bypass")
	}
	content := extractBypassContent(resp)
	if content != defaultBypassText {
		t.Errorf("content = %q, want %q", content, defaultBypassText)
	}
}

func TestBypassCount(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "count"}}},
		},
	}
	resp, bypassed := HandleBypassRequest(body, "claude-3", "claude-cli/1.0", false)
	if !bypassed {
		t.Fatal("expected bypass")
	}
	content := extractBypassContent(resp)
	if content != defaultBypassText {
		t.Errorf("content = %q, want %q", content, defaultBypassText)
	}
}

func TestBypassSkipPatterns(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "Please write a 5-10 word title for the following conversation:"}}},
		},
	}
	resp, bypassed := HandleBypassRequest(body, "claude-3", "claude-cli/1.0", false)
	if !bypassed {
		t.Fatal("expected bypass")
	}
	content := extractBypassContent(resp)
	if content != defaultBypassText {
		t.Errorf("content = %q, want %q", content, defaultBypassText)
	}
}

func TestBypassNamingIsNewTopic(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "Hello world from test"}}},
		},
		"system": []any{map[string]any{"type": "text", "text": "isNewTopic"}},
	}
	resp, bypassed := HandleBypassRequest(body, "claude-3", "claude-cli/1.0", true)
	if !bypassed {
		t.Fatal("expected bypass")
	}
	content := extractBypassContent(resp)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("content not JSON: %v", content)
	}
	if parsed["isNewTopic"] != true {
		t.Errorf("isNewTopic = %v, want true", parsed["isNewTopic"])
	}
	if parsed["title"] != "Hello world from" {
		t.Errorf("title = %q, want 'Hello world from'", parsed["title"])
	}
}

func TestBypassNoMatchNil(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "regular message"}}},
		},
	}
	resp, bypassed := HandleBypassRequest(body, "claude-3", "claude-cli/1.0", false)
	if bypassed {
		t.Error("expected no bypass")
	}
	if resp != nil {
		t.Errorf("expected nil response, got %v", resp)
	}
}

func extractBypassContent(resp []map[string]any) string {
	if len(resp) == 0 {
		return ""
	}
	// Non-streaming: single response with choices[0].message.content
	if len(resp) == 1 {
		choices, ok := resp[0]["choices"].([]any)
		if ok && len(choices) > 0 {
			choice := choices[0].(map[string]any)
			msg, ok := choice["message"].(map[string]any)
			if ok {
				return msg["content"].(string)
			}
			// Streaming final chunk
			delta, ok := choice["delta"].(map[string]any)
			if ok {
				if content, ok := delta["content"].(string); ok {
					return content
				}
			}
		}
	}
	// Streaming: concatenate content from chunks
	var parts []string
	for _, chunk := range resp {
		choices, ok := chunk["choices"].([]any)
		if !ok || len(choices) == 0 {
			continue
		}
		choice := choices[0].(map[string]any)
		delta, ok := choice["delta"].(map[string]any)
		if !ok {
			continue
		}
		if content, ok := delta["content"].(string); ok && content != "" {
			parts = append(parts, content)
		}
	}
	return strings.Join(parts, "")
}
