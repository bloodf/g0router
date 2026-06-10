package gemini

import (
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

func TestConvertChatRequestMultipleSystemMessages(t *testing.T) {
	req := &schemas.ChatRequest{
		Model: "gemini-1.5-pro",
		Messages: []schemas.Message{
			{Role: "system", Content: "First system instruction"},
			{Role: "system", Content: "Second system instruction"},
			{Role: "user", Content: "Hello"},
		},
	}

	converted := ConvertChatRequest(req)

	if converted.SystemInstruction == nil {
		t.Fatal("system_instruction should not be nil")
	}
	if len(converted.SystemInstruction.Parts) != 1 {
		t.Fatalf("system parts len = %d, want 1", len(converted.SystemInstruction.Parts))
	}
	want := "First system instruction\n\nSecond system instruction"
	if converted.SystemInstruction.Parts[0].Text != want {
		t.Errorf("system text = %q, want %q", converted.SystemInstruction.Parts[0].Text, want)
	}
	if len(converted.Contents) != 1 {
		t.Fatalf("contents len = %d, want 1", len(converted.Contents))
	}
}
