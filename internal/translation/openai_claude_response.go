package translation

import (
	"fmt"
	"time"
)

// claudeToOpenAIResponse converts a Claude-format stream chunk into OpenAI-
// shaped stream chunks. PAR-PR-1084: no literal <think> markers.
func claudeToOpenAIResponse(chunk map[string]any, state *StreamState) ([]map[string]any, error) {
	if chunk == nil {
		return nil, nil
	}

	event, _ := chunk["type"].(string)
	results := make([]map[string]any, 0)

	switch event {
	case "message_start":
		msg, _ := chunk["message"].(map[string]any)
		if id, ok := msg["id"].(string); ok && id != "" {
			state.MessageID = id
		} else {
			state.MessageID = fmt.Sprintf("msg_%d", time.Now().UnixMilli())
		}
		if model, ok := msg["model"].(string); ok {
			state.Model = model
		}
		state.ToolCallIndex = 0
		results = append(results, createOpenAIChunk(state, map[string]any{"role": "assistant"}, nil))

	case "content_block_start":
		block, _ := chunk["content_block"].(map[string]any)
		blockType, _ := block["type"].(string)
		blockIndex := int(numberValue(chunk, "index"))

		switch blockType {
		case "server_tool_use":
			state.ServerToolBlockIndex = blockIndex
		case "text":
			state.TextBlockStarted = true
		case "thinking":
			// PAR-PR-1084: do not emit <think> marker chunks.
			state.InThinkingBlock = true
			state.CurrentBlockIndex = blockIndex
		case "tool_use":
			toolCallIndex := state.ToolCallIndex
			state.ToolCallIndex++
			name, _ := block["name"].(string)
			if state.ToolNameMap != nil {
				if mapped, ok := state.ToolNameMap[name]; ok {
					name = mapped
				}
			}
			id, _ := block["id"].(string)
			toolCall := claudeOpenAIToolCall{
				Index: toolCallIndex,
				ID:    id,
				Name:  name,
			}
			state.ClaudeBlockTools[blockIndex] = toolCall
			results = append(results, createOpenAIChunk(state, map[string]any{
				"tool_calls": []any{map[string]any{
					"index": toolCallIndex,
					"id":    id,
					"type":  "function",
					"function": map[string]any{
						"name":      name,
						"arguments": "",
					},
				}},
			}, nil))
		}

	case "content_block_delta":
		blockIndex := int(numberValue(chunk, "index"))
		if blockIndex == state.ServerToolBlockIndex {
			break
		}
		delta, _ := chunk["delta"].(map[string]any)
		deltaType, _ := delta["type"].(string)
		switch deltaType {
		case "text_delta":
			if text, ok := delta["text"].(string); ok && text != "" {
				results = append(results, createOpenAIChunk(state, map[string]any{"content": text}, nil))
			}
		case "thinking_delta":
			if thinking, ok := delta["thinking"].(string); ok && thinking != "" {
				results = append(results, createOpenAIChunk(state, map[string]any{"reasoning_content": thinking}, nil))
			}
		case "input_json_delta":
			partial, _ := delta["partial_json"].(string)
			if tc, ok := state.ClaudeBlockTools[blockIndex]; ok && partial != "" {
				tc.Arguments += partial
				state.ClaudeBlockTools[blockIndex] = tc
				results = append(results, createOpenAIChunk(state, map[string]any{
					"tool_calls": []any{map[string]any{
						"index": tc.Index,
						"id":    tc.ID,
						"function": map[string]any{
							"arguments": partial,
						},
					}},
				}, nil))
			}
		}

	case "content_block_stop":
		blockIndex := int(numberValue(chunk, "index"))
		if blockIndex == state.ServerToolBlockIndex {
			state.ServerToolBlockIndex = -1
			break
		}
		if state.InThinkingBlock && blockIndex == state.CurrentBlockIndex {
			// PAR-PR-1084: no </think> emission.
			state.InThinkingBlock = false
		}
		state.TextBlockStarted = false
		state.ThinkingBlockStarted = false

	case "message_delta":
		if usageRaw, ok := chunk["usage"].(map[string]any); ok {
			inputTokens := int(numberValue(usageRaw, "input_tokens"))
			outputTokens := int(numberValue(usageRaw, "output_tokens"))
			cacheRead := int(numberValue(usageRaw, "cache_read_input_tokens"))
			cacheCreate := int(numberValue(usageRaw, "cache_creation_input_tokens"))
			promptTokens := inputTokens + cacheRead + cacheCreate
			state.Usage = map[string]any{
				"prompt_tokens":     promptTokens,
				"completion_tokens": outputTokens,
				"total_tokens":      promptTokens + outputTokens,
				"input_tokens":      inputTokens,
				"output_tokens":     outputTokens,
			}
			if cacheRead > 0 {
				state.Usage["cache_read_input_tokens"] = cacheRead
			}
			if cacheCreate > 0 {
				state.Usage["cache_creation_input_tokens"] = cacheCreate
			}
		}
		if delta, ok := chunk["delta"].(map[string]any); ok {
			if stopReason, ok := delta["stop_reason"].(string); ok && stopReason != "" {
				state.FinishReason = convertClaudeStopReason(stopReason)
				final := createOpenAIChunk(state, map[string]any{}, &state.FinishReason)
				if state.Usage != nil {
					usage := map[string]any{
						"prompt_tokens":     state.Usage["prompt_tokens"],
						"completion_tokens": state.Usage["completion_tokens"],
						"total_tokens":      state.Usage["total_tokens"],
					}
					cacheRead := int(numberValue(state.Usage, "cache_read_input_tokens"))
					cacheCreate := int(numberValue(state.Usage, "cache_creation_input_tokens"))
					if cacheRead > 0 || cacheCreate > 0 {
						details := map[string]any{}
						if cacheRead > 0 {
							details["cached_tokens"] = cacheRead
						}
						if cacheCreate > 0 {
							details["cache_creation_tokens"] = cacheCreate
						}
						usage["prompt_tokens_details"] = details
					}
					final["usage"] = usage
				}
				results = append(results, final)
				state.FinishReasonSent = true
			}
		}

	case "message_stop":
		if !state.FinishReasonSent {
			finishReason := state.FinishReason
			if finishReason == "" {
				if len(state.ClaudeBlockTools) > 0 {
					finishReason = "tool_calls"
				} else {
					finishReason = "stop"
				}
			}
			final := createOpenAIChunk(state, map[string]any{}, &finishReason)
			if state.Usage != nil {
				inputTokens := int(numberValue(state.Usage, "input_tokens"))
				outputTokens := int(numberValue(state.Usage, "output_tokens"))
				final["usage"] = map[string]any{
					"prompt_tokens":     inputTokens,
					"completion_tokens": outputTokens,
					"total_tokens":      inputTokens + outputTokens,
				}
			}
			results = append(results, final)
			state.FinishReasonSent = true
		}
	}

	if len(results) == 0 {
		return nil, nil
	}
	return results, nil
}

func createOpenAIChunk(state *StreamState, delta map[string]any, finishReason *string) map[string]any {
	choice := map[string]any{
		"index": 0,
		"delta": delta,
	}
	if finishReason != nil {
		choice["finish_reason"] = *finishReason
	} else {
		choice["finish_reason"] = nil
	}
	return map[string]any{
		"id":      fmt.Sprintf("chatcmpl-%s", state.MessageID),
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   state.Model,
		"choices": []any{choice},
	}
}

func convertClaudeStopReason(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "tool_use":
		return "tool_calls"
	case "stop_sequence":
		return "stop"
	default:
		return "stop"
	}
}
