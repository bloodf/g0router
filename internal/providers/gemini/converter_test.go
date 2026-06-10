package gemini

import (
	"regexp"
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

var chatIDRegex = regexp.MustCompile(`^chatcmpl-[A-Za-z0-9]+$`)

func TestConvertChatResponseGeneratesID(t *testing.T) {
	geminiResp := &GenerateContentResponse{
		Candidates: []Candidate{
			{
				Content: Content{
					Role:  "model",
					Parts: []Part{{Text: "Hello there"}},
				},
				FinishReason: "STOP",
			},
		},
	}

	converted := ConvertChatResponse(geminiResp, "gemini-1.5-pro")

	if converted.ID == "" {
		t.Fatal("id should not be empty")
	}
	if !chatIDRegex.MatchString(converted.ID) {
		t.Errorf("id = %q, does not match %s", converted.ID, chatIDRegex.String())
	}
}

func TestConvertStreamChunkGeneratesID(t *testing.T) {
	geminiResp := &GenerateContentResponse{
		Candidates: []Candidate{
			{
				Content: Content{
					Role:  "model",
					Parts: []Part{{Text: "Hello"}},
				},
			},
		},
	}

	chunk := ConvertStreamChunk(geminiResp, "gemini-1.5-pro")

	if chunk.ID == "" {
		t.Fatal("id should not be empty")
	}
	if !chatIDRegex.MatchString(chunk.ID) {
		t.Errorf("id = %q, does not match %s", chunk.ID, chatIDRegex.String())
	}
}

func TestConvertChatResponseDistinctToolCallIDs(t *testing.T) {
	geminiResp := &GenerateContentResponse{
		Candidates: []Candidate{
			{
				Content: Content{
					Role: "model",
					Parts: []Part{
						{
							FunctionCall: &FunctionCall{
								Name: "get_weather",
								Args: map[string]any{"city": "Paris"},
							},
						},
						{
							FunctionCall: &FunctionCall{
								Name: "get_weather",
								Args: map[string]any{"city": "London"},
							},
						},
					},
				},
				FinishReason: "STOP",
			},
		},
	}

	converted := ConvertChatResponse(geminiResp, "gemini-1.5-pro")

	if len(converted.Choices) != 1 {
		t.Fatalf("choices len = %d, want 1", len(converted.Choices))
	}
	msg := converted.Choices[0].Message
	if len(msg.ToolCalls) != 2 {
		t.Fatalf("tool_calls len = %d, want 2", len(msg.ToolCalls))
	}
	if msg.ToolCalls[0].ID == msg.ToolCalls[1].ID {
		t.Errorf("tool call IDs are equal (%q), expected distinct values", msg.ToolCalls[0].ID)
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
