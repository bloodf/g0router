package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
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

	for _, want := range []string{"auth", "healthcheck", "install", "keys", "login", "logout", "providers", "serve", "status", "uninstall", "version"} {
		if !names[want] {
			t.Fatalf("missing subcommand %q in %v", want, names)
		}
	}
}

func TestServeCommandStartsServerRunner(t *testing.T) {
	var got serveConfig
	called := false
	t.Setenv("API_KEY_SECRET", "test-api-key-secret")
	cmd := newRootCommand(rootConfig{
		Version: "0.1.0-test",
		Serve: func(ctx context.Context, config serveConfig) error {
			called = true
			got = config
			return nil
		},
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"serve", "--port", "20128", "--data-dir", t.TempDir()})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if !called {
		t.Fatal("serve runner was not called")
	}
	if got.Port != 20128 {
		t.Fatalf("port = %d, want 20128", got.Port)
	}
	if got.BindAddress != "127.0.0.1" {
		t.Fatalf("bind address = %q, want 127.0.0.1", got.BindAddress)
	}
	if got.Version != "0.1.0-test" {
		t.Fatalf("version = %q, want 0.1.0-test", got.Version)
	}
	if !got.RequireAPIKey {
		t.Fatal("RequireAPIKey = false, want true")
	}
	if got.APIKeySecret != "test-api-key-secret" {
		t.Fatalf("APIKeySecret = %q, want test-api-key-secret", got.APIKeySecret)
	}
	if got.DataDir == "" {
		t.Fatal("data dir should be passed to serve runner")
	}
}

