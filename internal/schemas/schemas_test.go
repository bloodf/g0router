package schemas

import (
	"encoding/json"
	"testing"
)

func ptrFloat64(v float64) *float64 { return &v }
func ptrInt(v int) *int             { return &v }
func ptrStr(v string) *string       { return &v }

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

func TestImageGenerationRequestRoundTrip(t *testing.T) {
	req := ImageGenerationRequest{
		Prompt: "a cat",
		Model:  "dall-e-3",
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ImageGenerationRequest
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Prompt != "a cat" {
		t.Errorf("prompt = %q, want a cat", got.Prompt)
	}
	if got.Model != "dall-e-3" {
		t.Errorf("model = %q, want dall-e-3", got.Model)
	}
}

func TestTranscriptionResponseRoundTrip(t *testing.T) {
	resp := TranscriptionResponse{
		Text: "hello world",
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got TranscriptionResponse
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Text != "hello world" {
		t.Errorf("text = %q, want hello world", got.Text)
	}
}

func TestFileObjectRoundTrip(t *testing.T) {
	f := FileObject{
		ID:       "file-123",
		Filename: "data.json",
		Purpose:  "batch",
	}
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got FileObject
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != "file-123" {
		t.Errorf("id = %q, want file-123", got.ID)
	}
	if got.Filename != "data.json" {
		t.Errorf("filename = %q, want data.json", got.Filename)
	}
}

func TestBatchRoundTrip(t *testing.T) {
	batch := Batch{
		ID:          "batch-123",
		Status:      "completed",
		InputFileID: "file-abc",
	}
	b, err := json.Marshal(batch)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Batch
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != "batch-123" {
		t.Errorf("id = %q, want batch-123", got.ID)
	}
	if got.Status != "completed" {
		t.Errorf("status = %q, want completed", got.Status)
	}
}

func TestResponsesRequestRoundTrip(t *testing.T) {
	req := ResponsesRequest{
		Model: "gpt-4o",
		Input: "hello",
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ResponsesRequest
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Model != "gpt-4o" {
		t.Errorf("model = %q, want gpt-4o", got.Model)
	}
}

func TestErrorResponseRoundTrip(t *testing.T) {
	resp := ErrorResponse{
		Error: APIError{
			Message: "bad request",
			Type:    "invalid_request_error",
			Code:    ptrStr("400"),
		},
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ErrorResponse
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Error.Message != "bad request" {
		t.Errorf("message = %q, want bad request", got.Error.Message)
	}
	if got.Error.Code == nil || *got.Error.Code != "400" {
		t.Errorf("code = %v, want 400", got.Error.Code)
	}
}

func TestSchemaTypesCompile(t *testing.T) {
	_ = ChatRequest{}
	_ = ChatResponse{}
	_ = TextCompletionRequest{}
	_ = TextCompletionResponse{}
	_ = EmbeddingRequest{}
	_ = EmbeddingResponse{}
	_ = ImageGenerationRequest{}
	_ = ImageGenerationResponse{}
	_ = SpeechRequest{}
	_ = SpeechResponse{}
	_ = TranscriptionRequest{}
	_ = TranscriptionResponse{}
	_ = FileObject{}
	_ = FileUploadRequest{}
	_ = Batch{}
	_ = BatchCreateRequest{}
	_ = ResponsesRequest{}
	_ = ResponsesResponse{}
	_ = APIError{}
	_ = ErrorResponse{}
	_ = ProviderError{}
	_ = GatewayContext{}
	_ = Key{}
	_ = NetworkConfig{}
	_ = ListModelsResponse{}
	_ = ModelEntry{}
	_ = TokenCountResponse{}
	_ = VirtualKey{}
	_ = ProviderConfig{}
	_ = Budget{}
	_ = PricingEntry{}
	_ = ModelCapability{}
	_ = Cost{}
	_ = MCPClient{}
	_ = MCPInstance{}
	_ = MCPTool{}
	_ = MCPToolGroup{}
}
