package translation

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// unsupportedSchemaConstraints lists JSON Schema keywords that Gemini does not
// support. Ported from geminiHelper.js:4-23.
var unsupportedSchemaConstraints = []string{
	// Basic constraints
	"minLength", "maxLength", "exclusiveMinimum", "exclusiveMaximum",
	"pattern", "minItems", "maxItems", "format",
	// Claude rejects these in VALIDATED mode
	"default", "examples",
	// JSON Schema meta keywords
	"$schema", "$defs", "definitions", "const", "$ref", "$comment",
	// Object validation keywords
	"additionalProperties", "propertyNames", "patternProperties", "enumDescriptions",
	// Complex schema keywords
	"anyOf", "oneOf", "allOf", "not",
	// Dependency keywords
	"dependencies", "dependentSchemas", "dependentRequired",
	// Other unsupported keywords
	"title", "if", "then", "else", "contentMediaType", "contentEncoding",
	// UI/Styling properties
	"cornerRadius", "fillColor", "fontFamily", "fontSize", "fontWeight",
	"gap", "padding", "strokeColor", "strokeThickness", "textColor",
}

// defaultSafetySettings returns the 5 Gemini safety categories with threshold OFF.
// Ported from geminiHelper.js:26-32. Returns a new slice each call (no shared mutable state).
func defaultSafetySettings() []map[string]any {
	return []map[string]any{
		{"category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "OFF"},
		{"category": "HARM_CATEGORY_DANGEROUS_CONTENT", "threshold": "OFF"},
		{"category": "HARM_CATEGORY_SEXUALLY_EXPLICIT", "threshold": "OFF"},
		{"category": "HARM_CATEGORY_HARASSMENT", "threshold": "OFF"},
		{"category": "HARM_CATEGORY_CIVIC_INTEGRITY", "threshold": "OFF"},
	}
}

// sanitizeGeminiFunctionName ensures a name complies with Gemini's function-name
// rules: starts with [a-zA-Z_], remainder [a-zA-Z0-9_.:\-], max 64 chars.
// Ported from openai-to-gemini.js:26-36 (row 027).
func sanitizeGeminiFunctionName(name string) string {
	if name == "" {
		return "_unknown"
	}
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == ':' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	sanitized := b.String()
	if sanitized == "" || !((sanitized[0] >= 'a' && sanitized[0] <= 'z') || (sanitized[0] >= 'A' && sanitized[0] <= 'Z') || sanitized[0] == '_') {
		sanitized = "_" + sanitized
	}
	if len(sanitized) > 64 {
		sanitized = sanitized[:64]
	}
	return sanitized
}

// convertOpenAIContentToParts turns OpenAI-style content into Gemini parts.
// Ported from geminiHelper.js:35-81.
func convertOpenAIContentToParts(content any) []map[string]any {
	var parts []map[string]any

	if s, ok := content.(string); ok {
		parts = append(parts, map[string]any{"text": s})
	} else if arr, ok := content.([]any); ok {
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			typ, _ := m["type"].(string)
			switch typ {
			case "text":
				if text, ok := m["text"].(string); ok {
					parts = append(parts, map[string]any{"text": text})
				}
			case "image_url":
				if imageURL, ok := m["image_url"].(map[string]any); ok {
					if url, ok := imageURL["url"].(string); ok {
						if strings.HasPrefix(url, "data:") {
							commaIdx := strings.Index(url, ",")
							if commaIdx != -1 {
								mimePart := url[5:commaIdx]
								data := url[commaIdx+1:]
								mimeType := strings.Split(mimePart, ";")[0]
								parts = append(parts, map[string]any{
									"inlineData": map[string]any{
										"mime_type": mimeType,
										"data":      data,
									},
								})
							}
						} else if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
							parts = append(parts, map[string]any{
								"fileData": map[string]any{
									"fileUri":  url,
									"mimeType": "image/*",
								},
							})
						}
					}
				}
			case "input_audio":
				if inputAudio, ok := m["input_audio"].(map[string]any); ok {
					if data, ok := inputAudio["data"].(string); ok {
						format := "wav"
						if f, ok := inputAudio["format"].(string); ok {
							format = f
						}
						mimeType := "audio/mpeg"
						if format != "mp3" {
							mimeType = "audio/" + format
						}
						parts = append(parts, map[string]any{
							"inlineData": map[string]any{
								"mime_type": mimeType,
								"data":      data,
							},
						})
					}
				}
			case "audio_url":
				if audioURL, ok := m["audio_url"].(map[string]any); ok {
					if url, ok := audioURL["url"].(string); ok && strings.HasPrefix(url, "data:") {
						commaIdx := strings.Index(url, ",")
						if commaIdx != -1 {
							mimePart := url[5:commaIdx]
							data := url[commaIdx+1:]
							mimeType := strings.Split(mimePart, ";")[0]
							parts = append(parts, map[string]any{
								"inlineData": map[string]any{
									"mime_type": mimeType,
									"data":      data,
								},
							})
						}
					}
				}
			}
		}
	}

	return parts
}

