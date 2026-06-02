package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/store"
)

func TestMCPAddAccountsToolsAndRemoveCommands(t *testing.T) {
	dataDir := t.TempDir()

	addA := NewRootCommand("test")
	var addOut bytes.Buffer
	addA.SetOut(&addOut)
	addA.SetErr(&addOut)
	addA.SetArgs([]string{
		"--data-dir", dataDir,
		"mcp", "add", "atlassian-a",
		"--server-key", "atlassian",
		"--launch-type", "http",
		"--transport", "streamable-http",
		"--url", "https://mcp.atlassian.com/mcp",
		"--account-label", "account-a",
		"--header", "Authorization=Bearer secret",
	})
	if err := addA.Execute(); err != nil {
		t.Fatalf("mcp add account a: %v", err)
	}
	if strings.Contains(addOut.String(), "secret") {
		t.Fatalf("add output leaked secret: %q", addOut.String())
	}

	addB := NewRootCommand("test")
	addB.SetOut(&bytes.Buffer{})
	addB.SetErr(&bytes.Buffer{})
	addB.SetArgs([]string{
		"--data-dir", dataDir,
		"mcp", "add", "atlassian-b",
		"--server-key", "atlassian",
		"--launch-type", "http",
		"--transport", "streamable-http",
		"--url", "https://mcp.atlassian.com/mcp",
		"--account-label", "account-b",
	})
	if err := addB.Execute(); err != nil {
		t.Fatalf("mcp add account b: %v", err)
	}

	s, err := store.NewStore(filepath.Join(dataDir, "g0router.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	instances, err := s.ListMCPInstances()
	if err != nil {
		t.Fatalf("ListMCPInstances: %v", err)
	}
	if len(instances) != 2 {
		t.Fatalf("instances = %+v, want two atlassian accounts", instances)
	}
	for _, instance := range instances {
		if err := s.UpdateMCPInstanceManifest(instance.ID, mcp.Manifest{ClientID: instance.ID, Tools: []mcp.Tool{{Name: "search", Description: "Search"}}}); err != nil {
			t.Fatalf("UpdateMCPInstanceManifest: %v", err)
		}
		if err := s.UpsertMCPOAuthAccount(&store.MCPOAuthAccount{
			InstanceID:   instance.ID,
			AccountLabel: *instance.AccountLabel,
			AccessToken:  "token-" + *instance.AccountLabel,
			ExpiresAt:    time.Now().Add(time.Hour),
		}); err != nil {
			t.Fatalf("UpsertMCPOAuthAccount: %v", err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	accounts := NewRootCommand("test")
	var accountsOut bytes.Buffer
	accounts.SetOut(&accountsOut)
	accounts.SetErr(&accountsOut)
	accounts.SetArgs([]string{"--data-dir", dataDir, "mcp", "accounts", "atlassian-a"})
	if err := accounts.Execute(); err != nil {
		t.Fatalf("mcp accounts: %v", err)
	}
	if !strings.Contains(accountsOut.String(), "account-a") || strings.Contains(accountsOut.String(), "token-account-a") {
		t.Fatalf("accounts output = %q, want redacted account label", accountsOut.String())
	}

	tools := NewRootCommand("test")
	var toolsOut bytes.Buffer
	tools.SetOut(&toolsOut)
	tools.SetErr(&toolsOut)
	tools.SetArgs([]string{"--data-dir", dataDir, "mcp", "tools", "atlassian-a"})
	if err := tools.Execute(); err != nil {
		t.Fatalf("mcp tools: %v", err)
	}
	if !strings.Contains(toolsOut.String(), "__search") || strings.Contains(toolsOut.String(), "atlassian-b") {
		t.Fatalf("tools output = %q, want only account a tools", toolsOut.String())
	}

	remove := NewRootCommand("test")
	remove.SetOut(&bytes.Buffer{})
	remove.SetErr(&bytes.Buffer{})
	remove.SetArgs([]string{"--data-dir", dataDir, "mcp", "remove", "atlassian-a"})
	if err := remove.Execute(); err != nil {
		t.Fatalf("mcp remove: %v", err)
	}

	list := NewRootCommand("test")
	var listOut bytes.Buffer
	list.SetOut(&listOut)
	list.SetErr(&listOut)
	list.SetArgs([]string{"--data-dir", dataDir, "mcp", "list"})
	if err := list.Execute(); err != nil {
		t.Fatalf("mcp list: %v", err)
	}
	if strings.Contains(listOut.String(), "atlassian-a") || !strings.Contains(listOut.String(), "atlassian-b") {
		t.Fatalf("list output = %q, want only remaining account", listOut.String())
	}
}

func TestMCPAuthStartCommandCreatesFlowWithoutLeakingState(t *testing.T) {
	dataDir := t.TempDir()
	s, err := store.NewStore(filepath.Join(dataDir, "g0router.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	instance := createCLIMCPInstance(t, s, "atlassian-a")
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	cmd := NewRootCommand("test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"--data-dir", dataDir,
		"mcp", "auth", "start", "atlassian-a",
		"--authorization-url", "https://auth.example/authorize",
		"--resource", "https://mcp.atlassian.com",
		"--redirect-url", "http://localhost:3000/api/mcp/oauth/callback",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("mcp auth start: %v", err)
	}
	if !strings.Contains(out.String(), "https://auth.example/authorize") || strings.Contains(out.String(), instance.ID) {
		t.Fatalf("auth start output = %q, want authorization URL without instance id", out.String())
	}
}
