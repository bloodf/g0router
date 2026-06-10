package translation

import "strings"

// normalizeResponsesInput normalizes Responses API input to an array of message
// items. Accepts string or array; returns nil for non-string/non-array.
// Empty string/array gets a placeholder "..." text item so providers never
// receive an empty messages list.
func normalizeResponsesInput(input any) []any {
	if s, ok := input.(string); ok {
		text := s
		if strings.TrimSpace(text) == "" {
			text = "..."
		}
		return []any{
			map[string]any{
				"type":    "message",
				"role":    "user",
				"content": []any{map[string]any{"type": "input_text", "text": text}},
			},
		}
	}
	if arr, ok := input.([]any); ok {
		if len(arr) == 0 {
			return []any{
				map[string]any{
					"type":    "message",
					"role":    "user",
					"content": []any{map[string]any{"type": "input_text", "text": "..."}},
				},
			}
		}
		return arr
	}
	return nil
}

const maxCallIDLen = 64

// clampCallID truncates string call IDs to 64 characters (Responses API limit).
// Non-string values pass through unchanged.
func clampCallID(id any) any {
	if s, ok := id.(string); ok && len(s) > maxCallIDLen {
		return s[:maxCallIDLen]
	}
	return id
}

// normalizeToolParameters ensures an object schema always has a properties field.
// nil → {type:object, properties:{}}; object without properties → copy with
// properties added. Others pass through unchanged.
func normalizeToolParameters(params map[string]any) map[string]any {
	if params == nil {
		return map[string]any{"type": "object", "properties": map[string]any{}}
	}
	if params["type"] == "object" {
		if _, ok := params["properties"]; !ok {
			// Return a copy so the input is not mutated.
			copyParams := make(map[string]any, len(params)+1)
			for k, v := range params {
				copyParams[k] = v
			}
			copyParams["properties"] = map[string]any{}
			return copyParams
		}
	}
	return params
}
