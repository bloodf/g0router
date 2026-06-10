package translation

import (
	"strings"
)

var validOpenAIContentTypes = map[string]bool{
	"text":        true,
	"image_url":   true,
	"image":       true,
	"input_audio": true,
	"audio_url":   true,
}

// FilterToOpenAIFormat normalizes a request body to strict OpenAI format:
// developer→system, drops thinking blocks, strips signature/cache_control,
// removes empty messages (preserving tool messages), normalizes tools and
// tool_choice shapes.
// Ported from openaiHelper.js:9-129 (row 010).
func FilterToOpenAIFormat(body map[string]any) map[string]any {
	if body == nil {
		return body
	}

	body = shallowCopyMap(body)

	// First pass: filter blocks and normalize roles.
	if msgsRaw, ok := body["messages"].([]any); ok {
		filtered := make([]any, 0, len(msgsRaw))
		for _, msgRaw := range msgsRaw {
			msg, ok := msgRaw.(map[string]any)
			if !ok {
				continue
			}

			// developer → system.
			if role, _ := msg["role"].(string); role == "developer" {
				msg = shallowCopyMap(msg)
				msg["role"] = "system"
			}

			// Keep tool messages untouched.
			if role, _ := msg["role"].(string); role == "tool" {
				filtered = append(filtered, msg)
				continue
			}

			// Keep assistant messages with tool_calls untouched.
			if role, _ := msg["role"].(string); role == "assistant" {
				if tc, ok := msg["tool_calls"].([]any); ok && len(tc) > 0 {
					filtered = append(filtered, msg)
					continue
				}
			}

			// String content passes through.
			if _, isString := msg["content"].(string); isString {
				filtered = append(filtered, msg)
				continue
			}

			// Array content: filter blocks.
			if blocks, ok := msg["content"].([]any); ok {
				cleaned := make([]any, 0, len(blocks))
				for _, bRaw := range blocks {
					b, ok := bRaw.(map[string]any)
					if !ok {
						continue
					}
					bt, _ := b["type"].(string)
					if bt == "thinking" || bt == "redacted_thinking" {
						continue
					}
					if validOpenAIContentTypes[bt] {
						cleanBlock := shallowCopyMap(b)
						delete(cleanBlock, "signature")
						delete(cleanBlock, "cache_control")
						cleaned = append(cleaned, cleanBlock)
					} else if bt == "tool_use" {
						continue
					} else if bt == "tool_result" {
						cleanBlock := shallowCopyMap(b)
						delete(cleanBlock, "signature")
						delete(cleanBlock, "cache_control")
						cleaned = append(cleaned, cleanBlock)
					}
				}
				if len(cleaned) == 0 {
					cleaned = append(cleaned, map[string]any{"type": "text", "text": ""})
				}
				msg = shallowCopyMap(msg)
				msg["content"] = cleaned
				filtered = append(filtered, msg)
				continue
			}

			filtered = append(filtered, msg)
		}

		// Second pass: drop empty messages (but NEVER tool or assistant-with-tool_calls).
		pruned := make([]any, 0, len(filtered))
		for _, msgRaw := range filtered {
			msg, ok := msgRaw.(map[string]any)
			if !ok {
				continue
			}

			role, _ := msg["role"].(string)
			if role == "tool" {
				pruned = append(pruned, msg)
				continue
			}
			if role == "assistant" {
				if tc, ok := msg["tool_calls"].([]any); ok && len(tc) > 0 {
					pruned = append(pruned, msg)
					continue
				}
			}

			keep := true
			if s, ok := msg["content"].(string); ok {
				keep = strings.TrimSpace(s) != ""
			} else if blocks, ok := msg["content"].([]any); ok {
				hasNonEmpty := false
				for _, bRaw := range blocks {
					b, ok := bRaw.(map[string]any)
					if !ok {
						continue
					}
					if b["type"] != "text" {
						hasNonEmpty = true
						break
					}
					if text, ok := b["text"].(string); ok && strings.TrimSpace(text) != "" {
						hasNonEmpty = true
						break
					}
				}
				keep = hasNonEmpty
			}
			if keep {
				pruned = append(pruned, msg)
			}
		}

		body["messages"] = pruned
	}

	// Remove empty tools array.
	if tools, ok := body["tools"].([]any); ok && len(tools) == 0 {
		delete(body, "tools")
	}

	// Normalize tools.
	if tools, ok := body["tools"].([]any); ok && len(tools) > 0 {
		normalized := make([]any, 0)
		for _, toolRaw := range tools {
			tool, ok := toolRaw.(map[string]any)
			if !ok {
				normalized = append(normalized, toolRaw)
				continue
			}
			// Already OpenAI format.
			if toolType, _ := tool["type"].(string); toolType == "function" && tool["function"] != nil {
				normalized = append(normalized, tool)
				continue
			}
			// Claude format: {name, description, input_schema}
			if name, ok := tool["name"].(string); ok && name != "" {
				desc := ""
				if d, ok := tool["description"].(string); ok {
					desc = d
				}
				schema := map[string]any{"type": "object", "properties": map[string]any{}}
				if is, ok := tool["input_schema"].(map[string]any); ok {
					schema = is
				}
				normalized = append(normalized, map[string]any{
					"type": "function",
					"function": map[string]any{
						"name":        name,
						"description": desc,
						"parameters":  schema,
					},
				})
				continue
			}
			// Gemini format: {functionDeclarations: [{name, description, parameters}]}
			if fdList, ok := tool["functionDeclarations"].([]any); ok {
				for _, fdRaw := range fdList {
					fd, ok := fdRaw.(map[string]any)
					if !ok {
						continue
					}
					name := ""
					if n, ok := fd["name"].(string); ok {
						name = n
					}
					desc := ""
					if d, ok := fd["description"].(string); ok {
						desc = d
					}
					schema := map[string]any{"type": "object", "properties": map[string]any{}}
					if p, ok := fd["parameters"].(map[string]any); ok {
						schema = p
					}
					normalized = append(normalized, map[string]any{
						"type": "function",
						"function": map[string]any{
							"name":        name,
							"description": desc,
							"parameters":  schema,
						},
					})
				}
				continue
			}
			normalized = append(normalized, tool)
		}
		body["tools"] = normalized
	}

	// Normalize tool_choice.
	if tc, ok := body["tool_choice"].(map[string]any); ok && tc != nil {
		tcType, _ := tc["type"].(string)
		switch tcType {
		case "auto":
			body["tool_choice"] = "auto"
		case "any":
			body["tool_choice"] = "required"
		case "tool":
			if name, ok := tc["name"].(string); ok && name != "" {
				body["tool_choice"] = map[string]any{
					"type": "function",
					"function": map[string]any{
						"name": name,
					},
				}
			}
		}
	}

	return body
}

func shallowCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
