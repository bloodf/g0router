package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestKeysAddListRemove(t *testing.T) {
	dataDir := t.TempDir()

	add := NewRootCommand("test")
	var addOut bytes.Buffer
	add.SetOut(&addOut)
	add.SetErr(&addOut)
	add.SetArgs([]string{"--data-dir", dataDir, "keys", "add", "local"})
	if err := add.Execute(); err != nil {
		t.Fatalf("keys add: %v", err)
	}
	if got := addOut.String(); !strings.Contains(got, "g0r_") {
		t.Fatalf("keys add output = %q, want raw key", got)
	}

	list := NewRootCommand("test")
	var listOut bytes.Buffer
	list.SetOut(&listOut)
	list.SetErr(&listOut)
	list.SetArgs([]string{"--data-dir", dataDir, "keys", "list"})
	if err := list.Execute(); err != nil {
		t.Fatalf("keys list: %v", err)
	}
	if got := listOut.String(); !strings.Contains(got, "local") {
		t.Fatalf("keys list output = %q, want local key", got)
	}

	rm := NewRootCommand("test")
	rm.SetOut(&bytes.Buffer{})
	rm.SetErr(&bytes.Buffer{})
	rm.SetArgs([]string{"--data-dir", dataDir, "keys", "rm", "local"})
	if err := rm.Execute(); err != nil {
		t.Fatalf("keys rm: %v", err)
	}

	after := NewRootCommand("test")
	var afterOut bytes.Buffer
	after.SetOut(&afterOut)
	after.SetErr(&afterOut)
	after.SetArgs([]string{"--data-dir", dataDir, "keys", "list"})
	if err := after.Execute(); err != nil {
		t.Fatalf("keys list after remove: %v", err)
	}
	if got := afterOut.String(); strings.Contains(got, "local") {
		t.Fatalf("keys list output = %q, should not contain removed key", got)
	}
}
