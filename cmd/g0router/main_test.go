package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/store"
)

func TestResetPasswordCLI(t *testing.T) {
	dir := t.TempDir()
	secret, err := store.LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	st, err := store.Open(filepath.Join(dir, "g0router.db"), secret)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	sessions := auth.NewSessions(st, time.Hour)
	if _, err := sessions.SeedAdmin("admin", "123456"); err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}

	// Normal login works before reset.
	if _, err := sessions.Login("admin", "123456"); err != nil {
		t.Fatalf("login before reset: %v", err)
	}

	st.Close()

	// Capture stdout during reset.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = resetPassword(dir)

	w.Close()
	os.Stdout = oldStdout
	if err != nil {
		t.Fatalf("resetPassword: %v", err)
	}
	out, _ := io.ReadAll(r)
	if !strings.Contains(string(out), "Password reset to default.") {
		t.Fatalf("reset output = %q, want to contain 'Password reset to default.'", string(out))
	}

	// Re-open store to verify reset.
	st2, err := store.Open(filepath.Join(dir, "g0router.db"), secret)
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer st2.Close()

	sessions = auth.NewSessions(st2, time.Hour)
	// After reset, the default password works again.
	if _, err := sessions.Login("admin", "123456"); err != nil {
		t.Fatalf("login after reset: %v", err)
	}
}

func TestResetPasswordThenDefaultLogin(t *testing.T) {
	st := newTestStore(t)
	sessions := auth.NewSessions(st, time.Hour)
	if _, err := sessions.SeedAdmin("admin", "123456"); err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}

	// Clear hash via store helper.
	if err := st.SetUserPasswordHash("admin", ""); err != nil {
		t.Fatalf("SetUserPasswordHash: %v", err)
	}

	// Default password "123456" works when hash is empty.
	if _, err := sessions.Login("admin", "123456"); err != nil {
		t.Fatalf("login with default password: %v", err)
	}
}

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	secret, err := store.LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	st, err := store.Open(filepath.Join(dir, "test.db"), secret)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}
