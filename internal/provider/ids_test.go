package provider

import "testing"

func TestCanonicalProviderIDNormalizesRuntimeAliases(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{provider: "codex", want: "openai"},
		{provider: "openai", want: "openai"},
		{provider: "github", want: "github-copilot"},
		{provider: "github-copilot", want: "github-copilot"},
		{provider: "  GitHub  ", want: "github-copilot"},
		{provider: "gitlab", want: "gitlab-duo"},
		{provider: "gitlab-duo", want: "gitlab-duo"},
		{provider: "minimax", want: "minimax"},
	}

	for _, tt := range tests {
		if got := CanonicalProviderID(tt.provider); got != tt.want {
			t.Fatalf("CanonicalProviderID(%q) = %q, want %q", tt.provider, got, tt.want)
		}
	}
}

func TestProviderAliasesIncludeLegacyIDs(t *testing.T) {
	tests := []struct {
		provider string
		want     []string
	}{
		{provider: "openai", want: []string{"openai", "codex"}},
		{provider: "codex", want: []string{"openai", "codex"}},
		{provider: "github-copilot", want: []string{"github-copilot", "github"}},
		{provider: "github", want: []string{"github-copilot", "github"}},
		{provider: "gitlab-duo", want: []string{"gitlab-duo", "gitlab"}},
		{provider: "gitlab", want: []string{"gitlab-duo", "gitlab"}},
		{provider: "minimax", want: []string{"minimax"}},
	}

	for _, tt := range tests {
		got := ProviderAliases(tt.provider)
		if len(got) != len(tt.want) {
			t.Fatalf("ProviderAliases(%q) = %#v, want %#v", tt.provider, got, tt.want)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Fatalf("ProviderAliases(%q) = %#v, want %#v", tt.provider, got, tt.want)
			}
		}
	}
}
