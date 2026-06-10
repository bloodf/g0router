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

// TestEnsureToolCallIDsSanitizesInvalid verifies that assistant tool_call IDs
// containing invalid characters are sanitized to keep only [a-zA-Z0-9_-].
func TestEnsureToolCallIDsSanitizesInvalid(t *testing.T) {
	req := &schemas.ChatRequest{
		Messages: []schemas.Message{
			{Role: "assistant", ToolCalls: []schemas.ToolCall{
				{ID: "abc!@#def", Type: "", Function: schemas.FunctionCall{Name: "foo", Arguments: "{}"}},
			}},
		},
	}

	EnsureToolCallIDs(req)

	if got := req.Messages[0].ToolCalls[0].ID; got != "abcdef" {
		t.Errorf("sanitized ID = %q, want %q", got, "abcdef")
	}
}

// TestEnsureToolCallIDsRegeneratesEmpty verifies that when an invalid ID
// becomes empty after sanitization, a deterministic ID is regenerated.
func TestEnsureToolCallIDsRegeneratesEmpty(t *testing.T) {
	req := &schemas.ChatRequest{
		Messages: []schemas.Message{
			{Role: "assistant", ToolCalls: []schemas.ToolCall{
				{ID: "!!!", Type: "", Function: schemas.FunctionCall{Name: "my-tool", Arguments: "{}"}},
			}},
		},
	}

	EnsureToolCallIDs(req)

	want := "call_msg0_tc0_my-tool"
	if got := req.Messages[0].ToolCalls[0].ID; got != want {
		t.Errorf("regenerated ID = %q, want %q", got, want)
	}
}

// TestEnsureToolCallIDsKeepsValid verifies that IDs already matching the
// allowed pattern are left untouched.
func TestEnsureToolCallIDsKeepsValid(t *testing.T) {
	req := &schemas.ChatRequest{
		Messages: []schemas.Message{
			{Role: "assistant", ToolCalls: []schemas.ToolCall{
				{ID: "valid_ID-123", Type: "", Function: schemas.FunctionCall{Name: "foo", Arguments: "{}"}},
			}},
		},
	}

	EnsureToolCallIDs(req)

	if got := req.Messages[0].ToolCalls[0].ID; got != "valid_ID-123" {
		t.Errorf("valid ID modified = %q, want %q", got, "valid_ID-123")
	}
}

// TestEnsureToolCallIDsFixesToolMessages verifies that role:tool messages
// with an invalid tool_call_id are sanitized/regenerated the same way.
func TestEnsureToolCallIDsFixesToolMessages(t *testing.T) {
	req := &schemas.ChatRequest{
		Messages: []schemas.Message{
			{Role: "tool", ToolCallID: strPtr("bad!id"), Content: "result"},
		},
	}

	EnsureToolCallIDs(req)

	if got := *req.Messages[0].ToolCallID; got != "badid" {
		t.Errorf("tool_call_id = %q, want %q", got, "badid")
	}
}

// TestEnsureToolCallIDsSetsType verifies that a missing Type field on
// assistant tool_calls is set to "function".
func TestEnsureToolCallIDsSetsType(t *testing.T) {
	req := &schemas.ChatRequest{
		Messages: []schemas.Message{
			{Role: "assistant", ToolCalls: []schemas.ToolCall{
				{ID: "x", Type: "", Function: schemas.FunctionCall{Name: "foo", Arguments: "{}"}},
			}},
		},
	}

	EnsureToolCallIDs(req)

	if got := req.Messages[0].ToolCalls[0].Type; got != "function" {
		t.Errorf("type = %q, want %q", got, "function")
	}
}

func strPtr(s string) *string { return &s }
