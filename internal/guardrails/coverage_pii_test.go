package guardrails

import (
	"testing"
)

func TestRedactUnknownType(t *testing.T) {
	cfg := PIIConfig{Enabled: true, Types: []string{"unknown_type", "email"}}
	input := "contact me at test@example.com"
	got := cfg.Redact(input)
	want := "contact me at [REDACTED:email]"
	if got != want {
		t.Fatalf("Redact = %q, want %q", got, want)
	}
}

func TestRedactContentNonMapItemInAnySlice(t *testing.T) {
	cfg := PIIConfig{Enabled: true, Types: []string{"email"}}
	content := []any{
		map[string]any{"type": "text", "text": "hello test@example.com"},
		"not a map",
		42,
	}
	got := redactContent(cfg, content)
	out, ok := got.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", got)
	}
	if len(out) != 3 {
		t.Fatalf("len = %d, want 3", len(out))
	}
	// First item should be redacted
	m, ok := out[0].(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", out[0])
	}
	if m["text"] != "hello [REDACTED:email]" {
		t.Fatalf("text = %q, want redacted", m["text"])
	}
	// Non-map items should pass through unchanged
	if out[1] != "not a map" {
		t.Fatalf("item 1 = %v, want 'not a map'", out[1])
	}
	if out[2] != 42 {
		t.Fatalf("item 2 = %v, want 42", out[2])
	}
}

func TestRedactContentDefaultCase(t *testing.T) {
	cfg := PIIConfig{Enabled: true, Types: []string{"email"}}
	content := 12345
	got := redactContent(cfg, content)
	if got != content {
		t.Fatalf("expected same content for default case, got %v", got)
	}
}
