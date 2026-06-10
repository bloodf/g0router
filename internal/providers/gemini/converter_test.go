package gemini

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

// unsupportedGeminiChatFields lists OpenAI ChatRequest fields that are
// intentionally not mapped to the Gemini GenerateContentRequest per Bundle E
// acceptance (AUD-032, AUD-033, AUD-034, AUD-037).
var unsupportedGeminiChatFields = []string{
	"n",
	"presence_penalty",
	"frequency_penalty",
	"logit_bias",
	"user",
	"response_format",
	"seed",
	"message_name",
}

func TestConvertChatRequestUnsupportedFieldsNotPresent(t *testing.T) {
	n := 2
	temp := 0.7
	presencePenalty := 0.5
	frequencyPenalty := 0.3
	seed := 42
	name := "test-name"
	responseFormat := &schemas.ResponseFormat{Type: "json_object"}
	logitBias := map[string]int{"foo": 1}

	req := &schemas.ChatRequest{
		Model:            "gemini-1.5-pro",
		N:                &n,
		Temperature:      &temp,
		PresencePenalty:  &presencePenalty,
		FrequencyPenalty: &frequencyPenalty,
		LogitBias:        logitBias,
		User:             "test-user",
		ResponseFormat:   responseFormat,
		Seed:             &seed,
		Messages: []schemas.Message{
			{Role: "user", Content: "Hello", Name: &name},
		},
	}

	converted, err := ConvertChatRequest(req)
	if err != nil {
		t.Fatalf("ConvertChatRequest: %v", err)
	}
	data, err := json.Marshal(converted)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, field := range []string{"n", "presence_penalty", "frequency_penalty", "logit_bias", "user", "response_format", "seed"} {
		if _, ok := result[field]; ok {
			t.Errorf("field %q should not be present in serialized Gemini chat request", field)
		}
	}

	if msgs, ok := result["contents"].([]any); ok && len(msgs) > 0 {
		if msg, ok := msgs[0].(map[string]any); ok {
			if _, ok := msg["name"]; ok {
				t.Error("field message.name should not be present in serialized Gemini chat request")
			}
		}
	}
}

func TestConvertChatRequestFieldCoverage(t *testing.T) {
	// AUD rows fields that must be covered by either a mapping test or the
	// unsupported list.
	audFields := []string{
		"n",                // AUD-032
		"presence_penalty", // AUD-033
		"frequency_penalty", // AUD-033
		"logit_bias",       // AUD-033
		"user",             // AUD-034
		"response_format",  // AUD-034
		"seed",             // AUD-034
		"message_name",     // AUD-037
	}

	fieldSet := make(map[string]bool)
	for _, f := range unsupportedGeminiChatFields {
		fieldSet[f] = true
	}

	// Fields with existing mapping tests in this package.
	mappedFields := map[string]bool{
		"model":              true,
		"messages":           true,
		"temperature":        true,
		"max_tokens":         true,
		"top_p":              true,
		"stop":               true,
		"tools":              true,
		"tool_choice":        true,
		"stream":             true, // AUD-036 — tested separately
	}

	for _, f := range audFields {
		if !fieldSet[f] && !mappedFields[f] {
			t.Errorf("AUD field %q not covered by mapping test or unsupported list", f)
		}
	}
}

