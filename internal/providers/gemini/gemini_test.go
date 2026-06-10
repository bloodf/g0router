package gemini

import (
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

func TestNewProvider(t *testing.T) {
	p := NewProvider()
	if p.GetProvider() != schemas.ProviderGemini {
		t.Errorf("provider = %q, want gemini", p.GetProvider())
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
		Model:    "gemini-1.5-pro",
		Messages: []schemas.Message{{Role: "user", Content: "Hello"}},
	}
	converted, err := ConvertChatRequest(req)
	if err != nil {
		t.Fatalf("ConvertChatRequest: %v", err)
	}

	if converted.Model != "gemini-1.5-pro" {
		t.Errorf("model = %q, want gemini-1.5-pro", converted.Model)
	}
	if len(converted.Contents) != 1 {
		t.Fatalf("contents len = %d, want 1", len(converted.Contents))
	}
	if converted.Contents[0].Role != "user" {
		t.Errorf("role = %q, want user", converted.Contents[0].Role)
	}
	if len(converted.Contents[0].Parts) != 1 {
		t.Fatalf("parts len = %d, want 1", len(converted.Contents[0].Parts))
	}
	if converted.Contents[0].Parts[0].Text != "Hello" {
		t.Errorf("text = %q, want Hello", converted.Contents[0].Parts[0].Text)
	}
}

func TestConvertRequestWithSystem(t *testing.T) {
	req := &schemas.ChatRequest{
		Model: "gemini-1.5-pro",
		Messages: []schemas.Message{
			{Role: "system", Content: "Be helpful"},
			{Role: "user", Content: "Hi"},
		},
	}
	converted, err := ConvertChatRequest(req)
	if err != nil {
		t.Fatalf("ConvertChatRequest: %v", err)
	}

	if converted.SystemInstruction == nil {
		t.Fatal("system_instruction should not be nil")
	}
	if len(converted.SystemInstruction.Parts) != 1 {
		t.Fatalf("system parts len = %d, want 1", len(converted.SystemInstruction.Parts))
	}
	if converted.SystemInstruction.Parts[0].Text != "Be helpful" {
		t.Errorf("system text = %q, want Be helpful", converted.SystemInstruction.Parts[0].Text)
	}
	for _, c := range converted.Contents {
		if c.Role == "system" {
			t.Error("system message should not appear in contents")
		}
	}
}

func TestConvertRequestAssistantRole(t *testing.T) {
	req := &schemas.ChatRequest{
		Model: "gemini-1.5-pro",
		Messages: []schemas.Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there"},
		},
	}
	converted, err := ConvertChatRequest(req)
	if err != nil {
		t.Fatalf("ConvertChatRequest: %v", err)
	}

	if len(converted.Contents) != 2 {
		t.Fatalf("contents len = %d, want 2", len(converted.Contents))
	}
	if converted.Contents[1].Role != "model" {
		t.Errorf("assistant role = %q, want model", converted.Contents[1].Role)
	}
}

