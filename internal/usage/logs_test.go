package usage

import (
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

func TestRecentLogsFormat(t *testing.T) {
	now := time.Date(2026, 6, 12, 14, 33, 22, 0, time.UTC)
	reader := &fakeUsageReader{}
	names := &fakeNameSource{conn: map[string]string{"conn-1": "Main Account"}}
	svc := NewStatsService(reader, names, nil, nil, func() time.Time { return now })

	reader.logs = []*store.RequestLogEntry{
		{
			Timestamp:        "2026-06-12T14:33:22Z",
			Provider:         "openai",
			Model:            "gpt-4o",
			ConnectionID:     "conn-1",
			PromptTokens:     100,
			CompletionTokens: 50,
			Status:           "ok",
		},
		{
			Timestamp:        "2026-06-12T14:30:00Z",
			Provider:         "anthropic",
			Model:            "claude-sonnet-4",
			ConnectionID:     "unknown-conn",
			PromptTokens:     0,
			CompletionTokens: 0,
			Status:           "error",
			Tokens:           map[string]int64{"prompt_tokens": 7, "completion_tokens": 3},
		},
		{
			Timestamp:        "2026-06-12T14:25:00Z",
			Provider:         "",
			Model:            "",
			ConnectionID:     "",
			PromptTokens:     0,
			CompletionTokens: 0,
			Status:           "",
		},
	}

	lines := svc.RecentLogs(200)
	if len(lines) != 3 {
		t.Fatalf("len = %d, want 3", len(lines))
	}

	want0 := "12-06-2026 14:33:22 | gpt-4o | OPENAI | Main Account | 100 | 50 | ok"
	if lines[0] != want0 {
		t.Errorf("line0 = %q, want %q", lines[0], want0)
	}
	want1 := "12-06-2026 14:30:00 | claude-sonnet-4 | ANTHROPIC | unknown- | 7 | 3 | error"
	if lines[1] != want1 {
		t.Errorf("line1 = %q, want %q", lines[1], want1)
	}
	want2 := "12-06-2026 14:25:00 | - | - | - | - | - | -"
	if lines[2] != want2 {
		t.Errorf("line2 = %q, want %q", lines[2], want2)
	}
}
