package gemini

import (
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

func TestConvertChatRequestToolMessageWithCallID(t *testing.T) {
	callID := "call_abc"
	fnName := "get_weather"
	req := &schemas.ChatRequest{
		Model: "gemini-1.5-pro",
		Messages: []schemas.Message{
			{Role: "user", Content: "What is the weather?"},
			{
				Role:       "tool",
				Content:    "25°C and sunny",
				ToolCallID: &callID,
				Name:       &fnName,
			},
		},
	}

	converted := ConvertChatRequest(req)

	if len(converted.Contents) != 2 {
		t.Fatalf("contents len = %d, want 2", len(converted.Contents))
	}
	// Tool message becomes user role with functionResponse part
	toolContent := converted.Contents[1]
	if toolContent.Role != "user" {
		t.Errorf("tool role = %q, want user", toolContent.Role)
	}
	if len(toolContent.Parts) != 1 {
		t.Fatalf("tool parts len = %d, want 1", len(toolContent.Parts))
	}
	fr := toolContent.Parts[0].FunctionResponse
	if fr == nil {
		t.Fatal("functionResponse should not be nil")
	}
	if fr.ID != "call_abc" {
		t.Errorf("functionResponse id = %q, want call_abc", fr.ID)
	}
	if fr.Name != "get_weather" {
		t.Errorf("functionResponse name = %q, want get_weather", fr.Name)
	}
}

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
