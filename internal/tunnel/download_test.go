package tunnel

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadCloudflaredSuccess(t *testing.T) {
	content := []byte("fake-cloudflared-binary")
	hash := sha256.Sum256(content)
	checksum := "sha256:" + hex.EncodeToString(hash[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := fmt.Sprintf("/%s/cloudflared-linux-amd64", cloudflaredVersion)
		if r.URL.Path != expectedPath {
			t.Errorf("path = %q, want %q", r.URL.Path, expectedPath)
		}
		w.Write(content)
	}))
	defer server.Close()

	dataDir := t.TempDir()
	binPath, err := downloadCloudflared(dataDir, "linux", "amd64", server.URL, map[string]string{
		"linux-amd64": checksum,
	})
	if err != nil {
		t.Fatalf("downloadCloudflared: %v", err)
	}

	wantPath := filepath.Join(dataDir, "bin", "cloudflared")
	if binPath != wantPath {
		t.Fatalf("path = %q, want %q", binPath, wantPath)
	}

	info, err := os.Stat(binPath)
	if err != nil {
		t.Fatalf("stat binary: %v", err)
	}

	if info.Mode().Perm()&0111 == 0 {
		t.Fatalf("binary is not executable: %o", info.Mode().Perm())
	}

	data, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("read binary: %v", err)
	}

	if string(data) != string(content) {
		t.Fatalf("binary content mismatch")
	}
}

func TestDownloadCloudflaredChecksumMismatch(t *testing.T) {
	content := []byte("fake-cloudflared-binary")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer server.Close()

	dataDir := t.TempDir()
	_, err := downloadCloudflared(dataDir, "linux", "amd64", server.URL, map[string]string{
		"linux-amd64": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
	})
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}
}

func TestDownloadCloudflaredUnsupportedPlatform(t *testing.T) {
	dataDir := t.TempDir()
	_, err := downloadCloudflared(dataDir, "windows", "386", "http://example.com", map[string]string{
		"linux-amd64": "sha256:0000",
	})
	if err == nil {
		t.Fatal("expected unsupported platform error")
	}
}

func TestDownloadCloudflaredHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	dataDir := t.TempDir()
	_, err := downloadCloudflared(dataDir, "linux", "amd64", server.URL, map[string]string{
		"linux-amd64": "sha256:0000",
	})
	if err == nil {
		t.Fatal("expected HTTP error")
	}
}

func TestDownloadCloudflaredPublicAPI(t *testing.T) {
	// Ensure DownloadCloudflared returns an error for unsupported platform
	// by using the real checksum map with a bogus OS/arch.
	_, err := downloadCloudflared(t.TempDir(), " Plan9", "mips", "http://example.com", cloudflaredChecksums)
	if err == nil {
		t.Fatal("expected unsupported platform error from public API path")
	}
}
