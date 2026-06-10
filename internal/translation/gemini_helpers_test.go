package translation

import (
	"strings"
	"testing"
)

// TestDefaultSafetySettings asserts the five categories from the reference
// (open-sse/translator/helpers/geminiHelper.js:26-32) all have threshold OFF.
func TestDefaultSafetySettings(t *testing.T) {
	settings := defaultSafetySettings()
	if len(settings) != 5 {
		t.Fatalf("expected 5 safety settings, got %d", len(settings))
	}
	wantCategories := []string{
		"HARM_CATEGORY_HATE_SPEECH",
		"HARM_CATEGORY_DANGEROUS_CONTENT",
		"HARM_CATEGORY_SEXUALLY_EXPLICIT",
		"HARM_CATEGORY_HARASSMENT",
		"HARM_CATEGORY_CIVIC_INTEGRITY",
	}
	for i, wantCat := range wantCategories {
		cat, _ := settings[i]["category"].(string)
		if cat != wantCat {
			t.Errorf("settings[%d].category = %q, want %q", i, cat, wantCat)
		}
		thr, _ := settings[i]["threshold"].(string)
		if thr != "OFF" {
			t.Errorf("settings[%d].threshold = %q, want OFF", i, thr)
		}
	}
}

func TestSanitizeGeminiFunctionName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "_unknown"},
		{"hello world", "hello_world"},
		{"123abc", "_123abc"},
		{"bad@chars#here", "bad_chars_here"},
		{"a" + strings.Repeat("b", 100), "a" + strings.Repeat("b", 63)},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := sanitizeGeminiFunctionName(tc.in)
			if got != tc.want {
				t.Errorf("sanitizeGeminiFunctionName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestCleanSchemaRemovesUnsupportedKeywords(t *testing.T) {
	for _, kw := range []string{"minLength", "maxLength", "format", "default", "$schema", "anyOf", "dependencies", "title", "cornerRadius"} {
		t.Run(kw, func(t *testing.T) {
			schema := map[string]any{kw: "x", "type": "string"}
			cleanJSONSchemaForGemini(schema)
			if _, ok := schema[kw]; ok {
				t.Errorf("expected %q to be removed", kw)
			}
			if schema["type"] != "string" {
				t.Errorf("expected type to remain")
			}
		})
	}
	t.Run("x-custom", func(t *testing.T) {
		schema := map[string]any{"x-custom": "y", "type": "string"}
		cleanJSONSchemaForGemini(schema)
		if _, ok := schema["x-custom"]; ok {
			t.Error("expected x-custom to be removed")
		}
	})
}

func TestCleanSchemaConstToEnum(t *testing.T) {
	schema := map[string]any{"const": "foo"}
	cleanJSONSchemaForGemini(schema)
	if _, ok := schema["const"]; ok {
		t.Error("expected const to be removed")
	}
	enum, ok := schema["enum"].([]any)
	if !ok || len(enum) != 1 || enum[0] != "foo" {
		t.Errorf("enum = %v, want [foo]", schema["enum"])
	}
}

func TestCleanSchemaEnumToStrings(t *testing.T) {
	schema := map[string]any{
		"enum": []any{
			nil,
			map[string]any{"a": float64(1)},
			[]any{float64(1), float64(2)},
			true,
			float64(2),
			float64(2.5),
			float64(1),
		},
	}
	cleanJSONSchemaForGemini(schema)
	enum, ok := schema["enum"].([]any)
	if !ok || len(enum) != 7 {
		t.Fatalf("enum = %v", schema["enum"])
	}
	want := []string{"null", "[object Object]", "1,2", "true", "2", "2.5", "1"}
	for i, w := range want {
		if got := enum[i]; got != w {
			t.Errorf("enum[%d] = %q, want %q", i, got, w)
		}
	}
	if schema["type"] != "string" {
		t.Errorf("type = %v, want string", schema["type"])
	}
}

func TestCleanSchemaFlattenAnyOf(t *testing.T) {
	schema := map[string]any{
		"anyOf": []any{
			map[string]any{"type": "null"},
			map[string]any{"type": "object", "properties": map[string]any{"a": map[string]any{"type": "string"}}},
			map[string]any{"type": "string"},
		},
	}
	cleanJSONSchemaForGemini(schema)
	if _, ok := schema["anyOf"]; ok {
		t.Error("expected anyOf to be removed")
	}
	if schema["type"] != "object" {
		t.Errorf("type = %v, want object", schema["type"])
	}
	if _, ok := schema["properties"]; !ok {
		t.Error("expected properties to be merged")
	}
}

func TestCleanSchemaMergeAllOf(t *testing.T) {
	schema := map[string]any{
		"allOf": []any{
			map[string]any{"properties": map[string]any{"a": map[string]any{"type": "string"}}, "required": []any{"a"}},
			map[string]any{"properties": map[string]any{"b": map[string]any{"type": "number"}}, "required": []any{"a", "b"}},
		},
	}
	cleanJSONSchemaForGemini(schema)
	if _, ok := schema["allOf"]; ok {
		t.Error("expected allOf to be removed")
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok || len(props) != 2 {
		t.Errorf("properties = %v", schema["properties"])
	}
	req, ok := schema["required"].([]any)
	if !ok || len(req) != 2 {
		t.Errorf("required = %v", schema["required"])
	}
}

func TestCleanSchemaTypeArrays(t *testing.T) {
	schema := map[string]any{"type": []any{"null", "string"}}
	cleanJSONSchemaForGemini(schema)
	if schema["type"] != "string" {
		t.Errorf("type = %v, want string", schema["type"])
	}
}

func TestCleanSchemaEnsureObjectType(t *testing.T) {
	schema := map[string]any{"properties": map[string]any{"a": map[string]any{"type": "string"}}}
	cleanJSONSchemaForGemini(schema)
	if schema["type"] != "object" {
		t.Errorf("type = %v, want object", schema["type"])
	}
}

func TestCleanSchemaRequiredCleanup(t *testing.T) {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{"a": map[string]any{"type": "string"}},
		"required":   []any{"a", "b"},
	}
	cleanJSONSchemaForGemini(schema)
	req, ok := schema["required"].([]any)
	if !ok || len(req) != 1 || req[0] != "a" {
		t.Errorf("required = %v, want [a]", schema["required"])
	}
}

func TestCleanSchemaEmptyObjectPlaceholder(t *testing.T) {
	schema := map[string]any{"type": "object"}
	cleanJSONSchemaForGemini(schema)
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties missing")
	}
	reason, ok := props["reason"].(map[string]any)
	if !ok {
		t.Fatal("reason property missing")
	}
	if reason["type"] != "string" {
		t.Errorf("reason.type = %v", reason["type"])
	}
	req, ok := schema["required"].([]any)
	if !ok || len(req) != 1 || req[0] != "reason" {
		t.Errorf("required = %v, want [reason]", schema["required"])
	}
}

