package tunnel

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const cloudflaredVersion = "2024.6.1"

var cloudflaredChecksums = map[string]string{
	"darwin-amd64": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
	"darwin-arm64": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
	"linux-amd64":  "sha256:0000000000000000000000000000000000000000000000000000000000000000",
	"linux-arm64":  "sha256:0000000000000000000000000000000000000000000000000000000000000000",
}

var downloadBaseURL = "https://github.com/cloudflare/cloudflared/releases/download"

// DownloadCloudflared downloads the pinned cloudflared binary for the current
// platform, verifies its SHA-256 checksum, makes it executable, and returns
// the absolute path to the binary.
func DownloadCloudflared(dataDir string) (string, error) {
	return downloadCloudflared(dataDir, runtime.GOOS, runtime.GOARCH, downloadBaseURL, cloudflaredChecksums)
}

func downloadCloudflared(dataDir, goos, goarch string, baseURL string, checksums map[string]string) (string, error) {
	key := goos + "-" + goarch
	checksum, ok := checksums[key]
	if !ok {
		return "", fmt.Errorf("unsupported platform %s", key)
	}

	url := fmt.Sprintf("%s/%s/cloudflared-%s", baseURL, cloudflaredVersion, key)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("download cloudflared: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download cloudflared: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	hash := sha256.Sum256(data)
	expectedHash := strings.TrimPrefix(checksum, "sha256:")
	if hex.EncodeToString(hash[:]) != expectedHash {
		return "", fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, hex.EncodeToString(hash[:]))
	}

	binDir := filepath.Join(dataDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("create bin directory: %w", err)
	}

	binPath := filepath.Join(binDir, "cloudflared")
	tmpPath := binPath + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return "", fmt.Errorf("write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, binPath); err != nil {
		return "", fmt.Errorf("rename temp file: %w", err)
	}

	if err := os.Chmod(binPath, 0755); err != nil {
		return "", fmt.Errorf("chmod binary: %w", err)
	}

	return binPath, nil
}
