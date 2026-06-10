package anthropic

import (
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

func TestConvertRequestMultipleSystemMessages(t *testing.T) {
	req := &schemas.ChatRequest{
		Model: "claude-3-5-sonnet",
		Messages: []schemas.Message{
			{Role: "system", Content: "First system prompt"},
			{Role: "system", Content: "Second system prompt"},
			{Role: "user", Content: "Hello"},
		},
	}

	converted := ConvertRequest(req)

	want := "First system prompt\n\nSecond system prompt"
	if converted.System != want {
		t.Errorf("system = %q, want %q", converted.System, want)
	}
	if len(converted.Messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(converted.Messages))
	}
	if converted.Messages[0].Role != "user" {
		t.Errorf("role = %q, want user", converted.Messages[0].Role)
	}
}
