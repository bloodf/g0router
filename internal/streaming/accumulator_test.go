package streaming

import (
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

func TestAccumulateTextResponse(t *testing.T) {
	role := "assistant"
	stop := "stop"
	acc := NewAccumulator()

	acc.AddChunk(providers.StreamChunk{
		ID:      "chatcmpl-1",
		Object:  "chat.completion.chunk",
		Created: 1700000000,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{
			{
				Index: 0,
				Delta: providers.StreamDelta{
					Role:    &role,
					Content: stringPtr("Hel"),
				},
			},
		},
	})
	acc.AddChunk(providers.StreamChunk{
		ID:      "chatcmpl-1",
		Object:  "chat.completion.chunk",
		Created: 1700000000,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{
			{Index: 0, Delta: providers.StreamDelta{Content: stringPtr("lo, ")}},
		},
	})
	acc.AddChunk(providers.StreamChunk{
		ID:      "chatcmpl-1",
		Object:  "chat.completion.chunk",
		Created: 1700000000,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{
			{
				Index:        0,
				Delta:        providers.StreamDelta{Content: stringPtr("world!")},
				FinishReason: &stop,
			},
		},
	})

	resp := acc.Response()
	if resp.ID != "chatcmpl-1" || resp.Object != "chat.completion" || resp.Created != 1700000000 || resp.Model != "gpt-4o" {
		t.Fatalf("response metadata mismatch: %+v", resp)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("choices len = %d, want 1", len(resp.Choices))
	}
	choice := resp.Choices[0]
	if choice.Index != 0 {
		t.Fatalf("choice index = %d, want 0", choice.Index)
	}
	if choice.Message.Role != "assistant" {
		t.Fatalf("role = %q, want assistant", choice.Message.Role)
	}
	if choice.Message.Content != "Hello, world!" {
		t.Fatalf("content = %q, want Hello, world!", choice.Message.Content)
	}
	if choice.FinishReason == nil || *choice.FinishReason != "stop" {
		t.Fatalf("finish reason = %v, want stop", choice.FinishReason)
	}
}

func TestAccumulateToolCall(t *testing.T) {
	role := "assistant"
	stop := "tool_calls"
	acc := NewAccumulator()

	acc.AddChunk(providers.StreamChunk{
		ID:      "chatcmpl-tool",
		Object:  "chat.completion.chunk",
		Created: 1700000001,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{
			{
				Index: 0,
				Delta: providers.StreamDelta{
					Role: &role,
					ToolCalls: []providers.ToolCall{
						{
							ID:   "call-1",
							Type: "function",
							Function: providers.ToolCallFunc{
								Name:      "get_weather",
								Arguments: `{"city":"`,
							},
						},
					},
				},
			},
		},
	})
	acc.AddChunk(providers.StreamChunk{
		ID:      "chatcmpl-tool",
		Object:  "chat.completion.chunk",
		Created: 1700000001,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{
			{
				Index: 0,
				Delta: providers.StreamDelta{
					ToolCalls: []providers.ToolCall{
						{Function: providers.ToolCallFunc{Arguments: `Paris"}`}},
					},
				},
				FinishReason: &stop,
			},
		},
	})

	resp := acc.Response()
	if len(resp.Choices) != 1 {
		t.Fatalf("choices len = %d, want 1", len(resp.Choices))
	}
	calls := resp.Choices[0].Message.ToolCalls
	if len(calls) != 1 {
		t.Fatalf("tool calls len = %d, want 1", len(calls))
	}
	if calls[0].ID != "call-1" || calls[0].Type != "function" || calls[0].Function.Name != "get_weather" {
		t.Fatalf("tool call metadata mismatch: %+v", calls[0])
	}
	if calls[0].Function.Arguments != `{"city":"Paris"}` {
		t.Fatalf("arguments = %q", calls[0].Function.Arguments)
	}
	if resp.Choices[0].FinishReason == nil || *resp.Choices[0].FinishReason != "tool_calls" {
		t.Fatalf("finish reason = %v, want tool_calls", resp.Choices[0].FinishReason)
	}
}

func TestAccumulateUsageFromFinalChunk(t *testing.T) {
	acc := NewAccumulator()

	acc.AddChunk(providers.StreamChunk{
		ID:      "chatcmpl-usage",
		Object:  "chat.completion.chunk",
		Created: 1700000002,
		Model:   "gpt-4o",
		Usage: &providers.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	})

	usage := acc.Usage()
	if usage == nil {
		t.Fatal("usage should be non-nil")
	}
	if usage.TotalTokens != 15 {
		t.Fatalf("total tokens = %d, want 15", usage.TotalTokens)
	}
	if acc.Response().Usage == nil || acc.Response().Usage.TotalTokens != 15 {
		t.Fatalf("response usage = %+v, want total 15", acc.Response().Usage)
	}
}

func TestAccumulateMultipleChoices(t *testing.T) {
	stop := "stop"
	acc := NewAccumulator()

	acc.AddChunk(providers.StreamChunk{
		ID:      "chatcmpl-multi",
		Object:  "chat.completion.chunk",
		Created: 1700000003,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{
			{Index: 1, Delta: providers.StreamDelta{Content: stringPtr("sec")}},
			{Index: 0, Delta: providers.StreamDelta{Content: stringPtr("fir")}},
		},
	})
	acc.AddChunk(providers.StreamChunk{
		ID:      "chatcmpl-multi",
		Object:  "chat.completion.chunk",
		Created: 1700000003,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{
			{Index: 0, Delta: providers.StreamDelta{Content: stringPtr("st")}, FinishReason: &stop},
			{Index: 1, Delta: providers.StreamDelta{Content: stringPtr("ond")}, FinishReason: &stop},
		},
	})

	resp := acc.Response()
	if len(resp.Choices) != 2 {
		t.Fatalf("choices len = %d, want 2", len(resp.Choices))
	}
	if resp.Choices[0].Index != 0 || resp.Choices[0].Message.Content != "first" {
		t.Fatalf("choice 0 = %+v", resp.Choices[0])
	}
	if resp.Choices[1].Index != 1 || resp.Choices[1].Message.Content != "second" {
		t.Fatalf("choice 1 = %+v", resp.Choices[1])
	}
}

func TestAccumulateEmpty(t *testing.T) {
	acc := NewAccumulator()

	resp := acc.Response()
	if resp.ID != "" || resp.Object != "chat.completion" || resp.Created != 0 || resp.Model != "" {
		t.Fatalf("empty metadata = %+v", resp)
	}
	if len(resp.Choices) != 0 {
		t.Fatalf("choices len = %d, want 0", len(resp.Choices))
	}
	if resp.Usage != nil {
		t.Fatalf("usage = %+v, want nil", resp.Usage)
	}
	if acc.Usage() != nil {
		t.Fatalf("Usage() = %+v, want nil", acc.Usage())
	}
}

func stringPtr(value string) *string {
	return &value
}
