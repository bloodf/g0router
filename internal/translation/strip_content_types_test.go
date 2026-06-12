package translation

import (
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

// TestStripContentTypes verifies that the Stage-1 implementation leaves plain
// string Message.Content untouched regardless of the strip flags.
func TestStripContentTypes(t *testing.T) {
	cases := []struct {
		name        string
		stripImages bool
		stripAudio  bool
	}{
		{"no stripping requested", false, false},
		{"strip images only", true, false},
		{"strip audio only", false, true},
		{"strip both", true, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := &schemas.ChatRequest{
				Messages: []schemas.Message{
					{Role: "user", Content: "hello"},
					{Role: "assistant", Content: "world"},
				},
			}
			original := make([]string, len(req.Messages))
			for i, m := range req.Messages {
				original[i] = m.Content
			}

			StripContentTypes(req, tc.stripImages, tc.stripAudio)

			for i, m := range req.Messages {
				if m.Content != original[i] {
					t.Errorf("message[%d].Content = %q, want %q", i, m.Content, original[i])
				}
			}
		})
	}
}
