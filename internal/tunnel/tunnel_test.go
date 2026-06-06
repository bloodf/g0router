package tunnel

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/store"
)

type fakeStore struct {
	configs []store.TunnelConfig
	status  map[string]struct {
		status    string
		lastError string
	}
}

func (f *fakeStore) ListTunnelConfigs() ([]store.TunnelConfig, error) {
	return f.configs, nil
}

func (f *fakeStore) UpsertTunnelConfig(cfg store.TunnelConfig) error {
	found := false
	for i := range f.configs {
		if f.configs[i].Type == cfg.Type {
			f.configs[i] = cfg
			found = true
			break
		}
	}
	if !found {
		f.configs = append(f.configs, cfg)
	}
	return nil
}

func (f *fakeStore) UpdateTunnelStatus(tunnelType, status, lastError string) error {
	if f.status == nil {
		f.status = make(map[string]struct{ status, lastError string })
	}
	f.status[tunnelType] = struct{ status, lastError string }{status, lastError}
	return nil
}

func TestManagerStartCloudflare(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	dataDir := t.TempDir()
	binDir := filepath.Join(dataDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	script := "#!/bin/sh\necho \"https://test-abc.trycloudflare.com\"\nsleep 3600\n"
	binPath := filepath.Join(binDir, "cloudflared")
	if err := os.WriteFile(binPath, []byte(script), 0755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	fs := &fakeStore{}
	mgr := NewManager(fs, dataDir)

	url, err := mgr.StartCloudflare("8080")
	if err != nil {
		t.Fatalf("StartCloudflare: %v", err)
	}
	if url != "https://test-abc.trycloudflare.com" {
		t.Fatalf("url = %q, want https://test-abc.trycloudflare.com", url)
	}

	st, ok := fs.status["cloudflare"]
	if !ok || st.status != "active" {
		t.Fatalf("status = %+v, want active", st)
	}
}

func TestManagerStopCloudflare(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	dataDir := t.TempDir()
	binDir := filepath.Join(dataDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	script := "#!/bin/sh\necho \"https://test-abc.trycloudflare.com\"\nsleep 3600\n"
	binPath := filepath.Join(binDir, "cloudflared")
	if err := os.WriteFile(binPath, []byte(script), 0755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	fs := &fakeStore{}
	mgr := NewManager(fs, dataDir)

	if _, err := mgr.StartCloudflare("8080"); err != nil {
		t.Fatalf("StartCloudflare: %v", err)
	}

	if err := mgr.StopCloudflare(); err != nil {
		t.Fatalf("StopCloudflare: %v", err)
	}

	st, ok := fs.status["cloudflare"]
	if !ok || st.status != "inactive" {
		t.Fatalf("status = %+v, want inactive", st)
	}
}

func TestManagerStopCloudflareWhenNotRunning(t *testing.T) {
	fs := &fakeStore{}
	mgr := NewManager(fs, t.TempDir())

	if err := mgr.StopCloudflare(); err != nil {
		t.Fatalf("StopCloudflare when not running: %v", err)
	}
}

func TestManagerStartTailscaleNotOnPath(t *testing.T) {
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "/nonexistent")
	defer os.Setenv("PATH", oldPath)

	fs := &fakeStore{}
	mgr := NewManager(fs, t.TempDir())

	_, err := mgr.StartTailscale("8080")
	if err == nil {
		t.Fatal("expected error when tailscale not on PATH")
	}

	st, ok := fs.status["tailscale"]
	if !ok || st.status != "error" {
		t.Fatalf("status = %+v, want error", st)
	}
}

func TestManagerPortValidation(t *testing.T) {
	fs := &fakeStore{}
	mgr := NewManager(fs, t.TempDir())

	invalidPorts := []string{"", "abc", "0", "70000", "80; rm -rf /"}
	for _, port := range invalidPorts {
		_, err := mgr.StartCloudflare(port)
		if err == nil {
			t.Fatalf("expected error for port %q", port)
		}
	}
}

func TestManagerPortValidationTailscale(t *testing.T) {
	fs := &fakeStore{}
	mgr := NewManager(fs, t.TempDir())

	_, err := mgr.StartTailscale("abc")
	if err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestValidateTunnelName(t *testing.T) {
	valid := []string{"my-tunnel", "tunnel123", "a", strings.Repeat("a", 63)}
	for _, name := range valid {
		if err := validateTunnelName(name); err != nil {
			t.Errorf("validateTunnelName(%q): unexpected error: %v", name, err)
		}
	}

	invalid := []string{"", "Tunnel_1", "tunnel.name", "tunnel space", "a$b", strings.Repeat("a", 64), "meta;rm -rf /"}
	for _, name := range invalid {
		if err := validateTunnelName(name); err == nil {
			t.Errorf("validateTunnelName(%q): expected error", name)
		}
	}
}