func TestConvertRequestWithTools(t *testing.T) {
	desc := "A test tool"
	req := &schemas.ChatRequest{
		Model: "gemini-1.5-pro",
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
	converted, err := ConvertChatRequest(req)
	if err != nil {
		t.Fatalf("ConvertChatRequest: %v", err)
	}

	if len(converted.Tools) != 1 {
		t.Fatalf("tools len = %d, want 1", len(converted.Tools))
	}
	if len(converted.Tools[0].FunctionDeclarations) != 1 {
		t.Fatalf("function declarations len = %d, want 1", len(converted.Tools[0].FunctionDeclarations))
	}
	if converted.Tools[0].FunctionDeclarations[0].Name != "test_fn" {
		t.Errorf("tool name = %q, want test_fn", converted.Tools[0].FunctionDeclarations[0].Name)
	}
}

func TestConvertChatResponse(t *testing.T) {
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
		UsageMetadata: &UsageMetadata{
			PromptTokenCount:     10,
			CandidatesTokenCount: 5,
			TotalTokenCount:      15,
		},
	}

	converted := ConvertChatResponse(geminiResp, "gemini-1.5-pro")

	if converted.Model != "gemini-1.5-pro" {
		t.Errorf("model = %q, want gemini-1.5-pro", converted.Model)
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

func TestConvertChatResponseWithToolCall(t *testing.T) {
	geminiResp := &GenerateContentResponse{
		Candidates: []Candidate{
			{
				Content: Content{
					Role: "model",
					Parts: []Part{{
						FunctionCall: &FunctionCall{
							Name: "get_weather",
							Args: map[string]any{"city": "Paris"},
						},
					}},
				},
				FinishReason: "STOP",
			},
		},
		UsageMetadata: &UsageMetadata{
			PromptTokenCount:     20,
			CandidatesTokenCount: 15,
			TotalTokenCount:      35,
		},
	}

	converted := ConvertChatResponse(geminiResp, "gemini-1.5-pro")

	if len(converted.Choices) != 1 {
		t.Fatalf("choices len = %d, want 1", len(converted.Choices))
	}
	msg := converted.Choices[0].Message
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("tool_calls len = %d, want 1", len(msg.ToolCalls))
	}
	if msg.ToolCalls[0].Function.Name != "get_weather" {
		t.Errorf("tool_call name = %q, want get_weather", msg.ToolCalls[0].Function.Name)
	}
	if msg.ToolCalls[0].Function.Arguments != `{"city":"Paris"}` {
		t.Errorf("tool_call args = %q, want {\"city\":\"Paris\"}", msg.ToolCalls[0].Function.Arguments)
	}
}

func TestConvertEmbeddingRequest(t *testing.T) {
	req := &schemas.EmbeddingRequest{
		Model: "text-embedding-004",
		Input: "hello world",
	}
	converted := ConvertEmbeddingRequest(req)

	if converted.Model != "text-embedding-004" {
		t.Errorf("model = %q, want text-embedding-004", converted.Model)
	}
	if len(converted.Content.Parts) != 1 {
		t.Fatalf("parts len = %d, want 1", len(converted.Content.Parts))
	}
	if converted.Content.Parts[0].Text != "hello world" {
		t.Errorf("text = %q, want hello world", converted.Content.Parts[0].Text)
	}
}

func TestConvertEmbeddingResponse(t *testing.T) {
	geminiResp := &EmbedContentResponse{
		Embedding: Embedding{
			Values: []float32{0.1, 0.2, 0.3},
		},
	}

	converted := ConvertEmbeddingResponse(geminiResp, "text-embedding-004")

	if converted.Model != "text-embedding-004" {
		t.Errorf("model = %q, want text-embedding-004", converted.Model)
	}
	if len(converted.Data) != 1 {
		t.Fatalf("data len = %d, want 1", len(converted.Data))
	}
	if len(converted.Data[0].Embedding) != 3 {
		t.Fatalf("embedding len = %d, want 3", len(converted.Data[0].Embedding))
	}
	if converted.Data[0].Embedding[0] < 0.09 || converted.Data[0].Embedding[0] > 0.11 {
		t.Errorf("embedding[0] = %f, want ~0.1", converted.Data[0].Embedding[0])
	}
}

func TestConvertStreamChunk(t *testing.T) {
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
	if chunk.Model != "gemini-1.5-pro" {
		t.Errorf("model = %q, want gemini-1.5-pro", chunk.Model)
	}
	if len(chunk.Choices) != 1 {
		t.Fatalf("choices len = %d, want 1", len(chunk.Choices))
	}
	if chunk.Choices[0].Delta.Content != "Hello" {
		t.Errorf("delta content = %q, want Hello", chunk.Choices[0].Delta.Content)
	}
}

func TestErrorConverter(t *testing.T) {
	ec := NewErrorConverter()
	body := []byte(`{"error":{"code":400,"message":"bad request","status":"INVALID_ARGUMENT"}}`)
	meta := schemas.ErrorMeta{Provider: "gemini", StatusCode: 400}
	err := ec.Convert(400, body, meta)

	if err.Message != "bad request" {
		t.Errorf("message = %q, want bad request", err.Message)
	}
	if err.StatusCode != 400 {
		t.Errorf("status = %d, want 400", err.StatusCode)
	}
}

func TestErrorConverterMalformed(t *testing.T) {
	ec := NewErrorConverter()
	body := []byte(`not json`)
	meta := schemas.ErrorMeta{Provider: "gemini", StatusCode: 500}
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
	key := schemas.Key{ID: "key-1", Provider: "gemini", Value: "test-key"}

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
