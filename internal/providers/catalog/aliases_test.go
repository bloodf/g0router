package catalog

import "testing"

func TestProviderAliasCount(t *testing.T) {
	if got := ProviderAliasCount(); got != 133 {
		t.Fatalf("ProviderAliasCount() = %d, want 133", got)
	}
}

func TestProviderAliasSamples(t *testing.T) {
	cases := map[string]string{
		"anthropic": "anthropic",
		"ds":        "deepseek",
		"hf":        "huggingface",
		"vx":        "vertex",
		"bb":        "blackbox",
	}
	for alias, want := range cases {
		got, ok := ResolveProviderAlias(alias)
		if !ok {
			t.Errorf("ResolveProviderAlias(%q) ok = false, want true", alias)
			continue
		}
		if got != want {
			t.Errorf("ResolveProviderAlias(%q) = %q, want %q", alias, got, want)
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
