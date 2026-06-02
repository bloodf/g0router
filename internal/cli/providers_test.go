package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestProvidersListShowsKnownProviders(t *testing.T) {
	cmd := NewRootCommand("test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"providers", "list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	output := out.String()
	for _, want := range []string{"anthropic", "gemini", "openai"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want provider %q", output, want)
		}
	}
}

func TestProvidersTestRejectsUnknownProvider(t *testing.T) {
	cmd := NewRootCommand("test")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"providers", "test", "unknown"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("execute error is nil")
	}
	if !strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("error = %q, want unknown provider", err.Error())
	}
}
