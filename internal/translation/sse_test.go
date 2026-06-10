package translation

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatSSEClaudeEventFraming(t *testing.T) {
	event := map[string]any{"type": "message_start", "message": map[string]any{"id": "msg_1"}}
	got := string(FormatSSE(FormatClaude, event))
	wantPrefix := "event: message_start\ndata: "
	if !strings.HasPrefix(got, wantPrefix) {
		t.Errorf("output = %q, want prefix %q", got, wantPrefix)
	}
	if !strings.HasSuffix(got, "\n\n") {
		t.Errorf("output missing terminating blank line: %q", got)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(got), "event: message_start\ndata: ")), &parsed); err != nil {
		t.Fatalf("data not valid JSON: %v", err)
	}
}

func TestFormatSSEDefaultDataFraming(t *testing.T) {
	event := map[string]any{"id": "c1"}
	got := string(FormatSSE(FormatOpenAI, event))
	want := "data: {\"id\":\"c1\"}\n\n"
	if got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestFormatSSEClaudeWithoutTypeUsesDefault(t *testing.T) {
	event := map[string]any{"id": "c1"}
	got := string(FormatSSE(FormatClaude, event))
	want := "data: {\"id\":\"c1\"}\n\n"
	if got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestHasValuableContentOpenAI(t *testing.T) {
	emptyDelta := map[string]any{
		"choices": []any{
			map[string]any{"delta": map[string]any{}},
		},
	}
	if HasValuableContent(emptyDelta, FormatOpenAI) {
		t.Errorf("empty delta should be false")
	}

	cases := []struct {
		name  string
		chunk map[string]any
	}{
		{"content", map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"content": "hi"}}}}},
		{"reasoning_content", map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"reasoning_content": "think"}}}}},
		{"tool_calls", map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"tool_calls": []any{map[string]any{}}}}}}},
		{"finish_reason", map[string]any{"choices": []any{map[string]any{"delta": map[string]any{}, "finish_reason": "stop"}}}},
		{"role", map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"role": "assistant"}}}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if !HasValuableContent(c.chunk, FormatOpenAI) {
				t.Errorf("expected true for %s", c.name)
			}
		})
	}
}

func TestHasValuableContentClaude(t *testing.T) {
	emptyContentBlockDelta := map[string]any{
		"type":  "content_block_delta",
		"delta": map[string]any{},
	}
	if HasValuableContent(emptyContentBlockDelta, FormatClaude) {
		t.Errorf("empty content_block_delta should be false")
	}

	messageStart := map[string]any{"type": "message_start", "message": map[string]any{"id": "msg_1"}}
	if !HasValuableContent(messageStart, FormatClaude) {
		t.Errorf("message_start should be true")
	}
}

func TestHasValuableContentOtherFormats(t *testing.T) {
	chunk := map[string]any{"id": "x"}
	for _, f := range []Format{FormatOllama, FormatGemini, FormatOpenAIResponses} {
		if !HasValuableContent(chunk, f) {
			t.Errorf("format %q should always be true", f)
		}
	}
}

func TestFixInvalidID(t *testing.T) {
	t.Run("chat", func(t *testing.T) {
		chunk := map[string]any{"id": "chat"}
		if !FixInvalidID(chunk) {
			t.Errorf("expected mutation for 'chat'")
		}
		id, ok := chunk["id"].(string)
		if !ok || !strings.HasPrefix(id, "chatcmpl-") {
			t.Errorf("expected chatcmpl- prefix, got %q", id)
		}
	})

	t.Run("completion", func(t *testing.T) {
		chunk := map[string]any{"id": "completion"}
		if !FixInvalidID(chunk) {
			t.Errorf("expected mutation for 'completion'")
		}
		id, ok := chunk["id"].(string)
		if !ok || !strings.HasPrefix(id, "chatcmpl-") {
			t.Errorf("expected chatcmpl- prefix, got %q", id)
		}
	})

	t.Run("too short", func(t *testing.T) {
		chunk := map[string]any{"id": "ab"}
		if !FixInvalidID(chunk) {
			t.Errorf("expected mutation for short id")
		}
		id, ok := chunk["id"].(string)
		if !ok || !strings.HasPrefix(id, "chatcmpl-") {
			t.Errorf("expected chatcmpl- prefix, got %q", id)
		}
	})

	t.Run("valid id untouched", func(t *testing.T) {
		chunk := map[string]any{"id": "chatcmpl-12345678"}
		if FixInvalidID(chunk) {
			t.Errorf("expected no mutation for valid id")
		}
		if chunk["id"] != "chatcmpl-12345678" {
			t.Errorf("id was mutated unexpectedly")
		}
	})

	t.Run("uses extend_fields requestId", func(t *testing.T) {
		chunk := map[string]any{"id": "chat", "extend_fields": map[string]any{"requestId": "req-42"}}
		FixInvalidID(chunk)
		if chunk["id"] != "chatcmpl-req-42" {
			t.Errorf("expected chatcmpl-req-42, got %v", chunk["id"])
		}
	})

	t.Run("uses extend_fields traceId fallback", func(t *testing.T) {
		chunk := map[string]any{"id": "chat", "extend_fields": map[string]any{"traceId": "trace-99"}}
		FixInvalidID(chunk)
		if chunk["id"] != "chatcmpl-trace-99" {
			t.Errorf("expected chatcmpl-trace-99, got %v", chunk["id"])
		}
	})
}

func TestFormatSSECleansNullUsage(t *testing.T) {
	// Null usage should be dropped.
	event := map[string]any{"id": "1", "usage": nil}
	got := string(FormatSSE(FormatOpenAI, event))
	if strings.Contains(got, "usage") {
		t.Errorf("null usage should be dropped, got %q", got)
	}

	// Null perf_metrics inside usage object should be dropped.
	event2 := map[string]any{"id": "2", "usage": map[string]any{"total_tokens": 10, "perf_metrics": nil}}
	got2 := string(FormatSSE(FormatOpenAI, event2))
	if strings.Contains(got2, "perf_metrics") {
		t.Errorf("null perf_metrics should be dropped, got %q", got2)
	}
	if !strings.Contains(got2, "total_tokens") {
		t.Errorf("total_tokens should remain, got %q", got2)
	}

	// Recurse into response key.
	event3 := map[string]any{"id": "3", "response": map[string]any{"usage": nil}}
	got3 := string(FormatSSE(FormatOpenAI, event3))
	if strings.Contains(got3, "usage") {
		t.Errorf("null usage in response should be dropped, got %q", got3)
	}
}
