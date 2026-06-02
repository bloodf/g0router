package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestAuthListShowsSupportedProviders(t *testing.T) {
	cmd := NewAuthCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	output := out.String()
	for _, want := range []string{"anthropic", "codex", "github", "gemini", "minimax"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want provider %q", output, want)
		}
	}
}

func TestAuthLoginStartsFlowWithoutBrowser(t *testing.T) {
	cmd := NewAuthCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"login", "minimax"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	output := out.String()
	for _, want := range []string{"minimax", "Open this URL", "Paste the resulting code"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want %q", output, want)
		}
	}
}

func TestAuthLoginRejectsUnknownProvider(t *testing.T) {
	cmd := NewAuthCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"login", "unknown"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("execute error is nil")
	}
	if !strings.Contains(err.Error(), "unknown oauth provider") {
		t.Fatalf("error = %q, want unknown provider", err.Error())
	}
}

func TestAuthCommandDoesNotExposeUnimplementedLogout(t *testing.T) {
	cmd := NewAuthCommand()
	names := commandNames(cmd.Commands())

	if names["logout"] {
		t.Fatal("auth logout should not be exposed until it has a credential-store implementation")
	}
}

func commandNames(commands []*cobra.Command) map[string]bool {
	names := make(map[string]bool, len(commands))
	for _, cmd := range commands {
		names[cmd.Name()] = true
	}
	return names
}
