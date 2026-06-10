package translation

import (
	"testing"
)

func TestNormalizeResponsesInput(t *testing.T) {
	t.Run("string input", func(t *testing.T) {
		result := normalizeResponsesInput("hello")
		if result == nil {
			t.Fatal("expected non-nil")
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 item, got %d", len(result))
		}
		item := result[0].(map[string]any)
		if item["type"] != "message" || item["role"] != "user" {
			t.Errorf("unexpected item: %v", item)
		}
		content := item["content"].([]any)
		if len(content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(content))
		}
		textBlock := content[0].(map[string]any)
		if textBlock["type"] != "input_text" || textBlock["text"] != "hello" {
			t.Errorf("unexpected text block: %v", textBlock)
		}
	})

	t.Run("empty string placeholder", func(t *testing.T) {
		result := normalizeResponsesInput("")
		content := result[0].(map[string]any)["content"].([]any)
		text := content[0].(map[string]any)["text"].(string)
		if text != "..." {
			t.Errorf("expected '...' placeholder, got %q", text)
		}
	})

	t.Run("whitespace string placeholder", func(t *testing.T) {
		result := normalizeResponsesInput("   ")
		content := result[0].(map[string]any)["content"].([]any)
		text := content[0].(map[string]any)["text"].(string)
		if text != "..." {
			t.Errorf("expected '...' placeholder, got %q", text)
		}
	})

	t.Run("array input passthrough", func(t *testing.T) {
		input := []any{map[string]any{"type": "message", "role": "user"}}
		result := normalizeResponsesInput(input)
		if len(result) != 1 {
			t.Fatalf("expected 1 item, got %d", len(result))
		}
	})

	t.Run("empty array placeholder", func(t *testing.T) {
		result := normalizeResponsesInput([]any{})
		if len(result) != 1 {
			t.Fatalf("expected 1 placeholder item, got %d", len(result))
		}
		content := result[0].(map[string]any)["content"].([]any)
		text := content[0].(map[string]any)["text"].(string)
		if text != "..." {
			t.Errorf("expected '...' placeholder, got %q", text)
		}
	})

	t.Run("non-string non-array returns nil", func(t *testing.T) {
		result := normalizeResponsesInput(123)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})
}

func TestClampCallID(t *testing.T) {
	t.Run("64 char unchanged", func(t *testing.T) {
		id := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl"
		if len(id) != 64 {
			t.Fatalf("test setup: expected 64 chars, got %d", len(id))
		}
		result := clampCallID(id)
		if result != id {
			t.Errorf("expected unchanged, got %v", result)
		}
	})

	t.Run("65 char truncated to 64", func(t *testing.T) {
		id := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklm"
		if len(id) != 65 {
			t.Fatalf("test setup: expected 65 chars, got %d", len(id))
		}
		result := clampCallID(id)
		s, ok := result.(string)
		if !ok {
			t.Fatalf("expected string, got %T", result)
		}
		if len(s) != 64 {
			t.Errorf("expected len 64, got %d", len(s))
		}
	})

	t.Run("non-string passthrough", func(t *testing.T) {
		result := clampCallID(123)
		if result != 123 {
			t.Errorf("expected 123, got %v", result)
		}
	})
}

func TestNormalizeToolParameters(t *testing.T) {
	t.Run("nil returns object with empty properties", func(t *testing.T) {
		result := normalizeToolParameters(nil)
		if result["type"] != "object" {
			t.Errorf("expected type object, got %v", result["type"])
		}
		props, ok := result["properties"].(map[string]any)
		if !ok || len(props) != 0 {
			t.Errorf("expected empty properties, got %v", result["properties"])
		}
	})

	t.Run("object without properties gets properties added", func(t *testing.T) {
		input := map[string]any{"type": "object"}
		result := normalizeToolParameters(input)
		if _, ok := result["properties"]; !ok {
			t.Errorf("expected properties key")
		}
		// Must be a copy, not mutate input
		if _, ok := input["properties"]; ok {
			t.Error("input should not be mutated")
		}
	})

	t.Run("object with properties unchanged", func(t *testing.T) {
		input := map[string]any{"type": "object", "properties": map[string]any{"x": map[string]any{}}}
		result := normalizeToolParameters(input)
		props, ok := result["properties"].(map[string]any)
		if !ok || len(props) != 1 {
			t.Errorf("expected 1 property, got %v", result["properties"])
		}
	})

	t.Run("non-object unchanged", func(t *testing.T) {
		input := map[string]any{"type": "string"}
		result := normalizeToolParameters(input)
		if result["type"] != "string" {
			t.Errorf("expected type string, got %v", result["type"])
		}
	})
}
