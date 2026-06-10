package translation

import (
	"encoding/json"
	"fmt"
)

// claudeToOpenAIRequest converts a Claude-format request body to an OpenAI-
// shaped request body.
func claudeToOpenAIRequest(model string, body map[string]any, stream bool) (map[string]any, error) {
	result := map[string]any{
		"model":    model,
		"messages": []any{},
		"stream":   stream,
	}

	// Max tokens.
	// PAR-TRANS-011 (w1-c): adjustMaxTokens applies here.
	result["max_tokens"] = AdjustMaxTokens(body)

	// Temperature.
	if temp, ok := body["temperature"]; ok && temp != nil {
		result["temperature"] = temp
	}

	// System message.
	if sys, ok := body["system"]; ok && sys != nil {
		var systemContent string
		switch v := sys.(type) {
		case string:
			systemContent = v
		case []any:
			parts := make([]string, 0, len(v))
			for _, item := range v {
				if m, ok := item.(map[string]any); ok {
					if text, ok := m["text"].(string); ok {
						parts = append(parts, text)
					}
				}
			}
			systemContent = joinStrings(parts, "\n")
		}
		if systemContent != "" {
			result["messages"] = append(result["messages"].([]any), map[string]any{
				"role":    "system",
				"content": systemContent,
			})
		}
	}

	// Convert messages.
	if msgs, ok := body["messages"].([]any); ok {
		for _, msg := range msgs {
			converted, err := convertClaudeMessage(msg)
			if err != nil {
				return nil, fmt.Errorf("convert claude message: %w", err)
			}
			if converted == nil {
				continue
			}
			if arr, ok := converted.([]any); ok {
				result["messages"] = append(result["messages"].([]any), arr...)
			} else {
				result["messages"] = append(result["messages"].([]any), converted)
			}
		}
	}

	// Fix missing tool responses.
	result["messages"] = fixMissingToolResponses(result["messages"].([]any))

	// Tools.
	if tools, ok := body["tools"].([]any); ok {
		openaiTools := make([]any, 0, len(tools))
		for _, tool := range tools {
			tm, ok := tool.(map[string]any)
			if !ok {
				continue
			}
			name, _ := tm["name"].(string)
			desc := ""
			if d, ok := tm["description"].(string); ok {
				desc = d
			}
			params := map[string]any{"type": "object", "properties": map[string]any{}}
			if is, ok := tm["input_schema"].(map[string]any); ok {
				params = is
			}
			openaiTools = append(openaiTools, map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        name,
					"description": desc,
					"parameters":  params,
				},
			})
		}
		result["tools"] = openaiTools
	}

	// Tool choice.
	if tc, ok := body["tool_choice"]; ok && tc != nil {
		result["tool_choice"] = convertToolChoice(tc)
	}

	return result, nil
}

func joinStrings(parts []string, sep string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}

