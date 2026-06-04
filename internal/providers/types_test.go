package providers

import (
	"encoding/json"
	"testing"
)

func TestChatRequestJSONRoundTrip(t *testing.T) {
	stream := true
	temp := 0.7
	req := ChatRequest{
		Model: "gpt-4o",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
		Stream:      &stream,
		Temperature: &temp,
		Tools: []Tool{
			{
				Type: "function",
				Function: ToolFunction{
					Name:        "get_weather",
					Description: "Get weather",
					Parameters:  json.RawMessage(`{"type":"object"}`),
				},
			},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got ChatRequest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Model != "gpt-4o" {
		t.Errorf("model: %q", got.Model)
	}
	if len(got.Messages) != 1 || got.Messages[0].Role != "user" {
		t.Errorf("messages: %+v", got.Messages)
	}
	if got.Stream == nil || !*got.Stream {
		t.Error("stream should be true")
	}
	if got.Temperature == nil || *got.Temperature != 0.7 {
		t.Error("temperature should be 0.7")
	}
	if len(got.Tools) != 1 || got.Tools[0].Function.Name != "get_weather" {
		t.Errorf("tools: %+v", got.Tools)
	}
}

func TestChatRequestMinimal(t *testing.T) {
	input := `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`
	var req ChatRequest
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.Model != "gpt-4o" {
		t.Errorf("model: %q", req.Model)
	}
	if req.Stream != nil {
		t.Error("stream should be nil")
	}
	if req.Temperature != nil {
		t.Error("temperature should be nil")
	}
	if req.Tools != nil {
		t.Error("tools should be nil")
	}
}

func TestKeyCarriesProviderAccountID(t *testing.T) {
	key := Key{
		Value:     "cf-token",
		Provider:  ProviderCloudflare,
		AccountID: "account-123",
	}

	data, err := json.Marshal(key)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Key
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.AccountID != "account-123" {
		t.Fatalf("account id = %q, want account-123", got.AccountID)
	}
}

func TestChatResponseJSONRoundTrip(t *testing.T) {
	finish := "stop"
	resp := ChatResponse{
		ID:      "chatcmpl-123",
		Object:  "chat.completion",
		Created: 1700000000,
		Model:   "gpt-4o",
		Choices: []Choice{
			{
				Index:        0,
				Message:      Message{Role: "assistant", Content: "Hello!"},
				FinishReason: &finish,
			},
		},
		Usage: &Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
			PromptTokensDetails: &PromptTokensDetails{
				CachedTokens: 3,
			},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got ChatResponse
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.ID != "chatcmpl-123" {
		t.Errorf("id: %q", got.ID)
	}
	if len(got.Choices) != 1 {
		t.Fatalf("choices len: %d", len(got.Choices))
	}
	if got.Usage == nil || got.Usage.TotalTokens != 15 {
		t.Errorf("usage: %+v", got.Usage)
	}
	if got.Usage.PromptTokensDetails == nil || got.Usage.PromptTokensDetails.CachedTokens != 3 {
		t.Error("cached tokens mismatch")
	}
}

func TestStreamChunkDelta(t *testing.T) {
	input := `{"id":"chatcmpl-1","object":"chat.completion.chunk","created":1700000000,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":"hello"},"finish_reason":null}]}`
	var chunk StreamChunk
	if err := json.Unmarshal([]byte(input), &chunk); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(chunk.Choices) != 1 {
		t.Fatalf("choices: %d", len(chunk.Choices))
	}
	if chunk.Choices[0].Delta.Content == nil || *chunk.Choices[0].Delta.Content != "hello" {
		t.Error("delta content mismatch")
	}
	if chunk.Choices[0].FinishReason != nil {
		t.Error("finish_reason should be nil")
	}
}

func TestStreamChunkFinal(t *testing.T) {
	input := `{"id":"chatcmpl-1","object":"chat.completion.chunk","created":1700000000,"model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`
	var chunk StreamChunk
	if err := json.Unmarshal([]byte(input), &chunk); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if chunk.Choices[0].FinishReason == nil || *chunk.Choices[0].FinishReason != "stop" {
		t.Error("finish_reason should be stop")
	}
	if chunk.Usage == nil || chunk.Usage.TotalTokens != 15 {
		t.Errorf("usage: %+v", chunk.Usage)
	}
}

func TestKeyJSONRoundTrip(t *testing.T) {
	key := Key{Value: "sk-test", Provider: ProviderOpenAI, ConnID: "abc", AuthType: "api_key"}
	data, err := json.Marshal(key)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Key
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Value != "sk-test" || got.Provider != ProviderOpenAI {
		t.Errorf("key: %+v", got)
	}
}

func TestModelProviderString(t *testing.T) {
	if ProviderOpenAI.String() != "openai" {
		t.Errorf("got %q", ProviderOpenAI.String())
	}
	if ProviderAnthropic.String() != "anthropic" {
		t.Errorf("got %q", ProviderAnthropic.String())
	}
}