func TestConvertContentParts(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		parts := convertOpenAIContentToParts("hello")
		if len(parts) != 1 || parts[0]["text"] != "hello" {
			t.Errorf("parts = %v", parts)
		}
	})
	t.Run("text array item", func(t *testing.T) {
		parts := convertOpenAIContentToParts([]any{map[string]any{"type": "text", "text": "hi"}})
		if len(parts) != 1 || parts[0]["text"] != "hi" {
			t.Errorf("parts = %v", parts)
		}
	})
	t.Run("data uri image", func(t *testing.T) {
		parts := convertOpenAIContentToParts([]any{map[string]any{"type": "image_url", "image_url": map[string]any{"url": "data:image/png;base64,abc123"}}})
		if len(parts) != 1 {
			t.Fatalf("len(parts) = %d", len(parts))
		}
		inlineData, ok := parts[0]["inlineData"].(map[string]any)
		if !ok {
			t.Fatalf("expected inlineData, got %v", parts[0])
		}
		if inlineData["mime_type"] != "image/png" || inlineData["data"] != "abc123" {
			t.Errorf("inlineData = %v", inlineData)
		}
	})
	t.Run("http image", func(t *testing.T) {
		parts := convertOpenAIContentToParts([]any{map[string]any{"type": "image_url", "image_url": map[string]any{"url": "https://example.com/img.png"}}})
		if len(parts) != 1 {
			t.Fatalf("len(parts) = %d", len(parts))
		}
		fileData, ok := parts[0]["fileData"].(map[string]any)
		if !ok {
			t.Fatalf("expected fileData, got %v", parts[0])
		}
		if fileData["fileUri"] != "https://example.com/img.png" || fileData["mimeType"] != "image/*" {
			t.Errorf("fileData = %v", fileData)
		}
	})
	t.Run("input_audio mp3", func(t *testing.T) {
		parts := convertOpenAIContentToParts([]any{map[string]any{"type": "input_audio", "input_audio": map[string]any{"data": "audioData", "format": "mp3"}}})
		if len(parts) != 1 {
			t.Fatalf("len(parts) = %d", len(parts))
		}
		inlineData, ok := parts[0]["inlineData"].(map[string]any)
		if !ok {
			t.Fatalf("expected inlineData, got %v", parts[0])
		}
		if inlineData["mime_type"] != "audio/mpeg" || inlineData["data"] != "audioData" {
			t.Errorf("inlineData = %v", inlineData)
		}
	})
	t.Run("input_audio wav", func(t *testing.T) {
		parts := convertOpenAIContentToParts([]any{map[string]any{"type": "input_audio", "input_audio": map[string]any{"data": "audioData", "format": "wav"}}})
		if len(parts) != 1 {
			t.Fatalf("len(parts) = %d", len(parts))
		}
		inlineData, ok := parts[0]["inlineData"].(map[string]any)
		if !ok {
			t.Fatalf("expected inlineData, got %v", parts[0])
		}
		if inlineData["mime_type"] != "audio/wav" || inlineData["data"] != "audioData" {
			t.Errorf("inlineData = %v", inlineData)
		}
	})
	t.Run("audio_url data uri", func(t *testing.T) {
		parts := convertOpenAIContentToParts([]any{map[string]any{"type": "audio_url", "audio_url": map[string]any{"url": "data:audio/wav;base64,abc123"}}})
		if len(parts) != 1 {
			t.Fatalf("len(parts) = %d", len(parts))
		}
		inlineData, ok := parts[0]["inlineData"].(map[string]any)
		if !ok {
			t.Fatalf("expected inlineData, got %v", parts[0])
		}
		if inlineData["mime_type"] != "audio/wav" || inlineData["data"] != "abc123" {
			t.Errorf("inlineData = %v", inlineData)
		}
	})
	t.Run("unknown dropped", func(t *testing.T) {
		parts := convertOpenAIContentToParts([]any{map[string]any{"type": "unknown", "data": "x"}})
		if len(parts) != 0 {
			t.Errorf("parts = %v", parts)
		}
	})
}

