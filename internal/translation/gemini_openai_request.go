package translation

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// geminiToOpenAIRequest converts a Gemini-format request body to an OpenAI-
// shaped request body. Ported from request/gemini-to-openai.js:6-146 (row 066).
func geminiToOpenAIRequest(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error) {
	result := map[string]any{
		"model":    model,
		"messages": []any{},
		"stream":   stream,
	}

	// Generation config (:14-26).
	if genConfigRaw, ok := body["generationConfig"]; ok && genConfigRaw != nil {
		if config, ok := genConfigRaw.(map[string]any); ok {
			if maxOutputTokens, ok := config["maxOutputTokens"]; ok && maxOutputTokens != nil {
				tempBody := map[string]any{
					"max_tokens": maxOutputTokens,
					"tools":      body["tools"],
				}
				result["max_tokens"] = AdjustMaxTokens(tempBody)
			}
			if temp, ok := config["temperature"]; ok && temp != nil {
				result["temperature"] = temp
			}
			if topP, ok := config["topP"]; ok && topP != nil {
				result["top_p"] = topP
			}
		}
	}

	// System instruction (:28-37).
	if sysInstRaw, ok := body["systemInstruction"]; ok && sysInstRaw != nil {
		systemText := extractGeminiText(sysInstRaw)
		if systemText != "" {
			result["messages"] = append(result["messages"].([]any), map[string]any{
				"role":    "system",
				"content": systemText,
			})
		}
	}

	// Convert contents to messages (:39-47).
	if contentsRaw, ok := body["contents"]; ok && contentsRaw != nil {
		if contents, ok := contentsRaw.([]any); ok {
			for _, contentRaw := range contents {
				if content, ok := contentRaw.(map[string]any); ok {
					converted := convertGeminiContent(content)
					if converted != nil {
						result["messages"] = append(result["messages"].([]any), converted)
					}
				}
			}
		}
	}

	// Tools (:50-66).
	if toolsRaw, ok := body["tools"]; ok && toolsRaw != nil {
		if tools, ok := toolsRaw.([]any); ok {
			var openAITools []any
			for _, toolRaw := range tools {
				if tool, ok := toolRaw.(map[string]any); ok {
					if funcDeclsRaw, ok := tool["functionDeclarations"]; ok && funcDeclsRaw != nil {
						if funcDecls, ok := funcDeclsRaw.([]any); ok {
							for _, funcRaw := range funcDecls {
								if fn, ok := funcRaw.(map[string]any); ok {
									name := ""
									if s, ok := fn["name"].(string); ok {
										name = s
									}
									description := ""
									if s, ok := fn["description"].(string); ok {
										description = s
									}
									parameters := map[string]any{
										"type":       "object",
										"properties": map[string]any{},
									}
									if params, ok := fn["parameters"].(map[string]any); ok && params != nil {
										parameters = params
									}
									openAITools = append(openAITools, map[string]any{
										"type": "function",
										"function": map[string]any{
											"name":        name,
											"description": description,
											"parameters":  parameters,
										},
									})
								}
							}
						}
					}
				}
			}
			if len(openAITools) > 0 {
				result["tools"] = openAITools
			}
		}
	}

	return result, nil
}

// jsTruthy approximates JavaScript truthiness for values found in Gemini
// functionResponse maps. Falsey values: nil, false, 0, 0.0, "".
func jsTruthy(v any) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != ""
	case int:
		return val != 0
	case int8:
		return val != 0
	case int16:
		return val != 0
	case int32:
		return val != 0
	case int64:
		return val != 0
	case uint:
		return val != 0
	case uint8:
		return val != 0
	case uint16:
		return val != 0
	case uint32:
		return val != 0
	case uint64:
		return val != 0
	case float32:
		return val != 0
	case float64:
		return val != 0
	case map[string]any:
		return len(val) > 0
	case []any:
		return len(val) > 0
	default:
		return true
	}
}

