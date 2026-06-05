package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/store"
)

// badDataDir returns a data-dir path whose parent is a regular file, forcing
// store.NewStore's os.MkdirAll (and therefore openCLIStore) to fail with
// ENOTDIR. No mocks, no network.
func badDataDir(t *testing.T) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return filepath.Join(f, "child")
}

func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := NewRootCommand("test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// --- openCLIStore error path for every mcp store-backed RunE ---

func TestMCPCommandsStoreOpenError(t *testing.T) {
	bad := badDataDir(t)
	cases := [][]string{
		{"mcp", "add", "x"},
		{"mcp", "list"},
		{"mcp", "remove", "x"},
		{"mcp", "accounts", "x"},
		{"mcp", "tools", "x"},
		{"mcp", "auth", "start", "x"},
		{"mcp", "auth", "complete", "x", "http://cb?code=c&state=s"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			if _, err := runCLI(t, append([]string{"--data-dir", bad}, args...)...); err == nil {
				t.Fatalf("expected store open error for %v", args)
			}
		})
	}
}

// --- mcp add / list / remove / accounts / tools success + not-found ---

func TestMCPAddAndListSuccess(t *testing.T) {
	dir := t.TempDir()
	out, err := runCLI(t, "--data-dir", dir, "mcp", "add", "expo",
		"--server-key", "k", "--command", "node", "--arg", "a", "--url", "https://m.example",
		"--header", "H=v", "--env", "E=1", "--cwd", "/tmp", "--account-label", "lbl")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if !strings.Contains(out, "added mcp instance expo") {
		t.Fatalf("add out = %q", out)
	}

	out, err = runCLI(t, "--data-dir", dir, "mcp", "list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "expo") {
		t.Fatalf("list out = %q", out)
	}
}

func TestMCPRemoveSuccess(t *testing.T) {
	dir := t.TempDir()
	s, err := store.NewStore(filepath.Join(dir, "g0router.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	createCLIMCPInstance(t, s, "expo")
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	out, err := runCLI(t, "--data-dir", dir, "mcp", "remove", "expo")
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if !strings.Contains(out, "removed mcp instance expo") {
		t.Fatalf("remove out = %q", out)
	}
}

func TestMCPInstanceNotFoundPaths(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"remove", "accounts", "tools"} {
		_, err := runCLI(t, "--data-dir", dir, "mcp", sub, "missing")
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("%s missing: err = %v", sub, err)
		}
	}
	for _, sub := range []string{"start"} {
		_, err := runCLI(t, "--data-dir", dir, "mcp", "auth", sub, "missing")
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("auth %s missing: err = %v", sub, err)
		}
	}
}

func TestMCPAccountsAndToolsEmptySuccess(t *testing.T) {
	dir := t.TempDir()
	s, err := store.NewStore(filepath.Join(dir, "g0router.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	createCLIMCPInstance(t, s, "expo")
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := runCLI(t, "--data-dir", dir, "mcp", "accounts", "expo"); err != nil {
		t.Fatalf("accounts: %v", err)
	}
	// tools with nil manifest returns nil early
	if _, err := runCLI(t, "--data-dir", dir, "mcp", "tools", "expo"); err != nil {
		t.Fatalf("tools: %v", err)
	}
}

func TestMCPAuthCompleteParseError(t *testing.T) {
	// callback URL with a control character forces url.Parse to error before
	// any store access.
	_, err := runCLI(t, "mcp", "auth", "complete", "x", "http://exa mple\x7f/cb")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

// --- auth.go residual branches ---

func TestPersistAPIKeyLoginStoreError(t *testing.T) {
	// openCLIStore fails inside persistAPIKeyLogin (provider supports api_key).
	err := persistAPIKeyLogin(badDataDir(t), "openai", "name", "sk-test")
	if err == nil {
		t.Fatal("expected store open error")
	}
}

func TestPersistAPIKeyLoginEmptyKey(t *testing.T) {
	if err := persistAPIKeyLogin(t.TempDir(), "openai", "name", "  "); err == nil {
		t.Fatal("expected empty key error")
	}
}

func TestPersistAPIKeyLoginDefaultsName(t *testing.T) {
	dir := t.TempDir()
	if err := persistAPIKeyLogin(dir, "openai", "  ", "sk-test"); err != nil {
		t.Fatalf("persist: %v", err)
	}
}

func TestAuthLogoutStoreError(t *testing.T) {
	if _, err := runCLI(t, "--data-dir", badDataDir(t), "auth", "logout", "openai"); err == nil {
		t.Fatal("expected store open error")
	}
}

func TestCompleteDeviceLoginStoreError(t *testing.T) {
	flow := fakeCLILoginFlow{
		provider: oauth.ProviderID("codex"),
		status:   oauth.PollStatusComplete,
		token:    &oauth.TokenResult{Provider: "codex", AccessToken: "t"},
	}
	session := oauth.AuthSession{Provider: "codex", UserCode: "U", Verification: "https://v"}
	_, err := completeDeviceLogin(context.Background(), badDataDir(t), "codex", flow, session)
	if err == nil {
		t.Fatal("expected store open error")
	}
}

func TestCompleteDeviceLoginNoSession(t *testing.T) {
	flow := fakeCLILoginFlow{provider: oauth.ProviderID("codex")}
	status, err := completeDeviceLogin(context.Background(), t.TempDir(), "codex", flow, oauth.AuthSession{})
	if err != nil || status != "" {
		t.Fatalf("status=%q err=%v", status, err)
	}
}
