package translation

import (
	"reflect"
	"testing"
)

func TestCursorOpenAIPassthrough(t *testing.T) {
	// Streaming chunk returned as-is.
	chunk := map[string]any{
		"object":  "chat.completion.chunk",
		"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"content": "hi"}}},
	}
	out, err := cursorToOpenAIResponse(chunk, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if !reflect.DeepEqual(out[0], chunk) {
		t.Errorf("out[0] = %v, want %v", out[0], chunk)
	}

	// Completion object returned as-is.
	completion := map[string]any{
		"object":  "chat.completion",
		"choices": []any{map[string]any{"index": 0, "message": map[string]any{"content": "ok"}}},
	}
	out, err = cursorToOpenAIResponse(completion, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if !reflect.DeepEqual(out[0], completion) {
		t.Errorf("out[0] = %v, want %v", out[0], completion)
	}

	// Unknown map returned as-is.
	unknown := map[string]any{"foo": "bar"}
	out, err = cursorToOpenAIResponse(unknown, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if !reflect.DeepEqual(out[0], unknown) {
		t.Errorf("out[0] = %v, want %v", out[0], unknown)
	}

	// nil returns nil.
	out, err = cursorToOpenAIResponse(nil, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if out != nil {
		t.Errorf("out = %v, want nil", out)
	}
}