// fixMissingToolResponses inserts [No response received] placeholders for
// tool_calls that are not immediately followed by a tool response.
func fixMissingToolResponses(messages []any) []any {
	for i := 0; i < len(messages); i++ {
		msg, ok := messages[i].(map[string]any)
		if !ok || msg["role"] != "assistant" {
			continue
		}
		toolCalls, ok := msg["tool_calls"].([]any)
		if !ok || len(toolCalls) == 0 {
			continue
		}

		var ids []string
		for _, tc := range toolCalls {
			tcm, ok := tc.(map[string]any)
			if !ok {
				continue
			}
			if id, ok := tcm["id"].(string); ok && id != "" {
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			continue
		}

		responded := make(map[string]bool)
		insertPos := i + 1
		for j := i + 1; j < len(messages); j++ {
			next, ok := messages[j].(map[string]any)
			if !ok || next["role"] != "tool" {
				break
			}
			if id, ok := next["tool_call_id"].(string); ok {
				responded[id] = true
				insertPos = j + 1
			} else {
				break
			}
		}

		var missing []map[string]any
		for _, id := range ids {
			if !responded[id] {
				missing = append(missing, map[string]any{
					"role":         "tool",
					"tool_call_id": id,
					"content":      "[No response received]",
				})
			}
		}
		if len(missing) == 0 {
			continue
		}

		for _, m := range missing {
			messages = append(messages[:insertPos], append([]any{m}, messages[insertPos:]...)...)
			insertPos++
		}
		i = insertPos - 1
	}
	return messages
}

func convertClaudeMessage(msg any) (any, error) {
	m, ok := msg.(map[string]any)
	if !ok {
		return nil, nil
	}

	role := "assistant"
	if r, ok := m["role"].(string); ok && (r == "user" || r == "tool") {
		role = "user"
	}

	// Simple string content.
	if content, ok := m["content"].(string); ok {
		return map[string]any{"role": role, "content": content}, nil
	}

	// Array content.
	blocks, ok := m["content"].([]any)
	if !ok {
		return nil, nil
	}
	if len(blocks) == 0 {
		return map[string]any{"role": role, "content": ""}, nil
	}

	var parts []any
	var toolCalls []any
	var toolResults []any

	for _, b := range blocks {
		block, ok := b.(map[string]any)
		if !ok {
			continue
		}
		bt, _ := block["type"].(string)
		switch bt {
		case "text":
			if text, ok := block["text"].(string); ok {
				parts = append(parts, map[string]any{"type": "text", "text": text})
			}
		case "image":
			if src, ok := block["source"].(map[string]any); ok {
				if srcType, _ := src["type"].(string); srcType == "base64" {
					mediaType, _ := src["media_type"].(string)
					data, _ := src["data"].(string)
					parts = append(parts, map[string]any{
						"type": "image_url",
						"image_url": map[string]any{
							"url": fmt.Sprintf("data:%s;base64,%s", mediaType, data),
						},
					})
				}
			}
		case "tool_use":
			id, _ := block["id"].(string)
			name, _ := block["name"].(string)
			input := block["input"]
			if input == nil {
				input = map[string]any{}
			}
			args, err := json.Marshal(input)
			if err != nil {
				return nil, fmt.Errorf("marshal tool_use input: %w", err)
			}
			toolCalls = append(toolCalls, map[string]any{
				"id":   id,
				"type": "function",
				"function": map[string]any{
					"name":      name,
					"arguments": string(args),
				},
			})
		case "tool_result":
			id, _ := block["tool_use_id"].(string)
			resultContent := ""
			switch v := block["content"].(type) {
			case string:
				resultContent = v
			case []any:
				texts := make([]string, 0, len(v))
				for _, item := range v {
					if im, ok := item.(map[string]any); ok {
						if it, _ := im["type"].(string); it == "text" {
							if text, ok := im["text"].(string); ok {
								texts = append(texts, text)
							}
						}
					}
				}
				resultContent = joinStrings(texts, "\n")
				if resultContent == "" {
					resultContent = jsonString(v)
				}
			default:
				if v != nil {
					resultContent = jsonString(v)
				}
			}
			toolResults = append(toolResults, map[string]any{
				"role":         "tool",
				"tool_call_id": id,
				"content":      resultContent,
			})
		}
	}

	// If has tool results, return array of tool messages.
	if len(toolResults) > 0 {
		if len(parts) > 0 {
			textContent := partsToContent(parts)
			return append(toolResults, map[string]any{"role": "user", "content": textContent}), nil
		}
		return toolResults, nil
	}

	// If has tool calls, return assistant message with tool_calls.
	if len(toolCalls) > 0 {
		result := map[string]any{"role": "assistant"}
		if len(parts) > 0 {
			result["content"] = partsToContent(parts)
		}
		result["tool_calls"] = toolCalls
		return result, nil
	}

	// Return content.
	if len(parts) > 0 {
		return map[string]any{
			"role":    role,
			"content": partsToContent(parts),
		}, nil
	}

	return map[string]any{"role": role, "content": ""}, nil
}

func partsToContent(parts []any) any {
	if len(parts) == 1 {
		if p, ok := parts[0].(map[string]any); ok && p["type"] == "text" {
			if text, ok := p["text"].(string); ok {
				return text
			}
		}
	}
	return parts
}

func jsonString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func convertToolChoice(choice any) any {
	if choice == nil {
		return "auto"
	}
	if s, ok := choice.(string); ok {
		switch s {
		case "auto":
			return "auto"
		case "any":
			return "required"
		default:
			return "auto"
		}
	}
	m, ok := choice.(map[string]any)
	if !ok {
		return "auto"
	}
	switch m["type"] {
	case "auto":
		return "auto"
	case "any":
		return "required"
	case "tool":
		name, _ := m["name"].(string)
		return map[string]any{
			"type":     "function",
			"function": map[string]any{"name": name},
		}
	default:
		return "auto"
	}
}
