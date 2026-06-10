package anthropic

import (
	"encoding/json"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

func TestNewProvider(t *testing.T) {
	p := NewProvider()
	if p.GetProvider() != schemas.ProviderAnthropic {
		t.Errorf("provider = %q, want anthropic", p.GetProvider())
	}
}

func TestProviderSetNetworkConfig(t *testing.T) {
	p := NewProvider()
	p.SetNetworkConfig(schemas.NetworkConfig{Timeout: 30, ProxyURL: "http://proxy"})
	if p.networkConfig.Timeout != 30 {
		t.Errorf("timeout = %d, want 30", p.networkConfig.Timeout)
	}
	if p.networkConfig.ProxyURL != "http://proxy" {
		t.Errorf("proxy = %q, want http://proxy", p.networkConfig.ProxyURL)
	}
}

func TestConvertRequestSimple(t *testing.T) {
	req := &schemas.ChatRequest{
		Model:    "claude-3-5-sonnet",
		Messages: []schemas.Message{{Role: "user", Content: "Hello"}},
	}
	converted := ConvertRequest(req)

	if converted.Model != "claude-3-5-sonnet" {
		t.Errorf("model = %q, want claude-3-5-sonnet", converted.Model)
	}
	if len(converted.Messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(converted.Messages))
	}
	if converted.Messages[0].Role != "user" {
		t.Errorf("role = %q, want user", converted.Messages[0].Role)
	}
	if len(converted.Messages[0].Content) != 1 {
		t.Fatalf("content blocks len = %d, want 1", len(converted.Messages[0].Content))
	}
	if converted.Messages[0].Content[0].Text != "Hello" {
		t.Errorf("content = %q, want Hello", converted.Messages[0].Content[0].Text)
	}
	// Anthropic requires max_tokens; default should be applied
	if converted.MaxTokens == 0 {
		t.Error("max_tokens should have a default value")
	}
}

func TestConvertRequestWithSystem(t *testing.T) {
	req := &schemas.ChatRequest{
		Model: "claude-3-5-sonnet",
		Messages: []schemas.Message{
			{Role: "system", Content: "Be helpful"},
			{Role: "user", Content: "Hi"},
		},
	}
	converted := ConvertRequest(req)

	if converted.System != "Be helpful" {
		t.Errorf("system = %q, want Be helpful", converted.System)
	}
	// System message should be extracted, not in messages
	for _, m := range converted.Messages {
		if m.Role == "system" {
			t.Error("system message should not appear in messages")
		}
	}
}

func TestConvertRequestWithTools(t *testing.T) {
	desc := "A test tool"
	req := &schemas.ChatRequest{
		Model: "claude-3-5-sonnet",
		Messages: []schemas.Message{
			{Role: "user", Content: "Call the tool"},
		},
		Tools: []schemas.Tool{
			{
				Type: "function",
				Function: schemas.FunctionDefinition{
					Name:        "test_fn",
					Description: &desc,
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"name": map[string]any{"type": "string"},
						},
					},
				},
			},
		},
	}
	converted := ConvertRequest(req)

	if len(converted.Tools) != 1 {
		t.Fatalf("tools len = %d, want 1", len(converted.Tools))
	}
	if converted.Tools[0].Name != "test_fn" {
		t.Errorf("tool name = %q, want test_fn", converted.Tools[0].Name)
	}
}

func TestConvertResponse(t *testing.T) {
	anthropicResp := &MessagesResponse{
		ID:         "msg_01",
		Type:       "message",
		Role:       "assistant",
		Model:      "claude-3-5-sonnet",
		StopReason: "end_turn",
		Content:    []ContentBlock{{Type: "text", Text: "Hello there"}},
		Usage:      Usage{InputTokens: 10, OutputTokens: 5},
	}

	converted := ConvertResponse(anthropicResp)

	if converted.ID != "msg_01" {
		t.Errorf("id = %q, want msg_01", converted.ID)
	}
	if converted.Model != "claude-3-5-sonnet" {
		t.Errorf("model = %q, want claude-3-5-sonnet", converted.Model)
	}
	if len(converted.Choices) != 1 {
		t.Fatalf("choices len = %d, want 1", len(converted.Choices))
	}
	if converted.Choices[0].Message.Content != "Hello there" {
		t.Errorf("content = %q, want Hello there", converted.Choices[0].Message.Content)
	}
	if converted.Choices[0].FinishReason != "stop" {
		t.Errorf("finish_reason = %q, want stop", converted.Choices[0].FinishReason)
	}
	if converted.Usage.PromptTokens != 10 {
		t.Errorf("prompt_tokens = %d, want 10", converted.Usage.PromptTokens)
	}
	if converted.Usage.CompletionTokens != 5 {
		t.Errorf("completion_tokens = %d, want 5", converted.Usage.CompletionTokens)
	}
}

