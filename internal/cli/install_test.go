package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallSystemWritesFilesAndRunsSystemctl(t *testing.T) {
	root := t.TempDir()
	binary := writeExecutable(t, root)
	recorder := &commandRecorder{fail: map[string]bool{"id -u g0router": true}}
	secrets := newSecretGenerator("jwt-secret", "api-key-secret")

	var out bytes.Buffer
	err := runInstall(installOptions{
		Root:            root,
		HomeDir:         filepath.Join(root, "home"),
		Executable:      binary,
		RunCommand:      recorder.run,
		SecretGenerator: secrets.next,
		Out:             &out,
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}

	assertFileContains(t, filepath.Join(root, "usr/local/bin/g0router"), "test binary")
	assertFileContains(t, filepath.Join(root, "etc/systemd/system/g0router.service"), "ExecStart=/usr/local/bin/g0router serve")
	assertFileContains(t, filepath.Join(root, "etc/default/g0router"), "DATA_DIR=/var/lib/g0router")
	assertFileContains(t, filepath.Join(root, "etc/default/g0router"), "JWT_SECRET=jwt-secret")
	assertFileContains(t, filepath.Join(root, "etc/default/g0router"), "API_KEY_SECRET=api-key-secret")
	if _, err := os.Stat(filepath.Join(root, "var/lib/g0router")); err != nil {
		t.Fatalf("data dir missing: %v", err)
	}
	recorder.assertRan(t, "systemctl daemon-reload")
	recorder.assertRan(t, "systemctl enable --now g0router")
	recorder.assertRan(t, "useradd --system --no-create-home --shell /usr/sbin/nologin g0router")
	recorder.assertRan(t, "chown g0router:g0router "+filepath.Join(root, "var/lib/g0router"))
	if got := out.String(); !strings.Contains(got, "installed system service") {
		t.Fatalf("output = %q, want installed system service", got)
	}
}

func TestInstallUsesDeployTemplatesOutsideCheckout(t *testing.T) {
	root := t.TempDir()
	binary := writeExecutable(t, root)
	recorder := &commandRecorder{fail: map[string]bool{"id -u g0router": true}}
	originalCWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalCWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	err = runInstall(installOptions{
		Root:            root,
		HomeDir:         filepath.Join(root, "home"),
		Executable:      binary,
		RunCommand:      recorder.run,
		SecretGenerator: newSecretGenerator("jwt-secret", "api-key-secret").next,
		Out:             io.Discard,
	})
	if err != nil {
		t.Fatalf("install outside checkout: %v", err)
	}

	assertFileContains(t, filepath.Join(root, "etc/systemd/system/g0router.service"), "ExecStart=/usr/local/bin/g0router serve")
	assertFileContains(t, filepath.Join(root, "etc/default/g0router"), "API_KEY_SECRET=api-key-secret")
}

func TestInstallSkipsCopyWhenExecutableAlreadyInstalled(t *testing.T) {
	root := t.TempDir()
	installed := filepath.Join(root, "usr/local/bin/g0router")
	mustWrite(t, installed, "already installed")
	if err := os.Chmod(installed, 0o755); err != nil {
		t.Fatalf("chmod installed binary: %v", err)
	}
	recorder := &commandRecorder{fail: map[string]bool{"id -u g0router": true}}

	err := runInstall(installOptions{
		Root:            root,
		HomeDir:         filepath.Join(root, "home"),
		Executable:      installed,
		RunCommand:      recorder.run,
		SecretGenerator: newSecretGenerator("jwt-secret", "api-key-secret").next,
		Out:             io.Discard,
	})
	if err != nil {
		t.Fatalf("install same binary: %v", err)
	}

	assertFileContains(t, installed, "already installed")
}

func TestInstallUserWritesFilesAndRunsUserSystemctl(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	binary := writeExecutable(t, root)
	recorder := &commandRecorder{}

	var out bytes.Buffer
	err := runInstall(installOptions{
		User:       true,
		Root:       root,
		HomeDir:    home,
		Executable: binary,
		RunCommand: recorder.run,
		Out:        &out,
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}

	assertFileContains(t, filepath.Join(home, ".local/bin/g0router"), "test binary")
	userService := filepath.Join(home, ".config/systemd/user/g0router.service")
	assertFileContains(t, userService, filepath.Join(home, ".local/bin/g0router")+" serve")
	assertFileContains(t, userService, "ReadWritePaths="+filepath.Join(home, ".g0router"))
	assertFileContains(t, userService, "WantedBy=default.target")
	assertFileOmits(t, userService, "User=")
	assertFileOmits(t, userService, "Group=")
	assertFileOmits(t, userService, "/etc/default/g0router")
	assertFileOmits(t, userService, "multi-user.target")
	assertFileOmits(t, userService, "ProtectHome=true")
	if _, err := os.Stat(filepath.Join(home, ".g0router")); err != nil {
		t.Fatalf("data dir missing: %v", err)
	}
	recorder.assertRan(t, "systemctl --user daemon-reload")
	recorder.assertRan(t, "systemctl --user enable --now g0router")
	if got := out.String(); !strings.Contains(got, "installed user service") {
		t.Fatalf("output = %q, want installed user service", got)
	}
}

func TestUninstallSystemDisablesAndRemovesManagedFiles(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "usr/local/bin/g0router"), "binary")
	mustWrite(t, filepath.Join(root, "etc/systemd/system/g0router.service"), "unit")
	mustWrite(t, filepath.Join(root, "etc/default/g0router"), "defaults")
	dataDir := filepath.Join(root, "var/lib/g0router")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}
	recorder := &commandRecorder{}

	var out bytes.Buffer
	err := runUninstall(installOptions{
		Root:       root,
		HomeDir:    filepath.Join(root, "home"),
		RunCommand: recorder.run,
		Out:        &out,
	})
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}

	for _, removed := range []string{
		filepath.Join(root, "usr/local/bin/g0router"),
		filepath.Join(root, "etc/systemd/system/g0router.service"),
		filepath.Join(root, "etc/default/g0router"),
	} {
		if _, err := os.Stat(removed); !os.IsNotExist(err) {
			t.Fatalf("%s still exists or stat failed: %v", removed, err)
		}
	}
	if _, err := os.Stat(dataDir); err != nil {
		t.Fatalf("data dir should be preserved: %v", err)
	}
	recorder.assertRan(t, "systemctl disable --now g0router")
	recorder.assertRan(t, "systemctl daemon-reload")
	if got := out.String(); !strings.Contains(got, "kept data") {
		t.Fatalf("output = %q, want kept data", got)
	}
}

