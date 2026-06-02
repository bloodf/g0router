package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandPrintsVersion(t *testing.T) {
	cmd := NewRootCommand("0.1.0-test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if got := strings.TrimSpace(out.String()); got != "g0router 0.1.0-test" {
		t.Fatalf("output = %q, want version", got)
	}
}

func TestRootCommandIncludesExpectedSubcommands(t *testing.T) {
	cmd := NewRootCommand("0.1.0-test")
	names := commandNames(cmd.Commands())

	for _, want := range []string{"auth", "install", "serve"} {
		if !names[want] {
			t.Fatalf("missing subcommand %q in %v", want, names)
		}
	}
}

func TestServeCommandPreservesSingleBinaryPlaceholder(t *testing.T) {
	cmd := NewRootCommand("0.1.0-test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"serve", "--port", "20128", "--data-dir", t.TempDir()})

	if err := cmd.Execute(); err == nil {
		t.Fatal("execute error is nil")
	}

	if got := out.String(); !strings.Contains(got, "not yet implemented") {
		t.Fatalf("output = %q, want placeholder message", got)
	}
}
