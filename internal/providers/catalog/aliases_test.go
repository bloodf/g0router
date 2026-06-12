package catalog

import "testing"

func TestProviderAliasCount(t *testing.T) {
	if got := len(ProviderAliases); got != 133 {
		t.Fatalf("len(ProviderAliases) = %d, want 133", got)
	}
}

func TestProviderAliasSamples(t *testing.T) {
	cases := map[string]string{
		"anthropic":   "anthropic",
		"ds":          "deepseek",
		"hf":          "huggingface",
		"vx":          "vertex",
		"bb":          "blackbox",
	}
	for alias, want := range cases {
		if got := ProviderAliases[alias]; got != want {
			t.Errorf("ProviderAliases[%q] = %q, want %q", alias, got, want)
		}
	}
}

func TestProviderAliasUnknown(t *testing.T) {
	got, ok := ResolveProviderAlias("nonexistent")
	if ok {
		t.Fatalf("ResolveProviderAlias(\"nonexistent\") ok = true, want false")
	}
	if got != "" {
		t.Errorf("ResolveProviderAlias(\"nonexistent\") = %q, want empty string", got)
	}
}
