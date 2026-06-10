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
