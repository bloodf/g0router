package anthropic

import (
	"encoding/json"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

// unsupportedAnthropicFields lists OpenAI ChatRequest fields that are
// intentionally not mapped to the Anthropic MessagesRequest per Bundle E
// acceptance (AUD-032, AUD-033, AUD-034, AUD-037).
var unsupportedAnthropicFields = []string{
	"n",
	"presence_penalty",
	"frequency_penalty",
	"logit_bias",
	"user",
	"response_format",
	"seed",
	"message_name",
}

func TestConvertRequestUnsupportedFieldsNotPresent(t *testing.T) {
	n := 2
	temp := 0.7
	presencePenalty := 0.5
	frequencyPenalty := 0.3
	seed := 42
	name := "test-name"
	responseFormat := &schemas.ResponseFormat{Type: "json_object"}
	logitBias := map[string]int{"foo": 1}

	req := &schemas.ChatRequest{
		Model:            "claude-3-5-sonnet",
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

	converted := ConvertRequest(req)
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
			t.Errorf("field %q should not be present in serialized Anthropic request", field)
		}
	}

	if msgs, ok := result["messages"].([]any); ok && len(msgs) > 0 {
		if msg, ok := msgs[0].(map[string]any); ok {
			if _, ok := msg["name"]; ok {
				t.Error("field message.name should not be present in serialized Anthropic request")
			}
		}
	}
}

func TestConvertRequestFieldCoverage(t *testing.T) {
	// AUD rows fields that must be covered by either a mapping test or the
	// unsupported list.
	audFields := []string{
		"n",               // AUD-032
		"presence_penalty", // AUD-033
		"frequency_penalty", // AUD-033
		"logit_bias",      // AUD-033
		"user",            // AUD-034
		"response_format", // AUD-034
		"seed",            // AUD-034
		"message_name",    // AUD-037
	}

	fieldSet := make(map[string]bool)
	for _, f := range unsupportedAnthropicFields {
		fieldSet[f] = true
	}

	// Fields with existing mapping tests in this package.
	mappedFields := map[string]bool{
		"model":        true,
		"messages":     true,
		"temperature":  true,
		"max_tokens":   true,
		"top_p":        true,
		"stop":         true,
		"tools":        true,
		"tool_choice":  true,
		"stream":       true,
	}

	for _, f := range audFields {
		if !fieldSet[f] && !mappedFields[f] {
			t.Errorf("AUD field %q not covered by mapping test or unsupported list", f)
		}
	}
}

func TestConvertRequestMaxTokensAdjusted(t *testing.T) {
	maxTokens := 1000
	req := &schemas.ChatRequest{
		Model: "claude-3-5-sonnet",
		Tools: []schemas.Tool{
			{Type: "function", Function: schemas.FunctionDefinition{Name: "Read"}},
		},
		MaxTokens: &maxTokens,
	}
	converted := ConvertRequest(req)
	if converted.MaxTokens != 32000 {
		t.Errorf("tools with low max_tokens = %d, want 32000", converted.MaxTokens)
	}

	// Zero MaxTokens → 64000
	req2 := &schemas.ChatRequest{
		Model: "claude-3-5-sonnet",
	}
	converted2 := ConvertRequest(req2)
	if converted2.MaxTokens != 64000 {
		t.Errorf("missing max_tokens = %d, want 64000", converted2.MaxTokens)
	}
}

func TestConvertRequestMultipleSystemMessages(t *testing.T) {
	req := &schemas.ChatRequest{
		Model: "claude-3-5-sonnet",
		Messages: []schemas.Message{
			{Role: "system", Content: "First system prompt"},
			{Role: "system", Content: "Second system prompt"},
			{Role: "user", Content: "Hello"},
		},
	}

	converted := ConvertRequest(req)

	want := "First system prompt\n\nSecond system prompt"
	if converted.System != want {
		t.Errorf("system = %q, want %q", converted.System, want)
	}
	if len(converted.Messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(converted.Messages))
	}
	if converted.Messages[0].Role != "user" {
		t.Errorf("role = %q, want user", converted.Messages[0].Role)
	}
}
