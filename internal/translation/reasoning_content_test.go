package translation

import (
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

// TestInjectReasoningContent verifies model-name based reasoning_content
// placeholder injection for DeepSeek and Kimi models.
func TestInjectReasoningContent(t *testing.T) {
	cases := []struct {
		name     string
		model    string
		messages []schemas.Message
		want     []string // expected reasoning_content values by message index
	}{
		{
			name:  "deepseek injects on assistant messages",
			model: "deepseek-chat",
			messages: []schemas.Message{
				{Role: "user", Content: "hi"},
				{Role: "assistant", Content: "hello"},
				{Role: "assistant", Content: "again"},
			},
			want: []string{"", " ", " "},
		},
		{
			name:  "deepseek skips non-assistant and existing reasoning_content",
			model: "deepseek-reasoner",
			messages: []schemas.Message{
				{Role: "user", Content: "hi"},
				{Role: "assistant", Content: "hello", ReasoningContent: strPtr("already")},
				{Role: "system", Content: "sys"},
			},
			want: []string{"", "already", ""},
		},
		{
			name:  "kimi injects only on assistant messages with tool_calls",
			model: "kimi-k1.5",
			messages: []schemas.Message{
				{Role: "user", Content: "hi"},
				{Role: "assistant", Content: "hello"},
				{Role: "assistant", Content: "call", ToolCalls: []schemas.ToolCall{
					{ID: "tc1", Type: "function", Function: schemas.FunctionCall{Name: "f"}},
				}},
			},
			want: []string{"", "", " "},
		},
		{
			name:  "other model is no-op",
			model: "gpt-4",
			messages: []schemas.Message{
				{Role: "assistant", Content: "hello"},
			},
			want: []string{""},
		},
		{
			name:  "empty model is no-op",
			model: "",
			messages: []schemas.Message{
				{Role: "assistant", Content: "hello"},
			},
			want: []string{""},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := &schemas.ChatRequest{Model: tc.model, Messages: tc.messages}
			InjectReasoningContent(req)

			if len(req.Messages) != len(tc.want) {
				t.Fatalf("len(messages) = %d, want %d", len(req.Messages), len(tc.want))
			}
			for i, m := range req.Messages {
				got := ""
				if m.ReasoningContent != nil {
					got = *m.ReasoningContent
				}
				if got != tc.want[i] {
					t.Errorf("message[%d].ReasoningContent = %q, want %q", i, got, tc.want[i])
				}
			}
		})
	}
}

// TestDeepseekProMaxExpansion verifies the deepseek-v4-pro alias mappings.
func TestDeepseekProMaxExpansion(t *testing.T) {
	cases := []struct {
		name            string
		model           string
		reasoningEffort string
		wantModel       string
		wantThinking    *schemas.ThinkingConfig
		wantEffort      string
	}{
		{
			name:            "max alias enables thinking and sets reasoning_effort",
			model:           "deepseek-v4-pro-max",
			reasoningEffort: "medium",
			wantModel:       "deepseek-v4-pro",
			wantThinking:    &schemas.ThinkingConfig{Type: "enabled"},
			wantEffort:      "max",
		},
		{
			name:            "none alias disables thinking and clears reasoning_effort",
			model:           "deepseek-v4-pro-none",
			reasoningEffort: "medium",
			wantModel:       "deepseek-v4-pro",
			wantThinking:    &schemas.ThinkingConfig{Type: "disabled"},
			wantEffort:      "",
		},
		{
			name:            "non alias model is untouched",
			model:           "deepseek-v4-pro",
			reasoningEffort: "medium",
			wantModel:       "deepseek-v4-pro",
			wantThinking:    nil,
			wantEffort:      "medium",
		},
		{
			name:            "unrelated model is untouched",
			model:           "gpt-4",
			reasoningEffort: "medium",
			wantModel:       "gpt-4",
			wantThinking:    nil,
			wantEffort:      "medium",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := &schemas.ChatRequest{
				Model:           tc.model,
				ReasoningEffort: tc.reasoningEffort,
			}
			ExpandDeepseekModel(req)

			if req.Model != tc.wantModel {
				t.Errorf("model = %q, want %q", req.Model, tc.wantModel)
			}
			if tc.wantThinking == nil {
				if req.Thinking != nil {
					t.Errorf("thinking = %+v, want nil", req.Thinking)
				}
			} else {
				if req.Thinking == nil {
					t.Fatalf("thinking = nil, want %+v", tc.wantThinking)
				}
				if req.Thinking.Type != tc.wantThinking.Type {
					t.Errorf("thinking.Type = %q, want %q", req.Thinking.Type, tc.wantThinking.Type)
				}
			}
			if req.ReasoningEffort != tc.wantEffort {
				t.Errorf("reasoning_effort = %q, want %q", req.ReasoningEffort, tc.wantEffort)
			}
		})
	}
}
