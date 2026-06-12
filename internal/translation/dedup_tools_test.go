package translation

import (
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

func tool(name string) schemas.Tool {
	return schemas.Tool{
		Type:     "function",
		Function: schemas.FunctionDefinition{Name: name},
	}
}

func TestDedupeTools(t *testing.T) {
	cases := []struct {
		name  string
		input []schemas.Tool
		want  []string
	}{
		{
			name:  "no tools",
			input: nil,
			want:  nil,
		},
		{
			name:  "unique tools no-op",
			input: []schemas.Tool{tool("a"), tool("b"), tool("c")},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "duplicate names keep last",
			input: []schemas.Tool{tool("a"), tool("b"), tool("a"), tool("c")},
			want:  []string{"b", "a", "c"},
		},
		{
			name:  "exa trigger strips built-in web tools",
			input: []schemas.Tool{tool("WebSearch"), tool("WebFetch"), tool("mcp__exa__web_search_exa")},
			want:  []string{"mcp__exa__web_search_exa"},
		},
		{
			name:  "tavily trigger strips built-in web tools",
			input: []schemas.Tool{tool("WebSearch"), tool("mcp__tavily__tavily_search")},
			want:  []string{"mcp__tavily__tavily_search"},
		},
		{
			name:  "browser mcp trigger strips chrome connector",
			input: []schemas.Tool{tool("mcp__browsermcp__navigate"), tool("mcp__Claude_in_Chrome__open")},
			want:  []string{"mcp__browsermcp__navigate"},
		},
		{
			name:  "mixed dedup and strip",
			input: []schemas.Tool{tool("WebSearch"), tool("a"), tool("a"), tool("mcp__exa__web_search_exa")},
			want:  []string{"a", "mcp__exa__web_search_exa"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := &schemas.ChatRequest{Tools: tc.input}
			DedupeTools(req)

			got := make([]string, len(req.Tools))
			for i, tl := range req.Tools {
				got[i] = tl.Function.Name
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("tool[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