// convertGeminiContent turns a single Gemini content object into an OpenAI
// message. Ported from request/gemini-to-openai.js:72-133.
func convertGeminiContent(content map[string]any) map[string]any {
	role := "assistant"
	if r, ok := content["role"].(string); ok && r == "user" {
		role = "user"
	}

	partsRaw, ok := content["parts"].([]any)
	if !ok || len(partsRaw) == 0 {
		return nil
	}

	var parts []any
	var toolCalls []any

	for i, partRaw := range partsRaw {
		part, ok := partRaw.(map[string]any)
		if !ok {
			continue
		}

		if text, ok := part["text"]; ok && text != nil {
			if s, ok := text.(string); ok {
				parts = append(parts, map[string]any{"type": "text", "text": s})
			}
		}

		if inlineDataRaw, ok := part["inlineData"]; ok && inlineDataRaw != nil {
			if inlineData, ok := inlineDataRaw.(map[string]any); ok {
				mimeType := ""
				if s, ok := inlineData["mimeType"].(string); ok {
					mimeType = s
				} else if s, ok := inlineData["mime_type"].(string); ok {
					mimeType = s
				}
				data := ""
				if s, ok := inlineData["data"].(string); ok {
					data = s
				}
				parts = append(parts, map[string]any{
					"type": "image_url",
					"image_url": map[string]any{
						"url": fmt.Sprintf("data:%s;base64,%s", mimeType, data),
					},
				})
			}
		}

		if functionCallRaw, ok := part["functionCall"]; ok && functionCallRaw != nil {
			if functionCall, ok := functionCallRaw.(map[string]any); ok {
				name := ""
				if s, ok := functionCall["name"].(string); ok {
					name = s
				}
				args := map[string]any{}
				if a, ok := functionCall["args"].(map[string]any); ok && a != nil {
					args = a
				}
				argsJSON, err := json.Marshal(args)
				if err != nil {
					argsJSON = []byte("{}")
				}
				toolCalls = append(toolCalls, map[string]any{
					"id":   fmt.Sprintf("call_%d_%d", time.Now().UnixMilli(), i),
					"type": "function",
					"function": map[string]any{
						"name":      name,
						"arguments": string(argsJSON),
					},
				})
			}
		}

		if functionResponseRaw, ok := part["functionResponse"]; ok && functionResponseRaw != nil {
			if functionResponse, ok := functionResponseRaw.(map[string]any); ok {
				toolCallID := ""
				if s, ok := functionResponse["id"].(string); ok && s != "" {
					toolCallID = s
				} else if s, ok := functionResponse["name"].(string); ok {
					toolCallID = s
				}

				var contentVal any
				if resp, ok := functionResponse["response"]; ok && resp != nil {
					if respMap, ok := resp.(map[string]any); ok {
						if result, ok := respMap["result"]; ok && jsTruthy(result) {
							contentVal = result
						} else if jsTruthy(respMap) {
							contentVal = respMap
						} else {
							contentVal = map[string]any{}
						}
					} else if jsTruthy(resp) {
						contentVal = resp
					} else {
						contentVal = map[string]any{}
					}
				} else {
					contentVal = map[string]any{}
				}
				contentBytes, err := json.Marshal(contentVal)
				if err != nil {
					contentBytes = []byte("{}")
				}

				return map[string]any{
					"role":         "tool",
					"tool_call_id": toolCallID,
					"content":      string(contentBytes),
				}
			}
		}
	}

	if len(toolCalls) > 0 {
		msg := map[string]any{
			"role":       "assistant",
			"tool_calls": toolCalls,
		}
		if len(parts) > 0 {
			if len(parts) == 1 {
				if p, ok := parts[0].(map[string]any); ok {
					msg["content"] = p["text"]
				} else {
					msg["content"] = parts
				}
			} else {
				msg["content"] = parts
			}
		}
		return msg
	}

	if len(parts) > 0 {
		if len(parts) == 1 {
			if p, ok := parts[0].(map[string]any); ok {
				if p["type"] == "text" {
					return map[string]any{
						"role":    role,
						"content": p["text"],
					}
				}
			}
		}
		return map[string]any{
			"role":    role,
			"content": parts,
		}
	}

	return nil
}

// extractGeminiText extracts plain text from a Gemini content value. Strings
// pass through; objects with a parts array yield the concatenation of text
// blocks. Ported from request/gemini-to-openai.js:136-142.
func extractGeminiText(content any) string {
	if s, ok := content.(string); ok {
		return s
	}
	if m, ok := content.(map[string]any); ok {
		if partsRaw, ok := m["parts"].([]any); ok {
			var texts []string
			for _, partRaw := range partsRaw {
				if part, ok := partRaw.(map[string]any); ok {
					if text, ok := part["text"].(string); ok {
						texts = append(texts, text)
					}
				}
			}
			return strings.Join(texts, "")
		}
	}
	return ""
}