func TestConvertResponseWithToolUse(t *testing.T) {
	anthropicResp := &MessagesResponse{
		ID:         "msg_02",
		Type:       "message",
		Role:       "assistant",
		Model:      "claude-3-5-sonnet",
		StopReason: "tool_use",
		Content: []ContentBlock{
			{Type: "text", Text: "I'll help"},
			{
				Type:  "tool_use",
				ID:    "toolu_01",
				Name:  "get_weather",
				Input: json.RawMessage(`{"city":"Paris"}`),
			},
		},
		Usage: Usage{InputTokens: 20, OutputTokens: 15},
	}

	converted := ConvertResponse(anthropicResp)

	if len(converted.Choices) != 1 {
		t.Fatalf("choices len = %d, want 1", len(converted.Choices))
	}
	msg := converted.Choices[0].Message
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("tool_calls len = %d, want 1", len(msg.ToolCalls))
	}
	if msg.ToolCalls[0].ID != "toolu_01" {
		t.Errorf("tool_call id = %q, want toolu_01", msg.ToolCalls[0].ID)
	}
	if msg.ToolCalls[0].Function.Name != "get_weather" {
		t.Errorf("tool_call name = %q, want get_weather", msg.ToolCalls[0].Function.Name)
	}
	if msg.ToolCalls[0].Function.Arguments != `{"city":"Paris"}` {
		t.Errorf("tool_call args = %q, want {\"city\":\"Paris\"}", msg.ToolCalls[0].Function.Arguments)
	}
	if converted.Choices[0].FinishReason != "tool_calls" {
		t.Errorf("finish_reason = %q, want tool_calls", converted.Choices[0].FinishReason)
	}
}

func TestConvertStreamEventToChunk(t *testing.T) {
	event := &StreamEvent{
		Type:  "content_block_delta",
		Index: 0,
		Delta: &StreamDelta{Type: "text_delta", Text: "Hello"},
	}

	chunk := ConvertStreamEventToChunk(event, "msg_01", "claude-3-5-sonnet")

	if chunk.ID != "msg_01" {
		t.Errorf("id = %q, want msg_01", chunk.ID)
	}
	if chunk.Model != "claude-3-5-sonnet" {
		t.Errorf("model = %q, want claude-3-5-sonnet", chunk.Model)
	}
	if len(chunk.Choices) != 1 {
		t.Fatalf("choices len = %d, want 1", len(chunk.Choices))
	}
	if chunk.Choices[0].Delta.Content != "Hello" {
		t.Errorf("delta content = %q, want Hello", chunk.Choices[0].Delta.Content)
	}
}

func TestConvertStreamEventStopReason(t *testing.T) {
	event := &StreamEvent{
		Type: "message_delta",
		Delta: &StreamDelta{
			StopReason: "end_turn",
		},
	}

	chunk := ConvertStreamEventToChunk(event, "msg_01", "claude-3-5-sonnet")
	if len(chunk.Choices) != 1 {
		t.Fatalf("choices len = %d, want 1", len(chunk.Choices))
	}
	if chunk.Choices[0].FinishReason == nil || *chunk.Choices[0].FinishReason != "stop" {
		reason := "nil"
		if chunk.Choices[0].FinishReason != nil {
			reason = *chunk.Choices[0].FinishReason
		}
		t.Errorf("finish_reason = %q, want stop", reason)
	}
}

func TestConvertStreamEventToChunkInputJSONDelta(t *testing.T) {
	fragments := []string{`{"city":`, `"Paris"}`}
	var concatenated string

	for i, frag := range fragments {
		event := &StreamEvent{
			Type:  "content_block_delta",
			Index: i,
			Delta: &StreamDelta{Type: "input_json_delta", PartialJSON: frag},
		}

		chunk := ConvertStreamEventToChunk(event, "msg_01", "claude-3-5-sonnet")

		if len(chunk.Choices) != 1 {
			t.Fatalf("choices len = %d, want 1", len(chunk.Choices))
		}

		if chunk.Choices[0].Delta.Content != "" {
			t.Errorf("delta content = %q, want empty", chunk.Choices[0].Delta.Content)
		}

		if len(chunk.Choices[0].Delta.ToolCalls) != 1 {
			t.Fatalf("tool_calls len = %d, want 1", len(chunk.Choices[0].Delta.ToolCalls))
		}

		if chunk.Choices[0].Delta.ToolCalls[0].Type != "function" {
			t.Errorf("tool_call type = %q, want function", chunk.Choices[0].Delta.ToolCalls[0].Type)
		}

		if chunk.Choices[0].Delta.ToolCalls[0].Function.Arguments != frag {
			t.Errorf("arguments = %q, want %q", chunk.Choices[0].Delta.ToolCalls[0].Function.Arguments, frag)
		}

		concatenated += frag
	}

	if concatenated != `{"city":"Paris"}` {
		t.Errorf("concatenated = %q, want {\"city\":\"Paris\"}", concatenated)
	}
}

