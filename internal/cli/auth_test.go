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
	providers := strings.Fields(output)
	providerSet := make(map[string]bool, len(providers))
	for _, provider := range providers {
		providerSet[provider] = true
	}
	for _, want := range []string{"anthropic", "codex", "github-copilot", "gemini", "kagi", "minimax", "qwen", "openrouter", "tavily"} {
		if !providerSet[want] {
			t.Fatalf("output = %q, want provider %q", output, want)
		}
	}
	if providerSet["github"] {
		t.Fatalf("output = %q, should list github-copilot instead of github", output)
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

func TestOAuthFlowAcceptsCanonicalProviderAliases(t *testing.T) {
	tests := []struct {
		provider string
		want     oauth.ProviderID
	}{
		{provider: "openai", want: oauth.ProviderID("codex")},
		{provider: "codex", want: oauth.ProviderID("codex")},
		{provider: "github", want: oauth.ProviderID("github-copilot")},
		{provider: "github-copilot", want: oauth.ProviderID("github-copilot")},
	}

	for _, tt := range tests {
		flow, err := newOAuthFlow(tt.provider)
		if err != nil {
			t.Fatalf("newOAuthFlow(%q): %v", tt.provider, err)
		}
		if flow.ProviderID() != tt.want {
			t.Fatalf("newOAuthFlow(%q) provider = %q, want %q", tt.provider, flow.ProviderID(), tt.want)
		}
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

func TestLoginCommandPersistsAPIKeyConnection(t *testing.T) {
	dataDir := t.TempDir()
	cmd := newRootCommand(rootConfig{
		Version: "test",
		Serve:   func(ctx context.Context, config serveConfig) error { return nil },
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--data-dir", dataDir, "login", "qwen", "--key", "--api-key", "qwen-secret", "--name", "work"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if strings.Contains(out.String(), "qwen-secret") {
		t.Fatalf("login output leaked provider key: %s", out.String())
	}
	if !strings.Contains(out.String(), "stored API key connection for qwen") {
		t.Fatalf("login output = %q, want stored connection message", out.String())
	}

	s := openCLIStoreForTest(t, dataDir)
	defer s.Close()
	connections, err := s.GetConnections("qwen")
	if err != nil {
		t.Fatalf("GetConnections: %v", err)
	}
	if len(connections) != 1 {
		t.Fatalf("qwen connections = %d, want 1", len(connections))
	}
	if connections[0].Name != "work" || connections[0].AuthType != store.AuthTypeAPIKey || connections[0].APIKey == nil || *connections[0].APIKey != "qwen-secret" || !connections[0].IsActive {
		t.Fatalf("qwen connection = %+v, want active named API-key connection", connections[0])
	}
}

func TestLoginCommandPersistsSearchProviderAPIKeyConnection(t *testing.T) {
	for _, provider := range []string{"kagi", "tavily"} {
		t.Run(provider, func(t *testing.T) {
			dataDir := t.TempDir()
			cmd := newRootCommand(rootConfig{
				Version: "test",
				Serve:   func(ctx context.Context, config serveConfig) error { return nil },
			})
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs([]string{"--data-dir", dataDir, "login", provider, "--key", "--api-key", provider + "-secret", "--name", "search"})

			if err := cmd.Execute(); err != nil {
				t.Fatalf("execute: %v", err)
			}
			if strings.Contains(out.String(), provider+"-secret") {
				t.Fatalf("login output leaked provider key: %s", out.String())
			}

			s := openCLIStoreForTest(t, dataDir)
			defer s.Close()
			connections, err := s.GetConnections(provider)
			if err != nil {
				t.Fatalf("GetConnections: %v", err)
			}
			if len(connections) != 1 {
				t.Fatalf("%s connections = %d, want 1", provider, len(connections))
			}
			if connections[0].Name != "search" || connections[0].AuthType != store.AuthTypeAPIKey || connections[0].APIKey == nil || *connections[0].APIKey != provider+"-secret" || !connections[0].IsActive {
				t.Fatalf("%s connection = %+v, want active search API-key connection", provider, connections[0])
			}
		})
	}
}

func TestLoginCommandRejectsAPIKeyForOAuthOnlyProvider(t *testing.T) {
	dataDir := t.TempDir()
	cmd := newRootCommand(rootConfig{
		Version: "test",
		Serve:   func(ctx context.Context, config serveConfig) error { return nil },
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--data-dir", dataDir, "login", "cursor", "--key", "--api-key", "cursor-secret"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("execute error is nil")
	}
	if !strings.Contains(err.Error(), "provider cursor does not support API-key auth") {
		t.Fatalf("error = %q, want API-key unsupported message", err.Error())
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

func TestLoginDevicePersistsVertexOAuthAsVertexConnection(t *testing.T) {
	dataDir := t.TempDir()
	cmd := newAuthLoginCommand("login", &dataDir, func(provider string) (oauth.Flow, error) {
		if provider != "vertex" {
			t.Fatalf("provider = %q, want vertex", provider)
		}
		return fakeCLILoginFlow{
			provider: oauth.ProviderID("gemini"),
			token: &oauth.TokenResult{
				Provider:     oauth.ProviderID("gemini"),
				AccessToken:  "vertex-access",
				RefreshToken: "vertex-refresh",
				TokenType:    "bearer",
			},
		}, nil
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"vertex", "--device"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	s := openCLIStoreForTest(t, dataDir)
	defer s.Close()
	vertexConnections, err := s.GetConnections("vertex")
	if err != nil {
		t.Fatalf("GetConnections vertex: %v", err)
	}
	if len(vertexConnections) != 1 {
		t.Fatalf("vertex connections = %d, want 1", len(vertexConnections))
	}
	if vertexConnections[0].ProviderSpecificData["oauth_provider"] != "gemini" {
		t.Fatalf("provider data = %+v, want oauth_provider gemini", vertexConnections[0].ProviderSpecificData)
	}
	geminiConnections, err := s.GetConnections("gemini")
	if err != nil {
		t.Fatalf("GetConnections gemini: %v", err)
	}
	if len(geminiConnections) != 0 {
		t.Fatalf("gemini connections = %d, want 0", len(geminiConnections))
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

func TestAuthLogoutRemovesCanonicalAliasConnections(t *testing.T) {
	dataDir := t.TempDir()
	s := openCLIStoreForTest(t, dataDir)
	for _, provider := range []string{"openai", "codex", "github-copilot", "github"} {
		conn := &store.Connection{
			Provider: provider,
			Name:     provider + "-test",
			AuthType: store.AuthTypeOAuth,
			IsActive: true,
		}
		if err := s.CreateConnection(conn); err != nil {
			t.Fatalf("CreateConnection %s: %v", provider, err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	runLogoutCommandForTest(t, dataDir, "codex")
	s = openCLIStoreForTest(t, dataDir)
	if conns, err := s.GetConnections("openai"); err != nil || len(conns) != 0 {
		t.Fatalf("openai connections after logout codex = %d, err=%v", len(conns), err)
	}
	if conns, err := s.GetConnections("codex"); err != nil || len(conns) != 0 {
		t.Fatalf("codex connections after logout codex = %d, err=%v", len(conns), err)
	}
	s.Close()

	runLogoutCommandForTest(t, dataDir, "github")
	s = openCLIStoreForTest(t, dataDir)
	if conns, err := s.GetConnections("github-copilot"); err != nil || len(conns) != 0 {
		t.Fatalf("github-copilot connections after logout github = %d, err=%v", len(conns), err)
	}
	if conns, err := s.GetConnections("github"); err != nil || len(conns) != 0 {
		t.Fatalf("github connections after logout github = %d, err=%v", len(conns), err)
	}
	s.Close()

	s = openCLIStoreForTest(t, dataDir)
	conn := &store.Connection{Provider: "github-copilot", Name: "github-test", AuthType: store.AuthTypeOAuth, IsActive: true}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection github-copilot: %v", err)
	}
	s.Close()

	runLogoutCommandForTest(t, dataDir, "github-copilot")
	s = openCLIStoreForTest(t, dataDir)
	defer s.Close()
	if conns, err := s.GetConnections("github-copilot"); err != nil || len(conns) != 0 {
		t.Fatalf("github-copilot connections after logout github-copilot = %d, err=%v", len(conns), err)
	}
}

func runLogoutCommandForTest(t *testing.T, dataDir string, provider string) {
	t.Helper()

	cmd := newRootCommand(rootConfig{
		Version: "test",
		Serve:   func(ctx context.Context, config serveConfig) error { return nil },
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--data-dir", dataDir, "logout", provider})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("logout %s: %v", provider, err)
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
