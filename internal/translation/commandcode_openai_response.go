package translation

import (
	"encoding/json"
	"fmt"
	"time"
)

// commandcodeToOpenAIResponse converts a CommandCode-format stream event into
// OpenAI-shaped stream chunks. Port of
// open-sse/translator/response/commandcode-to-openai.js.
//
// NOTE: the raw-string / "data:" framing tolerance from the ref (:66-79) is
// NOT ported — g0router's NDJSON scanner (w1-c) delivers parsed maps.
func commandcodeToOpenAIResponse(chunk map[string]any, state *StreamState) ([]map[string]any, error) {
	if chunk == nil {
		return nil, nil
	}

	// Already-OpenAI chunk: pass through.
	if obj, ok := chunk["object"].(string); ok && obj == "chat.completion.chunk" {
		return []map[string]any{chunk}, nil
	}

	eventType, _ := chunk["type"].(string)
	if eventType == "" {
		return nil, nil
	}

	ensureCommandCodeState(state, chunk)
	out := make([]map[string]any, 0)

	switch eventType {
	case "text-delta":
		text := ""
		if t, ok := chunk["text"].(string); ok && t != "" {
			text = t
		} else if d, ok := chunk["delta"].(string); ok && d != "" {
			text = d
		}
		if text == "" {
			break
		}
		delta := map[string]any{"content": text}
		if state.CommandCodeChunkIndex == 0 {
			delta["role"] = "assistant"
		}
		state.CommandCodeChunkIndex++
		out = append(out, makeCommandCodeChunk(state, delta, ""))

	case "reasoning-delta":
		text := ""
		if t, ok := chunk["text"].(string); ok {
			text = t
		}
		if text == "" {
			break
		}
		delta := map[string]any{"reasoning_content": text}
		if state.CommandCodeChunkIndex == 0 {
			delta["role"] = "assistant"
		}
		state.CommandCodeChunkIndex++
		out = append(out, makeCommandCodeChunk(state, delta, ""))

	case "tool-input-start":
		id := ""
		if i, ok := chunk["id"].(string); ok && i != "" {
			id = i
		} else if tcid, ok := chunk["toolCallId"].(string); ok {
			id = tcid
		}
		if id == "" {
			id = fmt.Sprintf("call_%d_%d", time.Now().UnixMilli(), state.CommandCodeToolIndex)
		}
		idx, ok := state.CommandCodeToolIndexByID[id]
		if !ok {
			idx = state.CommandCodeToolIndex
			state.CommandCodeToolIndex++
			state.CommandCodeToolIndexByID[id] = idx
		}
		delta := map[string]any{
			"tool_calls": []any{
				map[string]any{
					"index": idx,
					"id":    id,
					"type":  "function",
					"function": map[string]any{
						"name":      chunk["toolName"],
						"arguments": "",
					},
				},
			},
		}
		if state.CommandCodeChunkIndex == 0 {
			delta["role"] = "assistant"
		}
		state.CommandCodeChunkIndex++
		out = append(out, makeCommandCodeChunk(state, delta, ""))

	case "tool-input-delta":
		id := ""
		if i, ok := chunk["id"].(string); ok {
			id = i
		} else if tcid, ok := chunk["toolCallId"].(string); ok {
			id = tcid
		}
		idx, ok := state.CommandCodeToolIndexByID[id]
		if !ok {
			break
		}
		deltaArg := ""
		if d, ok := chunk["delta"].(string); ok {
			deltaArg = d
		} else if d, ok := chunk["inputTextDelta"].(string); ok {
			deltaArg = d
		}
		delta := map[string]any{
			"tool_calls": []any{
				map[string]any{
					"index": idx,
					"function": map[string]any{
						"arguments": deltaArg,
					},
				},
			},
		}
		out = append(out, makeCommandCodeChunk(state, delta, ""))

	case "tool-call":
		id := ""
		if i, ok := chunk["toolCallId"].(string); ok {
			id = i
		}
		if id == "" {
			break
		}
		if _, ok := state.CommandCodeToolIndexByID[id]; ok {
			break
		}
		idx := state.CommandCodeToolIndex
		state.CommandCodeToolIndex++
		state.CommandCodeToolIndexByID[id] = idx
		var argsStr string
		if input, ok := chunk["input"]; ok {
			switch v := input.(type) {
			case string:
				argsStr = v
			default:
				b, _ := json.Marshal(v)
				argsStr = string(b)
			}
		} else {
			argsStr = "{}"
		}
		delta := map[string]any{
			"tool_calls": []any{
				map[string]any{
					"index": idx,
					"id":    id,
					"type":  "function",
					"function": map[string]any{
						"name":      chunk["toolName"],
						"arguments": argsStr,
					},
				},
			},
		}
		if state.CommandCodeChunkIndex == 0 {
			delta["role"] = "assistant"
		}
		state.CommandCodeChunkIndex++
		out = append(out, makeCommandCodeChunk(state, delta, ""))

	case "finish-step":
		if reason, ok := chunk["finishReason"].(string); ok {
			state.CommandCodeFinishReason = mapCommandCodeFinishReason(reason)
		}
		if usage, ok := chunk["usage"].(map[string]any); ok && usage != nil {
			state.CommandCodeUsage = usage
		}

	case "finish":
		finishReason := state.CommandCodeFinishReason
		if finishReason == "" {
			if reason, ok := chunk["finishReason"].(string); ok && reason != "" {
				finishReason = mapCommandCodeFinishReason(reason)
			} else {
				finishReason = "stop"
			}
		}
		finalChunk := makeCommandCodeChunk(state, map[string]any{}, finishReason)
		totalUsage := state.CommandCodeUsage
		if tu, ok := chunk["totalUsage"].(map[string]any); ok && tu != nil {
			totalUsage = tu
		}
		if totalUsage != nil {
			promptTokens := 0
			if n, ok := totalUsage["inputTokens"].(float64); ok {
				promptTokens = int(n)
			} else if n, ok := totalUsage["inputTokens"].(int); ok {
				promptTokens = n
			}
			completionTokens := 0
			if n, ok := totalUsage["outputTokens"].(float64); ok {
				completionTokens = int(n)
			} else if n, ok := totalUsage["outputTokens"].(int); ok {
				completionTokens = n
			}
			totalTokens := promptTokens + completionTokens
			if n, ok := totalUsage["totalTokens"].(float64); ok {
				totalTokens = int(n)
			} else if n, ok := totalUsage["totalTokens"].(int); ok {
				totalTokens = n
			}
			finalChunk["usage"] = map[string]any{
				"prompt_tokens":     promptTokens,
				"completion_tokens": completionTokens,
				"total_tokens":      totalTokens,
			}
		}
		out = append(out, finalChunk)

	case "error":
		state.CommandCodeFinishReason = "stop"
		errVal := "unknown"
		if e, ok := chunk["error"]; ok && e != nil {
			switch v := e.(type) {
			case string:
				errVal = v
			default:
				b, _ := json.Marshal(v)
				errVal = string(b)
			}
		} else if m, ok := chunk["message"].(string); ok && m != "" {
			errVal = m
		}
		out = append(out,
			makeCommandCodeChunk(state, map[string]any{"content": fmt.Sprintf("\n\n[CommandCode error: %s]", errVal)}, ""),
			makeCommandCodeChunk(state, map[string]any{}, "stop"),
		)

	default:
		// Silently ignore: start, start-step, reasoning-start, reasoning-end,
		// text-start, text-end, provider-metadata, message-metadata, etc.
	}

	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func ensureCommandCodeState(state *StreamState, chunk map[string]any) {
	if state.CommandCodeID != "" {
		return
	}
	state.CommandCodeID = fmt.Sprintf("chatcmpl-%d", time.Now().UnixMilli())
	state.CommandCodeCreated = time.Now().Unix()
	if model, ok := chunk["model"].(string); ok && model != "" {
		state.CommandCodeModel = model
	} else if state.Model != "" {
		state.CommandCodeModel = state.Model
	} else {
		state.CommandCodeModel = "commandcode"
	}
	state.CommandCodeChunkIndex = 0
	state.CommandCodeToolIndex = 0
	state.CommandCodeToolIndexByID = make(map[string]int)
	state.CommandCodeFinishReason = ""
	state.CommandCodeUsage = make(map[string]any)
}

func makeCommandCodeChunk(state *StreamState, delta map[string]any, finishReason string) map[string]any {
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
		"id":      state.CommandCodeID,
		"object":  "chat.completion.chunk",
		"created": state.CommandCodeCreated,
		"model":   state.CommandCodeModel,
		"choices": []any{choice},
	}
}

func mapCommandCodeFinishReason(reason string) string {
	switch reason {
	case "stop":
		return "stop"
	case "length":
		return "length"
	case "tool-calls", "tool_use":
		return "tool_calls"
	case "content-filter":
		return "content_filter"
	case "error":
		return "stop"
	default:
		if reason != "" {
			return reason
		}
		return "stop"
	}
}
