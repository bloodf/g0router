//go:build e2ebin

// Package g0router_test contains an opt-in end-to-end smoke test that builds the
// real g0router binary, runs it as a process, and exercises the HTTP surface the
// way a deployed instance is used. It is excluded from the default test suite
// (build tag e2ebin) because it compiles a binary and binds a socket; run it
// with `make e2e-binary`.
package g0router_test

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestE2EBinaryServesAndAuthenticates(t *testing.T) {
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "g0router")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/g0router")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("build binary: %v", err)
	}

	dataDir := t.TempDir()
	port := freePort(t)
	const secret = "e2e-secret"

	// Mint an API key against the same data dir the server will use.
	keyCmd := exec.Command(binPath, "keys", "add", "e2e", "--data-dir", dataDir)
	keyCmd.Env = append(os.Environ(), "API_KEY_SECRET="+secret)
	out, err := keyCmd.Output()
	if err != nil {
		t.Fatalf("keys add: %v", err)
	}
	fields := strings.Fields(string(out))
	if len(fields) < 2 || !strings.HasPrefix(fields[1], "g0r_") {
		t.Fatalf("unexpected keys add output: %q", string(out))
	}
	apiKey := fields[1]

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := exec.CommandContext(ctx, binPath, "serve", "--port", fmt.Sprint(port), "--data-dir", dataDir)
	srv.Env = append(os.Environ(),
		"API_KEY_SECRET="+secret,
		"REQUIRE_API_KEY=true",
		"BIND_ADDRESS=127.0.0.1",
	)
	srv.Stderr = os.Stderr
	if err := srv.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer func() {
		cancel()
		_ = srv.Wait()
	}()

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	waitReady(t, base+"/healthz")

	auth := func(req *http.Request) { req.Header.Set("Authorization", "Bearer "+apiKey) }

	assertStatus(t, "GET", base+"/healthz", nil, http.StatusOK)
	assertStatus(t, "GET", base+"/", nil, http.StatusOK)
	// Inference + control plane require a valid key.
	assertStatus(t, "GET", base+"/v1/models", nil, http.StatusUnauthorized)
	assertStatus(t, "GET", base+"/v1/models", auth, http.StatusOK)
	assertStatus(t, "GET", base+"/api/connections", nil, http.StatusUnauthorized)
	assertStatus(t, "GET", base+"/api/connections", auth, http.StatusOK)
	assertStatus(t, "GET", base+"/api/providers", auth, http.StatusOK)
	assertStatus(t, "GET", base+"/api/usage", auth, http.StatusOK)
	assertStatus(t, "GET", base+"/api/mcp/instances", auth, http.StatusOK)
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("free port: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func waitReady(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("server did not become ready at %s", url)
}

func assertStatus(t *testing.T, method, url string, decorate func(*http.Request), want int) {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("new request %s %s: %v", method, url, err)
	}
	if decorate != nil {
		decorate(req)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	_ = bufio.NewReader(resp.Body)
	if resp.StatusCode != want {
		t.Fatalf("%s %s = %d, want %d", method, url, resp.StatusCode, want)
	}
}
