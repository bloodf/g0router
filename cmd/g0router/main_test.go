package main

import (
	"bytes"
	"testing"

	"github.com/bloodf/g0router/internal/cli"
)

// TestMainWiringHelp exercises the same root-command construction that main()
// performs, then runs --help so the wiring is exercised without binding a
// socket or calling os.Exit.
func TestMainWiringHelp(t *testing.T) {
	cmd := cli.NewRootCommand(version)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute --help: %v", err)
	}
	if out.Len() == 0 {
		t.Fatal("expected help output")
	}
}

func TestMainWiringVersionVar(t *testing.T) {
	if version == "" {
		t.Fatal("version should be set")
	}
}
