package translation

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// geminiToOpenAIResponse converts a Gemini response chunk to OpenAI stream
// chunks. It may return zero, one, or many chunks per input chunk.
// Ported from response/gemini-to-openai.js:5-245 (row 043).
func geminiToOpenAIResponse(chunk map[string]any, state *StreamState) ([]map[string]any, error) {
	if chunk == nil {
		return nil, nil
	}

	// Wrapper tolerance (row 043: chunk.response || chunk).
	response := chunk
	if r, ok := chunk["response"].(map[string]any); ok && r != nil {
		response = r
	}

	candidatesRaw, ok := response["candidates"].([]any)
	if !ok || len(candidatesRaw) == 0 {
		return nil, nil
	}
	candidate, ok := candidatesRaw[0].(map[string]any)
	if !ok {
		return nil, nil
	}

	results := []map[string]any{}
	now := time.Now().Unix()

	// First chunk: state init + role chunk.
	if state.MessageID == "" {
		state.MessageID = fmt.Sprintf("msg_%d", time.Now().UnixMilli())
		if id, ok := response["responseId"].(string); ok && id != "" {
			state.MessageID = id
		}
		state.Model = "gemini"
		if mv, ok := response["modelVersion"].(string); ok && mv != "" {
			state.Model = mv
		}
		state.FunctionIndex = 0
		results = append(results, map[string]any{
			"id":      "chatcmpl-" + state.MessageID,
			"object":  "chat.completion.chunk",
			"created": now,
			"model":   state.Model,
			"choices": []any{map[string]any{
				"index":         0,
				"delta":         map[string]any{"role": "assistant"},
				"finish_reason": nil,
			}},
		})
	}

	// Process parts.
	if content, ok := candidate["content"].(map[string]any); ok {
		if partsRaw, ok := content["parts"].([]any); ok {
			for _, partRaw := range partsRaw {
				part, ok := partRaw.(map[string]any)
				if !ok {
					continue
				}

				_, hasThoughtSig := part["thoughtSignature"]
				if !hasThoughtSig {
					_, hasThoughtSig = part["thought_signature"]
				}
				isThought := part["thought"] == true

				if hasThoughtSig {
					text := ""
					if s, ok := part["text"].(string); ok {
						text = s
					}
					hasTextContent := text != ""
					hasFunctionCall := false
					if fc, ok := part["functionCall"].(map[string]any); ok && fc != nil {
						hasFunctionCall = true
					}

					if hasTextContent {
						delta := map[string]any{}
						if isThought {
							delta["reasoning_content"] = text
						} else {
							delta["content"] = text
						}
						results = append(results, map[string]any{
							"id":      "chatcmpl-" + state.MessageID,
							"object":  "chat.completion.chunk",
							"created": now,
							"model":   state.Model,
							"choices": []any{map[string]any{
								"index":         0,
								"delta":         delta,
								"finish_reason": nil,
							}},
						})
					}

					if hasFunctionCall {
						results = append(results, geminiFunctionCallToToolCall(part, state, now)...)
					}
					continue
				}

				// Plain text parts.
				if text, ok := part["text"].(string); ok && text != "" {
					results = append(results, map[string]any{
						"id":      "chatcmpl-" + state.MessageID,
						"object":  "chat.completion.chunk",
						"created": now,
						"model":   state.Model,
						"choices": []any{map[string]any{
							"index":         0,
							"delta":         map[string]any{"content": text},
							"finish_reason": nil,
						}},
					})
				}

				// Function call parts.
				if fc, ok := part["functionCall"].(map[string]any); ok && fc != nil {
					results = append(results, geminiFunctionCallToToolCall(part, state, now)...)
				}

				// Inline data (images).
				inlineData := part["inlineData"]
				if inlineData == nil {
					inlineData = part["inline_data"]
				}
				if idata, ok := inlineData.(map[string]any); ok {
					if data, ok := idata["data"].(string); ok && data != "" {
						mimeType := "image/png"
						if mt, ok := idata["mimeType"].(string); ok && mt != "" {
							mimeType = mt
						} else if mt, ok := idata["mime_type"].(string); ok && mt != "" {
							mimeType = mt
						}
						results = append(results, map[string]any{
							"id":      "chatcmpl-" + state.MessageID,
							"object":  "chat.completion.chunk",
							"created": now,
							"model":   state.Model,
							"choices": []any{map[string]any{
								"index": 0,
								"delta": map[string]any{
									"images": []any{map[string]any{
										"type": "image_url",
										"image_url": map[string]any{
											"url": fmt.Sprintf("data:%s;base64,%s", mimeType, data),
										},
									}},
								},
								"finish_reason": nil,
							}},
						})
					}
				}
			}
		}
	}

	// Usage metadata.
	usageMetaRaw := response["usageMetadata"]
	if usageMetaRaw == nil {
		usageMetaRaw = chunk["usageMetadata"]
	}
	if usageMeta, ok := usageMetaRaw.(map[string]any); ok && usageMeta != nil {
		cachedTokens := float64(0)
		if v, ok := usageMeta["cachedContentTokenCount"].(float64); ok {
			cachedTokens = v
		}
		promptTokenCountRaw := float64(0)
		if v, ok := usageMeta["promptTokenCount"].(float64); ok {
			promptTokenCountRaw = v
		}
		thoughtsTokens := float64(0)
		if v, ok := usageMeta["thoughtsTokenCount"].(float64); ok {
			thoughtsTokens = v
		}
		candidatesTokens := float64(0)
		if v, ok := usageMeta["candidatesTokenCount"].(float64); ok {
			candidatesTokens = v
		}
		totalTokens := float64(0)
		if v, ok := usageMeta["totalTokenCount"].(float64); ok {
			totalTokens = v
		}

		if candidatesTokens == 0 && totalTokens > 0 {
			candidatesTokens = totalTokens - promptTokenCountRaw - thoughtsTokens
			if candidatesTokens < 0 {
				candidatesTokens = 0
			}
		}

		completionTokens := candidatesTokens + thoughtsTokens

		usage := map[string]any{
			"prompt_tokens":     promptTokenCountRaw,
			"completion_tokens": completionTokens,
			"total_tokens":      totalTokens,
		}

		if cachedTokens > 0 {
			usage["prompt_tokens_details"] = map[string]any{
				"cached_tokens": cachedTokens,
			}
		}

		if thoughtsTokens > 0 {
			usage["completion_tokens_details"] = map[string]any{
				"reasoning_tokens": thoughtsTokens,
			}
		}

		state.Usage = usage
	}

	// Finish reason.
	if finishReasonRaw, ok := candidate["finishReason"].(string); ok && finishReasonRaw != "" {
		finishReason := strings.ToLower(finishReasonRaw)
		if finishReason == "stop" && len(state.ToolCalls) > 0 {
			finishReason = "tool_calls"
		}

		finalChunk := map[string]any{
			"id":      "chatcmpl-" + state.MessageID,
			"object":  "chat.completion.chunk",
			"created": now,
			"model":   state.Model,
			"choices": []any{map[string]any{
				"index":         0,
				"delta":         map[string]any{},
				"finish_reason": finishReason,
			}},
		}

		if state.Usage != nil {
			finalChunk["usage"] = state.Usage
		}

		results = append(results, finalChunk)
		state.FinishReason = finishReason
	}

	if len(results) == 0 {
		return nil, nil
	}
	return results, nil
}

