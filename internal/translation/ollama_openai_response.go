package translation

import (
	"encoding/json"
	"fmt"
	"time"
)

// ollamaToOpenAIResponse converts an Ollama-format NDJSON stream chunk into
// OpenAI-shaped stream chunks. Port of
// open-sse/translator/response/ollama-to-openai.js.
func ollamaToOpenAIResponse(chunk map[string]any, state *StreamState) ([]map[string]any, error) {
	if chunk == nil {
		return nil, nil
	}

	// Initialize state on first chunk.
	if state.OllamaID == "" {
		state.OllamaID = fmt.Sprintf("chatcmpl-%d", time.Now().UnixMilli())
		state.OllamaCreated = time.Now().Unix()
		if model, ok := chunk["model"].(string); ok && model != "" {
			state.OllamaModel = model
		} else if state.Model != "" {
			state.OllamaModel = state.Model
		}
	}

	id := state.OllamaID
	created := state.OllamaCreated
	model := state.OllamaModel

	// Final chunk with done=true.
	if done, ok := chunk["done"].(bool); ok && done {
		usage := extractOllamaUsage(chunk)

		finishReason := "stop"
		if reason, ok := chunk["done_reason"].(string); ok && reason == "tool_calls" {
			finishReason = "tool_calls"
		} else if state.OllamaHadToolCalls {
			finishReason = "tool_calls"
		}

		return []map[string]any{
			{
				"id":      id,
				"object":  "chat.completion.chunk",
				"created": created,
				"model":   model,
				"choices": []any{
					map[string]any{
						"index":         0,
						"delta":         map[string]any{},
						"finish_reason": finishReason,
					},
				},
				"usage": usage,
			},
		}, nil
	}

	// Content chunk.
	message, _ := chunk["message"].(map[string]any)
	if message == nil {
		return nil, nil
	}

	content := ""
	if c, ok := message["content"].(string); ok {
		content = c
	}
	thinking := ""
	if t, ok := message["thinking"].(string); ok {
		thinking = t
	}
	var toolCalls []any
	if tc, ok := message["tool_calls"].([]any); ok {
		toolCalls = tc
	}

	// Skip empty chunks.
	if content == "" && thinking == "" && len(toolCalls) == 0 {
		return nil, nil
	}

	// Accumulate content in state.
	if content != "" {
		state.OllamaContent += content
	}
	if thinking != "" {
		state.OllamaThinking += thinking
	}

	delta := map[string]any{}
	if content != "" {
		delta["content"] = content
	}
	if thinking != "" {
		delta["reasoning_content"] = thinking
	}
	if len(toolCalls) > 0 {
		state.OllamaHadToolCalls = true
		delta["tool_calls"] = convertOllamaToolCalls(toolCalls)
	}

	return []map[string]any{
		{
			"id":      id,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   model,
			"choices": []any{
				map[string]any{
					"index":         0,
					"delta":         delta,
					"finish_reason": nil,
				},
			},
		},
	}, nil
}

func extractOllamaUsage(ollamaChunk map[string]any) map[string]any {
	promptEval := 0
	if n, ok := ollamaChunk["prompt_eval_count"].(float64); ok {
		promptEval = int(n)
	} else if n, ok := ollamaChunk["prompt_eval_count"].(int); ok {
		promptEval = n
	}
	evalCount := 0
	if n, ok := ollamaChunk["eval_count"].(float64); ok {
		evalCount = int(n)
	} else if n, ok := ollamaChunk["eval_count"].(int); ok {
		evalCount = n
	}
	return map[string]any{
		"prompt_tokens":     promptEval,
		"completion_tokens": evalCount,
		"total_tokens":      promptEval + evalCount,
	}
}

func convertOllamaToolCalls(toolCalls []any) []any {
	now := time.Now().UnixMilli()
	result := make([]any, 0, len(toolCalls))
	for i, tc := range toolCalls {
		tcMap, ok := tc.(map[string]any)
		if !ok {
			continue
		}
		var fn map[string]any
		if f, ok := tcMap["function"].(map[string]any); ok {
			fn = f
		}
		idx := i
		if n, ok := fn["index"].(float64); ok {
			idx = int(n)
		} else if n, ok := fn["index"].(int); ok {
			idx = n
		}
		id := ""
		if idStr, ok := tcMap["id"].(string); ok && idStr != "" {
			id = idStr
		} else {
			id = fmt.Sprintf("call_%d_%d", i, now)
		}
		name := ""
		if n, ok := fn["name"].(string); ok {
			name = n
		}
		var arguments string
		if args, ok := fn["arguments"]; ok {
			switch v := args.(type) {
			case string:
				arguments = v
			default:
				b, _ := json.Marshal(v)
				arguments = string(b)
			}
		} else {
			arguments = "{}"
		}
		result = append(result, map[string]any{
			"index": idx,
			"id":    id,
			"type":  "function",
			"function": map[string]any{
				"name":      name,
				"arguments": arguments,
			},
		})
	}
	return result
}

// OllamaBodyToOpenAI converts an Ollama non-streaming response body to an
// OpenAI chat.completion shape. Port of ollama-to-openai.js:126-149.
func OllamaBodyToOpenAI(body map[string]any) map[string]any {
	msg := map[string]any{}
	if m, ok := body["message"].(map[string]any); ok {
		msg = m
	}
	content := ""
	if c, ok := msg["content"].(string); ok {
		content = c
	}
	thinking := ""
	if t, ok := msg["thinking"].(string); ok {
		thinking = t
	}
	var toolCalls []any
	if tc, ok := msg["tool_calls"].([]any); ok {
		toolCalls = tc
	}

	message := map[string]any{
		"role": "assistant",
	}
	if content != "" {
		message["content"] = content
	}
	if thinking != "" {
		message["reasoning_content"] = thinking
	}
	if len(toolCalls) > 0 {
		message["tool_calls"] = convertOllamaToolCalls(toolCalls)
	}
	if message["content"] == nil && message["tool_calls"] == nil {
		message["content"] = ""
	}

	finishReason := "stop"
	if reason, ok := body["done_reason"].(string); ok && reason != "" {
		finishReason = reason
	}
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	}

	model := "ollama"
	if m, ok := body["model"].(string); ok && m != "" {
		model = m
	}

	return map[string]any{
		"id":      fmt.Sprintf("chatcmpl-%d", time.Now().UnixMilli()),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []any{
			map[string]any{
				"index":          0,
				"message":        message,
				"finish_reason":  finishReason,
			},
		},
		"usage": extractOllamaUsage(body),
	}
}
