package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/store"
	"github.com/spf13/cobra"
)

type fakeCLILoginFlow struct {
	provider oauth.ProviderID
	token    *oauth.TokenResult
	status   oauth.PollStatus
}

func (f fakeCLILoginFlow) ProviderID() oauth.ProviderID {
	return f.provider
}

func (f fakeCLILoginFlow) Start(ctx context.Context) (oauth.AuthSession, error) {
	return oauth.AuthSession{
		Provider:     f.provider,
		AuthURL:      "https://auth.example/device",
		SessionID:    "device-session",
		UserCode:     "USER-CODE",
		Verification: "https://auth.example",
		PollInterval: 1,
	}, nil
}

func (f fakeCLILoginFlow) Exchange(ctx context.Context, session oauth.AuthSession, code string) (oauth.TokenResult, error) {
	return oauth.TokenResult{}, nil
}

func (f fakeCLILoginFlow) Poll(ctx context.Context, session oauth.AuthSession) (oauth.PollResult, error) {
	status := f.status
	if status == "" {
		status = oauth.PollStatusComplete
	}
	return oauth.PollResult{Status: status, Token: f.token}, nil
}

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
	cmd := NewRootCommand("test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"login", "minimax", "--device"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("device login execute: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "minimax") || !strings.Contains(got, "Open this URL") {
		t.Fatalf("device login output = %q, want oauth flow", got)
	}

	cmd = NewRootCommand("test")
	out.Reset()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"login", "minimax", "--key"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("key login execute: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "API key") || strings.Contains(got, "Open this URL") {
		t.Fatalf("key login output = %q, want api key flow", got)
	}
}

func TestLoginCommandRejectsConflictingModes(t *testing.T) {
	cmd := NewRootCommand("test")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"login", "minimax", "--device", "--key"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("execute error is nil")
	}
	if !strings.Contains(err.Error(), "choose either --device or --key") {
		t.Fatalf("error = %q, want conflicting mode message", err.Error())
	}
}

func TestLoginDevicePersistsCompletedConnection(t *testing.T) {
	dataDir := t.TempDir()
	expiresAt := time.Now().Add(time.Hour)
	cmd := newAuthLoginCommand("login", &dataDir, func(provider string) (oauth.Flow, error) {
		return fakeCLILoginFlow{
			provider: oauth.ProviderID("codex"),
			token: &oauth.TokenResult{
				Provider:     oauth.ProviderID("codex"),
				AccessToken:  "codex-access",
				RefreshToken: "codex-refresh",
				TokenType:    "bearer",
				ExpiresAt:    expiresAt,
			},
		}, nil
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"codex", "--device"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	s := openCLIStoreForTest(t, dataDir)
	defer s.Close()
	connections, err := s.GetConnections("openai")
	if err != nil {
		t.Fatalf("GetConnections: %v", err)
	}
	if len(connections) != 1 {
		t.Fatalf("connections = %d, want 1", len(connections))
	}
	if connections[0].AccessToken == nil || *connections[0].AccessToken != "codex-access" {
		t.Fatalf("access token = %v, want codex-access", connections[0].AccessToken)
	}
	if connections[0].RefreshToken == nil || *connections[0].RefreshToken != "codex-refresh" {
		t.Fatalf("refresh token = %v, want codex-refresh", connections[0].RefreshToken)
	}
	if got := out.String(); strings.Contains(got, "codex-access") || strings.Contains(got, "codex-refresh") {
		t.Fatalf("login output leaked token material: %s", got)
	}
}

func TestLoginDevicePersistsGitHubAliasAsCopilot(t *testing.T) {
	dataDir := t.TempDir()
	cmd := newAuthLoginCommand("login", &dataDir, func(provider string) (oauth.Flow, error) {
		if provider != "github" {
			t.Fatalf("provider = %q, want github alias", provider)
		}
		return fakeCLILoginFlow{
			provider: oauth.ProviderID("github-copilot"),
			token: &oauth.TokenResult{
				Provider:    oauth.ProviderID("github-copilot"),
				AccessToken: "gh-access",
				TokenType:   "bearer",
			},
		}, nil
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"github", "--device"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	s := openCLIStoreForTest(t, dataDir)
	defer s.Close()
	connections, err := s.GetConnections("github-copilot")
	if err != nil {
		t.Fatalf("GetConnections: %v", err)
	}
	if len(connections) != 1 {
		t.Fatalf("github-copilot connections = %d, want 1", len(connections))
	}
	aliasConnections, err := s.GetConnections("github")
	if err != nil {
		t.Fatalf("GetConnections github: %v", err)
	}
	if len(aliasConnections) != 0 {
		t.Fatalf("github alias connections = %d, want 0", len(aliasConnections))
	}
}

func TestLoginDeviceDoesNotPersistPendingPoll(t *testing.T) {
	dataDir := t.TempDir()
	cmd := newAuthLoginCommand("login", &dataDir, func(provider string) (oauth.Flow, error) {
		return fakeCLILoginFlow{provider: oauth.ProviderID("codex"), status: oauth.PollStatusPending}, nil
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"codex", "--device"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	s := openCLIStoreForTest(t, dataDir)
	defer s.Close()
	connections, err := s.GetConnections("openai")
	if err != nil {
		t.Fatalf("GetConnections: %v", err)
	}
	if len(connections) != 0 {
		t.Fatalf("connections = %d, want 0", len(connections))
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
