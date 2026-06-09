package schemas

import (
	"encoding/json"
	"testing"
)

func ptrFloat64(v float64) *float64 { return &v }
func ptrInt(v int) *int             { return &v }

func TestChatRequestRoundTrip(t *testing.T) {
	req := ChatRequest{
		Model:       "gpt-4o",
		Messages:    []Message{{Role: "user", Content: "hello"}},
		Temperature: ptrFloat64(0.7),
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ChatRequest
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Model != "gpt-4o" {
		t.Errorf("model = %q, want gpt-4o", got.Model)
	}
	if len(got.Messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(got.Messages))
	}
	if got.Messages[0].Role != "user" {
		t.Errorf("role = %q, want user", got.Messages[0].Role)
	}
	if got.Messages[0].Content != "hello" {
		t.Errorf("content = %q, want hello", got.Messages[0].Content)
	}
	if got.Temperature == nil || *got.Temperature != 0.7 {
		t.Errorf("temperature = %v, want 0.7", got.Temperature)
	}
}

func TestChatResponseRoundTrip(t *testing.T) {
	resp := ChatResponse{
		ID:      "chat-123",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "gpt-4o",
		Choices: []Choice{{
			Index: 0,
			Message: &Message{
				Role:    "assistant",
				Content: "hi",
			},
			FinishReason: "stop",
		}},
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ChatResponse
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != "chat-123" {
		t.Errorf("id = %q, want chat-123", got.ID)
	}
	if len(got.Choices) != 1 {
		t.Fatalf("choices len = %d, want 1", len(got.Choices))
	}
	if got.Choices[0].Message == nil || got.Choices[0].Message.Content != "hi" {
		t.Errorf("choice content = %v, want hi", got.Choices[0].Message)
	}
}

func TestEmbeddingRequestRoundTrip(t *testing.T) {
	req := EmbeddingRequest{
		Input: "hello",
		Model: "text-embedding-3-small",
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got EmbeddingRequest
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Input != "hello" {
		t.Errorf("input = %v, want hello", got.Input)
	}
	if got.Model != "text-embedding-3-small" {
		t.Errorf("model = %q, want text-embedding-3-small", got.Model)
	}
}

func TestStreamChunkRoundTrip(t *testing.T) {
	chunk := StreamChunk{
		ID:     "chatcmpl-123",
		Object: "chat.completion.chunk",
		Model:  "gpt-4o",
		Choices: []StreamChoice{{
			Index: 0,
			Delta: Message{Role: "assistant", Content: "hi"},
		}},
	}
	b, err := json.Marshal(chunk)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got StreamChunk
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != "chatcmpl-123" {
		t.Errorf("id = %q, want chatcmpl-123", got.ID)
	}
	if len(got.Choices) != 1 {
		t.Fatalf("choices len = %d, want 1", len(got.Choices))
	}
	if got.Choices[0].Delta.Content != "hi" {
		t.Errorf("delta content = %q, want hi", got.Choices[0].Delta.Content)
	}
}