func TestCleanSchemaMergeAllOfNonComparableRequired(t *testing.T) {
	schema := map[string]any{
		"allOf": []any{
			map[string]any{"required": []any{map[string]any{"nested": true}}},
			map[string]any{"required": []any{map[string]any{"nested": true}, "field"}},
		},
	}
	cleanJSONSchemaForGemini(schema) // must not panic on non-string required dedupe
	if _, ok := schema["allOf"]; ok {
		t.Fatalf("allOf should be merged away, got %v", schema["allOf"])
	}
}

func TestCleanSchemaNestedEmptyObjectPlaceholder(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"config": map[string]any{"type": "object", "properties": map[string]any{}},
		},
	}
	cleanJSONSchemaForGemini(schema)
	nested := schema["properties"].(map[string]any)["config"].(map[string]any)
	props, ok := nested["properties"].(map[string]any)
	if !ok {
		t.Fatalf("nested properties missing: %v", nested)
	}
	reason, ok := props["reason"].(map[string]any)
	if !ok {
		t.Fatalf("nested reason placeholder missing: %v", props)
	}
	if reason["type"] != "string" {
		t.Errorf("nested reason.type = %v", reason["type"])
	}
	req, ok := nested["required"].([]any)
	if !ok || len(req) != 1 || req[0] != "reason" {
		t.Errorf("nested required = %v, want [reason]", nested["required"])
	}
}
