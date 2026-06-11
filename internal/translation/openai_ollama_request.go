package translation

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// openaiToOllamaRequest converts an OpenAI-format request body to an Ollama-
// shaped request body. Port of open-sse/translator/request/openai-to-ollama.js.
func openaiToOllamaRequest(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error) {
	msgs, err := normalizeOllamaMessages(body)
	if err != nil {
		return nil, fmt.Errorf("normalize ollama messages: %w", err)
	}
	result := map[string]any{
		"model":    model,
		"messages": msgs,
		"stream":   stream,
	}

	// Temperature
	if temp, ok := body["temperature"]; ok && temp != nil {
		options, _ := result["options"].(map[string]any)
		if options == nil {
			options = map[string]any{}
			result["options"] = options
		}
		options["temperature"] = temp
	}

	// Max tokens (Ollama uses num_predict)
	if maxTokens, ok := body["max_tokens"]; ok && maxTokens != nil {
		options, _ := result["options"].(map[string]any)
		if options == nil {
			options = map[string]any{}
			result["options"] = options
		}
		options["num_predict"] = maxTokens
	}

	// Top_p
	if topP, ok := body["top_p"]; ok && topP != nil {
		options, _ := result["options"].(map[string]any)
		if options == nil {
			options = map[string]any{}
			result["options"] = options
		}
		options["top_p"] = topP
	}

	// Tools (Ollama supports tools in OpenAI format)
	if tools, ok := body["tools"].([]any); ok && len(tools) > 0 {
		result["tools"] = tools
	}

	// Tool choice
	if toolChoice, ok := body["tool_choice"]; ok && toolChoice != nil {
		result["tool_choice"] = toolChoice
	}

	return result, nil
}

// normalizeOllamaMessages normalizes OpenAI messages to Ollama format.
func normalizeOllamaMessages(body map[string]any) ([]any, error) {
	rawMsgs, ok := body["messages"].([]any)
	if !ok {
		return nil, nil
	}

	// First pass: build tool_call_id -> tool_name map from assistant messages.
	toolCallMap := make(map[string]string)
	for _, msg := range rawMsgs {
		m, ok := msg.(map[string]any)
		if !ok {
			continue
		}
		if role, _ := m["role"].(string); role != "assistant" {
			continue
		}
		rawToolCalls, ok := m["tool_calls"].([]any)
		if !ok {
			continue
		}
		for _, tc := range rawToolCalls {
			tcMap, ok := tc.(map[string]any)
			if !ok {
				continue
			}
			id, _ := tcMap["id"].(string)
			var fn map[string]any
			if f, ok := tcMap["function"].(map[string]any); ok {
				fn = f
			}
			name, _ := fn["name"].(string)
			if id != "" && name != "" {
				toolCallMap[id] = name
			}
		}
	}

	// Second pass: convert messages.
	result := make([]any, 0, len(rawMsgs))
	for _, msg := range rawMsgs {
		m, ok := msg.(map[string]any)
		if !ok {
			continue
		}
		role, _ := m["role"].(string)

		// Handle tool result messages.
		if role == "tool" {
			toolResult := normalizeOllamaContent(m["content"])
			if toolResult == "" {
				continue
			}
			toolCallID, _ := m["tool_call_id"].(string)
			toolName := toolCallMap[toolCallID]
			if toolName == "" {
				toolName, _ = m["name"].(string)
			}
			if toolName == "" {
				toolName = "unknown_tool"
			}
			result = append(result, map[string]any{
				"role":      "tool",
				"tool_name": toolName,
				"content":   toolResult,
			})
			continue
		}

		// Handle assistant messages with tool_calls.
		if role == "assistant" {
			rawToolCalls, ok := m["tool_calls"].([]any)
			if ok && len(rawToolCalls) > 0 {
				content := normalizeOllamaContent(m["content"])
				ollamaToolCalls := make([]any, 0, len(rawToolCalls))
				for _, tc := range rawToolCalls {
					tcMap, ok := tc.(map[string]any)
					if !ok {
						continue
					}
					var fn map[string]any
					if f, ok := tcMap["function"].(map[string]any); ok {
						fn = f
					}
					name := ""
					if n, ok := fn["name"].(string); ok {
						name = n
					}
					index := 0
					if i, ok := tcMap["index"].(float64); ok {
						index = int(i)
					} else if i, ok := tcMap["index"].(int); ok {
						index = i
					}
					var arguments any
					if args, ok := fn["arguments"]; ok {
						switch v := args.(type) {
						case string:
							var parsed any
							if err := json.Unmarshal([]byte(v), &parsed); err != nil {
								return nil, fmt.Errorf("parse tool call arguments: %w", err)
							}
							arguments = parsed
						case map[string]any:
							arguments = v
						default:
							arguments = map[string]any{}
						}
					} else {
						arguments = map[string]any{}
					}
					ollamaToolCalls = append(ollamaToolCalls, map[string]any{
						"type": "function",
						"function": map[string]any{
							"index":     index,
							"name":      name,
							"arguments": arguments,
						},
					})
				}
				result = append(result, map[string]any{
					"role":       "assistant",
					"content":    content,
					"tool_calls": ollamaToolCalls,
				})
				continue
			}
		}

		// Normal messages.
		content := normalizeOllamaContent(m["content"])
		images := extractOllamaImages(m["content"])

		// Skip empty messages (except assistant).
		if content == "" && role != "assistant" {
			continue
		}

		out := map[string]any{
			"role":    role,
			"content": content,
		}
		if len(images) > 0 {
			out["images"] = images
		}
		result = append(result, out)
	}

	return result, nil
}

func normalizeOllamaContent(content any) string {
	if s, ok := content.(string); ok {
		return s
	}
	if arr, ok := content.([]any); ok {
		var parts []string
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if t, _ := m["type"].(string); t != "text" {
				continue
			}
			if text, ok := m["text"].(string); ok && text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

func extractOllamaImages(content any) []any {
	arr, ok := content.([]any)
	if !ok {
		return nil
	}
	dataURIPattern, err := regexp.Compile(`^data:[^;]+;base64,([\s\S]+)$`)
	if err != nil {
		return nil
	}
	images := make([]any, 0)
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if t, _ := m["type"].(string); t != "image_url" {
			continue
		}
		var url string
		if imageURL, ok := m["image_url"].(map[string]any); ok {
			url, _ = imageURL["url"].(string)
		} else if s, ok := m["image_url"].(string); ok {
			url = s
		}
		if url == "" {
			continue
		}
		match := dataURIPattern.FindStringSubmatch(url)
		if len(match) != 2 {
			continue
		}
		images = append(images, match[1])
	}
	return images
}