func TestServeCommandUsesEnvironmentDefaults(t *testing.T) {
	var got serveConfig
	dataDir := filepath.Join(t.TempDir(), "env-data")
	t.Setenv("PORT", "22345")
	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("BIND_ADDRESS", "0.0.0.0")
	t.Setenv("API_KEY_SECRET", "test-api-key-secret")
	cmd := newRootCommand(rootConfig{
		Version: "0.1.0-test",
		Serve: func(ctx context.Context, config serveConfig) error {
			got = config
			return nil
		},
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"serve"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if got.Port != 22345 {
		t.Fatalf("port = %d, want env port", got.Port)
	}
	if got.BindAddress != "0.0.0.0" {
		t.Fatalf("bind address = %q, want env bind address", got.BindAddress)
	}
	if got.DataDir != dataDir {
		t.Fatalf("data dir = %q, want env data dir %q", got.DataDir, dataDir)
	}
}

func TestServeCommandFlagsOverrideEnvironmentDefaults(t *testing.T) {
	var got serveConfig
	flagDataDir := filepath.Join(t.TempDir(), "flag-data")
	t.Setenv("PORT", "22345")
	t.Setenv("DATA_DIR", filepath.Join(t.TempDir(), "env-data"))
	t.Setenv("API_KEY_SECRET", "test-api-key-secret")
	cmd := newRootCommand(rootConfig{
		Version: "0.1.0-test",
		Serve: func(ctx context.Context, config serveConfig) error {
			got = config
			return nil
		},
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--data-dir", flagDataDir, "serve", "--port", "20129"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if got.Port != 20129 {
		t.Fatalf("port = %d, want flag port", got.Port)
	}
	if got.DataDir != flagDataDir {
		t.Fatalf("data dir = %q, want flag data dir %q", got.DataDir, flagDataDir)
	}
}

func TestServeCommandFailsInvalidPortEnv(t *testing.T) {
	t.Setenv("PORT", "99999")
	t.Setenv("API_KEY_SECRET", "test-api-key-secret")
	cmd := newRootCommand(rootConfig{
		Version: "0.1.0-test",
		Serve: func(ctx context.Context, config serveConfig) error {
			t.Fatal("serve runner should not be called")
			return nil
		},
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"serve"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("execute should fail")
	}
	if !strings.Contains(err.Error(), "port must be 1-65535") {
		t.Fatalf("error = %q", err)
	}
}

func TestServeCommandFailsInvalidBooleanEnv(t *testing.T) {
	t.Setenv("RTK_ENABLED", "maybe")
	t.Setenv("API_KEY_SECRET", "test-api-key-secret")
	cmd := newRootCommand(rootConfig{
		Version: "0.1.0-test",
		Serve: func(ctx context.Context, config serveConfig) error {
			t.Fatal("serve runner should not be called")
			return nil
		},
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"serve"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("execute should fail")
	}
	if !strings.Contains(err.Error(), "RTK_ENABLED must be a boolean") {
		t.Fatalf("error = %q", err)
	}
}

func TestServeCommandRequiresAPIKeySecretByDefault(t *testing.T) {
	cmd := newRootCommand(rootConfig{
		Version: "0.1.0-test",
		Serve: func(ctx context.Context, config serveConfig) error {
			t.Fatal("serve runner should not be called")
			return nil
		},
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"serve", "--data-dir", t.TempDir()})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("execute should fail")
	}
	if !strings.Contains(err.Error(), "API_KEY_SECRET required when REQUIRE_API_KEY=true") {
		t.Fatalf("error = %q", err)
	}
}

func TestExpandServeDataDirExpandsHome(t *testing.T) {
	got, err := expandServeDataDir("~/.g0router")
	if err != nil {
		t.Fatalf("expandServeDataDir: %v", err)
	}

	if strings.Contains(got, "~") {
		t.Fatalf("expanded path = %q, should not contain ~", got)
	}
	if !strings.HasSuffix(got, ".g0router") {
		t.Fatalf("expanded path = %q, want .g0router suffix", got)
	}
}

func TestVersionCommandPrintsVersion(t *testing.T) {
	cmd := NewRootCommand("0.1.0-test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if got := strings.TrimSpace(out.String()); got != "g0router 0.1.0-test" {
		t.Fatalf("output = %q, want version", got)
	}
}

func TestStatusCommandUsesDataDir(t *testing.T) {
	cmd := NewRootCommand("0.1.0-test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--data-dir", t.TempDir(), "status"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if got := out.String(); !strings.Contains(got, "store: ok") {
		t.Fatalf("output = %q, want store status", got)
	}
}

func TestHealthcheckCommandFailsWhenServerUnavailable(t *testing.T) {
	cmd := NewRootCommand("test")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"healthcheck", "--url", "http://127.0.0.1:1/healthz"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("execute error is nil")
	}
}

func TestHealthcheckCommandChecksServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			t.Fatalf("path = %q, want /healthz", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cmd := NewRootCommand("test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"healthcheck", "--url", server.URL + "/healthz"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if got := out.String(); !strings.Contains(got, "healthcheck: ok") {
		t.Fatalf("output = %q, want healthcheck status", got)
	}
}

func TestDefaultServerConfigWiresWave4ADependencies(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	cfg := newServerConfig(serveConfig{Port: 20128, Version: "test"}, s)
	if cfg.OAuthFlows["minimax"] == nil {
		t.Fatal("minimax oauth flow should be wired")
	}

	models, err := cfg.ModelSource.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	foundOpenAI := false
	for _, model := range models {
		if model.Provider == providers.ProviderOpenAI {
			foundOpenAI = true
		}
	}
	if !foundOpenAI {
		t.Fatalf("models = %+v, want openai model", models)
	}

	if cfg.QuotaFetchers[providers.ProviderOpenAI] == nil {
		t.Fatal("openai quota fetcher should be wired")
	}
}

func TestDefaultServerConfigUsesAuthEnvironment(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	cfg := newServerConfig(serveConfig{
		Port:          20128,
		Version:       "test",
		RequireAPIKey: true,
		APIKeySecret:  "env-secret",
	}, s)
	if !cfg.RequireAPIKey {
		t.Fatal("RequireAPIKey = false, want true")
	}
	if cfg.APIKeySecret != "env-secret" {
		t.Fatalf("APIKeySecret = %q, want env-secret", cfg.APIKeySecret)
	}
	if cfg.APIKeyValidator == nil {
		t.Fatal("APIKeyValidator is nil")
	}

	_, raw, err := s.CreateAPIKey("default", "env-secret")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	ok, err := cfg.APIKeyValidator.ValidateAPIKey(raw, cfg.APIKeySecret)
	if err != nil {
		t.Fatalf("ValidateAPIKey: %v", err)
	}
	if !ok {
		t.Fatal("ValidateAPIKey = false, want true")
	}
}
