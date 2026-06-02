package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/store"
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

func TestLoginCommandAcceptsAdvertisedFlags(t *testing.T) {
	for _, args := range [][]string{
		{"login", "minimax", "--device"},
		{"login", "minimax", "--key"},
	} {
		cmd := NewRootCommand("test")
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs(args)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("%v execute: %v", args, err)
		}
		if got := out.String(); !strings.Contains(got, "minimax") {
			t.Fatalf("%v output = %q, want provider", args, got)
		}
	}
}

func TestAuthCommandExposesLogout(t *testing.T) {
	cmd := NewAuthCommand()
	names := commandNames(cmd.Commands())

	if !names["logout"] {
		t.Fatal("auth logout should be exposed")
	}
}

func TestAuthLogoutRemovesProviderConnections(t *testing.T) {
	dataDir := t.TempDir()
	s := openCLIStoreForTest(t, dataDir)
	conn := &store.Connection{
		Provider: "minimax",
		Name:     "test",
		AuthType: store.AuthTypeOAuth,
		IsActive: true,
	}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	cmd := newRootCommand(rootConfig{
		Version: "test",
		Serve:   func(ctx context.Context, config serveConfig) error { return nil },
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--data-dir", dataDir, "logout", "minimax"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	s = openCLIStoreForTest(t, dataDir)
	defer s.Close()
	conns, err := s.GetConnections("minimax")
	if err != nil {
		t.Fatalf("GetConnections: %v", err)
	}
	if len(conns) != 0 {
		t.Fatalf("connections = %d, want 0", len(conns))
	}
}

func commandNames(commands []*cobra.Command) map[string]bool {
	names := make(map[string]bool, len(commands))
	for _, cmd := range commands {
		names[cmd.Name()] = true
	}
	return names
}

func openCLIStoreForTest(t *testing.T, dataDir string) *store.Store {
	t.Helper()

	s, err := store.NewStore(filepath.Join(dataDir, "g0router.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s
}
