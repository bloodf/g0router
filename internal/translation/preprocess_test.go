package translation

import (
	"encoding/json"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

// TestChatRequestNewFieldsRoundTrip verifies that the newly added schema
// fields (top_k, max_completion_tokens, reasoning_effort, thinking) survive
// a JSON marshal/unmarshal round-trip.
func TestChatRequestNewFieldsRoundTrip(t *testing.T) {
	topK := 40
	maxComp := 4096
	thinking := schemas.ThinkingConfig{
		Type:         "enabled",
		BudgetTokens: 8192,
	}

	req := schemas.ChatRequest{
		Model:               "gpt-4",
		Messages:            []schemas.Message{{Role: "user", Content: "hi"}},
		TopK:                &topK,
		MaxCompletionTokens: &maxComp,
		ReasoningEffort:     "medium",
		Thinking:            &thinking,
	}

	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got schemas.ChatRequest
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.TopK == nil || *got.TopK != 40 {
		t.Errorf("top_k = %v, want 40", got.TopK)
	}
	if got.MaxCompletionTokens == nil || *got.MaxCompletionTokens != 4096 {
		t.Errorf("max_completion_tokens = %v, want 4096", got.MaxCompletionTokens)
	}
	if got.ReasoningEffort != "medium" {
		t.Errorf("reasoning_effort = %q, want %q", got.ReasoningEffort, "medium")
	}
	if got.Thinking == nil || got.Thinking.Type != "enabled" || got.Thinking.BudgetTokens != 8192 {
		t.Errorf("thinking = %+v, want {Type:enabled BudgetTokens:8192}", got.Thinking)
	}
}
