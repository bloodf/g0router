package translation

import (
	"encoding/json"
)

// openaiToResponsesRequest converts an OpenAI Chat Completions request body to
// an OpenAI Responses API request body.
func openaiToResponsesRequest(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error) {
	// Body already in Responses API format (e.g. Cursor CLI calling /chat/completions with input[])
	if _, ok := body["input"]; ok {
		result := make(map[string]any, len(body)+2)
		for k, v := range body {
			result[k] = v
		}
		result["model"] = model
		result["stream"] = true
		return result, nil
	}

	result := map[string]any{
		"model":   model,
		"input":   []any{},
		"stream":  true,
		"store":   false,
	}

	hasSystemMessage := false
	var messages []any
	if m, ok := body["messages"].([]any); ok {
		messages = m
	}

	for _, rawMsg := range messages {
		msg, ok := rawMsg.(map[string]any)
		if !ok {
			continue
		}
		role, _ := msg["role"].(string)

		if role == "system" {
			if !hasSystemMessage {
				content := ""
				if c, ok := msg["content"].(string); ok {
					content = c
				}
				result["instructions"] = content
				hasSystemMessage = true
			}
			continue
		}

		if role == "user" || role == "assistant" {
			contentType := "input_text"
			if role == "assistant" {
				contentType = "output_text"
			}

			var blocks []any
			if contentStr, ok := msg["content"].(string); ok {
				blocks = []any{map[string]any{"type": contentType, "text": contentStr}}
			} else if contentArr, ok := msg["content"].([]any); ok {
				for _, rawC := range contentArr {
					c, ok := rawC.(map[string]any)
					if !ok {
						continue
					}
					ct, _ := c["type"].(string)
					switch ct {
					case "text":
						blocks = append(blocks, map[string]any{"type": contentType, "text": c["text"]})
					case "image_url":
						var url string
						if u, ok := c["image_url"].(string); ok {
							url = u
						} else if uMap, ok := c["image_url"].(map[string]any); ok {
							if u, ok := uMap["url"].(string); ok {
								url = u
							}
						}
						detail := "auto"
						if dMap, ok := c["image_url"].(map[string]any); ok {
							if d, ok := dMap["detail"].(string); ok && d != "" {
								detail = d
							}
						}
						blocks = append(blocks, map[string]any{"type": "input_image", "image_url": url, "detail": detail})
					case "input_image":
						blocks = append(blocks, c)
					default:
						// Serialize unknown type as text
						text := ""
						if t, ok := c["text"].(string); ok && t != "" {
							text = t
						} else if t, ok := c["content"].(string); ok && t != "" {
							text = t
						} else {
							b, _ := json.Marshal(c)
							text = string(b)
						}
						blocks = append(blocks, map[string]any{"type": contentType, "text": text})
					}
				}
			}

			// Only push a message block if content is non-empty.
			// Assistant messages with only tool_calls have content: nil — skip the
			// message block in that case; the tool_calls are pushed separately below.
			if len(blocks) > 0 {
				result["input"] = append(result["input"].([]any), map[string]any{
					"type":    "message",
					"role":    role,
					"content": blocks,
				})
			}
		}

		// Convert tool calls
		if role == "assistant" {
			if rawToolCalls, ok := msg["tool_calls"].([]any); ok {
				for _, rawTC := range rawToolCalls {
					tc, ok := rawTC.(map[string]any)
					if !ok {
						continue
					}
					name := "_unknown"
					arguments := "{}"
					if fn, ok := tc["function"].(map[string]any); ok {
						if n, ok := fn["name"].(string); ok && n != "" {
							name = n
						}
						if a, ok := fn["arguments"].(string); ok {
							arguments = a
						}
					}
					result["input"] = append(result["input"].([]any), map[string]any{
						"type":      "function_call",
						"call_id":   clampCallID(tc["id"]),
						"name":      name,
						"arguments": arguments,
					})
				}
			}
		}

		// Convert tool results
		if role == "tool" {
			var output string
			if o, ok := msg["content"].(string); ok {
				output = o
			} else if arr, ok := msg["content"].([]any); ok {
				parts := make([]string, 0, len(arr))
				for _, rawC := range arr {
					c, ok := rawC.(map[string]any)
					if ok {
						if t, ok := c["text"].(string); ok && t != "" {
							parts = append(parts, t)
						} else {
							b, _ := json.Marshal(c)
							parts = append(parts, string(b))
						}
					} else {
						b, _ := json.Marshal(rawC)
						parts = append(parts, string(b))
					}
				}
				output = joinStrings(parts, "")
			} else {
				b, _ := json.Marshal(msg["content"])
				output = string(b)
			}
			result["input"] = append(result["input"].([]any), map[string]any{
				"type":     "function_call_output",
				"call_id":  clampCallID(msg["tool_call_id"]),
				"output":   output,
			})
		}
	}

	// If no system message, leave instructions empty
	if !hasSystemMessage {
		result["instructions"] = ""
	}

	// Convert tools format
	if rawTools, ok := body["tools"].([]any); ok && len(rawTools) > 0 {
		convertedTools := make([]any, 0, len(rawTools))
		for _, rawTool := range rawTools {
			tool, ok := rawTool.(map[string]any)
			if !ok {
				continue
			}
			if toolType, _ := tool["type"].(string); toolType == "function" {
				var fn map[string]any
				if f, ok := tool["function"].(map[string]any); ok {
					fn = f
				}
				if fn == nil {
					continue
				}
				name := ""
				if n, ok := fn["name"].(string); ok {
					name = n
				}
				desc := ""
				if d, ok := fn["description"].(string); ok {
					desc = d
				}
				var params map[string]any
				if p, ok := fn["parameters"].(map[string]any); ok {
					params = p
				}
				converted := map[string]any{
					"type":        "function",
					"name":        name,
					"description": desc,
					"parameters":  normalizeToolParameters(params),
				}
				if strict, ok := fn["strict"]; ok {
					converted["strict"] = strict
				}
				convertedTools = append(convertedTools, converted)
			} else {
				convertedTools = append(convertedTools, tool)
			}
		}
		if len(convertedTools) > 0 {
			result["tools"] = convertedTools
		}
	}

	// Pass through other relevant fields
	if temp, ok := body["temperature"]; ok && temp != nil {
		result["temperature"] = temp
	}
	if maxTokens, ok := body["max_tokens"]; ok && maxTokens != nil {
		result["max_tokens"] = maxTokens
	}
	if topP, ok := body["top_p"]; ok && topP != nil {
		result["top_p"] = topP
	}

	return result, nil
}
