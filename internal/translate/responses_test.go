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

func TestResponsesRequestToOpenAIChat(t *testing.T) {
	instructions := "Be brief."
	maxOutputTokens := 64
	stream := true
	req := &ResponsesRequest{
		Model:           "gpt-4o-mini",
		Instructions:    &instructions,
		MaxOutputTokens: &maxOutputTokens,
		Stream:          &stream,
		Input: []ResponsesInput{
			{
				Role: "user",
				Content: []ResponsesContent{
					{Type: "input_text", Text: "Hello"},
					{Type: "input_text", Text: " world"},
				},
			},
		},
		Tools: []ResponsesTool{
			{Type: "function", Name: "lookup", Description: "Look up a value", Parameters: json.RawMessage(`{"type":"object"}`)},
		},
	}

	got, err := ResponsesRequestToOpenAIChat(req)
	if err != nil {
		t.Fatalf("ResponsesRequestToOpenAIChat: %v", err)
	}
	if got.Model != "gpt-4o-mini" {
		t.Fatalf("model = %q", got.Model)
	}
	if got.Stream == nil || !*got.Stream {
		t.Fatalf("stream = %+v, want true", got.Stream)
	}
	if got.MaxCompletionTokens == nil || *got.MaxCompletionTokens != 64 {
		t.Fatalf("max completion tokens = %+v, want 64", got.MaxCompletionTokens)
	}
	if got.System != "Be brief." {
		t.Fatalf("system = %#v, want instructions", got.System)
	}
	if len(got.Messages) != 1 || got.Messages[0].Role != "user" || got.Messages[0].Content != "Hello world" {
		t.Fatalf("messages = %+v", got.Messages)
	}
	if len(got.Tools) != 1 || got.Tools[0].Function.Name != "lookup" {
		t.Fatalf("tools = %+v", got.Tools)
	}
}

func TestResponsesRequestToOpenAIChatRejectsUnsupportedInputContent(t *testing.T) {
	_, err := ResponsesRequestToOpenAIChat(&ResponsesRequest{
		Model: "gpt-4o-mini",
		Input: []ResponsesInput{
			{
				Role: "user",
				Content: []ResponsesContent{
					{Type: "input_image"},
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected unsupported content error")
	}
}

func TestResponsesRequestToOpenAIChatRejectsUnsupportedInputItem(t *testing.T) {
	_, err := ResponsesRequestToOpenAIChat(&ResponsesRequest{
		Model: "gpt-4o-mini",
		Input: []ResponsesInput{
			{
				Type: "function_call_output",
				Role: "user",
				Content: []ResponsesContent{
					{Type: "input_text", Text: "tool result"},
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected unsupported input item error")
	}
}

func TestOpenAIChatToResponsesResponse(t *testing.T) {
	resp := &providers.ChatResponse{
		ID:      "chatcmpl-123",
		Created: 1700000000,
		Model:   "gpt-4o-mini",
		Choices: []providers.Choice{
			{
				Message: providers.Message{Role: "assistant", Content: "Hello world"},
			},
		},
		Usage: &providers.Usage{PromptTokens: 4, CompletionTokens: 3, TotalTokens: 7},
	}

	got := OpenAIChatToResponsesResponse(resp)
	if got.ID != "chatcmpl-123" || got.Object != "response" || got.Status != "completed" {
		t.Fatalf("metadata = %+v", got)
	}
	if got.OutputText != "Hello world" {
		t.Fatalf("output text = %q, want Hello world", got.OutputText)
	}
	if len(got.Output) != 1 || got.Output[0].Role != "assistant" || got.Output[0].Content[0].Type != "output_text" {
		t.Fatalf("output = %+v", got.Output)
	}
	if got.Usage == nil || got.Usage.TotalTokens != 7 {
		t.Fatalf("usage = %+v", got.Usage)
	}
}

func TestOpenAIChatToResponsesResponsePreservesToolCalls(t *testing.T) {
	resp := &providers.ChatResponse{
		ID:      "chatcmpl-tool",
		Created: 1700000000,
		Model:   "gpt-4o-mini",
		Choices: []providers.Choice{
			{
				Message: providers.Message{
					Role: "assistant",
					ToolCalls: []providers.ToolCall{{
						ID:   "call_lookup",
						Type: "function",
						Function: providers.ToolCallFunc{
							Name:      "lookup",
							Arguments: `{"query":"docs"}`,
						},
					}},
				},
			},
		},
		Usage: &providers.Usage{PromptTokens: 4, CompletionTokens: 3, TotalTokens: 7},
	}

	got := OpenAIChatToResponsesResponse(resp)
	body, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	var decoded struct {
		Output []struct {
			Type      string `json:"type"`
			CallID    string `json:"call_id"`
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"output"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal response: %v; body=%s", err, body)
	}
	if len(decoded.Output) != 1 {
		t.Fatalf("output len = %d, want 1: %s", len(decoded.Output), body)
	}
	call := decoded.Output[0]
	if call.Type != "function_call" || call.CallID != "call_lookup" || call.Name != "lookup" || call.Arguments != `{"query":"docs"}` {
		t.Fatalf("function call output = %+v", call)
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
