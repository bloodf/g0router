package translate

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

func TestOpenAIToGeminiConvertsMessagesAndGenerationConfig(t *testing.T) {
	temp := 0.2
	topP := 0.9
	maxTokens := 128
	req := &providers.ChatRequest{
		Model:               "gemini-2.5-flash",
		System:              "top level system",
		Temperature:         &temp,
		TopP:                &topP,
		MaxCompletionTokens: &maxTokens,
		Messages: []providers.Message{
			{Role: "system", Content: "message system"},
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hello back"},
		},
	}

	got, err := OpenAIToGemini(req)
	if err != nil {
		t.Fatalf("OpenAIToGemini() error = %v", err)
	}

	if len(got.Contents) != 2 {
		t.Fatalf("contents len = %d", len(got.Contents))
	}
	if got.Contents[0].Role != "user" || got.Contents[0].Parts[0].Text != "hello" {
		t.Fatalf("user content = %+v", got.Contents[0])
	}
	if got.Contents[1].Role != "model" || got.Contents[1].Parts[0].Text != "hello back" {
		t.Fatalf("assistant content = %+v", got.Contents[1])
	}
	if got.SystemInstruction == nil || got.SystemInstruction.Parts[0].Text != "top level system\n\nmessage system" {
		t.Fatalf("system instruction = %+v", got.SystemInstruction)
	}
	if got.GenerationConfig == nil {
		t.Fatal("generation config is nil")
	}
	if got.GenerationConfig.Temperature == nil || *got.GenerationConfig.Temperature != 0.2 {
		t.Fatalf("temperature = %+v", got.GenerationConfig.Temperature)
	}
	if got.GenerationConfig.TopP == nil || *got.GenerationConfig.TopP != 0.9 {
		t.Fatalf("topP = %+v", got.GenerationConfig.TopP)
	}
	if got.GenerationConfig.MaxOutputTokens == nil || *got.GenerationConfig.MaxOutputTokens != 128 {
		t.Fatalf("maxOutputTokens = %+v", got.GenerationConfig.MaxOutputTokens)
	}
}

func TestOpenAIToGeminiConvertsTextBlocksToolsAndToolCalls(t *testing.T) {
	params := json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}}}`)
	req := &providers.ChatRequest{
		Model: "gemini-2.5-flash",
		Tools: []providers.Tool{
			{
				Type: "function",
				Function: providers.ToolFunction{
					Name:        "weather",
					Description: "Get weather",
					Parameters:  params,
				},
			},
		},
		Messages: []providers.Message{
			{
				Role: "user",
				Content: []map[string]any{
					{"type": "text", "text": "What is the weather?"},
				},
			},
			{
				Role:    "assistant",
				Content: "Calling weather.",
				ToolCalls: []providers.ToolCall{
					{
						ID:   "call-1",
						Type: "function",
						Function: providers.ToolCallFunc{
							Name:      "weather",
							Arguments: `{"city":"Paris"}`,
						},
					},
				},
			},
			{
				Role:       "tool",
				ToolCallID: stringPtr("call-1"),
				Content:    `{"temp_c":19}`,
			},
		},
	}

	got, err := OpenAIToGemini(req)
	if err != nil {
		t.Fatalf("OpenAIToGemini() error = %v", err)
	}

	if len(got.Tools) != 1 || len(got.Tools[0].FunctionDeclarations) != 1 {
		t.Fatalf("tools = %+v", got.Tools)
	}
	declaration := got.Tools[0].FunctionDeclarations[0]
	if declaration.Name != "weather" || declaration.Description != "Get weather" {
		t.Fatalf("declaration = %+v", declaration)
	}
	if string(declaration.Parameters) != `{"type":"object","properties":{"city":{"type":"string"}}}` {
		t.Fatalf("parameters = %s", declaration.Parameters)
	}
	if got.Contents[0].Parts[0].Text != "What is the weather?" {
		t.Fatalf("text block part = %+v", got.Contents[0].Parts[0])
	}
	if got.Contents[1].Role != "model" {
		t.Fatalf("tool call role = %q", got.Contents[1].Role)
	}
	if got.Contents[1].Parts[0].Text != "Calling weather." {
		t.Fatalf("assistant text part = %+v", got.Contents[1].Parts[0])
	}
	if got.Contents[1].Parts[1].FunctionCall == nil {
		t.Fatalf("function call part = %+v", got.Contents[1].Parts[1])
	}
	if got.Contents[1].Parts[1].FunctionCall.ID != "call-1" {
		t.Fatalf("function call id = %q", got.Contents[1].Parts[1].FunctionCall.ID)
	}
	if got.Contents[1].Parts[1].FunctionCall.Name != "weather" {
		t.Fatalf("function call name = %q", got.Contents[1].Parts[1].FunctionCall.Name)
	}
	if got.Contents[1].Parts[1].FunctionCall.Args["city"] != "Paris" {
		t.Fatalf("function call args = %+v", got.Contents[1].Parts[1].FunctionCall.Args)
	}
	if got.Contents[2].Role != "user" || got.Contents[2].Parts[0].FunctionResponse == nil {
		t.Fatalf("tool result content = %+v", got.Contents[2])
	}
	if got.Contents[2].Parts[0].FunctionResponse.Name != "weather" {
		t.Fatalf("function response name = %q", got.Contents[2].Parts[0].FunctionResponse.Name)
	}
	if got.Contents[2].Parts[0].FunctionResponse.ID != "call-1" {
		t.Fatalf("function response id = %q", got.Contents[2].Parts[0].FunctionResponse.ID)
	}
	if got.Contents[2].Parts[0].FunctionResponse.Response["content"] != `{"temp_c":19}` {
		t.Fatalf("function response = %+v", got.Contents[2].Parts[0].FunctionResponse.Response)
	}
}

func TestOpenAIToGeminiOmitsEmptyGenerationConfig(t *testing.T) {
	got, err := OpenAIToGemini(&providers.ChatRequest{
		Model: "gemini-2.5-flash",
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("OpenAIToGemini() error = %v", err)
	}
	if got.GenerationConfig != nil {
		t.Fatalf("generation config = %+v", got.GenerationConfig)
	}
}

func TestOpenAIToGeminiRejectsUnsupportedContent(t *testing.T) {
	_, err := OpenAIToGemini(&providers.ChatRequest{
		Model: "gemini-2.5-flash",
		Messages: []providers.Message{
			{Role: "user", Content: []map[string]any{{"type": "image_url", "image_url": map[string]any{"url": "x"}}}},
		},
	})
	if err == nil {
		t.Fatal("OpenAIToGemini() error = nil")
	}
	if !strings.Contains(err.Error(), "unsupported content block type") {
		t.Fatalf("error = %v", err)
	}
}

func TestOpenAIToGeminiNilRequest(t *testing.T) {
	_, err := OpenAIToGemini(nil)
	if err == nil {
		t.Fatal("OpenAIToGemini() error = nil")
	}
}

func stringPtr(value string) *string {
	return &value
}
