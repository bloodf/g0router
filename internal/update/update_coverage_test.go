package update

import (
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
)

func TestCheckerWithServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tag_name":"v1.1.0","html_url":"https://example.com/v1.1.0"}`)
	}))
	defer server.Close()

	checker := NewChecker()
	checker.url = server.URL

	result, err := checker.Check("1.0.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !result.UpdateAvailable {
		t.Fatal("expected update available")
	}
	if result.Latest != "1.1.0" {
		t.Fatalf("latest = %q, want 1.1.0", result.Latest)
	}
}

func TestCheckerNoUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tag_name":"v1.0.0","html_url":"https://example.com/v1.0.0"}`)
	}))
	defer server.Close()

	checker := NewChecker()
	checker.url = server.URL

	result, err := checker.Check("1.0.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.UpdateAvailable {
		t.Fatal("expected no update")
	}
}

func TestCheckerServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	checker := NewChecker()
	checker.url = server.URL

	_, err := checker.Check("1.0.0")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCheckerInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{invalid`)
	}))
	defer server.Close()

	checker := NewChecker()
	checker.url = server.URL

	_, err := checker.Check("1.0.0")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdaterFetchChecksum(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "abc123  my-asset\n")
	}))
	defer server.Close()

	updater := NewUpdater()
	updater.client = server.Client()

	got, err := updater.fetchChecksum(server.URL, "my-asset")
	if err != nil {
		t.Fatalf("fetchChecksum: %v", err)
	}
	if got != "abc123" {
		t.Fatalf("checksum = %q, want abc123", got)
	}
}

func TestUpdaterFetchChecksumNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "abc123  other-asset\n")
	}))
	defer server.Close()

	updater := NewUpdater()
	updater.client = server.Client()

	_, err := updater.fetchChecksum(server.URL, "my-asset")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdaterDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	}))
	defer server.Close()

	updater := NewUpdater()
	updater.client = server.Client()

	got, err := updater.download(server.URL)
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("content = %q, want hello", string(got))
	}
}

func TestUpdaterDownloadServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	updater := NewUpdater()
	updater.client = server.Client()

	_, err := updater.download(server.URL)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyStagesBinary(t *testing.T) {
	binaryContent := []byte("fake-binary-content")
	sum := sha256sum(binaryContent)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "checksums.txt") {
			fmt.Fprintf(w, "%s  g0router-%s-%s\n", sum, runtime.GOOS, runtime.GOARCH)
			return
		}
		if strings.Contains(path, "g0router-") {
			w.Write(binaryContent)
			return
		}
		if strings.Contains(path, "releases/latest") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tag_name":"v1.1.0","html_url":"https://example.com/v1.1.0"}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	updater := NewUpdater()
	updater.client = server.Client()
	updater.baseURL = server.URL
	updater.checker = &Checker{client: server.Client(), url: server.URL + "/releases/latest"}

	dataDir := t.TempDir()
	if err := updater.Apply("1.0.0", dataDir); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	stagePath := filepath.Join(dataDir, "update", "g0router.new")
	staged, err := os.ReadFile(stagePath)
	if err != nil {
		t.Fatalf("read staged file: %v", err)
	}
	if string(staged) != string(binaryContent) {
		t.Fatalf("staged content mismatch")
	}
}

func TestApplyChecksumMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "checksums.txt") {
			fmt.Fprintf(w, "badchecksum  g0router-%s-%s\n", runtime.GOOS, runtime.GOARCH)
			return
		}
		if strings.Contains(path, "g0router-") {
			w.Write([]byte("fake-binary"))
			return
		}
		if strings.Contains(path, "releases/latest") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tag_name":"v1.1.0","html_url":"https://example.com/v1.1.0"}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	updater := NewUpdater()
	updater.client = server.Client()
	updater.baseURL = server.URL
	updater.checker = &Checker{client: server.Client(), url: server.URL + "/releases/latest"}

	dataDir := t.TempDir()
	err := updater.Apply("1.0.0", dataDir)
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("error = %q, want checksum mismatch", err.Error())
	}
}

func TestApplyNoUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tag_name":"v1.0.0","html_url":"https://example.com/v1.0.0"}`)
	}))
	defer server.Close()

	updater := NewUpdater()
	updater.baseURL = server.URL
	updater.checker = &Checker{client: server.Client(), url: server.URL}

	dataDir := t.TempDir()
	if err := updater.Apply("1.0.0", dataDir); err != nil {
		t.Fatalf("Apply: %v", err)
	}
}

func TestApplyMkdirError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "checksums.txt") {
			fmt.Fprintf(w, "abc  g0router-%s-%s\n", runtime.GOOS, runtime.GOARCH)
			return
		}
		if strings.Contains(path, "g0router-") {
			w.Write([]byte("x"))
			return
		}
		if strings.Contains(path, "releases/latest") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tag_name":"v1.1.0","html_url":"https://example.com/v1.1.0"}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	updater := NewUpdater()
	updater.client = server.Client()
	updater.baseURL = server.URL
	updater.checker = &Checker{client: server.Client(), url: server.URL + "/releases/latest"}

	// Use a file as dataDir so MkdirAll fails
	f, err := os.CreateTemp("", "g0router-apply-test-*")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	if err := updater.Apply("1.0.0", f.Name()); err == nil {
		t.Fatal("expected mkdir error")
	}
}

func TestDownloadNetworkError(t *testing.T) {
	updater := NewUpdater()
	_, err := updater.download("http://localhost:1/nonexistent")
	if err == nil {
		t.Fatal("expected network error")
	}
}

func TestFetchChecksumNetworkError(t *testing.T) {
	updater := NewUpdater()
	_, err := updater.fetchChecksum("http://localhost:1/nonexistent", "asset")
	if err == nil {
		t.Fatal("expected network error")
	}
}

func TestCheckerNetworkError(t *testing.T) {
	checker := NewChecker()
	checker.url = "http://localhost:1/nonexistent"
	_, err := checker.Check("1.0.0")
	if err == nil {
		t.Fatal("expected network error")
	}
}

func TestVersionGreaterLongerSemver(t *testing.T) {
	if !versionGreater("1.0.0.1", "1.0.0") {
		t.Fatal("expected 1.0.0.1 > 1.0.0")
	}
}

func sha256sum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
