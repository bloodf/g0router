package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/store"
)

func TestKeysCommandsStoreOpenError(t *testing.T) {
	bad := badDataDir(t)
	for _, args := range [][]string{
		{"keys", "add", "k"},
		{"keys", "list"},
		{"keys", "rm", "k"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			if _, err := runCLI(t, append([]string{"--data-dir", bad}, args...)...); err == nil {
				t.Fatalf("expected store error for %v", args)
			}
		})
	}
}

func TestKeysAddMissingSecret(t *testing.T) {
	t.Setenv("API_KEY_SECRET", "")
	if _, err := runCLI(t, "--data-dir", t.TempDir(), "keys", "add", "k"); err == nil ||
		!strings.Contains(err.Error(), "API_KEY_SECRET required to create API keys") {
		t.Fatalf("err = %v", err)
	}
}

func TestKeysAddReadsSecretFromDB(t *testing.T) {
	dir := t.TempDir()
	s, err := store.NewStore(filepath.Join(dir, "g0router.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := s.SetAPIKeySecret("db-secret"); err != nil {
		t.Fatalf("SetAPIKeySecret: %v", err)
	}
	s.Close()

	out, err := runCLI(t, "--data-dir", dir, "keys", "add", "mykey")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if !strings.Contains(out, "mykey") {
		t.Fatalf("add out = %q", out)
	}
}

func TestKeysAddAndListAndRemove(t *testing.T) {
	t.Setenv("API_KEY_SECRET", "test-secret-value")
	dir := t.TempDir()

	out, err := runCLI(t, "--data-dir", dir, "keys", "add", "mykey")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if !strings.Contains(out, "mykey") {
		t.Fatalf("add out = %q", out)
	}

	out, err = runCLI(t, "--data-dir", dir, "keys", "list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "mykey") {
		t.Fatalf("list out = %q", out)
	}

	out, err = runCLI(t, "--data-dir", dir, "keys", "rm", "mykey")
	if err != nil {
		t.Fatalf("rm: %v", err)
	}
	if !strings.Contains(out, "removed mykey") {
		t.Fatalf("rm out = %q", out)
	}
}

func TestKeysRemoveNotFound2(t *testing.T) {
	if _, err := runCLI(t, "--data-dir", t.TempDir(), "keys", "rm", "ghost"); err == nil ||
		!strings.Contains(err.Error(), "not found") {
		t.Fatalf("err = %v", err)
	}
}

func TestProvidersTestUnknown(t *testing.T) {
	if _, err := runCLI(t, "providers", "test", "definitely-not-a-provider"); err == nil ||
		!strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("err = %v", err)
	}
}

func TestProvidersTestPublicUnavailable(t *testing.T) {
	// cursor has OAuth wiring but no inference adapter -> public unavailable.
	if _, err := runCLI(t, "providers", "test", "cursor"); err == nil ||
		!strings.Contains(err.Error(), "public inference unavailable") {
		t.Fatalf("err = %v", err)
	}
}

func TestProvidersTestNoAuthSucceeds(t *testing.T) {
	out, err := runCLI(t, "providers", "test", "ollama")
	if err != nil {
		t.Fatalf("ollama test: %v", err)
	}
	if !strings.Contains(out, "no credentials required") {
		t.Fatalf("out = %q", out)
	}
}

func TestProvidersTestNoActiveConnection(t *testing.T) {
	// openai needs a connection; none present -> error.
	if _, err := runCLI(t, "--data-dir", t.TempDir(), "providers", "test", "openai"); err == nil ||
		!strings.Contains(err.Error(), "no active connection") {
		t.Fatalf("err = %v", err)
	}
}

func TestProvidersTestStoreOpenError(t *testing.T) {
	if _, err := runCLI(t, "--data-dir", badDataDir(t), "providers", "test", "openai"); err == nil {
		t.Fatal("expected store open error")
	}
}

func TestProvidersTestActiveConnection(t *testing.T) {
	dir := t.TempDir()
	s, err := store.NewStore(filepath.Join(dir, "g0router.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	apiKey := "sk-test"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "default",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	out, err := runCLI(t, "--data-dir", dir, "providers", "test", "openai")
	if err != nil {
		t.Fatalf("providers test: %v", err)
	}
	if !strings.Contains(out, "active connection") {
		t.Fatalf("out = %q", out)
	}
}

func TestStatusStoreOpenError(t *testing.T) {
	if _, err := runCLI(t, "--data-dir", badDataDir(t), "status"); err == nil {
		t.Fatal("expected store open error")
	}
}
