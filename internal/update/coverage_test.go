package update

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type errorReader struct{}

func (e *errorReader) Read(p []byte) (int, error) { return 0, errors.New("read fail") }
func (e *errorReader) Close() error               { return nil }

type errorBodyTransport struct{}

func (t *errorBodyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       &errorReader{},
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func TestCheckerReadBodyError(t *testing.T) {
	checker := NewChecker()
	checker.client = &http.Client{Transport: &errorBodyTransport{}}
	checker.url = "http://example.com/release"
	_, err := checker.Check("1.0.0")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCheckerBadURL(t *testing.T) {
	checker := NewChecker()
	checker.url = "://bad-url"
	_, err := checker.Check("1.0.0")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFetchChecksumStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	updater := NewUpdater()
	updater.client = server.Client()

	_, err := updater.fetchChecksum(server.URL, "asset")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "status") {
		t.Fatalf("error = %q, want status error", err.Error())
	}
}

func TestFetchChecksumReadBodyError(t *testing.T) {
	updater := NewUpdater()
	updater.client = &http.Client{Transport: &errorBodyTransport{}}

	_, err := updater.fetchChecksum("http://example.com/checksums.txt", "asset")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyWriteFileError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "checksums.txt") {
			sum := sha256sum([]byte("x"))
			fmt.Fprintf(w, "%s  g0router-%s-%s\n", sum, runtime.GOOS, runtime.GOARCH)
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

	dataDir := t.TempDir()
	stageDir := filepath.Join(dataDir, "update")
	if err := os.MkdirAll(stageDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Chmod(stageDir, 0555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer os.Chmod(stageDir, 0755)

	err := updater.Apply("1.0.0", dataDir)
	if err == nil {
		t.Fatal("expected write file error")
	}
}

func TestVersionGreaterPrerelease(t *testing.T) {
	if !versionGreater("1.0.0-beta", "1.0.0-alpha") {
		t.Fatal("expected beta > alpha")
	}
	if versionGreater("1.0.0-alpha", "1.0.0-beta") {
		t.Fatal("expected alpha < beta")
	}
}