func TestErrorConverter(t *testing.T) {
	ec := NewErrorConverter()
	body := []byte(`{"type":"error","error":{"type":"invalid_request_error","message":"bad request"}}`)
	meta := schemas.ErrorMeta{Provider: "anthropic", StatusCode: 400}
	err := ec.Convert(400, body, meta)

	if err.Message != "bad request" {
		t.Errorf("message = %q, want bad request", err.Message)
	}
	if err.Type != "invalid_request_error" {
		t.Errorf("type = %q, want invalid_request_error", err.Type)
	}
	if err.StatusCode != 400 {
		t.Errorf("status = %d, want 400", err.StatusCode)
	}
}

func TestErrorConverterMalformed(t *testing.T) {
	ec := NewErrorConverter()
	body := []byte(`not json`)
	meta := schemas.ErrorMeta{Provider: "anthropic", StatusCode: 500}
	err := ec.Convert(500, body, meta)

	if err.Type != "api_error" {
		t.Errorf("type = %q, want api_error", err.Type)
	}
	if err.StatusCode != 500 {
		t.Errorf("status = %d, want 500", err.StatusCode)
	}
}

func TestNotImplementedStubs(t *testing.T) {
	p := NewProvider()
	ctx := &schemas.GatewayContext{RequestID: "test-1"}
	key := schemas.Key{ID: "key-1", Provider: "anthropic", Value: "sk-test"}

	tests := []struct {
		name string
		err  *schemas.ProviderError
	}{
		{"TextCompletion", func() *schemas.ProviderError { _, e := p.TextCompletion(ctx, key, nil); return e }()},
		{"TextCompletionStream", func() *schemas.ProviderError { _, e := p.TextCompletionStream(ctx, nil, key, nil); return e }()},
		{"Responses", func() *schemas.ProviderError { _, e := p.Responses(ctx, key, nil); return e }()},
		{"ResponsesStream", func() *schemas.ProviderError { _, e := p.ResponsesStream(ctx, nil, key, nil); return e }()},
		{"ImageGeneration", func() *schemas.ProviderError { _, e := p.ImageGeneration(ctx, key, nil); return e }()},
		{"ImageEdit", func() *schemas.ProviderError { _, e := p.ImageEdit(ctx, key, nil); return e }()},
		{"ImageVariation", func() *schemas.ProviderError { _, e := p.ImageVariation(ctx, key, nil); return e }()},
		{"Speech", func() *schemas.ProviderError { _, e := p.Speech(ctx, key, nil); return e }()},
		{"Transcription", func() *schemas.ProviderError { _, e := p.Transcription(ctx, key, nil); return e }()},
		{"FileUpload", func() *schemas.ProviderError { _, e := p.FileUpload(ctx, key, nil); return e }()},
		{"FileList", func() *schemas.ProviderError { _, e := p.FileList(ctx, key); return e }()},
		{"FileRetrieve", func() *schemas.ProviderError { _, e := p.FileRetrieve(ctx, key, "f-1"); return e }()},
		{"FileDelete", func() *schemas.ProviderError { _, e := p.FileDelete(ctx, key, "f-1"); return e }()},
		{"FileContent", func() *schemas.ProviderError { _, e := p.FileContent(ctx, key, "f-1"); return e }()},
		{"BatchCreate", func() *schemas.ProviderError { _, e := p.BatchCreate(ctx, key, nil); return e }()},
		{"BatchList", func() *schemas.ProviderError { _, e := p.BatchList(ctx, key); return e }()},
		{"BatchRetrieve", func() *schemas.ProviderError { _, e := p.BatchRetrieve(ctx, key, "b-1"); return e }()},
		{"BatchCancel", func() *schemas.ProviderError { _, e := p.BatchCancel(ctx, key, "b-1"); return e }()},
		{"CountTokens", func() *schemas.ProviderError { _, e := p.CountTokens(ctx, key, nil); return e }()},
		{"Embedding", func() *schemas.ProviderError { _, e := p.Embedding(ctx, key, nil); return e }()},
		{"ListModels", func() *schemas.ProviderError { _, e := p.ListModels(ctx, key); return e }()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatalf("expected error, got nil")
			}
			if tt.err.Type != "not_implemented" {
				t.Errorf("type = %q, want not_implemented", tt.err.Type)
			}
			if tt.err.StatusCode != 501 {
				t.Errorf("status = %d, want 501", tt.err.StatusCode)
			}
		})
	}
}
