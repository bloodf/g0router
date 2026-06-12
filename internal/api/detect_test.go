package api

import "testing"

func TestFormatAutoDetect(t *testing.T) {
	tests := []struct {
		name string
		body map[string]any
		want string
	}{
		{
			name: "openai-responses: input array no messages",
			body: map[string]any{"input": []any{"hello"}, "model": "gpt-4"},
			want: "openai-responses",
		},
		{
			name: "openai-responses: input string no messages",
			body: map[string]any{"input": "hello"},
			want: "openai-responses",
		},
		{
			name: "openai-responses: input ignored when messages present",
			body: map[string]any{
				"input":    []any{"hello"},
				"messages": []any{},
			},
			// messages present → falls through to later checks → default openai
			want: "openai",
		},
		{
			name: "antigravity: body.request.contents + userAgent=antigravity",
			body: map[string]any{
				"request":   map[string]any{"contents": []any{}},
				"userAgent": "antigravity",
			},
			want: "antigravity",
		},
		{
			name: "gemini: contents array",
			body: map[string]any{"contents": []any{map[string]any{"role": "user"}}},
			want: "gemini",
		},
		{
			name: "openai: stream_options field",
			body: map[string]any{"messages": []any{}, "stream_options": map[string]any{}},
			want: "openai",
		},
		{
			name: "openai: response_format field",
			body: map[string]any{"response_format": map[string]any{"type": "json_object"}},
			want: "openai",
		},
		{
			name: "openai: n field",
			body: map[string]any{"n": 2, "messages": []any{}},
			want: "openai",
		},
		{
			name: "openai: user field",
			body: map[string]any{"user": "u123", "messages": []any{}},
			want: "openai",
		},
		{
			name: "claude: system field",
			body: map[string]any{
				"messages": []any{map[string]any{"role": "user", "content": "hi"}},
				"system":   "you are helpful",
			},
			want: "claude",
		},
		{
			name: "claude: anthropic_version field",
			body: map[string]any{
				"messages":          []any{map[string]any{"role": "user", "content": "hi"}},
				"anthropic_version": "bedrock-2023-05-31",
			},
			want: "claude",
		},
		{
			name: "claude: content array with text type and system field",
			body: map[string]any{
				"model": "claude-3-5-sonnet",
				"messages": []any{map[string]any{
					"role": "user",
					"content": []any{map[string]any{
						"type": "text",
						"text": "hello",
					}},
				}},
				"system": "be helpful",
			},
			want: "claude",
		},
		{
			name: "default openai: plain messages with string content",
			body: map[string]any{
				"messages": []any{map[string]any{"role": "user", "content": "hi"}},
			},
			want: "openai",
		},
		{
			name: "default openai: empty body",
			body: map[string]any{},
			want: "openai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectFormat(tt.body)
			if got != tt.want {
				t.Errorf("DetectFormat() = %q, want %q", got, tt.want)
			}
		})
	}
}
