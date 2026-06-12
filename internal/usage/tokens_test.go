package usage

import (
	"testing"
)

func TestNormalizeTokens(t *testing.T) {
	cases := []struct {
		name string
		in   map[string]int64
		want TokenSet
	}{
		{
			name: "canonical names",
			in:   map[string]int64{"prompt_tokens": 100, "completion_tokens": 50, "cached_tokens": 20, "reasoning_tokens": 10, "cache_creation_input_tokens": 5},
			want: TokenSet{Prompt: 100, Completion: 50, Cached: 20, Reasoning: 10, CacheCreation: 5},
		},
		{
			name: "synonyms",
			in:   map[string]int64{"input_tokens": 100, "output_tokens": 50, "cache_read_input_tokens": 20},
			want: TokenSet{Prompt: 100, Completion: 50, Cached: 20},
		},
		{
			name: "canonical overrides synonym",
			in:   map[string]int64{"prompt_tokens": 100, "input_tokens": 200},
			want: TokenSet{Prompt: 100},
		},
		{
			name: "nil tokens",
			in:   nil,
			want: TokenSet{},
		},
		{
			name: "empty tokens",
			in:   map[string]int64{},
			want: TokenSet{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeTokens(tc.in)
			if got != tc.want {
				t.Errorf("NormalizeTokens = %+v, want %+v", got, tc.want)
			}
		})
	}
}
