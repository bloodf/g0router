package api

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestHealthz(t *testing.T) {
	srv := NewServer(ServerConfig{Port: 0, Version: "test-version"})
	ln := srv.listener()
	if ln == nil {
		t.Fatal("listener failed")
	}

	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Stop() })

	resp, err := httpClient().Get("http://" + localhostAddr(t, ln) + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("status: %q", result["status"])
	}
	if result["version"] != "test-version" {
		t.Errorf("version: %q", result["version"])
	}
}

func TestUnknownRoute(t *testing.T) {
	srv := NewServer(ServerConfig{Port: 0, Version: "test"})
	ln := srv.listener()
	if ln == nil {
		t.Fatal("listener failed")
	}

	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Stop() })

	resp, err := httpClient().Get("http://" + localhostAddr(t, ln) + "/nope")
	if err != nil {
		t.Fatalf("GET /nope: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 2 * time.Second}
}

func localhostAddr(t *testing.T, ln net.Listener) string {
	t.Helper()

	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener addr is %T, want *net.TCPAddr", ln.Addr())
	}
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(tcpAddr.Port))
}
