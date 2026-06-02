package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

func TestMCPAuthCompleteCommandCompletesPastedCallback(t *testing.T) {
	dataDir := t.TempDir()
	s, err := store.NewStore(filepath.Join(dataDir, "g0router.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	instance := createCLIMCPInstance(t, s, "expo")
	if err := s.CreateMCPOAuthFlow(&store.MCPOAuthFlow{
		InstanceID:         instance.ID,
		State:              "state-1",
		CodeVerifierSecret: "verifier",
		ResourceURI:        "https://mcp.example",
		ExpiresAt:          time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateMCPOAuthFlow: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	cmd := NewRootCommand("test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--data-dir", dataDir, "mcp", "auth", "complete", "expo", "http://localhost:3000/callback?code=secret-code&state=state-1"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "completed mcp auth for expo") {
		t.Fatalf("output = %q, want completion", output)
	}
	if strings.Contains(output, "secret-code") {
		t.Fatalf("output leaked code: %q", output)
	}
}

func TestMCPAuthCompleteCommandRejectsMissingCode(t *testing.T) {
	cmd := NewRootCommand("test")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mcp", "auth", "complete", "expo", "http://localhost:3000/callback?state=state-1"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute error is nil")
	}
	if !strings.Contains(err.Error(), "code is required") {
		t.Fatalf("err = %v, want missing code", err)
	}
}

func createCLIMCPInstance(t *testing.T, s *store.Store, name string) *store.MCPInstance {
	t.Helper()
	instance := &store.MCPInstance{
		Name:       name,
		ServerKey:  name,
		LaunchType: "http",
		Transport:  "streamable-http",
		URL:        stringPtr("https://mcp.example/mcp"),
		IsActive:   true,
	}
	if err := s.CreateMCPInstance(instance); err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}
	return instance
}

func stringPtr(value string) *string {
	return &value
}
