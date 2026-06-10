package translation

import (
	"fmt"
	"time"
)

// claudeToOpenAIResponse converts a Claude-format stream chunk into OpenAI-
// shaped stream chunks.
func claudeToOpenAIResponse(chunk map[string]any, state *StreamState) ([]map[string]any, error) {
	if chunk == nil {
		return nil, nil
	}

	results := []map[string]any{}
	event, _ := chunk["type"].(string)

	switch event {
	case "message_start":
		msg, _ := chunk["message"].(map[string]any)
		state.MessageID = ""
		if id, ok := msg["id"].(string); ok && id != "" {
			state.MessageID = id
		}
		if state.MessageID == "" {
			state.MessageID = fmt.Sprintf("msg_%d", time.Now().UnixMilli())
		}
		state.Model = "unknown"
		if m, ok := msg["model"].(string); ok && m != "" {
			state.Model = m
		}
		state.ToolCallIndex = 0
		results = append(results, createOpenAIChunk(state, map[string]any{"role": "assistant"}, ""))

	case "content_block_start":
		block, _ := chunk["content_block"].(map[string]any)
		blockType, _ := block["type"].(string)
		index := 0
		if n, ok := chunk["index"].(float64); ok {
			index = int(n)
		} else if n, ok := chunk["index"].(int); ok {
			index = n
		}

		if blockType == "server_tool_use" {
			// Built-in tool (web search) - Claude handles internally, skip.
			state.ServerToolBlockIndex = index
			break
		}
		if blockType == "text" {
			state.TextBlockStarted = true
		} else if blockType == "thinking" {
			// PAR-PR-1084: do NOT emit the literal <think> content chunk.
			// Thinking text still flows as reasoning_content via thinking_delta.
			state.InThinkingBlock = true
			state.CurrentBlockIndex = index
		} else if blockType == "tool_use" {
			toolCallIndex := state.ToolCallIndex
			state.ToolCallIndex++
			toolName := ""
			if name, ok := block["name"].(string); ok {
				toolName = name
			}
			// Restore original tool name from mapping (Claude OAuth).
			if state.ToolNameMap != nil {
				if orig, ok := state.ToolNameMap[toolName]; ok {
					toolName = orig
				}
			}
			toolID := ""
			if id, ok := block["id"].(string); ok {
				toolID = id
			}
			state.ClaudeBlockTools[index] = claudeOpenAIToolCall{
				Index: toolCallIndex,
				ID:    toolID,
				Name:  toolName,
			}
			toolCall := map[string]any{
				"index": toolCallIndex,
				"id":    toolID,
				"type":  "function",
				"function": map[string]any{
					"name":      toolName,
					"arguments": "",
				},
			}
			results = append(results, createOpenAIChunk(state, map[string]any{"tool_calls": []any{toolCall}}, ""))
		}

	case "content_block_delta":
		index := 0
		if n, ok := chunk["index"].(float64); ok {
			index = int(n)
		} else if n, ok := chunk["index"].(int); ok {
			index = n
		}
		// Skip deltas for built-in server tool blocks.
		if index == state.ServerToolBlockIndex {
			break
		}
		delta, _ := chunk["delta"].(map[string]any)
		deltaType, _ := delta["type"].(string)

		if deltaType == "text_delta" {
			if text, ok := delta["text"].(string); ok && text != "" {
				results = append(results, createOpenAIChunk(state, map[string]any{"content": text}, ""))
			}
		} else if deltaType == "thinking_delta" {
			if thinking, ok := delta["thinking"].(string); ok && thinking != "" {
				results = append(results, createOpenAIChunk(state, map[string]any{"reasoning_content": thinking}, ""))
			}
		} else if deltaType == "input_json_delta" {
			if partialJSON, ok := delta["partial_json"].(string); ok && partialJSON != "" {
				if info, ok := state.ClaudeBlockTools[index]; ok {
					info.Arguments += partialJSON
					state.ClaudeBlockTools[index] = info
					results = append(results, createOpenAIChunk(state, map[string]any{
						"tool_calls": []any{map[string]any{
							"index": info.Index,
							"id":    info.ID,
							"function": map[string]any{
								"arguments": partialJSON,
							},
						}},
					}, ""))
				}
			}
		}

	case "content_block_stop":
		index := 0
		if n, ok := chunk["index"].(float64); ok {
			index = int(n)
		} else if n, ok := chunk["index"].(int); ok {
			index = n
		}
		// Skip stop for built-in server tool blocks.
		if index == state.ServerToolBlockIndex {
			state.ServerToolBlockIndex = -1
			break
		}
		if state.InThinkingBlock && index == state.CurrentBlockIndex {
			// PAR-PR-1084: do NOT emit the literal </think> content chunk.
			state.InThinkingBlock = false
		}
		state.TextBlockStarted = false

	case "message_delta":
		delta, _ := chunk["delta"].(map[string]any)
		usage, _ := chunk["usage"].(map[string]any)

		if usage != nil {
			inputTokens := 0
			if n, ok := usage["input_tokens"].(float64); ok {
				inputTokens = int(n)
			} else if n, ok := usage["input_tokens"].(int); ok {
				inputTokens = n
			}
			outputTokens := 0
			if n, ok := usage["output_tokens"].(float64); ok {
				outputTokens = int(n)
			} else if n, ok := usage["output_tokens"].(int); ok {
				outputTokens = n
			}
			cacheReadTokens := 0
			if n, ok := usage["cache_read_input_tokens"].(float64); ok {
				cacheReadTokens = int(n)
			} else if n, ok := usage["cache_read_input_tokens"].(int); ok {
				cacheReadTokens = n
			}
			cacheCreationTokens := 0
			if n, ok := usage["cache_creation_input_tokens"].(float64); ok {
				cacheCreationTokens = int(n)
			} else if n, ok := usage["cache_creation_input_tokens"].(int); ok {
				cacheCreationTokens = n
			}

			promptTokens := inputTokens + cacheReadTokens + cacheCreationTokens
			state.Usage = map[string]any{
				"prompt_tokens":     promptTokens,
				"completion_tokens": outputTokens,
				"total_tokens":      promptTokens + outputTokens,
				"input_tokens":      inputTokens,
				"output_tokens":     outputTokens,
			}
			if cacheReadTokens > 0 {
				state.Usage["cache_read_input_tokens"] = cacheReadTokens
			}
			if cacheCreationTokens > 0 {
				state.Usage["cache_creation_input_tokens"] = cacheCreationTokens
			}
		}

		if stopReason, ok := delta["stop_reason"].(string); ok && stopReason != "" {
			state.FinishReason = convertClaudeStopReason(stopReason)
			finalChunk := createOpenAIChunk(state, map[string]any{}, state.FinishReason)

			if state.Usage != nil {
				usageCopy := map[string]any{
					"prompt_tokens":     state.Usage["prompt_tokens"],
					"completion_tokens": state.Usage["completion_tokens"],
					"total_tokens":      state.Usage["total_tokens"],
				}
				cacheRead := 0
				if v, ok := state.Usage["cache_read_input_tokens"].(int); ok {
					cacheRead = v
				}
				cacheCreate := 0
				if v, ok := state.Usage["cache_creation_input_tokens"].(int); ok {
					cacheCreate = v
				}
				if cacheRead > 0 || cacheCreate > 0 {
					usageCopy["prompt_tokens_details"] = map[string]any{}
					if cacheRead > 0 {
						usageCopy["prompt_tokens_details"].(map[string]any)["cached_tokens"] = cacheRead
					}
					if cacheCreate > 0 {
						usageCopy["prompt_tokens_details"].(map[string]any)["cache_creation_tokens"] = cacheCreate
					}
				}
				finalChunk["usage"] = usageCopy
			}

			results = append(results, finalChunk)
			state.FinishReasonSent = true
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
			chunk := createOpenAIChunk(state, map[string]any{}, finishReason)
			if state.Usage != nil {
				// Same shape as the message_delta final chunk: prompt_tokens
				// already includes cache tokens, and cache accounting is
				// preserved via prompt_tokens_details (diff gate w1-d).
				usageCopy := map[string]any{
					"prompt_tokens":     state.Usage["prompt_tokens"],
					"completion_tokens": state.Usage["completion_tokens"],
					"total_tokens":      state.Usage["total_tokens"],
				}
				cacheRead := 0
				if v, ok := state.Usage["cache_read_input_tokens"].(int); ok {
					cacheRead = v
				}
				cacheCreate := 0
				if v, ok := state.Usage["cache_creation_input_tokens"].(int); ok {
					cacheCreate = v
				}
				if cacheRead > 0 || cacheCreate > 0 {
					details := map[string]any{}
					if cacheRead > 0 {
						details["cached_tokens"] = cacheRead
					}
					if cacheCreate > 0 {
						details["cache_creation_tokens"] = cacheCreate
					}
					usageCopy["prompt_tokens_details"] = details
				}
				chunk["usage"] = usageCopy
			}
			results = append(results, chunk)
			state.FinishReasonSent = true
		}
	}

	if len(results) == 0 {
		return nil, nil
	}
	return results, nil
}

func createOpenAIChunk(state *StreamState, delta map[string]any, finishReason string) map[string]any {
	choice := map[string]any{
		"index": 0,
		"delta": delta,
	}
	if finishReason != "" {
		choice["finish_reason"] = finishReason
	} else {
		choice["finish_reason"] = nil
	}
	choices := []any{choice}
	return map[string]any{
		"id":      fmt.Sprintf("chatcmpl-%s", state.MessageID),
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   state.Model,
		"choices": choices,
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
