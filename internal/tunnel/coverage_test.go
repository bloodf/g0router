package tunnel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

// ========== download.go error branches ==========

func TestDownloadCloudflaredNetworkError(t *testing.T) {
	dataDir := t.TempDir()
	_, err := downloadCloudflared(dataDir, "linux", "amd64", "http://[::1]:1", map[string]string{
		"linux-amd64": "sha256:0000",
	})
	if err == nil {
		t.Fatal("expected network error")
	}
}

func TestDownloadCloudflaredReadBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h, ok := w.(http.Hijacker)
		if !ok {
			t.Skip("server does not support hijacking")
		}
		conn, _, err := h.Hijack()
		if err != nil {
			t.Fatalf("hijack: %v", err)
		}
		conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\npartial"))
		conn.Close()
	}))
	defer server.Close()

	dataDir := t.TempDir()
	_, err := downloadCloudflared(dataDir, "linux", "amd64", server.URL, map[string]string{
		"linux-amd64": "sha256:0000",
	})
	if err == nil {
		t.Fatal("expected read body error")
	}
}

func TestDownloadCloudflaredMkdirAllError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	content := []byte("fake-cloudflared-binary")
	hash := sha256.Sum256(content)
	checksum := "sha256:" + hex.EncodeToString(hash[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer server.Close()

	dataDir := "/dev/null/not-a-dir"
	_, err := downloadCloudflared(dataDir, "linux", "amd64", server.URL, map[string]string{
		"linux-amd64": checksum,
	})
	if err == nil {
		t.Fatal("expected mkdirall error")
	}
}

func TestDownloadCloudflaredWriteFileError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	content := []byte("fake-cloudflared-binary")
	hash := sha256.Sum256(content)
	checksum := "sha256:" + hex.EncodeToString(hash[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer server.Close()

	dataDir := t.TempDir()
	binDir := filepath.Join(dataDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Chmod(binDir, 0555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer os.Chmod(binDir, 0755)

	_, err := downloadCloudflared(dataDir, "linux", "amd64", server.URL, map[string]string{
		"linux-amd64": checksum,
	})
	if err == nil {
		t.Fatal("expected write file error")
	}
}

func TestDownloadCloudflaredRenameError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	content := []byte("fake-cloudflared-binary")
	hash := sha256.Sum256(content)
	checksum := "sha256:" + hex.EncodeToString(hash[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer server.Close()

	dataDir := t.TempDir()
	binDir := filepath.Join(dataDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Create a directory at the destination path so os.Rename fails.
	binPath := filepath.Join(binDir, "cloudflared")
	if err := os.MkdirAll(binPath, 0755); err != nil {
		t.Fatalf("mkdir binPath: %v", err)
	}

	_, err := downloadCloudflared(dataDir, "linux", "amd64", server.URL, map[string]string{
		"linux-amd64": checksum,
	})
	if err == nil {
		t.Fatal("expected rename error")
	}
}

func TestDownloadCloudflaredPublicAPICoverage(t *testing.T) {
	oldBaseURL := downloadBaseURL
	oldChecksums := cloudflaredChecksums
	defer func() {
		downloadBaseURL = oldBaseURL
		cloudflaredChecksums = oldChecksums
	}()

	content := []byte("fake-cloudflared-binary")
	hash := sha256.Sum256(content)
	checksum := "sha256:" + hex.EncodeToString(hash[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer server.Close()

	downloadBaseURL = server.URL
	key := runtime.GOOS + "-" + runtime.GOARCH
	cloudflaredChecksums = map[string]string{
		key: checksum,
	}

	dataDir := t.TempDir()
	binPath, err := DownloadCloudflared(dataDir)
	if err != nil {
		t.Fatalf("DownloadCloudflared: %v", err)
	}
	if binPath == "" {
		t.Fatal("expected non-empty binPath")
	}
}

// ========== supervisor.go error branches ==========

func TestSupervisorStartProcessFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	// Use a directory as the binary path — exec.Start will fail
	dir := t.TempDir()
	s := &Supervisor{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.Start(ctx, dir, []string{})
	if err == nil {
		t.Fatal("expected process start error")
	}
}

func TestSupervisorStartStderrOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	script := "#!/bin/sh\necho 'error message' >&2\necho 'https://test-abc.trycloudflare.com'\nsleep 3600\n"
	binPath := writeFakeBinary(t, t.TempDir(), "fake-stderr", script)

	s := &Supervisor{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Start(ctx, binPath, []string{}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()

	// Give discardOutput time to consume stderr
	time.Sleep(200 * time.Millisecond)
}

// ========== tunnel.go error branches ==========

func TestValidatePortAtoiOverflow(t *testing.T) {
	port := strings.Repeat("9", 30)
	err := validatePort(port)
	if err == nil {
		t.Fatal("expected error for Atoi overflow")
	}
}

func TestManagerStartCloudflareDownloadError(t *testing.T) {
	oldBaseURL := downloadBaseURL
	defer func() { downloadBaseURL = oldBaseURL }()

	downloadBaseURL = "http://[::1]:1"

	fs := &fakeStore{}
	mgr := NewManager(fs, t.TempDir())

	_, err := mgr.StartCloudflare("8080")
	if err == nil {
		t.Fatal("expected error when download fails")
	}

	st, ok := fs.status["cloudflare"]
	if !ok || st.status != "error" {
		t.Fatalf("status = %+v, want error", st)
	}
}

func TestManagerStartCloudflareBinaryStartFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	dataDir := t.TempDir()
	binDir := filepath.Join(dataDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	binPath := filepath.Join(binDir, "cloudflared")
	if err := os.WriteFile(binPath, []byte("not executable"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	fs := &fakeStore{}
	mgr := NewManager(fs, dataDir)

	_, err := mgr.StartCloudflare("8080")
	if err == nil {
		t.Fatal("expected error when binary fails to start")
	}
}

func TestManagerStartCloudflareWaitForURLTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	dataDir := t.TempDir()
	binDir := filepath.Join(dataDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	script := "#!/bin/sh\nsleep 3600\n"
	binPath := filepath.Join(binDir, "cloudflared")
	if err := os.WriteFile(binPath, []byte(script), 0755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	fs := &fakeStore{}
	mgr := NewManager(fs, dataDir)

	_, err := mgr.StartCloudflare("8080")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestManagerStopCloudflareSupervisorTimeout(t *testing.T) {
	s := &Supervisor{}
	s.done = make(chan struct{}) // never closed
	s.cancel = func() {}         // non-nil cancel

	fs := &fakeStore{}
	mgr := NewManager(fs, t.TempDir())
	mgr.supervisor = s

	start := time.Now()
	err := mgr.StopCloudflare()
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("error = %q, want timeout", err.Error())
	}
	if time.Since(start) < 4*time.Second {
		t.Fatalf("Stop returned too quickly")
	}
}

func TestManagerStartTailscaleSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	tmpDir := t.TempDir()
	script := "#!/bin/sh\necho 'https://test-abc.tailnet.com'\nsleep 3600\n"
	fakeTailscale := filepath.Join(tmpDir, "tailscale")
	if err := os.WriteFile(fakeTailscale, []byte(script), 0755); err != nil {
		t.Fatalf("write fake tailscale: %v", err)
	}

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+string(filepath.ListSeparator)+oldPath)
	defer os.Setenv("PATH", oldPath)

	fs := &fakeStore{}
	mgr := NewManager(fs, t.TempDir())

	url, err := mgr.StartTailscale("8080")
	if err != nil {
		t.Fatalf("StartTailscale: %v", err)
	}
	if url != "https://test-abc.tailnet.com" {
		t.Fatalf("url = %q, want https://test-abc.tailnet.com", url)
	}

	st, ok := fs.status["tailscale"]
	if !ok || st.status != "active" {
		t.Fatalf("status = %+v, want active", st)
	}

	if err := mgr.StopTailscale(); err != nil {
		t.Fatalf("StopTailscale: %v", err)
	}

	st, ok = fs.status["tailscale"]
	if !ok || st.status != "inactive" {
		t.Fatalf("status = %+v, want inactive", st)
	}
}

func TestManagerStartTailscaleStartFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	tmpDir := t.TempDir()
	fakeTailscale := filepath.Join(tmpDir, "tailscale")
	// Use a bad interpreter so exec.LookPath finds it but cmd.Start fails.
	script := "#!/nonexistent\nexit 1\n"
	if err := os.WriteFile(fakeTailscale, []byte(script), 0755); err != nil {
		t.Fatalf("write fake tailscale: %v", err)
	}

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+string(filepath.ListSeparator)+oldPath)
	defer os.Setenv("PATH", oldPath)

	fs := &fakeStore{}
	mgr := NewManager(fs, t.TempDir())

	_, err := mgr.StartTailscale("8080")
	if err == nil {
		t.Fatal("expected error when tailscale binary fails to start")
	}

	st, ok := fs.status["tailscale"]
	if !ok || st.status != "error" {
		t.Fatalf("status = %+v, want error", st)
	}
}

func TestManagerStartTailscaleWaitForURLTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	tmpDir := t.TempDir()
	script := "#!/bin/sh\nsleep 3600\n"
	fakeTailscale := filepath.Join(tmpDir, "tailscale")
	if err := os.WriteFile(fakeTailscale, []byte(script), 0755); err != nil {
		t.Fatalf("write fake tailscale: %v", err)
	}

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+string(filepath.ListSeparator)+oldPath)
	defer os.Setenv("PATH", oldPath)

	fs := &fakeStore{}
	mgr := NewManager(fs, t.TempDir())

	_, err := mgr.StartTailscale("8080")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestManagerStopTailscaleWhenNotRunning(t *testing.T) {
	fs := &fakeStore{}
	mgr := NewManager(fs, t.TempDir())

	if err := mgr.StopTailscale(); err != nil {
		t.Fatalf("StopTailscale when not running: %v", err)
	}
}

func TestManagerStopTailscaleSupervisorTimeout(t *testing.T) {
	s := &Supervisor{}
	s.done = make(chan struct{}) // never closed
	s.cancel = func() {}         // non-nil cancel

	fs := &fakeStore{}
	mgr := NewManager(fs, t.TempDir())
	mgr.supervisor = s

	start := time.Now()
	err := mgr.StopTailscale()
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("error = %q, want timeout", err.Error())
	}
	if time.Since(start) < 4*time.Second {
		t.Fatalf("Stop returned too quickly")
	}
}

// errorStore returns errors from all methods to exercise ignored-error lines.
type errorStore struct{}

func (e *errorStore) ListTunnelConfigs() ([]store.TunnelConfig, error) {
	return nil, fmt.Errorf("list error")
}

func (e *errorStore) UpsertTunnelConfig(cfg store.TunnelConfig) error {
	return fmt.Errorf("upsert error")
}

func (e *errorStore) UpdateTunnelStatus(tunnelType, status, lastError string) error {
	return fmt.Errorf("update error")
}

func TestManagerStartCloudflareWithStoreErrors(t *testing.T) {
	oldBaseURL := downloadBaseURL
	defer func() { downloadBaseURL = oldBaseURL }()

	downloadBaseURL = "http://[::1]:1"

	es := &errorStore{}
	mgr := NewManager(es, t.TempDir())

	_, err := mgr.StartCloudflare("8080")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestManagerStartTailscaleWithStoreErrors(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	tmpDir := t.TempDir()
	script := "#!/bin/sh\necho 'https://test-abc.tailnet.com'\nsleep 3600\n"
	fakeTailscale := filepath.Join(tmpDir, "tailscale")
	if err := os.WriteFile(fakeTailscale, []byte(script), 0755); err != nil {
		t.Fatalf("write fake tailscale: %v", err)
	}

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+string(filepath.ListSeparator)+oldPath)
	defer os.Setenv("PATH", oldPath)

	es := &errorStore{}
	mgr := NewManager(es, t.TempDir())

	_, err := mgr.StartTailscale("8080")
	if err != nil {
		t.Fatalf("StartTailscale: %v", err)
	}

	if err := mgr.StopTailscale(); err != nil {
		t.Fatalf("StopTailscale: %v", err)
	}
}
