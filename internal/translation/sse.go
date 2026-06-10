package translation

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// HasValuableContent returns whether a stream chunk carries meaningful data.
func HasValuableContent(chunk map[string]any, format Format) bool {
	if format == FormatOpenAI {
		choicesRaw, ok := chunk["choices"]
		if !ok {
			return true
		}
		choices, ok := choicesRaw.([]any)
		if !ok || len(choices) == 0 {
			return true
		}
		choice, ok := choices[0].(map[string]any)
		if !ok {
			return true
		}
		deltaRaw, ok := choice["delta"]
		if !ok {
			return true
		}
		delta, ok := deltaRaw.(map[string]any)
		if !ok {
			return true
		}
		if content, ok := delta["content"].(string); ok && content != "" {
			return true
		}
		if reasoning, ok := delta["reasoning_content"].(string); ok && reasoning != "" {
			return true
		}
		if toolCalls, ok := delta["tool_calls"].([]any); ok && len(toolCalls) > 0 {
			return true
		}
		if finishReason, ok := choice["finish_reason"].(string); ok && finishReason != "" {
			return true
		}
		if role, ok := delta["role"].(string); ok && role != "" {
			return true
		}
		return false
	}

	if format == FormatClaude {
		typ, _ := chunk["type"].(string)
		if typ != "content_block_delta" {
			return true
		}
		deltaRaw, ok := chunk["delta"]
		if !ok {
			return true
		}
		delta, ok := deltaRaw.(map[string]any)
		if !ok {
			return true
		}
		hasText := false
		if text, ok := delta["text"].(string); ok && text != "" {
			hasText = true
		}
		hasThinking := false
		if thinking, ok := delta["thinking"].(string); ok && thinking != "" {
			hasThinking = true
		}
		hasPartialJSON := false
		if partialJSON, ok := delta["partial_json"].(string); ok && partialJSON != "" {
			hasPartialJSON = true
		}
		if !hasText && !hasThinking && !hasPartialJSON {
			return false
		}
		return true
	}

	return true
}

// FixInvalidID replaces generic or too-short IDs with a chatcmpl- prefix.
// It returns true when the chunk was mutated.
func FixInvalidID(chunk map[string]any) bool {
	idRaw, ok := chunk["id"]
	if !ok {
		return false
	}
	id, ok := idRaw.(string)
	if !ok {
		return false
	}
	if id == "chat" || id == "completion" || len(id) < 8 {
		fallback := ""
		if extendFieldsRaw, ok := chunk["extend_fields"]; ok {
			if extendFields, ok := extendFieldsRaw.(map[string]any); ok {
				if reqID, ok := extendFields["requestId"].(string); ok && reqID != "" {
					fallback = reqID
				} else if traceID, ok := extendFields["traceId"].(string); ok && traceID != "" {
					fallback = traceID
				}
			}
		}
		if fallback == "" {
			fallback = strconv.FormatInt(time.Now().UnixMilli(), 36)
		}
		chunk["id"] = "chatcmpl-" + fallback
		return true
	}
	return false
}

// cleanUsagePayload drops null usage and null perf_metrics, recursing into
// the "response" key.
func cleanUsagePayload(payload map[string]any) map[string]any {
	if payload == nil {
		return nil
	}
	if usageRaw, ok := payload["usage"]; ok && usageRaw == nil {
		delete(payload, "usage")
	} else if usage, ok := usageRaw.(map[string]any); ok {
		if perfRaw, ok := usage["perf_metrics"]; ok && perfRaw == nil {
			delete(usage, "perf_metrics")
		}
	}
	if responseRaw, ok := payload["response"]; ok {
		if response, ok := responseRaw.(map[string]any); ok {
			cleanUsagePayload(response)
		}
	}
	return payload
}

// FormatSSE formats an event map for Server-Sent Events. For Claude-format
// events carrying a "type" key, it prefixes the frame with "event: <type>".
// All other formats emit the standard "data: <json>" frame.
func FormatSSE(format Format, event map[string]any) []byte {
	cleanUsagePayload(event)
	b, err := json.Marshal(event)
	if err != nil {
		return []byte("data: null\n\n")
	}
	if format == FormatClaude && event != nil {
		if _, ok := event["type"]; ok {
			return []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event["type"], b))
		}
	}
	return []byte(fmt.Sprintf("data: %s\n\n", b))
}
