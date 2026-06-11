package translation

import (
	"strings"
	"testing"
)

func TestFormatIncompleteResponsesStreamFailure(t *testing.T) {
	frame := formatIncompleteResponsesStreamFailure()
	s := string(frame)
	if !strings.HasPrefix(s, "event: response.failed\ndata: ") {
		t.Errorf("bad frame prefix: %q", s)
	}
	if !strings.Contains(s, `"status":"failed"`) {
		t.Errorf("missing failed status: %q", s)
	}
	if !strings.Contains(s, `"code":"stream_disconnected"`) {
		t.Errorf("missing error code: %q", s)
	}
	if !strings.Contains(s, `"type":"stream_error"`) {
		t.Errorf("missing error type: %q", s)
	}
	if !strings.Contains(s, "stream closed before response.completed") {
		t.Errorf("missing error message: %q", s)
	}
}

func TestIsResponsesTerminalEvent(t *testing.T) {
	tests := []struct {
		name  string
		chunk map[string]any
		want  bool
	}{
		{
			name:  "completed by event",
			chunk: map[string]any{"event": "response.completed"},
			want:  true,
		},
		{
			name:  "failed by event",
			chunk: map[string]any{"event": "response.failed"},
			want:  true,
		},
		{
			name:  "completed by type",
			chunk: map[string]any{"type": "response.completed"},
			want:  true,
		},
		{
			name:  "failed by type",
			chunk: map[string]any{"type": "response.failed"},
			want:  true,
		},
		{
			name:  "completed by status",
			chunk: map[string]any{"response": map[string]any{"status": "completed"}},
			want:  true,
		},
		{
			name:  "failed by status",
			chunk: map[string]any{"response": map[string]any{"status": "failed"}},
			want:  true,
		},
		{
			name:  "in progress",
			chunk: map[string]any{"event": "response.in_progress"},
			want:  false,
		},
		{
			name:  "non-terminal type",
			chunk: map[string]any{"type": "response.output_text.delta"},
			want:  false,
		},
		{
			name:  "no relevant fields",
			chunk: map[string]any{"type": "foo"},
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isResponsesTerminalEvent(tt.chunk)
			if got != tt.want {
				t.Errorf("isResponsesTerminalEvent(%+v) = %v, want %v", tt.chunk, got, tt.want)
			}
		})
	}
}
