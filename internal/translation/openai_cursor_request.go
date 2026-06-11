package translation

import (
	"encoding/json"
	"fmt"
	"strings"
)

// buildCursorRequest converts an OpenAI-format request body to a Cursor-shaped
// request body. Port of open-sse/translator/request/openai-to-cursor.js.
func buildCursorRequest(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error) {
	var rawMessages []any
	if m, ok := body["messages"].([]any); ok {
		rawMessages = m
	}
	messages := convertCursorMessages(rawMessages)

	result := make(map[string]any, len(body)+1)
	for k, v := range body {
		if k == "user" || k == "metadata" || k == "tool_choice" || k == "stream_options" || k == "system" {
			continue
		}
		result[k] = v
	}
	result["messages"] = messages
	result["max_tokens"] = 32000

	return result, nil
}

func extractContent(content any) string {
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
			if text, ok := m["text"].(string); ok {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "")
	}
	return ""
}

func sanitizeToolResultText(text string) string {
	// Strip non-printable control chars that can produce backend request errors.
	var b strings.Builder
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c <= 0x08 || c == 0x0B || c == 0x0C || (c >= 0x0E && c <= 0x1F) || c == 0x7F {
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func escapeXml(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	return text
}

func buildToolResultBlock(toolName, toolCallID, resultText string) string {
	cleanResult := sanitizeToolResultText(resultText)
	name := toolName
	if name == "" {
		name = "tool"
	}
	return strings.Join([]string{
		"<tool_result>",
		fmt.Sprintf("<tool_name>%s</tool_name>", escapeXml(name)),
		fmt.Sprintf("<tool_call_id>%s</tool_call_id>", escapeXml(toolCallID)),
		fmt.Sprintf("<result>%s</result>", escapeXml(cleanResult)),
		"</tool_result>",
	}, "\n")
}

func normalizeToolCallID(id string) string {
	if idx := strings.Index(id, "\n"); idx != -1 {
		return id[:idx]
	}
	return id
}

func convertCursorMessages(messages []any) []any {
	result := make([]any, 0, len(messages))

	// Build a map of tool_call_id -> tool name from assistant tool calls.
	toolCallMetaMap := make(map[string]string)
	rememberToolMeta := func(toolCallID, toolName string) {
		if toolCallID == "" {
			return
		}
		name := toolName
		if name == "" {
			name = "tool"
		}
		toolCallMetaMap[toolCallID] = name
		normalized := normalizeToolCallID(toolCallID)
		if normalized != "" && normalized != toolCallID {
			toolCallMetaMap[normalized] = name
		}
	}

	for _, msg := range messages {
		m, ok := msg.(map[string]any)
		if !ok {
			continue
		}
		role, _ := m["role"].(string)
		if role == "assistant" {
			if tcs, ok := m["tool_calls"].([]any); ok {
				for _, tc := range tcs {
					tcMap, ok := tc.(map[string]any)
					if !ok {
						continue
					}
					fn, _ := tcMap["function"].(map[string]any)
					name, _ := fn["name"].(string)
					rememberToolMeta(coalesceString(tcMap["id"]), name)
				}
			}
			if arr, ok := m["content"].([]any); ok {
				for _, part := range arr {
					pMap, ok := part.(map[string]any)
					if !ok {
						continue
					}
					if t, _ := pMap["type"].(string); t != "tool_use" {
						continue
					}
					rememberToolMeta(coalesceString(pMap["id"]), coalesceString(pMap["name"]))
				}
			}
		}
	}

	for _, msg := range messages {
		m, ok := msg.(map[string]any)
		if !ok {
			continue
		}
		role, _ := m["role"].(string)

		if role == "system" {
			result = append(result, map[string]any{
				"role":    "user",
				"content": "[System Instructions]\n" + extractContent(m["content"]),
			})
			continue
		}

		if role == "tool" {
			toolContent := extractContent(m["content"])
			toolCallID := coalesceString(m["tool_call_id"])
			toolName := coalesceString(m["name"])
			if toolName == "" {
				if name, ok := toolCallMetaMap[toolCallID]; ok {
					toolName = name
				} else {
					toolName = "tool"
				}
			}
			result = append(result, map[string]any{
				"role":    "user",
				"content": buildToolResultBlock(toolName, toolCallID, toolContent),
			})
			continue
		}

		if role == "user" || role == "assistant" {
			if role == "user" {
				if arr, ok := m["content"].([]any); ok {
					parts := make([]string, 0)
					for _, block := range arr {
						bMap, ok := block.(map[string]any)
						if !ok {
							continue
						}
						bType, _ := bMap["type"].(string)
						if bType == "text" {
							if text, ok := bMap["text"].(string); ok {
								parts = append(parts, text)
							}
							continue
						}
						if bType == "tool_result" {
							toolCallID := coalesceString(bMap["tool_use_id"])
							toolName := "tool"
							if name, ok := toolCallMetaMap[toolCallID]; ok {
								toolName = name
							} else {
								normalized := normalizeToolCallID(toolCallID)
								if name, ok := toolCallMetaMap[normalized]; ok {
									toolName = name
								}
							}
							toolContent := extractContent(bMap["content"])
							parts = append(parts, buildToolResultBlock(toolName, toolCallID, toolContent))
						}
					}
					joined := strings.Join(parts, "\n")
					if joined != "" {
						result = append(result, map[string]any{
							"role":    "user",
							"content": joined,
						})
					}
					continue
				}
			}

			content := extractContent(m["content"])

			if role == "assistant" {
				if tcs, ok := m["tool_calls"].([]any); ok && len(tcs) > 0 {
					assistantMsg := map[string]any{
						"role":    "assistant",
						"content": content,
					}
					if content == "" {
						assistantMsg["content"] = ""
					}
					stripped := make([]any, 0, len(tcs))
					for _, tc := range tcs {
						tcMap, ok := tc.(map[string]any)
						if !ok {
							continue
						}
						cp := make(map[string]any, len(tcMap))
						for k, v := range tcMap {
							if k == "index" {
								continue
							}
							cp[k] = v
						}
						stripped = append(stripped, cp)
					}
					assistantMsg["tool_calls"] = stripped
					result = append(result, assistantMsg)
					continue
				}

				if arr, ok := m["content"].([]any); ok {
					extractedToolCalls := make([]any, 0)
					for _, b := range arr {
						bMap, ok := b.(map[string]any)
						if !ok {
							continue
						}
						if t, _ := bMap["type"].(string); t != "tool_use" {
							continue
						}
						id := coalesceString(bMap["id"])
						if id == "" {
							continue
						}
						name := coalesceString(bMap["name"])
						if name == "" {
							name = "tool"
						}
						input := bMap["input"]
						if input == nil {
							input = map[string]any{}
						}
						args, err := json.Marshal(input)
						if err != nil {
							args = []byte("{}")
						}
						extractedToolCalls = append(extractedToolCalls, map[string]any{
							"id":   id,
							"type": "function",
							"function": map[string]any{
								"name":      name,
								"arguments": string(args),
							},
						})
					}
					if len(extractedToolCalls) > 0 {
						result = append(result, map[string]any{
							"role":       "assistant",
							"content":    content,
							"tool_calls": extractedToolCalls,
						})
						continue
					} else if content != "" {
						result = append(result, map[string]any{
							"role":    "assistant",
							"content": content,
						})
						continue
					}
				} else {
					if content != "" {
						result = append(result, map[string]any{
							"role":    msgRole(m),
							"content": content,
						})
					}
				}
			} else {
				if content != "" {
					result = append(result, map[string]any{
						"role":    msgRole(m),
						"content": content,
					})
				}
			}
		}
	}

	return result
}

func msgRole(m map[string]any) string {
	role, _ := m["role"].(string)
	return role
}
