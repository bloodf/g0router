package translation

import "testing"

func TestDebug2ToolUseOrdering(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "text", "text": "before"},
					map[string]any{"type": "tool_use", "id": "tu1", "name": "Read"},
					map[string]any{"type": "text", "text": "after"},
				},
			},
		},
	}
	msgs := body["messages"].([]any)
	filtered := make([]any, 0, len(msgs))
	filtered = append(filtered, msgs...)
	
	result := fixToolUseOrdering(filtered)
	for i, m := range result {
		msg := m.(map[string]any)
		content := msg["content"].([]any)
		t.Logf("result[%d] content len=%d", i, len(content))
		for j, c := range content {
			t.Logf("  [%d] type=%v", j, c.(map[string]any)["type"])
		}
	}
}