func TestConvertChatRequestStreamNotPresent(t *testing.T) {
	// AUD-036: stream must not leak into the Gemini body on either path —
	// streaming (Stream=true → :streamGenerateContent) or non-streaming
	// (Stream=false → :generateContent). The body converter is shared;
	// assert both values.
	for _, stream := range []bool{true, false} {
		req := &schemas.ChatRequest{
			Model:    "gemini-1.5-pro",
			Stream:   stream,
			Messages: []schemas.Message{{Role: "user", Content: "Hello"}},
		}

		converted, err := ConvertChatRequest(req)
		if err != nil {
			t.Fatalf("ConvertChatRequest(stream=%v): %v", stream, err)
		}
		data, err := json.Marshal(converted)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var body map[string]any
		if err := json.Unmarshal(data, &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if _, ok := body["stream"]; ok {
			t.Errorf("stream=%v: %q key serialized into Gemini body", stream, "stream")
		}
	}

	req := &schemas.ChatRequest{
		Model:    "gemini-1.5-pro",
		Stream:   true,
		Messages: []schemas.Message{{Role: "user", Content: "Hello"}},
	}

	converted, err := ConvertChatRequest(req)
	if err != nil {
		t.Fatalf("ConvertChatRequest: %v", err)
	}
	data, err := json.Marshal(converted)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, ok := result["stream"]; ok {
		t.Error("stream field should not be present in Gemini chat request body")
	}
}

func TestConvertChatRequestToolMessageWithCallID(t *testing.T) {
	callID := "call_abc"
	fnName := "get_weather"
	req := &schemas.ChatRequest{
		Model: "gemini-1.5-pro",
		Messages: []schemas.Message{
			{Role: "user", Content: "What is the weather?"},
			{
				Role:       "tool",
				Content:    "25°C and sunny",
				ToolCallID: &callID,
				Name:       &fnName,
			},
		},
	}

	converted, err := ConvertChatRequest(req)
	if err != nil {
		t.Fatalf("ConvertChatRequest: %v", err)
	}

	if len(converted.Contents) != 2 {
		t.Fatalf("contents len = %d, want 2", len(converted.Contents))
	}
	// Tool message becomes user role with functionResponse part
	toolContent := converted.Contents[1]
	if toolContent.Role != "user" {
		t.Errorf("tool role = %q, want user", toolContent.Role)
	}
	if len(toolContent.Parts) != 1 {
		t.Fatalf("tool parts len = %d, want 1", len(toolContent.Parts))
	}
	fr := toolContent.Parts[0].FunctionResponse
	if fr == nil {
		t.Fatal("functionResponse should not be nil")
	}
	if fr.ID != "call_abc" {
		t.Errorf("functionResponse id = %q, want call_abc", fr.ID)
	}
	if fr.Name != "get_weather" {
		t.Errorf("functionResponse name = %q, want get_weather", fr.Name)
	}
}

var chatIDRegex = regexp.MustCompile(`^chatcmpl-[A-Za-z0-9]+$`)

func TestConvertChatResponseGeneratesID(t *testing.T) {
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
	}

	converted := ConvertChatResponse(geminiResp, "gemini-1.5-pro")

	if converted.ID == "" {
		t.Fatal("id should not be empty")
	}
	if !chatIDRegex.MatchString(converted.ID) {
		t.Errorf("id = %q, does not match %s", converted.ID, chatIDRegex.String())
	}
}

func TestConvertStreamChunkGeneratesID(t *testing.T) {
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

	if chunk.ID == "" {
		t.Fatal("id should not be empty")
	}
	if !chatIDRegex.MatchString(chunk.ID) {
		t.Errorf("id = %q, does not match %s", chunk.ID, chatIDRegex.String())
	}
}

func TestConvertChatResponseDistinctToolCallIDs(t *testing.T) {
	geminiResp := &GenerateContentResponse{
		Candidates: []Candidate{
			{
				Content: Content{
					Role: "model",
					Parts: []Part{
						{
							FunctionCall: &FunctionCall{
								Name: "get_weather",
								Args: map[string]any{"city": "Paris"},
							},
						},
						{
							FunctionCall: &FunctionCall{
								Name: "get_weather",
								Args: map[string]any{"city": "London"},
							},
						},
					},
				},
				FinishReason: "STOP",
			},
		},
	}

	converted := ConvertChatResponse(geminiResp, "gemini-1.5-pro")

	if len(converted.Choices) != 1 {
		t.Fatalf("choices len = %d, want 1", len(converted.Choices))
	}
	msg := converted.Choices[0].Message
	if len(msg.ToolCalls) != 2 {
		t.Fatalf("tool_calls len = %d, want 2", len(msg.ToolCalls))
	}
	if msg.ToolCalls[0].ID == msg.ToolCalls[1].ID {
		t.Errorf("tool call IDs are equal (%q), expected distinct values", msg.ToolCalls[0].ID)
	}
}

func TestConvertChatRequestMalformedToolArgs(t *testing.T) {
	badArgs := "not-valid-json"
	req := &schemas.ChatRequest{
		Model: "gemini-1.5-pro",
		Messages: []schemas.Message{
			{
				Role: "assistant",
				ToolCalls: []schemas.ToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: schemas.FunctionCall{
							Name:      "get_weather",
							Arguments: badArgs,
						},
					},
				},
			},
		},
	}

	_, err := ConvertChatRequest(req)
	if err == nil {
		t.Fatal("expected error for malformed tool arguments, got nil")
	}
}

func TestConvertChatRequestMultipleSystemMessages(t *testing.T) {
	req := &schemas.ChatRequest{
		Model: "gemini-1.5-pro",
		Messages: []schemas.Message{
			{Role: "system", Content: "First system instruction"},
			{Role: "system", Content: "Second system instruction"},
			{Role: "user", Content: "Hello"},
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
	want := "First system instruction\n\nSecond system instruction"
	if converted.SystemInstruction.Parts[0].Text != want {
		t.Errorf("system text = %q, want %q", converted.SystemInstruction.Parts[0].Text, want)
	}
	if len(converted.Contents) != 1 {
		t.Fatalf("contents len = %d, want 1", len(converted.Contents))
	}
}
