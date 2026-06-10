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

// TestFixMissingToolResponsesInserts verifies that when an assistant message
// has tool_calls and the next message does not contain tool results for those
// IDs, empty role:tool messages are inserted directly after the assistant.
func TestFixMissingToolResponsesInserts(t *testing.T) {
	req := &schemas.ChatRequest{
		Messages: []schemas.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", ToolCalls: []schemas.ToolCall{
				{ID: "tc1", Type: "function", Function: schemas.FunctionCall{Name: "f1", Arguments: "{}"}},
				{ID: "tc2", Type: "function", Function: schemas.FunctionCall{Name: "f2", Arguments: "{}"}},
			}},
			{Role: "user", Content: "next"},
		},
	}

	FixMissingToolResponses(req)

	if len(req.Messages) != 5 {
		t.Fatalf("len(messages) = %d, want 5", len(req.Messages))
	}
	if req.Messages[2].Role != "tool" || *req.Messages[2].ToolCallID != "tc1" || req.Messages[2].Content != "" {
		t.Errorf("msg[2] = %+v, want role:tool tool_call_id:tc1 content:\"\"", req.Messages[2])
	}
	if req.Messages[3].Role != "tool" || *req.Messages[3].ToolCallID != "tc2" || req.Messages[3].Content != "" {
		t.Errorf("msg[3] = %+v, want role:tool tool_call_id:tc2 content:\"\"", req.Messages[3])
	}
}

// TestFixMissingToolResponsesSkipsAnswered verifies that when the next message
// already answers the tool calls (via tool_call_id), nothing is inserted.
func TestFixMissingToolResponsesSkipsAnswered(t *testing.T) {
	req := &schemas.ChatRequest{
		Messages: []schemas.Message{
			{Role: "assistant", ToolCalls: []schemas.ToolCall{
				{ID: "tc1", Type: "function", Function: schemas.FunctionCall{Name: "f1", Arguments: "{}"}},
			}},
			{Role: "tool", ToolCallID: strPtr("tc1"), Content: "done"},
		},
	}

	FixMissingToolResponses(req)

	if len(req.Messages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(req.Messages))
	}
}

// TestFixMissingToolResponsesIgnoresTrailing verifies that when assistant
// tool_calls are the final message, no empty tool responses are appended.
func TestFixMissingToolResponsesIgnoresTrailing(t *testing.T) {
	req := &schemas.ChatRequest{
		Messages: []schemas.Message{
			{Role: "assistant", ToolCalls: []schemas.ToolCall{
				{ID: "tc1", Type: "function", Function: schemas.FunctionCall{Name: "f1", Arguments: "{}"}},
			}},
		},
	}

	FixMissingToolResponses(req)

	if len(req.Messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(req.Messages))
	}
}

// TestNormalizeThinkingConfigClears verifies that when the last message is not
// from the user, Thinking and ReasoningEffort are cleared.
func TestNormalizeThinkingConfigClears(t *testing.T) {
	thinking := schemas.ThinkingConfig{Type: "enabled", BudgetTokens: 1024}
	req := &schemas.ChatRequest{
		Messages: []schemas.Message{
			{Role: "user", Content: "hi"},
			{Role: "assistant", Content: "hello"},
		},
		ReasoningEffort: "medium",
		Thinking:        &thinking,
	}

	NormalizeThinkingConfig(req)

	if req.Thinking != nil {
		t.Errorf("thinking = %+v, want nil", req.Thinking)
	}
	if req.ReasoningEffort != "" {
		t.Errorf("reasoning_effort = %q, want empty", req.ReasoningEffort)
	}
}

// TestNormalizeThinkingConfigKeepsForUser verifies that when the last message
// is from the user, Thinking and ReasoningEffort are left untouched.
func TestNormalizeThinkingConfigKeepsForUser(t *testing.T) {
	thinking := schemas.ThinkingConfig{Type: "enabled", BudgetTokens: 1024}
	req := &schemas.ChatRequest{
		Messages: []schemas.Message{
			{Role: "user", Content: "hi"},
		},
		ReasoningEffort: "medium",
		Thinking:        &thinking,
	}

	NormalizeThinkingConfig(req)

	if req.Thinking == nil || req.Thinking.Type != "enabled" {
		t.Errorf("thinking = %+v, want kept", req.Thinking)
	}
	if req.ReasoningEffort != "medium" {
		t.Errorf("reasoning_effort = %q, want %q", req.ReasoningEffort, "medium")
	}
}