// extractTextContentGemini extracts plain text from OpenAI content. Strings pass
// through; arrays yield the concatenation of text-block texts.
// Ported from geminiHelper.js:85-91.
func extractTextContentGemini(content any) string {
	if s, ok := content.(string); ok {
		return s
	}
	if arr, ok := content.([]any); ok {
		var parts []string
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				if m["type"] == "text" {
					if text, ok := m["text"].(string); ok {
						parts = append(parts, text)
					}
				}
			}
		}
		return strings.Join(parts, "")
	}
	return ""
}

// tryParseJSONValue parses a string as JSON; non-strings pass through unchanged.
// Parse failure returns nil.
// Ported from geminiHelper.js:94-101.
func tryParseJSONValue(input any) any {
	if input == nil {
		return nil
	}
	if s, ok := input.(string); ok {
		var result any
		if err := json.Unmarshal([]byte(s), &result); err != nil {
			return nil
		}
		return result
	}
	return input
}

// jsString approximates JavaScript's String(v) for values found in JSON schema
// enums. Ported from geminiHelper.js:168.
func jsString(v any) string {
	if v == nil {
		return "null"
	}
	if s, ok := v.(string); ok {
		return s
	}
	if b, ok := v.(bool); ok {
		if b {
			return "true"
		}
		return "false"
	}
	if f, ok := v.(float64); ok {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	if arr, ok := v.([]any); ok {
		parts := make([]string, len(arr))
		for i, item := range arr {
			parts[i] = jsString(item)
		}
		return strings.Join(parts, ",")
	}
	if _, ok := v.(map[string]any); ok {
		return "[object Object]"
	}
	return fmt.Sprintf("%v", v)
}

// cleanJSONSchemaForGemini removes unsupported keywords and normalizes a JSON
// schema for Gemini compatibility. It mutates the input map in-place.
// Ported from cleanJSONSchemaForAntigravity (geminiHelper.js:298-371).
func cleanJSONSchemaForGemini(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}
	// Phase 1: Convert and prepare
	convertConstToEnum(schema)
	convertEnumValuesToStrings(schema)

	// Phase 2: Flatten complex structures
	mergeAllOf(schema)
	flattenAnyOfOneOf(schema)
	flattenTypeArrays(schema)

	// Phase 2.5: Infer missing type=object when properties exist
	ensureObjectType(schema)

	// Phase 3: Remove all unsupported keywords
	removeUnsupportedKeywords(schema, unsupportedSchemaConstraints)

	// Phase 4: Cleanup required fields
	cleanupRequired(schema)

	// Phase 5: Add placeholders for empty object schemas
	addPlaceholders(schema)

	return schema
}

func sliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func removeUnsupportedKeywords(obj any, keywords []string) {
	if obj == nil {
		return
	}
	if arr, ok := obj.([]any); ok {
		for _, item := range arr {
			removeUnsupportedKeywords(item, keywords)
		}
		return
	}
	m, ok := obj.(map[string]any)
	if !ok {
		return
	}
	for key := range m {
		if sliceContains(keywords, key) || strings.HasPrefix(key, "x-") {
			delete(m, key)
			continue
		}
		removeUnsupportedKeywords(m[key], keywords)
	}
}

func convertConstToEnum(obj any) {
	m, ok := obj.(map[string]any)
	if !ok {
		if arr, ok := obj.([]any); ok {
			for _, item := range arr {
				convertConstToEnum(item)
			}
		}
		return
	}
	if m["const"] != nil {
		if _, hasEnum := m["enum"]; !hasEnum {
			m["enum"] = []any{m["const"]}
			delete(m, "const")
		}
	}
	for _, v := range m {
		convertConstToEnum(v)
	}
}

func convertEnumValuesToStrings(obj any) {
	m, ok := obj.(map[string]any)
	if !ok {
		if arr, ok := obj.([]any); ok {
			for _, item := range arr {
				convertEnumValuesToStrings(item)
			}
		}
		return
	}
	if enum, ok := m["enum"].([]any); ok {
		strEnum := make([]any, len(enum))
		for i, v := range enum {
			strEnum[i] = jsString(v)
		}
		m["enum"] = strEnum
		if _, hasType := m["type"]; !hasType {
			m["type"] = "string"
		}
	}
	for _, v := range m {
		convertEnumValuesToStrings(v)
	}
}

func requiredFieldKey(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%T", v)
	}
	return string(b)
}

func mergeRequiredFields(existing, add []any) []any {
	seen := make(map[string]struct{}, len(existing)+len(add))
	out := make([]any, 0, len(existing)+len(add))
	for _, r := range existing {
		k := requiredFieldKey(r)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, r)
	}
	for _, r := range add {
		k := requiredFieldKey(r)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, r)
	}
	return out
}

func mergeAllOf(obj any) {
	m, ok := obj.(map[string]any)
	if !ok {
		if arr, ok := obj.([]any); ok {
			for _, item := range arr {
				mergeAllOf(item)
			}
		}
		return
	}
	if allOf, ok := m["allOf"].([]any); ok && len(allOf) > 0 {
		mergedProps := make(map[string]any)
		var mergedReq []any
		seenReq := make(map[string]struct{})
		for _, item := range allOf {
			im, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if props, ok := im["properties"].(map[string]any); ok {
				for k, v := range props {
					mergedProps[k] = v
				}
			}
			if req, ok := im["required"].([]any); ok {
				for _, r := range req {
					k := requiredFieldKey(r)
					if _, seen := seenReq[k]; seen {
						continue
					}
					seenReq[k] = struct{}{}
					mergedReq = append(mergedReq, r)
				}
			}
		}
		delete(m, "allOf")
		if len(mergedProps) > 0 {
			if existingProps, ok := m["properties"].(map[string]any); ok {
				for k, v := range mergedProps {
					existingProps[k] = v
				}
			} else {
				m["properties"] = mergedProps
			}
		}
		if len(mergedReq) > 0 {
			if existingReq, ok := m["required"].([]any); ok {
				m["required"] = mergeRequiredFields(existingReq, mergedReq)
			} else {
				m["required"] = mergedReq
			}
		}
	}
	for _, v := range m {
		mergeAllOf(v)
	}
}

func selectBest(items []any) int {
	bestIdx := 0
	bestScore := -1
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		score := 0
		typ, _ := m["type"].(string)
		if typ == "object" || m["properties"] != nil {
			score = 3
		} else if typ == "array" || m["items"] != nil {
			score = 2
		} else if typ != "" && typ != "null" {
			score = 1
		}
		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}
	return bestIdx
}

