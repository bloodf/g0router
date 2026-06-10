package translation

import (
	"encoding/json"
	"fmt"
	"time"
)

func openaiToAntigravityResponse(chunk map[string]any, state *StreamState) ([]map[string]any, error) {
	if chunk == nil {
		return nil, nil
	}

	choicesRaw, ok := chunk["choices"].([]any)
	if !ok || len(choicesRaw) == 0 {
		if usage, ok := chunk["usage"].(map[string]any); ok && usage != nil {
			state.AntigravityUsage = usage
		}
		return nil, nil
	}
	choice, ok := choicesRaw[0].(map[string]any)
	if !ok {
		return nil, nil
	}

	delta := map[string]any{}
	if d, ok := choice["delta"].(map[string]any); ok {
		delta = d
	}
	finishReason := ""
	if fr, ok := choice["finish_reason"].(string); ok {
		finishReason = fr
	}

	if state.AntigravityResponseID == "" {
		state.AntigravityResponseID = fmt.Sprintf("resp_%d", time.Now().UnixMilli())
		if id, ok := chunk["id"].(string); ok && id != "" {
			state.AntigravityResponseID = id
		}
	}
	if state.AntigravityModelVersion == "" {
		if model, ok := chunk["model"].(string); ok {
			state.AntigravityModelVersion = model
		}
	}

	parts := []map[string]any{}

	// Thinking/reasoning → thought part
	if reasoningContent, ok := delta["reasoning_content"].(string); ok && reasoningContent != "" {
		parts = append(parts, map[string]any{"thought": true, "text": reasoningContent})
	}

	// Text content
	if content, ok := delta["content"].(string); ok && content != "" {
		parts = append(parts, map[string]any{"text": content})
	}

	// Accumulate tool calls silently
	if rawToolCalls, ok := delta["tool_calls"].([]any); ok {
		for _, tcRaw := range rawToolCalls {
			tc, ok := tcRaw.(map[string]any)
			if !ok {
				continue
			}
			idx := 0
			if i, ok := tc["index"].(float64); ok {
				idx = int(i)
			}
			if state.AntigravityToolCallAccum[idx] == nil {
				state.AntigravityToolCallAccum[idx] = map[string]any{
					"id":        "",
					"name":      "",
					"arguments": "",
				}
			}
			accum := state.AntigravityToolCallAccum[idx]
			if id, ok := tc["id"].(string); ok && id != "" {
				accum["id"] = id
			}
			if fn, ok := tc["function"].(map[string]any); ok {
				if name, ok := fn["name"].(string); ok && name != "" {
					accum["name"] = accum["name"].(string) + name
				}
				if args, ok := fn["arguments"].(string); ok && args != "" {
					accum["arguments"] = accum["arguments"].(string) + args
				}
			}
		}
		if len(parts) == 0 && finishReason == "" {
			return nil, nil
		}
	}

	// On finish, emit accumulated tool calls
	if finishReason != "" {
		for _, accum := range state.AntigravityToolCallAccum {
			args := map[string]any{}
			if argStr, ok := accum["arguments"].(string); ok && argStr != "" {
				json.Unmarshal([]byte(argStr), &args)
			}
			name := ""
			if n, ok := accum["name"].(string); ok {
				name = n
			}
			if state.ToolNameMap != nil {
				if mapped, ok := state.ToolNameMap[name]; ok {
					name = mapped
				}
			}
			parts = append(parts, map[string]any{
				"functionCall": map[string]any{
					"name": name,
					"args": args,
				},
			})
		}
	}

	// Skip empty non-finish chunks
	if len(parts) == 0 && finishReason == "" {
		return nil, nil
	}

	// Ensure at least empty text part on finish with no content
	if len(parts) == 0 && finishReason != "" {
		parts = append(parts, map[string]any{"text": ""})
	}

	candidate := map[string]any{
		"content": map[string]any{
			"role":  "model",
			"parts": toAnySlice(parts),
		},
	}

	if finishReason != "" {
		reasonMap := map[string]string{
			"stop":           "STOP",
			"length":         "MAX_TOKENS",
			"tool_calls":     "STOP",
			"content_filter": "SAFETY",
		}
		candidate["finishReason"] = reasonMap[finishReason]
		if candidate["finishReason"] == "" {
			candidate["finishReason"] = "STOP"
		}
	}

	response := map[string]any{
		"candidates":   []any{candidate},
		"modelVersion": state.AntigravityModelVersion,
		"responseId":   state.AntigravityResponseID,
	}

	usage := chunk["usage"]
	if usage == nil {
		usage = state.AntigravityUsage
	}
	if um, ok := usage.(map[string]any); ok && um != nil {
		response["usageMetadata"] = map[string]any{
			"promptTokenCount":     float64(0),
			"candidatesTokenCount": float64(0),
			"totalTokenCount":      float64(0),
		}
		if v, ok := um["prompt_tokens"].(float64); ok {
			response["usageMetadata"].(map[string]any)["promptTokenCount"] = v
		}
		if v, ok := um["completion_tokens"].(float64); ok {
			response["usageMetadata"].(map[string]any)["candidatesTokenCount"] = v
		}
		if v, ok := um["total_tokens"].(float64); ok {
			response["usageMetadata"].(map[string]any)["totalTokenCount"] = v
		}
		if details, ok := um["completion_tokens_details"].(map[string]any); ok && details != nil {
			if v, ok := details["reasoning_tokens"].(float64); ok {
				response["usageMetadata"].(map[string]any)["thoughtsTokenCount"] = v
			}
		}
		if details, ok := um["prompt_tokens_details"].(map[string]any); ok && details != nil {
			if v, ok := details["cached_tokens"].(float64); ok {
				response["usageMetadata"].(map[string]any)["cachedContentTokenCount"] = v
			}
		}
	}

	return []map[string]any{{"response": response}}, nil
}
