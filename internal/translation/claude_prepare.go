package translation

import (
	"strings"
)

var claudeFormatProvidersWithoutOutputConfig = map[string]struct{}{
	"minimax":    {},
	"minimax-cn": {},
}

func hasValidContent(msg map[string]any) bool {
	if content, ok := msg["content"].(string); ok && strings.TrimSpace(content) != "" {
		return true
	}
	if arr, ok := msg["content"].([]any); ok {
		for _, item := range arr {
			block, ok := item.(map[string]any)
			if !ok {
				continue
			}
			bt, _ := block["type"].(string)
			if bt == "text" {
				if text, ok := block["text"].(string); ok && strings.TrimSpace(text) != "" {
					return true
				}
			}
			if bt == "tool_use" || bt == "tool_result" {
				return true
			}
		}
	}
	return false
}

func fixToolUseOrdering(messages []any) []any {
	if len(messages) <= 1 {
		return messages
	}

	// Pass 1: Fix assistant messages with tool_use - remove text after tool_use
	for _, m := range messages {
		msg, ok := m.(map[string]any)
		if !ok {
			continue
		}
		if msg["role"] != "assistant" {
			continue
		}
		content, ok := msg["content"].([]any)
		if !ok {
			continue
		}
		hasToolUse := false
		for _, item := range content {
			block, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if block["type"] == "tool_use" {
				hasToolUse = true
				break
			}
		}
		if !hasToolUse {
			continue
		}
		newContent := make([]any, 0, len(content))
		foundToolUse := false
		for _, item := range content {
			block, ok := item.(map[string]any)
			if !ok {
				continue
			}
			bt, _ := block["type"].(string)
			if bt == "tool_use" {
				foundToolUse = true
				newContent = append(newContent, block)
			} else if bt == "thinking" || bt == "redacted_thinking" {
				newContent = append(newContent, block)
			} else if !foundToolUse {
				newContent = append(newContent, block)
			}
			// Skip text blocks after tool_use
		}
		msg["content"] = newContent
	}

	// Pass 2: Merge consecutive same-role messages
	merged := make([]any, 0, len(messages))
	for _, m := range messages {
		msg, ok := m.(map[string]any)
		if !ok {
			continue
		}
		if len(merged) == 0 {
			merged = append(merged, cloneMessage(msg))
			continue
		}
		last, ok := merged[len(merged)-1].(map[string]any)
		if !ok {
			merged = append(merged, cloneMessage(msg))
			continue
		}
		if last["role"] != msg["role"] {
			merged = append(merged, cloneMessage(msg))
			continue
		}
		lastContent := toContentArray(last["content"])
		msgContent := toContentArray(msg["content"])
		toolResults := make([]any, 0)
		otherContent := make([]any, 0)
		for _, item := range lastContent {
			block, ok := item.(map[string]any)
			if ok && block["type"] == "tool_result" {
				toolResults = append(toolResults, block)
			} else {
				otherContent = append(otherContent, item)
			}
		}
		for _, item := range msgContent {
			block, ok := item.(map[string]any)
			if ok && block["type"] == "tool_result" {
				toolResults = append(toolResults, block)
			} else {
				otherContent = append(otherContent, item)
			}
		}
		last["content"] = append(toolResults, otherContent...)
	}

	return merged
}

func cloneMessage(msg map[string]any) map[string]any {
	cloned := make(map[string]any, len(msg))
	for k, v := range msg {
		cloned[k] = v
	}
	return cloned
}

func toContentArray(content any) []any {
	if arr, ok := content.([]any); ok {
		return arr
	}
	if s, ok := content.(string); ok {
		return []any{map[string]any{"type": "text", "text": s}}
	}
	return []any{}
}

