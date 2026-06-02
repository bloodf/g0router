package translate

import (
	"encoding/json"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

func TestNormalizeOpenAILeavesOpenAIRequestsUnchanged(t *testing.T) {
	body := []byte(`{
		"model": "gpt-4o-mini",
		"messages": [{"role": "user", "content": "hi"}],
		"max_completion_tokens": 128
	}`)

	req, format, err := NormalizeOpenAI(body)
	if err != nil {
		t.Fatalf("NormalizeOpenAI() error = %v", err)
	}
	if format != FormatOpenAI {
		t.Fatalf("format = %q, want %q", format, FormatOpenAI)
	}
	if req.Model != "gpt-4o-mini" {
		t.Fatalf("model = %q", req.Model)
	}
	if len(req.Messages) != 1 || req.Messages[0].Role != "user" {
		t.Fatalf("messages = %+v", req.Messages)
	}
	if req.MaxCompletionTokens == nil || *req.MaxCompletionTokens != 128 {
		t.Fatalf("max_completion_tokens = %+v", req.MaxCompletionTokens)
	}
}

func TestNormalizeOpenAIConvertsAnthropicSystemToSystemMessage(t *testing.T) {
	body := []byte(`{
		"model": "claude-sonnet-4-20250514",
		"system": "answer tersely",
		"max_tokens": 256,
		"messages": [{"role": "user", "content": "hi"}]
	}`)

	req, format, err := NormalizeOpenAI(body)
	if err != nil {
		t.Fatalf("NormalizeOpenAI() error = %v", err)
	}
	if format != FormatAnthropic {
		t.Fatalf("format = %q, want %q", format, FormatAnthropic)
	}
	if len(req.Messages) != 2 {
		t.Fatalf("messages len = %d", len(req.Messages))
	}
	if req.Messages[0].Role != "system" || req.Messages[0].Content != "answer tersely" {
		t.Fatalf("system message = %+v", req.Messages[0])
	}
	if req.Messages[1].Role != "user" || req.Messages[1].Content != "hi" {
		t.Fatalf("user message = %+v", req.Messages[1])
	}
	if req.MaxTokens == nil || *req.MaxTokens != 256 {
		t.Fatalf("max_tokens = %+v", req.MaxTokens)
	}
	if req.System != nil {
		t.Fatalf("system should be nil after normalization, got %#v", req.System)
	}
}

func TestOpenAIToAnthropicExtractsSystemMessages(t *testing.T) {
	req := &providers.ChatRequest{
		Model: "claude-sonnet-4-20250514",
		Messages: []providers.Message{
			{Role: "system", Content: "first"},
			{Role: "user", Content: "hi"},
			{Role: "system", Content: "second"},
			{Role: "assistant", Content: "hello"},
		},
	}

	got, err := OpenAIToAnthropic(req)
	if err != nil {
		t.Fatalf("OpenAIToAnthropic() error = %v", err)
	}
	if got.System != "first\n\nsecond" {
		t.Fatalf("system = %#v", got.System)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("messages len = %d", len(got.Messages))
	}
	if got.Messages[0].Role != "user" || got.Messages[1].Role != "assistant" {
		t.Fatalf("messages = %+v", got.Messages)
	}
}

func TestOpenAIToAnthropicPreservesContentBlocks(t *testing.T) {
	content := []map[string]any{
		{"type": "thinking", "thinking": "private reasoning", "signature": "sig"},
		{"type": "text", "text": "visible answer"},
	}
	req := &providers.ChatRequest{
		Model: "claude-sonnet-4-20250514",
		Messages: []providers.Message{
			{Role: "assistant", Content: content},
		},
	}

	got, err := OpenAIToAnthropic(req)
	if err != nil {
		t.Fatalf("OpenAIToAnthropic() error = %v", err)
	}
	data, err := json.Marshal(got.Messages[0].Content)
	if err != nil {
		t.Fatalf("marshal content: %v", err)
	}
	if string(data) != `[{"signature":"sig","thinking":"private reasoning","type":"thinking"},{"text":"visible answer","type":"text"}]` {
		t.Fatalf("content JSON = %s", data)
	}
}

func TestOpenAIToAnthropicNilRequest(t *testing.T) {
	_, err := OpenAIToAnthropic(nil)
	if err == nil {
		t.Fatal("OpenAIToAnthropic() error = nil")
	}
}
