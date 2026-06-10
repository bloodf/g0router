package translation

import (
	"encoding/json"
)

// responsesToOpenAIRequest converts an OpenAI Responses API request body to an
// OpenAI Chat Completions request body.
func responsesToOpenAIRequest(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error) {
	if _, ok := body["input"]; !ok {
		return body, nil
	}

	result := make(map[string]any, len(body)+1)
	for k, v := range body {
		result[k] = v
	}
	result["messages"] = []any{}

	// Convert instructions to system message
	if instructions, ok := body["instructions"].(string); ok && instructions != "" {
		result["messages"] = append(result["messages"].([]any), map[string]any{
			"role":    "system",
			"content": instructions,
		})
	}

	var currentAssistantMsg map[string]any
	var pendingToolResults []any
	var pendingReasoning string

	inputItems := normalizeResponsesInput(body["input"])
	if inputItems == nil {
		return body, nil
	}

	for _, rawItem := range inputItems {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}

		itemType := ""
		if t, ok := item["type"].(string); ok && t != "" {
			itemType = t
		} else if _, hasRole := item["role"]; hasRole {
			itemType = "message"
		}

		switch itemType {
		case "message":
			// Flush any pending assistant message with tool calls
			if currentAssistantMsg != nil {
				result["messages"] = append(result["messages"].([]any), currentAssistantMsg)
				currentAssistantMsg = nil
			}
			// Flush pending tool results
			if len(pendingToolResults) > 0 {
				result["messages"] = append(result["messages"].([]any), pendingToolResults...)
				pendingToolResults = nil
			}

			var content any
			if contentArr, ok := item["content"].([]any); ok {
				converted := make([]any, 0, len(contentArr))
				for _, rawC := range contentArr {
					c, ok := rawC.(map[string]any)
					if !ok {
						converted = append(converted, rawC)
						continue
					}
					ct, _ := c["type"].(string)
					switch ct {
					case "input_text":
						converted = append(converted, map[string]any{"type": "text", "text": c["text"]})
					case "output_text":
						converted = append(converted, map[string]any{"type": "text", "text": c["text"]})
					case "input_image":
						url := ""
						if u, ok := c["image_url"].(string); ok && u != "" {
							url = u
						} else if u, ok := c["file_id"].(string); ok && u != "" {
							url = u
						}
						detail := "auto"
						if d, ok := c["detail"].(string); ok && d != "" {
							detail = d
						}
						converted = append(converted, map[string]any{
							"type": "image_url",
							"image_url": map[string]any{
								"url":    url,
								"detail": detail,
							},
						})
					default:
						converted = append(converted, c)
					}
				}
				content = converted
			} else {
				content = item["content"]
			}

			msg := map[string]any{
				"role":    item["role"],
				"content": content,
			}
			if role, _ := item["role"].(string); role == "assistant" && pendingReasoning != "" {
				msg["reasoning_content"] = pendingReasoning
			}
			pendingReasoning = ""
			result["messages"] = append(result["messages"].([]any), msg)

		case "function_call":
			name := ""
			if n, ok := item["name"].(string); ok {
				name = n
			}
			if name == "" {
				continue
			}
			if currentAssistantMsg == nil {
				currentAssistantMsg = map[string]any{
					"role":       "assistant",
					"content":    nil,
					"tool_calls": []any{},
				}
				if pendingReasoning != "" {
					currentAssistantMsg["reasoning_content"] = pendingReasoning
					pendingReasoning = ""
				}
			}
			callID := item["call_id"]
			arguments := "{}"
			if a, ok := item["arguments"].(string); ok {
				arguments = a
			}
			toolCall := map[string]any{
				"id":   callID,
				"type": "function",
				"function": map[string]any{
					"name":      name,
					"arguments": arguments,
				},
			}
			currentAssistantMsg["tool_calls"] = append(currentAssistantMsg["tool_calls"].([]any), toolCall)

		case "function_call_output":
			// Flush assistant message first if exists
			if currentAssistantMsg != nil {
				result["messages"] = append(result["messages"].([]any), currentAssistantMsg)
				currentAssistantMsg = nil
			}
			// Flush any pending tool results first
			if len(pendingToolResults) > 0 {
				result["messages"] = append(result["messages"].([]any), pendingToolResults...)
				pendingToolResults = nil
			}
			output := ""
			if o, ok := item["output"].(string); ok {
				output = o
			} else {
				b, _ := json.Marshal(item["output"])
				output = string(b)
			}
			result["messages"] = append(result["messages"].([]any), map[string]any{
				"role":         "tool",
				"tool_call_id": item["call_id"],
				"content":      output,
			})

		case "reasoning":
			txt := extractReasoningText(item)
			if txt != "" {
				if pendingReasoning != "" {
					pendingReasoning = pendingReasoning + "\n" + txt
				} else {
					pendingReasoning = txt
				}
			}
		}
	}

	// Flush remaining
	if currentAssistantMsg != nil {
		result["messages"] = append(result["messages"].([]any), currentAssistantMsg)
	}
	if len(pendingToolResults) > 0 {
		result["messages"] = append(result["messages"].([]any), pendingToolResults...)
	}

	// Convert tools format
	if rawTools, ok := body["tools"].([]any); ok && len(rawTools) > 0 {
		convertedTools := make([]any, 0, len(rawTools))
		for _, rawTool := range rawTools {
			tool, ok := rawTool.(map[string]any)
			if !ok {
				continue
			}
			// Already in Chat Completions format
			if _, hasFunction := tool["function"]; hasFunction {
				convertedTools = append(convertedTools, tool)
				continue
			}
			name := ""
			if n, ok := tool["name"].(string); ok {
				name = n
			}
			if name == "" {
				continue
			}
			desc := ""
			if d, ok := tool["description"].(string); ok {
				desc = d
			}
			var params map[string]any
			if p, ok := tool["parameters"].(map[string]any); ok {
				params = p
			}
			converted := map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        name,
					"description": desc,
					"parameters":  normalizeToolParameters(params),
				},
			}
			if strict, ok := tool["strict"]; ok {
				converted["function"].(map[string]any)["strict"] = strict
			}
			convertedTools = append(convertedTools, converted)
		}
		if len(convertedTools) > 0 {
			result["tools"] = convertedTools
		}
	}

	// Cleanup Responses API specific fields
	delete(result, "input")
	delete(result, "instructions")
	delete(result, "include")
	delete(result, "prompt_cache_key")
	delete(result, "store")
	delete(result, "reasoning")

	return result, nil
}

func extractReasoningText(item map[string]any) string {
	if summary, ok := item["summary"].([]any); ok && len(summary) > 0 {
		parts := make([]string, 0, len(summary))
		for _, rawS := range summary {
			s, ok := rawS.(map[string]any)
			if !ok {
				continue
			}
			if text, ok := s["text"].(string); ok && text != "" {
				parts = append(parts, text)
			}
		}
		if len(parts) > 0 {
			return joinStrings(parts, "\n")
		}
	}
	if content, ok := item["content"].([]any); ok && len(content) > 0 {
		parts := make([]string, 0, len(content))
		for _, rawC := range content {
			c, ok := rawC.(map[string]any)
			if !ok {
				continue
			}
			if text, ok := c["text"].(string); ok && text != "" {
				parts = append(parts, text)
			}
		}
		if len(parts) > 0 {
			return joinStrings(parts, "\n")
		}
	}
	return ""
}


