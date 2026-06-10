package translation

import (
	"fmt"
	"time"
)

// responsesToOpenAIResponse converts OpenAI Responses API events into OpenAI
// Chat Completions streaming chunks.
func responsesToOpenAIResponse(chunk map[string]any, state *StreamState) ([]map[string]any, error) {
	if chunk == nil {
		// Flush: send final chunk with finish_reason
		if state.FinishReasonSent || !state.ResponsesStarted {
			return nil, nil
		}

		finishReason := computeResponsesFinishReason(state)
		state.FinishReasonSent = true
		state.FinishReason = finishReason

		finalChunk := createResponsesOpenAIChunk(state, map[string]any{}, finishReason)
		if state.Usage != nil {
			finalChunk["usage"] = state.Usage
		}
		return []map[string]any{finalChunk}, nil
	}

	// Handle different event types from Responses API
	eventType := ""
	if t, ok := chunk["type"].(string); ok && t != "" {
		eventType = t
	} else if t, ok := chunk["event"].(string); ok && t != "" {
		eventType = t
	}

	data := chunk
	if d, ok := chunk["data"].(map[string]any); ok {
		data = d
	}

	// Initialize state
	if !state.ResponsesStarted {
		state.ResponsesStarted = true
		state.ResponsesChatID = fmt.Sprintf("chatcmpl-%d", time.Now().UnixMilli())
		state.ResponsesCreated = time.Now().Unix()
		state.ResponsesToolCallIndex = 0
		state.ResponsesCurrentToolCallID = ""
	}

	// Text content delta
	if eventType == "response.output_text.delta" {
		delta := ""
		if d, ok := data["delta"].(string); ok {
			delta = d
		}
		if delta == "" {
			return nil, nil
		}
		return []map[string]any{createResponsesOpenAIChunk(state, map[string]any{"content": delta}, "")}, nil
	}

	// Text content done (ignore, we handle via delta)
	if eventType == "response.output_text.done" {
		return nil, nil
	}

	// Function call started (standard function_call or custom_tool_call)
	if eventType == "response.output_item.added" {
		item, _ := data["item"].(map[string]any)
		itemType := ""
		if t, ok := item["type"].(string); ok {
			itemType = t
		}
		if itemType == "function_call" || itemType == "custom_tool_call" {
			callID := ""
			if c, ok := item["call_id"].(string); ok && c != "" {
				callID = c
			} else {
				callID = fmt.Sprintf("call_%d", time.Now().UnixMilli())
			}
			state.ResponsesCurrentToolCallID = callID
			name := ""
			if n, ok := item["name"].(string); ok {
				name = n
			}
			return []map[string]any{createResponsesOpenAIChunk(state, map[string]any{
				"tool_calls": []any{
					map[string]any{
						"index": state.ResponsesToolCallIndex,
						"id":    callID,
						"type":  "function",
						"function": map[string]any{
							"name":      name,
							"arguments": "",
						},
					},
				},
			}, "")}, nil
		}
	}

	// Function call arguments delta (standard or custom_tool_call variant)
	if eventType == "response.function_call_arguments.delta" || eventType == "response.custom_tool_call_input.delta" {
		argsDelta := ""
		if d, ok := data["delta"].(string); ok {
			argsDelta = d
		}
		if argsDelta == "" {
			return nil, nil
		}
		return []map[string]any{createResponsesOpenAIChunk(state, map[string]any{
			"tool_calls": []any{
				map[string]any{
					"index": state.ResponsesToolCallIndex,
					"function": map[string]any{
						"arguments": argsDelta,
					},
				},
			},
		}, "")}, nil
	}

	// Function call done (standard or custom_tool_call variant)
	if eventType == "response.output_item.done" {
		item, _ := data["item"].(map[string]any)
		itemType := ""
		if t, ok := item["type"].(string); ok {
			itemType = t
		}
		if itemType == "function_call" || itemType == "custom_tool_call" {
			state.ResponsesToolCallIndex++
			return nil, nil
		}
	}

	// Response completed
	if eventType == "response.completed" || eventType == "response.done" {
		// Extract usage from response.completed event
		responseUsage := extractNestedMap(data, "response", "usage")
		if responseUsage != nil {
			inputTokens := extractInt(responseUsage, "input_tokens")
			if inputTokens == 0 {
				inputTokens = extractInt(responseUsage, "prompt_tokens")
			}
			outputTokens := extractInt(responseUsage, "output_tokens")
			if outputTokens == 0 {
				outputTokens = extractInt(responseUsage, "completion_tokens")
			}

			cacheReadTokens := 0
			if details, ok := responseUsage["input_tokens_details"].(map[string]any); ok {
				cacheReadTokens = extractInt(details, "cached_tokens")
			}
			if cacheReadTokens == 0 {
				cacheReadTokens = extractInt(responseUsage, "cache_read_input_tokens")
			}

			state.Usage = map[string]any{
				"prompt_tokens":     inputTokens,
				"completion_tokens": outputTokens,
				"total_tokens":      inputTokens + outputTokens,
			}
			if cacheReadTokens > 0 {
				state.Usage["prompt_tokens_details"] = map[string]any{
					"cached_tokens": cacheReadTokens,
				}
			}
		}

		if !state.FinishReasonSent {
			finishReason := computeResponsesFinishReason(state)
			state.FinishReasonSent = true
			state.FinishReason = finishReason

			finalChunk := createResponsesOpenAIChunk(state, map[string]any{}, finishReason)
			if state.Usage != nil {
				finalChunk["usage"] = state.Usage
			}
			return []map[string]any{finalChunk}, nil
		}
		return nil, nil
	}

	// Error events from Responses API (e.g. model_not_found)
	if eventType == "error" || eventType == "response.failed" {
		// Avoid emitting duplicate errors
		if state.FinishReasonSent {
			return nil, nil
		}

		var errObj map[string]any
		if e, ok := data["error"].(map[string]any); ok {
			errObj = e
		} else if resp, ok := data["response"].(map[string]any); ok {
			if e, ok := resp["error"].(map[string]any); ok {
				errObj = e
			}
		}
		if errObj != nil {
			msg := ""
			if m, ok := errObj["message"].(string); ok {
				msg = m
			} else {
				msg = fmt.Sprintf("%v", errObj)
			}
			state.FinishReasonSent = true
			return []map[string]any{createResponsesOpenAIChunk(state, map[string]any{
				"content": fmt.Sprintf("[Error] %s", msg),
			}, "stop")}, nil
		}
		return nil, nil
	}

	// Reasoning summary delta → emit as reasoning_content for client thinking display
	if eventType == "response.reasoning_summary_text.delta" {
		delta := ""
		if d, ok := data["delta"].(string); ok {
			delta = d
		}
		if delta == "" {
			return nil, nil
		}
		return []map[string]any{createResponsesOpenAIChunk(state, map[string]any{
			"reasoning_content": delta,
		}, "")}, nil
	}

	// Ignore other events
	return nil, nil
}

func computeResponsesFinishReason(state *StreamState) string {
	if state.ResponsesToolCallIndex > 0 || state.ResponsesCurrentToolCallID != "" {
		return "tool_calls"
	}
	return "stop"
}

func createResponsesOpenAIChunk(state *StreamState, delta map[string]any, finishReason string) map[string]any {
	choice := map[string]any{
		"index": 0,
		"delta": delta,
	}
	if finishReason != "" {
		choice["finish_reason"] = finishReason
	} else {
		choice["finish_reason"] = nil
	}
	return map[string]any{
		"id":      state.ResponsesChatID,
		"object":  "chat.completion.chunk",
		"created": state.ResponsesCreated,
		"model":   "unknown",
		"choices": []any{choice},
	}
}

func extractInt(m map[string]any, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	if v, ok := m[key].(int); ok {
		return v
	}
	return 0
}

func extractNestedMap(m map[string]any, keys ...string) map[string]any {
	current := m
	for _, key := range keys {
		next, ok := current[key].(map[string]any)
		if !ok {
			return nil
		}
		current = next
	}
	return current
}
