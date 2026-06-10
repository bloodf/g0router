package translation

import (
	"encoding/json"
	"fmt"
	"strings"
)

func antigravityToOpenAIRequest(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error) {
	req := body
	if r, ok := body["request"].(map[string]any); ok && r != nil {
		req = r
	}

	result := map[string]any{
		"model":    model,
		"messages": []any{},
		"stream":   stream,
	}

	if genConfig, ok := req["generationConfig"].(map[string]any); ok && genConfig != nil {
		if maxOutputTokens, ok := genConfig["maxOutputTokens"]; ok && maxOutputTokens != nil {
			tempBody := map[string]any{
				"max_tokens": maxOutputTokens,
				"tools":      req["tools"],
			}
			result["max_tokens"] = AdjustMaxTokens(tempBody)
		}
		if temp, ok := genConfig["temperature"]; ok && temp != nil {
			result["temperature"] = temp
		}
		if topP, ok := genConfig["topP"]; ok && topP != nil {
			result["top_p"] = topP
		}
		if topK, ok := genConfig["topK"]; ok && topK != nil {
			result["top_k"] = topK
		}

		if thinkingConfig, ok := genConfig["thinkingConfig"].(map[string]any); ok && thinkingConfig != nil {
			budget := 0
			if b, ok := thinkingConfig["thinkingBudget"]; ok {
				switch v := b.(type) {
				case int:
					budget = v
				case float64:
					budget = int(v)
				}
			}
			if budget > 0 {
				if budget <= 2048 {
					result["reasoning_effort"] = "low"
				} else if budget <= 16384 {
					result["reasoning_effort"] = "medium"
				} else {
					result["reasoning_effort"] = "high"
				}
			}
		}
	}

	if sysInstr, ok := req["systemInstruction"]; ok && sysInstr != nil {
		systemText := extractTextAntigravity(sysInstr)
		if systemText != "" {
			result["messages"] = append(result["messages"].([]any), map[string]any{
				"role":    "system",
				"content": systemText,
			})
		}
	}

	if contents, ok := req["contents"].([]any); ok {
		for _, content := range contents {
			converted, err := convertAntigravityContent(content)
			if err != nil {
				return nil, fmt.Errorf("antigravityToOpenAIRequest: %w", err)
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

	if tools, ok := req["tools"].([]any); ok && len(tools) > 0 {
		result["tools"] = []any{}
		for _, tool := range tools {
			tm, ok := tool.(map[string]any)
			if !ok {
				continue
			}
			if fds, ok := tm["functionDeclarations"].([]any); ok {
				for _, fd := range fds {
					fdm, ok := fd.(map[string]any)
					if !ok {
						continue
					}
					name := ""
					if s, ok := fdm["name"].(string); ok {
						name = s
					}
					desc := ""
					if s, ok := fdm["description"].(string); ok {
						desc = s
					}
					params := map[string]any{"type": "object", "properties": map[string]any{}}
					if p, ok := fdm["parameters"].(map[string]any); ok && p != nil {
						params = normalizeSchemaTypes(p)
					}
					result["tools"] = append(result["tools"].([]any), map[string]any{
						"type": "function",
						"function": map[string]any{
							"name":        name,
							"description": desc,
							"parameters":  params,
						},
					})
				}
			}
		}
	}

	return result, nil
}

func extractTextAntigravity(instruction any) string {
	if s, ok := instruction.(string); ok {
		return s
	}
	if m, ok := instruction.(map[string]any); ok {
		if parts, ok := m["parts"].([]any); ok {
			var texts []string
			for _, p := range parts {
				if pm, ok := p.(map[string]any); ok {
					if text, ok := pm["text"].(string); ok {
						texts = append(texts, text)
					}
				}
			}
			return strings.Join(texts, "")
		}
	}
	return ""
}

func normalizeSchemaTypes(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}
	result := make(map[string]any, len(schema))
	for k, v := range schema {
		result[k] = v
	}

	if typ, ok := result["type"].(string); ok {
		result["type"] = strings.ToLower(typ)
	}

	delete(result, "enumDescriptions")

	if props, ok := result["properties"].(map[string]any); ok {
		normalized := make(map[string]any, len(props))
		for key, val := range props {
			if vm, ok := val.(map[string]any); ok {
				normalized[key] = normalizeSchemaTypes(vm)
			} else {
				normalized[key] = val
			}
		}
		result["properties"] = normalized
	}

	if items, ok := result["items"].(map[string]any); ok {
		result["items"] = normalizeSchemaTypes(items)
	}

	return result
}

func convertAntigravityContent(content any) (any, error) {
	m, ok := content.(map[string]any)
	if !ok {
		return nil, nil
	}

	role := "user"
	if r, ok := m["role"].(string); ok {
		switch r {
		case "model":
			role = "assistant"
		case "user":
			role = "user"
		}
	}

	partsRaw, ok := m["parts"].([]any)
	if !ok {
		return nil, nil
	}

	textParts := []map[string]any{}
	toolCalls := []map[string]any{}
	toolResults := []map[string]any{}
	reasoningContent := ""

	for _, partRaw := range partsRaw {
		part, ok := partRaw.(map[string]any)
		if !ok {
			continue
		}

		if part["thought"] == true {
			if text, ok := part["text"].(string); ok && text != "" {
				reasoningContent += text
			}
			continue
		}

		if _, ok := part["thoughtSignature"]; ok {
			if text, ok := part["text"].(string); ok && text != "" {
				textParts = append(textParts, map[string]any{"type": "text", "text": text})
			}
			continue
		}

		if text, ok := part["text"].(string); ok {
			textParts = append(textParts, map[string]any{"type": "text", "text": text})
		}

		if inlineData, ok := part["inlineData"].(map[string]any); ok {
			mimeType := "image/png"
			if mt, ok := inlineData["mimeType"].(string); ok && mt != "" {
				mimeType = mt
			} else if mt, ok := inlineData["mime_type"].(string); ok && mt != "" {
				mimeType = mt
			}
			if data, ok := inlineData["data"].(string); ok && data != "" {
				textParts = append(textParts, map[string]any{
					"type": "image_url",
					"image_url": map[string]any{
						"url": fmt.Sprintf("data:%s;base64,%s", mimeType, data),
					},
				})
			}
		}

		if fc, ok := part["functionCall"].(map[string]any); ok {
			id := ""
			if s, ok := fc["id"].(string); ok {
				id = s
			}
			name := ""
			if s, ok := fc["name"].(string); ok {
				name = s
			}
			args := map[string]any{}
			if a, ok := fc["args"].(map[string]any); ok {
				args = a
			}
			argsJSON, err := json.Marshal(args)
			if err != nil {
				return nil, fmt.Errorf("marshal functionCall args: %w", err)
			}
			toolCalls = append(toolCalls, map[string]any{
				"id":   id,
				"type": "function",
				"function": map[string]any{
					"name":      name,
					"arguments": string(argsJSON),
				},
			})
		}

		if fr, ok := part["functionResponse"].(map[string]any); ok {
			id := ""
			if s, ok := fr["id"].(string); ok {
				id = s
			}
			if id == "" {
				if s, ok := fr["name"].(string); ok {
					id = s
				}
			}
			resp := map[string]any{}
			if r, ok := fr["response"].(map[string]any); ok {
				resp = r
			}
			resultVal := resp["result"]
			if resultVal == nil {
				resultVal = resp
			}
			contentJSON, err := json.Marshal(resultVal)
			if err != nil {
				return nil, fmt.Errorf("marshal functionResponse result: %w", err)
			}
			toolResults = append(toolResults, map[string]any{
				"role":         "tool",
				"tool_call_id": id,
				"content":      string(contentJSON),
			})
		}
	}

	if len(toolResults) > 0 {
		results := make([]any, len(toolResults))
		for i, r := range toolResults {
			results[i] = r
		}
		return results, nil
	}

	if len(toolCalls) > 0 {
		msg := map[string]any{
			"role":       "assistant",
			"tool_calls": toAnySlice(toolCalls),
		}
		if len(textParts) > 0 {
			if len(textParts) == 1 {
				msg["content"] = textParts[0]["text"]
			} else {
				msg["content"] = toAnySlice(textParts)
			}
		}
		if reasoningContent != "" {
			msg["reasoning_content"] = reasoningContent
		}
		return msg, nil
	}

	if len(textParts) > 0 || reasoningContent != "" {
		msg := map[string]any{
			"role": role,
		}
		if len(textParts) > 0 {
			if len(textParts) == 1 {
				msg["content"] = textParts[0]["text"]
			} else {
				msg["content"] = toAnySlice(textParts)
			}
		}
		if reasoningContent != "" {
			msg["reasoning_content"] = reasoningContent
		}
		return msg, nil
	}

	return nil, nil
}
