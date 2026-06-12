package store

import (
	"testing"
)

func TestDisabledModelTracking(t *testing.T) {
	st := newTestStore(t)

	// Initially nothing disabled.
	all, err := st.ListDisabledModels()
	if err != nil {
		t.Fatalf("ListDisabledModels: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("initial disabled = %v, want empty", all)
	}

	// Disable two models under one alias, one under another.
	if err := st.DisableModels("openai", []string{"gpt-4", "gpt-3.5"}); err != nil {
		t.Fatalf("DisableModels openai: %v", err)
	}
	if err := st.DisableModels("anthropic", []string{"claude-3-opus"}); err != nil {
		t.Fatalf("DisableModels anthropic: %v", err)
	}

	all, err = st.ListDisabledModels()
	if err != nil {
		t.Fatalf("ListDisabledModels after disable: %v", err)
	}
	if len(all["openai"]) != 2 {
		t.Fatalf("openai disabled = %v, want 2 entries", all["openai"])
	}
	if len(all["anthropic"]) != 1 || all["anthropic"][0] != "claude-3-opus" {
		t.Fatalf("anthropic disabled = %v", all["anthropic"])
	}

	// IsDisabled checks.
	ok, err := st.IsDisabled("openai", "gpt-4")
	if err != nil {
		t.Fatalf("IsDisabled gpt-4: %v", err)
	}
	if !ok {
		t.Fatal("gpt-4 should be disabled")
	}
	ok, err = st.IsDisabled("openai", "gpt-5")
	if err != nil {
		t.Fatalf("IsDisabled gpt-5: %v", err)
	}
	if ok {
		t.Fatal("gpt-5 should not be disabled")
	}

	// Enable one model: only that one is removed.
	if err := st.EnableModels("openai", []string{"gpt-4"}); err != nil {
		t.Fatalf("EnableModels gpt-4: %v", err)
	}
	all, err = st.ListDisabledModels()
	if err != nil {
		t.Fatalf("ListDisabledModels after enable: %v", err)
	}
	if len(all["openai"]) != 1 || all["openai"][0] != "gpt-3.5" {
		t.Fatalf("openai after enable = %v, want [gpt-3.5]", all["openai"])
	}

	// Enable all (empty slice) clears all for alias.
	if err := st.EnableModels("anthropic", []string{}); err != nil {
		t.Fatalf("EnableModels all anthropic: %v", err)
	}
	all, err = st.ListDisabledModels()
	if err != nil {
		t.Fatalf("ListDisabledModels after enable all: %v", err)
	}
	if len(all["anthropic"]) != 0 {
		t.Fatalf("anthropic after enable-all = %v, want empty", all["anthropic"])
	}

	// Re-disabling an already-disabled model is idempotent.
	if err := st.DisableModels("openai", []string{"gpt-3.5"}); err != nil {
		t.Fatalf("DisableModels idempotent: %v", err)
	}
	all, err = st.ListDisabledModels()
	if err != nil {
		t.Fatalf("ListDisabledModels idempotent: %v", err)
	}
	if len(all["openai"]) != 1 {
		t.Fatalf("openai after re-disable = %v, want 1 entry", all["openai"])
	}
}
