package translate

import "testing"

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name string
		body string
		want Format
	}{
		{
			name: "openai messages",
			body: `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`,
			want: FormatOpenAI,
		},
		{
			name: "anthropic top level system",
			body: `{"model":"claude-sonnet-4-20250514","system":"be brief","messages":[{"role":"user","content":"hi"}]}`,
			want: FormatAnthropic,
		},
		{
			name: "gemini contents",
			body: `{"contents":[{"role":"user","parts":[{"text":"hi"}]}]}`,
			want: FormatGemini,
		},
		{
			name: "unknown object",
			body: `{"model":"x"}`,
			want: FormatUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DetectFormat([]byte(tt.body))
			if err != nil {
				t.Fatalf("DetectFormat() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("DetectFormat() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetectFormatInvalidJSON(t *testing.T) {
	_, err := DetectFormat([]byte(`{"messages":`))
	if err == nil {
		t.Fatal("DetectFormat() error = nil")
	}
}

func TestFormatString(t *testing.T) {
	if FormatAnthropic.String() != "anthropic" {
		t.Fatalf("String() = %q, want anthropic", FormatAnthropic.String())
	}
}
