package translation

import (
	"strings"
)

// defaultThinkingAGSignature is the default thinking signature used when no
// signature is provided from the thinking store.
// Ported from defaultThinkingSignature.js.
const defaultThinkingAGSignature = "EuwGCukGAXLI2nxwZIq54WWSoL/YN0P3TsDZ7zRnLi8g0S4aVr2HUGxvaHKySuY6HAVzcE0GPGjXrytLIldxthSvfxgUlJh6Qa9Z+Oj5QZBlYdg6HaJ6yuY5R7waE6rdwBsRf7Ft2j3DJ9rMi9qhWFqApewYtPhls3VHtuvND3l8Rm09+lbAXQs6KKWEWrxNLKTBkfpMgXhRERc/TQRMZu1twAablm6/Zk1tsYRvfWKLsNbeKF+CCojJdXJKvnR/8Ouuoa+Y2Ti20hcW7aZIIjZDFYPU//k6Ybmhg69J/imbFai2ckhfLaisqdDkdoIiBJScTOUvYqP6AE9d4MsydSC+UlhIMk4hoP76R8vUSCZRMkjOaDXstf/QoVZKbt94wyRZgAJ1G0BqI8L5ow86kLpA4wJEtxsRGymOE4bKUvApveBakYDNM9APkf+LbtbzWSseGjoZcSlycF9iN8Q2XNYKRrHbv3Lr5Y8JjdH/5y/6SHkNehTEZugaeGnSPSyCTWto1kQgHpxdWmhkLfJGNUGLmue7Mesj4TSms4J33mRpYVhNB/J333FCqIP0hr/E7BkkjEn7yZ4X7SQlh+xKPurapsnHRwiKmtsilmEFrnTE9iQr+pMr6M29qqFNv1tr5yumbaJw8JW9sB15tNsRv+dW6BjNanbsKz7HCgKUBc8tGy+7YuhXzAfViyRefcjK7eZW0Fbyt7AbybJTKz78W8NH7ye6LAwzOebXpeZ4D43fNIt8bKh26qgduSQv/7o+pAflkuqHZ99YWgHQ8h8OkZFi3eOiSYjsjhdZ/czWOdoPI/OnqIldzMPF5YlrKBLFX8VhRKVmqgsmWf5PHGulHhMkVlS+XG2UIseGy69ARa93D78Gsa+1n1kJr7EEB7Rh+27vUMxVYLdz1yMSvE5nalTAlg/ZeG8+XQ0cHuAI3KbQpHW2Q++RdXfm5JzD5WdJZUU+Zn8t8UUn85BH4RxZLeE0qJikgSsKoYVBc6YhiMjhPgkR95ReimY4Z0xCJdRo1gjexOFeODZMpQF6Yxnoic7IrdgsFA3iePTbFnPp3IAM1fAThWhXJUn3QInUOTd5o1qmTmn6REbL15g/JQNl+dqUoPkhleeb2V3kjqp1okmO3wMZbPknR3S1LZNmlS72/iBQUm+n2b/RCn4PjmM2"

// openaiToGeminiRequest converts an OpenAI-format request body to a Gemini-
// shaped request body.
func openaiToGeminiRequest(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error) {
	return openaiToGeminiBase(model, body, stream, defaultThinkingAGSignature, credentials)
}