// PrepareClaudeRequest prepares a Claude-format request body for the provider.
func PrepareClaudeRequest(body map[string]any, provider string, apiKey string, connectionId string) map[string]any {
	if _, ok := claudeFormatProvidersWithoutOutputConfig[provider]; ok {
		delete(body, "output_config")
	}

	// 1. System: remove all cache_control, add only to last block with ttl 1h
	if sys, ok := body["system"].([]any); ok && len(sys) > 0 {
		newSys := make([]any, 0, len(sys))
		for i, s := range sys {
			block, ok := s.(map[string]any)
			if !ok {
				newSys = append(newSys, s)
				continue
			}
			cloned := make(map[string]any, len(block))
			for k, v := range block {
				if k != "cache_control" {
					cloned[k] = v
				}
			}
			if i == len(sys)-1 {
				cloned["cache_control"] = map[string]any{"type": "ephemeral", "ttl": "1h"}
			}
			newSys = append(newSys, cloned)
		}
		body["system"] = newSys
	}

	// 2. Messages: process in optimized passes
	if rawMsgs, ok := body["messages"].([]any); ok && len(rawMsgs) > 0 {
		lenMsgs := len(rawMsgs)
		filtered := make([]any, 0, lenMsgs)

		// Pass 1: remove cache_control + filter empty messages
		for i := 0; i < lenMsgs; i++ {
			msg, ok := rawMsgs[i].(map[string]any)
			if !ok {
				continue
			}
			if content, ok := msg["content"].([]any); ok {
				for _, item := range content {
					block, ok := item.(map[string]any)
					if ok {
						delete(block, "cache_control")
					}
				}
			}
			isFinalAssistant := i == lenMsgs-1 && msg["role"] == "assistant"
			if isFinalAssistant || hasValidContent(msg) {
				filtered = append(filtered, msg)
			}
		}

		// Pass 1.5: Fix tool_use/tool_result ordering
		filtered = fixToolUseOrdering(filtered)
		body["messages"] = filtered

		// Check if thinking is enabled AND last message is from user
		var lastMessageIsUser bool
		var thinkingEnabled bool
		if len(filtered) > 0 {
			lastMsg, ok := filtered[len(filtered)-1].(map[string]any)
			if ok {
				lastMessageIsUser = lastMsg["role"] == "user"
			}
		}
		if thinking, ok := body["thinking"].(map[string]any); ok {
			if t, _ := thinking["type"].(string); t == "enabled" && lastMessageIsUser {
				thinkingEnabled = true
			}
		}

		// Pass 2 (reverse): add cache_control to last assistant + handle thinking
		lastAssistantProcessed := false
		for i := len(filtered) - 1; i >= 0; i-- {
			msg, ok := filtered[i].(map[string]any)
			if !ok || msg["role"] != "assistant" {
				continue
			}
			content, ok := msg["content"].([]any)
			if !ok || len(content) == 0 {
				continue
			}

			if !lastAssistantProcessed {
				for j := len(content) - 1; j >= 0; j-- {
					block, ok := content[j].(map[string]any)
					if !ok {
						continue
					}
					bt, _ := block["type"].(string)
					if bt != "thinking" && bt != "redacted_thinking" {
						block["cache_control"] = map[string]any{"type": "ephemeral"}
						break
					}
				}
				lastAssistantProcessed = true
			}

			if provider == "claude" || strings.HasPrefix(provider, "anthropic-compatible") {
				hasToolUse := false
				hasThinking := false
				for _, item := range content {
					block, ok := item.(map[string]any)
					if !ok {
						continue
					}
					bt, _ := block["type"].(string)
					if bt == "thinking" || bt == "redacted_thinking" {
						block["signature"] = defaultThinkingClaudeSignature
						hasThinking = true
					}
					if bt == "tool_use" {
						hasToolUse = true
					}
				}
				if thinkingEnabled && !hasThinking && hasToolUse {
					newContent := make([]any, 0, len(content)+1)
					newContent = append(newContent, map[string]any{
						"type":      "thinking",
						"thinking":  ".",
						"signature": defaultThinkingClaudeSignature,
					})
					newContent = append(newContent, content...)
					msg["content"] = newContent
				}
			}
		}
	}

	// 3. Tools: filter built-in tools for non-Anthropic providers, then handle cache_control
	if rawTools, ok := body["tools"].([]any); ok && len(rawTools) > 0 {
		if provider != "claude" {
			filtered := make([]any, 0, len(rawTools))
			for _, rt := range rawTools {
				tool, ok := rt.(map[string]any)
				if !ok {
					continue
				}
				toolType, _ := tool["type"].(string)
				if toolType == "" || toolType == "function" {
					filtered = append(filtered, tool)
				}
			}
			body["tools"] = filtered
			rawTools = filtered
		}

		for i, rt := range rawTools {
			tool, ok := rt.(map[string]any)
			if !ok {
				continue
			}
			cloned := make(map[string]any, len(tool))
			for k, v := range tool {
				if k != "cache_control" {
					cloned[k] = v
				}
			}
			if i == len(rawTools)-1 {
				cloned["cache_control"] = map[string]any{"type": "ephemeral", "ttl": "1h"}
			}
			rawTools[i] = cloned
		}

		if len(rawTools) == 0 {
			delete(body, "tools")
			delete(body, "tool_choice")
		}
	}

	// Apply cloaking for OAuth tokens
	if (provider == "claude" || strings.HasPrefix(provider, "anthropic-compatible")) && apiKey != "" {
		sessionId, err := deriveSessionId(connectionId)
		if err != nil {
			// Silently skip cloaking on derive error to avoid panics.
			return body
		}
		body = applyCloaking(body, apiKey, sessionId)
	}

	return body
}
