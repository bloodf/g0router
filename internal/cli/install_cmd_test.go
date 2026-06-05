package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// NewInstallCommand / newUninstallCommand cobra wrappers are exercised only via
// their structure, not Execute(), because runInstall writes to real systemd
// paths under the host root. The underlying runInstall/runUninstall logic is
// fully covered through install_errors_test.go with injected options.
func TestInstallCommandsConstruct(t *testing.T) {
	if c := NewInstallCommand(); c.Use != "install" || c.RunE == nil {
		t.Fatalf("install command = %+v", c)
	}
	if c := newUninstallCommand(); c.Use != "uninstall" || c.RunE == nil {
		t.Fatalf("uninstall command = %+v", c)
	}
}

func TestServiceTemplateUserOmitsLines(t *testing.T) {
	out, err := serviceTemplate("/bin/g0router", "/data", true)
	if err != nil {
		t.Fatalf("serviceTemplate: %v", err)
	}
	if strings.Contains(out, "User=g0router") {
		t.Fatalf("user template still has User= line: %q", out)
	}
	if !strings.Contains(out, "WantedBy=default.target") {
		t.Fatalf("user template missing default.target: %q", out)
	}
}

func TestRenderDefaultTemplateNoPlaceholder(t *testing.T) {
	content := []byte("PORT=1\n")
	out, err := renderDefaultTemplate(content, failingSecret{}.next)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if string(out) != "PORT=1\n" {
		t.Fatalf("out = %q", out)
	}
}

func TestReadDeployTemplateMissing(t *testing.T) {
	if _, err := readDeployTemplate("does-not-exist.tmpl"); err == nil ||
		!strings.Contains(err.Error(), "not found") {
		t.Fatalf("err = %v", err)
	}
}

func TestCopyFileRenameTargetDirMissing(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.WriteFile(src, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	// dst whose parent does not exist -> CreateTemp in missing dir fails.
	dst := filepath.Join(dir, "no-such-dir", "dst")
	if err := copyFile(src, dst, 0o644); err == nil {
		t.Fatal("expected error for missing dst dir")
	}
}