func openaiToGeminiBase(model string, body map[string]any, stream bool, signature string, credentials map[string]any) (map[string]any, error) {
	result := map[string]any{
		"model":            model,
		"contents":         []any{},
		"generationConfig": map[string]any{},
		"safetySettings":   defaultSafetySettings(),
	}

	// Generation config (row 023).
	if temp, ok := body["temperature"]; ok && temp != nil {
		result["generationConfig"].(map[string]any)["temperature"] = temp
	}
	if topP, ok := body["top_p"]; ok && topP != nil {
		result["generationConfig"].(map[string]any)["topP"] = topP
	}
	if topK, ok := body["top_k"]; ok && topK != nil {
		result["generationConfig"].(map[string]any)["topK"] = topK
	}
	if maxTokens, ok := body["max_tokens"]; ok && maxTokens != nil {
		result["generationConfig"].(map[string]any)["maxOutputTokens"] = maxTokens
	}

	// Build tool_call_id -> name map from assistant tool_calls.
	tcID2Name := make(map[string]string)
	if rawMsgs, ok := body["messages"].([]any); ok {
		for _, msg := range rawMsgs {
			m, ok := msg.(map[string]any)
			if !ok {
				continue
			}
			if role, _ := m["role"].(string); role == "assistant" {
				if rawToolCalls, ok := m["tool_calls"].([]any); ok {
					for _, tc := range rawToolCalls {
						tcMap, ok := tc.(map[string]any)
						if !ok {
							continue
						}
						if tcType, _ := tcMap["type"].(string); tcType != "function" {
							continue
						}
						var fn map[string]any
						if f, ok := tcMap["function"].(map[string]any); ok {
							fn = f
						}
						id := ""
						if s, ok := tcMap["id"].(string); ok {
							id = s
						}
						name := ""
						if s, ok := fn["name"].(string); ok {
							name = s
						}
						if id != "" && name != "" {
							tcID2Name[id] = name
						}
					}
				}
			}
		}
	}

	// Build tool responses cache from role:tool messages.
	toolResponses := make(map[string]any)
	if rawMsgs, ok := body["messages"].([]any); ok {
		for _, msg := range rawMsgs {
			m, ok := msg.(map[string]any)
			if !ok {
				continue
			}
			if role, _ := m["role"].(string); role == "tool" {
				if tcID, ok := m["tool_call_id"].(string); ok {
					toolResponses[tcID] = m["content"]
				}
			}
		}
	}

	// Convert messages (rows 024-026).
	if rawMsgs, ok := body["messages"].([]any); ok {
		for _, msg := range rawMsgs {
			m, ok := msg.(map[string]any)
			if !ok {
				continue
			}
			role, _ := m["role"].(string)
			content := m["content"]

			if role == "system" && len(rawMsgs) > 1 {
				text := extractTextContentGemini(content)
				result["systemInstruction"] = map[string]any{
					"role":  "user",
					"parts": []any{map[string]any{"text": text}},
				}
			} else if role == "user" || (role == "system" && len(rawMsgs) == 1) {
				parts := convertOpenAIContentToParts(content)
				if len(parts) > 0 {
					result["contents"] = append(result["contents"].([]any), map[string]any{
						"role":  "user",
						"parts": toAnySlice(parts),
					})
				}
			} else if role == "assistant" {
				parts := []map[string]any{}

				// Row 025: reasoning_content → thought part + signature separator.
				if reasoningContent, ok := m["reasoning_content"].(string); ok && reasoningContent != "" {
					parts = append(parts, map[string]any{"thought": true, "text": reasoningContent})
					parts = append(parts, map[string]any{"thoughtSignature": signature, "text": ""})
				}

				if content != nil {
					text := extractTextContentGemini(content)
					if text != "" {
						parts = append(parts, map[string]any{"text": text})
					}
				}

				if rawToolCalls, ok := m["tool_calls"].([]any); ok {
					toolCallIDs := []string{}
					for _, tc := range rawToolCalls {
						tcMap, ok := tc.(map[string]any)
						if !ok {
							continue
						}
						if tcType, _ := tcMap["type"].(string); tcType != "function" {
							continue
						}
						var fn map[string]any
						if f, ok := tcMap["function"].(map[string]any); ok {
							fn = f
						}
						id := ""
						if s, ok := tcMap["id"].(string); ok {
							id = s
						}
						name := ""
						if s, ok := fn["name"].(string); ok {
							name = s
						}
						args := "{}"
						if s, ok := fn["arguments"].(string); ok {
							args = s
						}
						parsedArgs := tryParseJSONValue(args)
						if parsedArgs == nil {
							parsedArgs = map[string]any{}
						}
						parts = append(parts, map[string]any{
							"thoughtSignature": signature,
							"functionCall": map[string]any{
								"id":   id,
								"name": sanitizeGeminiFunctionName(name),
								"args": parsedArgs,
							},
						})
						toolCallIDs = append(toolCallIDs, id)
					}

					if len(parts) > 0 {
						result["contents"] = append(result["contents"].([]any), map[string]any{
							"role":  "model",
							"parts": toAnySlice(parts),
						})
					}

					// Row 026: emit functionResponse parts when tool responses exist.
					hasActualResponses := false
					for _, fid := range toolCallIDs {
						if _, ok := toolResponses[fid]; ok {
							hasActualResponses = true
							break
						}
					}

					if hasActualResponses {
						toolParts := []map[string]any{}
						for _, fid := range toolCallIDs {
							resp, ok := toolResponses[fid]
							if !ok {
								continue
							}

							name := tcID2Name[fid]
							if name == "" {
								idParts := strings.Split(fid, "-")
								if len(idParts) > 2 {
									name = strings.Join(idParts[:len(idParts)-2], "-")
								} else {
									name = fid
								}
							}

							parsedResp := tryParseJSONValue(resp)
							if parsedResp == nil {
								parsedResp = map[string]any{"result": resp}
							} else if _, isObj := parsedResp.(map[string]any); !isObj {
								parsedResp = map[string]any{"result": parsedResp}
							}

							toolParts = append(toolParts, map[string]any{
								"functionResponse": map[string]any{
									"id":   fid,
									"name": sanitizeGeminiFunctionName(name),
									"response": map[string]any{
										"result": parsedResp,
									},
								},
							})
						}
						if len(toolParts) > 0 {
							result["contents"] = append(result["contents"].([]any), map[string]any{
								"role":  "user",
								"parts": toAnySlice(toolParts),
							})
						}
					}
				} else if len(parts) > 0 {
					result["contents"] = append(result["contents"].([]any), map[string]any{
						"role":  "model",
						"parts": toAnySlice(parts),
					})
				}
			}
		}
	}

	// Convert tools.
	if rawTools, ok := body["tools"].([]any); ok && len(rawTools) > 0 {
		functionDeclarations := []map[string]any{}
		for _, tool := range rawTools {
			toolMap, ok := tool.(map[string]any)
			if !ok {
				continue
			}

			// Claude format: {name, description, input_schema}
			if toolName, ok := toolMap["name"].(string); ok && toolName != "" {
				var schema map[string]any
				if is, ok := toolMap["input_schema"].(map[string]any); ok {
					schema = deepCopyMap(is)
				} else {
					schema = map[string]any{"type": "object", "properties": map[string]any{}}
				}
				cleanJSONSchemaForGemini(schema)
				desc := ""
				if d, ok := toolMap["description"].(string); ok {
					desc = d
				}
				functionDeclarations = append(functionDeclarations, map[string]any{
					"name":        sanitizeGeminiFunctionName(toolName),
					"description": desc,
					"parameters":  schema,
				})
			} else if toolType, _ := toolMap["type"].(string); toolType == "function" {
				// OpenAI format: {type, function: {name, description, parameters}}
				var fn map[string]any
				if f, ok := toolMap["function"].(map[string]any); ok {
					fn = f
				}
				if fn == nil {
					continue
				}
				fnName := ""
				if s, ok := fn["name"].(string); ok {
					fnName = s
				}
				var schema map[string]any
				if params, ok := fn["parameters"].(map[string]any); ok {
					schema = deepCopyMap(params)
				} else {
					schema = map[string]any{"type": "object", "properties": map[string]any{}}
				}
				cleanJSONSchemaForGemini(schema)
				desc := ""
				if d, ok := fn["description"].(string); ok {
					desc = d
				}
				functionDeclarations = append(functionDeclarations, map[string]any{
					"name":        sanitizeGeminiFunctionName(fnName),
					"description": desc,
					"parameters":  schema,
				})
			}
		}

		if len(functionDeclarations) > 0 {
			fdAny := make([]any, len(functionDeclarations))
			for i, fd := range functionDeclarations {
				fdAny[i] = fd
			}
			result["tools"] = []any{map[string]any{"functionDeclarations": fdAny}}
		}
	}

	return result, nil
}

func toAnySlice(m []map[string]any) []any {
	if len(m) == 0 {
		return nil
	}
	result := make([]any, len(m))
	for i, v := range m {
		result[i] = v
	}
	return result
}

func deepCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case map[string]any:
			result[k] = deepCopyMap(val)
		case []any:
			result[k] = deepCopySlice(val)
		default:
			result[k] = val
		}
	}
	return result
}

func deepCopySlice(s []any) []any {
	if s == nil {
		return nil
	}
	result := make([]any, len(s))
	for i, v := range s {
		switch val := v.(type) {
		case map[string]any:
			result[i] = deepCopyMap(val)
		case []any:
			result[i] = deepCopySlice(val)
		default:
			result[i] = val
		}
	}
	return result
}
