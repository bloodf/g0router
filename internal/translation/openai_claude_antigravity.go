package translation

import (
	"fmt"
	"strings"
)

// claudeAntigravityToolPrefix is verbatim openai-to-claude.js:8 (empty string).
const claudeAntigravityToolPrefix = ""

func openaiToClaudeRequestForAntigravity(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error) {
	result, err := openaiToClaudeRequest(model, body, stream, credentials)
	if err != nil {
		return nil, fmt.Errorf("openaiToClaudeRequestForAntigravity: %w", err)
	}

	// Remove Claude Code system prompt, keep only user's system messages
	if sys, ok := result["system"]; ok && sys != nil {
		switch v := sys.(type) {
		case []any:
			filtered := make([]any, 0, len(v))
			for _, block := range v {
				if b, ok := block.(map[string]any); ok {
					if text, ok := b["text"].(string); ok && strings.Contains(text, claudeSystemPrompt) {
						continue
					}
				}
				filtered = append(filtered, block)
			}
			if len(filtered) > 0 {
				result["system"] = filtered
			} else {
				delete(result, "system")
			}
		case string:
			if strings.Contains(v, claudeSystemPrompt) {
				delete(result, "system")
			}
		}
	}

	// Strip prefix from tool names for Antigravity (doesn't use Claude OAuth)
	if tools, ok := result["tools"].([]any); ok {
		for i, tool := range tools {
			tm, ok := tool.(map[string]any)
			if !ok {
				continue
			}
			name, _ := tm["name"].(string)
			if name != "" && strings.HasPrefix(name, claudeAntigravityToolPrefix) {
				tm["name"] = strings.TrimPrefix(name, claudeAntigravityToolPrefix)
				tools[i] = tm
			}
		}
	}

	// Strip prefix from tool_use in messages
	if msgs, ok := result["messages"].([]any); ok {
		for i, msg := range msgs {
			m, ok := msg.(map[string]any)
			if !ok {
				continue
			}
			content, ok := m["content"].([]any)
			if !ok {
				continue
			}
			for j, block := range content {
				b, ok := block.(map[string]any)
				if !ok {
					continue
				}
				if b["type"] == "tool_use" {
					name, _ := b["name"].(string)
					if name != "" && strings.HasPrefix(name, claudeAntigravityToolPrefix) {
						b["name"] = strings.TrimPrefix(name, claudeAntigravityToolPrefix)
						content[j] = b
					}
				}
			}
			m["content"] = content
			msgs[i] = m
		}
	}

	return result, nil
}
