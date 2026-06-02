package translate

import (
	"encoding/json"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

func TestOpenAIChatToResponsesRequest(t *testing.T) {
	temp := 0.2
	maxTokens := 128
	req := &providers.ChatRequest{
		Model:       "gpt-4o-mini",
		Temperature: &temp,
		MaxTokens:   &maxTokens,
		Messages: []providers.Message{
			{Role: "system", Content: "Be brief."},
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi"},
		},
		Tools: []providers.Tool{
			{
				Type: "function",
				Function: providers.ToolFunction{
					Name:        "lookup",
					Description: "Look up a value",
					Parameters:  json.RawMessage(`{"type":"object"}`),
				},
			},
		},
	}

	got, err := OpenAIChatToResponses(req)
	if err != nil {
		t.Fatalf("OpenAIChatToResponses: %v", err)
	}

	if got.Model != "gpt-4o-mini" {
		t.Fatalf("model = %q", got.Model)
	}
	if got.Instructions == nil || *got.Instructions != "Be brief." {
		t.Fatalf("instructions = %+v", got.Instructions)
	}
	if got.Temperature == nil || *got.Temperature != 0.2 {
		t.Fatalf("temperature = %+v", got.Temperature)
	}
	if got.MaxOutputTokens == nil || *got.MaxOutputTokens != 128 {
		t.Fatalf("max output tokens = %+v", got.MaxOutputTokens)
	}
	if len(got.Input) != 2 {
		t.Fatalf("input len = %d", len(got.Input))
	}
	if got.Input[0].Role != "user" || got.Input[0].Content[0].Type != "input_text" || got.Input[0].Content[0].Text != "Hello" {
		t.Fatalf("first input = %+v", got.Input[0])
	}
	if got.Input[1].Role != "assistant" || got.Input[1].Content[0].Text != "Hi" {
		t.Fatalf("second input = %+v", got.Input[1])
	}
	if len(got.Tools) != 1 || got.Tools[0].Name != "lookup" || got.Tools[0].Type != "function" {
		t.Fatalf("tools = %+v", got.Tools)
	}
}

func TestResponsesToOpenAIChatResponse(t *testing.T) {
	resp := &ResponsesResponse{
		ID:        "resp_123",
		Object:    "response",
		CreatedAt: 1700000000,
		Model:     "gpt-4o-mini",
		Output: []ResponsesOutput{
			{
				Type: "message",
				Role: "assistant",
				Content: []ResponsesContent{
					{Type: "output_text", Text: "Hello"},
					{Type: "output_text", Text: " world"},
				},
			},
		},
		Usage: &ResponsesUsage{
			InputTokens:  4,
			OutputTokens: 3,
			TotalTokens:  7,
		},
	}

	got := ResponsesToOpenAIChat(resp)
	if got.ID != "resp_123" || got.Object != "chat.completion" || got.Created != 1700000000 || got.Model != "gpt-4o-mini" {
		t.Fatalf("metadata = %+v", got)
	}
	if len(got.Choices) != 1 {
		t.Fatalf("choices len = %d", len(got.Choices))
	}
	if got.Choices[0].Message.Role != "assistant" || got.Choices[0].Message.Content != "Hello world" {
		t.Fatalf("message = %+v", got.Choices[0].Message)
	}
	if got.Usage == nil || got.Usage.PromptTokens != 4 || got.Usage.CompletionTokens != 3 || got.Usage.TotalTokens != 7 {
		t.Fatalf("usage = %+v", got.Usage)
	}
}

func TestOpenAIChatToResponsesRejectsUnsupportedContent(t *testing.T) {
	_, err := OpenAIChatToResponses(&providers.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []providers.Message{
			{Role: "user", Content: 42},
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
