package translation

import (
	"encoding/json"
	"fmt"
	"time"
)

// kiroToOpenAIResponse converts a Kiro-format stream event into OpenAI-shaped
// stream chunks. Port of open-sse/translator/response/kiro-to-openai.js:12-192.
//
// NOTE: the raw-SSE-string parsing branch (:23-53) is NOT ported — g0router's
// scanner/executor delivers parsed maps carrying _eventType or wrapped event keys.
func kiroToOpenAIResponse(chunk map[string]any, state *StreamState) ([]map[string]any, error) {
	if chunk == nil {
		return nil, nil
	}

	// Pass through chunks already shaped as chat.completion.chunk with choices.
	if obj, ok := chunk["object"].(string); ok && obj == "chat.completion.chunk" {
		if _, ok := chunk["choices"]; ok {
			return []map[string]any{chunk}, nil
		}
	}

	// Initialise state on first real event.
	if state.KiroID == "" {
		state.KiroID = fmt.Sprintf("chatcmpl-%d", time.Now().UnixMilli())
		state.KiroCreated = time.Now().Unix()
		if state.Model != "" {
			state.KiroModel = state.Model
		}
	}

	eventType, eventData := resolveKiroEvent(chunk)
	if eventType == "" {
		return nil, nil
	}

	switch eventType {
	case "assistantResponseEvent":
		return handleAssistantResponseEvent(eventData, state)
	case "reasoningContentEvent":
		return handleReasoningContentEvent(eventData, state)
	case "toolUseEvent":
		return handleToolUseEvent(eventData, state)
	case "messageStopEvent", "done":
		return handleMessageStopEvent(state)
	case "usageEvent":
		return handleUsageEvent(eventData, state)
	default:
		return nil, nil
	}
}

// resolveKiroEvent extracts the event type and its payload from a Kiro chunk.
// Supports both _eventType/event top-level keys and wrapped-key forms.
func resolveKiroEvent(chunk map[string]any) (string, map[string]any) {
	if et, ok := chunk["_eventType"].(string); ok && et != "" {
		return et, chunk
	}
	if et, ok := chunk["event"].(string); ok && et != "" {
		return et, chunk
	}
	// Wrapped-key detection: one of the keys is the event type itself.
	for k, v := range chunk {
		if k == "_eventType" || k == "event" {
			continue
		}
		if vm, ok := v.(map[string]any); ok {
			return k, vm
		}
	}
	return "", nil
}

func handleAssistantResponseEvent(data map[string]any, state *StreamState) ([]map[string]any, error) {
	text := ""
	if t, ok := data["content"].(string); ok {
		text = t
	}
	if text == "" {
		return nil, nil
	}

	delta := map[string]any{"content": text}
	if state.KiroChunkIndex == 0 {
		delta["role"] = "assistant"
	}
	state.KiroChunkIndex++

	return []map[string]any{makeKiroChunk(state, delta, "")}, nil
}

func handleReasoningContentEvent(data map[string]any, state *StreamState) ([]map[string]any, error) {
	var text string
	if t, ok := data["text"].(string); ok && t != "" {
		text = t
	} else if t, ok := data["content"].(string); ok && t != "" {
		text = t
	} else if t, ok := data["string"].(string); ok && t != "" {
		text = t
	}
	if text == "" {
		return nil, nil
	}

	delta := map[string]any{"reasoning_content": text}
	if state.KiroChunkIndex == 0 {
		delta["role"] = "assistant"
	}
	state.KiroChunkIndex++

	return []map[string]any{makeKiroChunk(state, delta, "")}, nil
}

func handleToolUseEvent(data map[string]any, state *StreamState) ([]map[string]any, error) {
	id := ""
	if i, ok := data["toolUseId"].(string); ok && i != "" {
		id = i
	}
	if id == "" {
		id = fmt.Sprintf("call_%d", time.Now().UnixMilli())
	}

	name := ""
	if n, ok := data["name"].(string); ok {
		name = n
	}

	var argsStr string
	if input, ok := data["input"]; ok {
		b, err := json.Marshal(input)
		if err != nil {
			return nil, fmt.Errorf("marshal tool input: %w", err)
		}
		argsStr = string(b)
	} else {
		argsStr = "{}"
	}

	delta := map[string]any{
		"tool_calls": []any{
			map[string]any{
				"index": 0,
				"id":    id,
				"type":  "function",
				"function": map[string]any{
					"name":      name,
					"arguments": argsStr,
				},
			},
		},
	}
	if state.KiroChunkIndex == 0 {
		delta["role"] = "assistant"
	}
	state.KiroChunkIndex++

	return []map[string]any{makeKiroChunk(state, delta, "")}, nil
}

func handleMessageStopEvent(state *StreamState) ([]map[string]any, error) {
	state.KiroFinishReason = "stop"
	chunk := makeKiroChunk(state, map[string]any{}, "stop")
	if state.KiroUsage != nil {
		chunk["usage"] = state.KiroUsage
	}
	return []map[string]any{chunk}, nil
}

func handleUsageEvent(data map[string]any, state *StreamState) ([]map[string]any, error) {
	promptTokens := 0
	if n, ok := data["inputTokens"].(float64); ok {
		promptTokens = int(n)
	} else if n, ok := data["inputTokens"].(int); ok {
		promptTokens = n
	}

	completionTokens := 0
	if n, ok := data["outputTokens"].(float64); ok {
		completionTokens = int(n)
	} else if n, ok := data["outputTokens"].(int); ok {
		completionTokens = n
	}

	state.KiroUsage = map[string]any{
		"prompt_tokens":     promptTokens,
		"completion_tokens": completionTokens,
		"total_tokens":      promptTokens + completionTokens,
	}
	return nil, nil
}

func makeKiroChunk(state *StreamState, delta map[string]any, finishReason string) map[string]any {
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
		"id":      state.KiroID,
		"object":  "chat.completion.chunk",
		"created": state.KiroCreated,
		"model":   state.KiroModel,
		"choices": []any{choice},
	}
}