func geminiFunctionCallToToolCall(part map[string]any, state *StreamState, now int64) []map[string]any {
	fc, ok := part["functionCall"].(map[string]any)
	if !ok || fc == nil {
		return nil
	}
	rawName := ""
	if s, ok := fc["name"].(string); ok {
		rawName = s
	}
	fcName := rawName
	if state.ToolNameMap != nil {
		if mapped, ok := state.ToolNameMap[rawName]; ok {
			fcName = mapped
		}
	}
	fcArgs := map[string]any{}
	if args, ok := fc["args"].(map[string]any); ok && args != nil {
		fcArgs = args
	}
	toolCallIndex := state.FunctionIndex
	state.FunctionIndex++

	argsJSON, _ := json.Marshal(fcArgs)

	toolCall := map[string]any{
		"id":    fmt.Sprintf("%s-%d-%d", fcName, time.Now().UnixMilli(), toolCallIndex),
		"index": toolCallIndex,
		"type":  "function",
		"function": map[string]any{
			"name":      fcName,
			"arguments": string(argsJSON),
		},
	}

	state.ToolCalls[toolCallIndex] = ToolCallInfo{
		ID:   toolCall["id"].(string),
		Name: fcName,
	}

	return []map[string]any{{
		"id":      "chatcmpl-" + state.MessageID,
		"object":  "chat.completion.chunk",
		"created": now,
		"model":   state.Model,
		"choices": []any{map[string]any{
			"index":         0,
			"delta":         map[string]any{"tool_calls": []any{toolCall}},
			"finish_reason": nil,
		}},
	}}
}