func TestDeployTemplatesExist(t *testing.T) {
	service, err := os.ReadFile(filepath.Join("..", "..", "deploy", "g0router.service"))
	if err != nil {
		t.Fatalf("read service: %v", err)
	}
	for _, want := range []string{"[Unit]", "ExecStart=/usr/local/bin/g0router serve", "ReadWritePaths=/var/lib/g0router"} {
		if !strings.Contains(string(service), want) {
			t.Fatalf("service = %q, want %q", string(service), want)
		}
	}

	defaults, err := os.ReadFile(filepath.Join("..", "..", "deploy", "g0router.default"))
	if err != nil {
		t.Fatalf("read defaults: %v", err)
	}
	for _, want := range []string{"PORT=20128", "DATA_DIR=/var/lib/g0router", "REQUIRE_API_KEY=true"} {
		if !strings.Contains(string(defaults), want) {
			t.Fatalf("defaults = %q, want %q", string(defaults), want)
		}
	}
}

type commandRecorder struct {
	commands []string
	fail     map[string]bool
}

type secretGenerator struct {
	values []string
	index  int
}

func newSecretGenerator(values ...string) *secretGenerator {
	return &secretGenerator{values: values}
}

func (g *secretGenerator) next() (string, error) {
	if g.index >= len(g.values) {
		return "", os.ErrNotExist
	}
	value := g.values[g.index]
	g.index++
	return value, nil
}

func (r *commandRecorder) run(name string, args ...string) error {
	command := strings.TrimSpace(name + " " + strings.Join(args, " "))
	r.commands = append(r.commands, command)
	if r.fail[command] {
		return os.ErrNotExist
	}
	return nil
}

func (r *commandRecorder) assertRan(t *testing.T, want string) {
	t.Helper()
	for _, got := range r.commands {
		if got == want {
			return
		}
	}
	t.Fatalf("commands = %#v, want %q", r.commands, want)
}

func writeExecutable(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "g0router-test")
	mustWrite(t, path, "test binary")
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod binary: %v", err)
	}
	return path
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(content), want) {
		t.Fatalf("%s = %q, want %q", path, string(content), want)
	}
}

func assertFileOmits(t *testing.T, path, unwanted string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if strings.Contains(string(content), unwanted) {
		t.Fatalf("%s = %q, should omit %q", path, string(content), unwanted)
	}
}
