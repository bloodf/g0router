package cli

import (
	"bytes"
	"context"
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

	for _, want := range []string{"auth", "healthcheck", "install", "keys", "login", "logout", "providers", "serve", "status", "uninstall", "version"} {
		if !names[want] {
			t.Fatalf("missing subcommand %q in %v", want, names)
		}
	}
}

func TestServeCommandStartsServerRunner(t *testing.T) {
	var got serveConfig
	called := false
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
	if got.Version != "0.1.0-test" {
		t.Fatalf("version = %q, want 0.1.0-test", got.Version)
	}
	if got.DataDir == "" {
		t.Fatal("data dir should be passed to serve runner")
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

func TestHealthcheckCommandAcceptsDefaultConfiguration(t *testing.T) {
	cmd := NewRootCommand("test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"healthcheck"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if got := out.String(); !strings.Contains(got, "healthcheck: ok") {
		t.Fatalf("output = %q, want healthcheck status", got)
	}
}
