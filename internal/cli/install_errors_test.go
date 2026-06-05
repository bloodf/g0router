package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// failingSecret always errors, exercising the secret-generation error branch.
type failingSecret struct{}

func (failingSecret) next() (string, error) { return "", os.ErrInvalid }

func baseInstallOptions(t *testing.T) installOptions {
	t.Helper()
	root := t.TempDir()
	binary := writeExecutable(t, root)
	return installOptions{
		Root:            root,
		HomeDir:         filepath.Join(root, "home"),
		Executable:      binary,
		RunCommand:      (&commandRecorder{}).run,
		SecretGenerator: newSecretGenerator("a", "b").next,
		Out:             &bytes.Buffer{},
	}
}

func TestRunInstallEnsureUserFails(t *testing.T) {
	opts := baseInstallOptions(t)
	// id check passes (no fail) so useradd is skipped; instead fail useradd by
	// failing id then useradd.
	rec := &commandRecorder{fail: map[string]bool{
		"id -u g0router": true,
		"useradd --system --no-create-home --shell /usr/sbin/nologin g0router": true,
	}}
	opts.RunCommand = rec.run
	if err := runInstall(opts); err == nil || !strings.Contains(err.Error(), "create g0router user") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunInstallCopyBinaryFails(t *testing.T) {
	opts := baseInstallOptions(t)
	opts.Executable = filepath.Join(t.TempDir(), "does-not-exist")
	if err := runInstall(opts); err == nil || !strings.Contains(err.Error(), "install binary") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunInstallChownFails(t *testing.T) {
	opts := baseInstallOptions(t)
	rec := &commandRecorder{fail: map[string]bool{
		"id -u g0router": true,
	}}
	// chown command embeds the root-prefixed data dir; fail any chown.
	rec.fail["chown g0router:g0router "+filepath.Join(opts.Root, "var/lib/g0router")] = true
	opts.RunCommand = rec.run
	if err := runInstall(opts); err == nil || !strings.Contains(err.Error(), "set data dir owner") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunInstallSecretGenFails(t *testing.T) {
	opts := baseInstallOptions(t)
	opts.RunCommand = (&commandRecorder{fail: map[string]bool{"id -u g0router": true}}).run
	opts.SecretGenerator = failingSecret{}.next
	if err := runInstall(opts); err == nil || !strings.Contains(err.Error(), "render defaults") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunInstallDaemonReloadFails(t *testing.T) {
	opts := baseInstallOptions(t)
	opts.RunCommand = (&commandRecorder{fail: map[string]bool{
		"id -u g0router":          true,
		"systemctl daemon-reload": true,
	}}).run
	if err := runInstall(opts); err == nil || !strings.Contains(err.Error(), "reload systemd") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunInstallEnableFails(t *testing.T) {
	opts := baseInstallOptions(t)
	opts.RunCommand = (&commandRecorder{fail: map[string]bool{
		"id -u g0router":                  true,
		"systemctl enable --now g0router": true,
	}}).run
	if err := runInstall(opts); err == nil || !strings.Contains(err.Error(), "enable service") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunInstallUserModeSucceeds(t *testing.T) {
	opts := baseInstallOptions(t)
	opts.User = true
	var out bytes.Buffer
	opts.Out = &out
	if err := runInstall(opts); err != nil {
		t.Fatalf("install user: %v", err)
	}
	if !strings.Contains(out.String(), "installed user service") {
		t.Fatalf("out = %q", out.String())
	}
}

func TestRunUninstallDisableFails(t *testing.T) {
	opts := baseInstallOptions(t)
	opts.RunCommand = (&commandRecorder{fail: map[string]bool{
		"systemctl disable --now g0router": true,
	}}).run
	if err := runUninstall(opts); err == nil || !strings.Contains(err.Error(), "disable service") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunUninstallReloadFails(t *testing.T) {
	opts := baseInstallOptions(t)
	opts.RunCommand = (&commandRecorder{fail: map[string]bool{
		"systemctl daemon-reload": true,
	}}).run
	if err := runUninstall(opts); err == nil || !strings.Contains(err.Error(), "reload systemd") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunUninstallSucceeds(t *testing.T) {
	opts := baseInstallOptions(t)
	var out bytes.Buffer
	opts.Out = &out
	if err := runUninstall(opts); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if !strings.Contains(out.String(), "removed service") {
		t.Fatalf("out = %q", out.String())
	}
}

func TestCopyFileSourceMissing(t *testing.T) {
	if err := copyFile(filepath.Join(t.TempDir(), "nope"), filepath.Join(t.TempDir(), "dst"), 0o644); err == nil {
		t.Fatal("expected error for missing source")
	}
}

func TestCopyFileSameFileChmod(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "f")
	if err := os.WriteFile(src, []byte("hi"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	// dst == src -> SameFile path, just chmods.
	if err := copyFile(src, src, 0o640); err != nil {
		t.Fatalf("copyFile same: %v", err)
	}
	info, err := os.Stat(src)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %v", info.Mode().Perm())
	}
}

func TestCopyFileCopies(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	if err := os.WriteFile(src, []byte("payload"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := copyFile(src, dst, 0o644); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil || string(got) != "payload" {
		t.Fatalf("dst = %q err = %v", got, err)
	}
}