func flattenAnyOfOneOf(obj any) {
	m, ok := obj.(map[string]any)
	if !ok {
		if arr, ok := obj.([]any); ok {
			for _, item := range arr {
				flattenAnyOfOneOf(item)
			}
		}
		return
	}

	if anyOf, ok := m["anyOf"].([]any); ok && len(anyOf) > 0 {
		nonNull := make([]any, 0)
		for _, item := range anyOf {
			im, ok := item.(map[string]any)
			if !ok {
				nonNull = append(nonNull, item)
				continue
			}
			typ, _ := im["type"].(string)
			if typ != "null" {
				nonNull = append(nonNull, item)
			}
		}
		if len(nonNull) > 0 {
			bestIdx := selectBest(nonNull)
			selected, ok := nonNull[bestIdx].(map[string]any)
			if ok {
				delete(m, "anyOf")
				for k, v := range selected {
					m[k] = v
				}
			}
		}
	}

	if oneOf, ok := m["oneOf"].([]any); ok && len(oneOf) > 0 {
		nonNull := make([]any, 0)
		for _, item := range oneOf {
			im, ok := item.(map[string]any)
			if !ok {
				nonNull = append(nonNull, item)
				continue
			}
			typ, _ := im["type"].(string)
			if typ != "null" {
				nonNull = append(nonNull, item)
			}
		}
		if len(nonNull) > 0 {
			bestIdx := selectBest(nonNull)
			selected, ok := nonNull[bestIdx].(map[string]any)
			if ok {
				delete(m, "oneOf")
				for k, v := range selected {
					m[k] = v
				}
			}
		}
	}

	for _, v := range m {
		flattenAnyOfOneOf(v)
	}
}

func flattenTypeArrays(obj any) {
	m, ok := obj.(map[string]any)
	if !ok {
		if arr, ok := obj.([]any); ok {
			for _, item := range arr {
				flattenTypeArrays(item)
			}
		}
		return
	}
	if typArr, ok := m["type"].([]any); ok && len(typArr) > 0 {
		found := false
		for _, t := range typArr {
			if ts, ok := t.(string); ok && ts != "null" {
				m["type"] = ts
				found = true
				break
			}
		}
		if !found {
			m["type"] = "string"
		}
	}
	for _, v := range m {
		flattenTypeArrays(v)
	}
}

func ensureObjectType(obj any) {
	m, ok := obj.(map[string]any)
	if !ok {
		if arr, ok := obj.([]any); ok {
			for _, item := range arr {
				ensureObjectType(item)
			}
		}
		return
	}
	if m["properties"] != nil {
		if _, hasType := m["type"]; !hasType {
			m["type"] = "object"
		}
	}
	for _, v := range m {
		ensureObjectType(v)
	}
}

func cleanupRequired(obj any) {
	m, ok := obj.(map[string]any)
	if !ok {
		if arr, ok := obj.([]any); ok {
			for _, item := range arr {
				cleanupRequired(item)
			}
		}
		return
	}
	if req, ok := m["required"].([]any); ok && m["properties"] != nil {
		props, _ := m["properties"].(map[string]any)
		valid := make([]any, 0)
		for _, r := range req {
			if rs, ok := r.(string); ok {
				if _, exists := props[rs]; exists {
					valid = append(valid, rs)
				}
			}
		}
		if len(valid) == 0 {
			delete(m, "required")
		} else {
			m["required"] = valid
		}
	}
	for _, v := range m {
		cleanupRequired(v)
	}
}

func addPlaceholders(obj any) {
	m, ok := obj.(map[string]any)
	if !ok {
		if arr, ok := obj.([]any); ok {
			for _, item := range arr {
				addPlaceholders(item)
			}
		}
		return
	}
	if typ, ok := m["type"].(string); ok && typ == "object" {
		props, hasProps := m["properties"].(map[string]any)
		if !hasProps || len(props) == 0 {
			m["properties"] = map[string]any{
				"reason": map[string]any{
					"type":        "string",
					"description": "Brief explanation of why you are calling this tool",
				},
			}
			m["required"] = []any{"reason"}
		}
	}
	for _, v := range m {
		addPlaceholders(v)
	}
}
