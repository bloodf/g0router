package mitm

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProxyToggleOffStopsListener(t *testing.T) {
	dir := t.TempDir()
	p, err := NewProxy(dir, 0)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	if p.IsRunning() {
		t.Fatal("proxy should not be running initially")
	}

	if err := p.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !p.IsRunning() {
		t.Fatal("proxy should be running after Start")
	}

	if err := p.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	// Wait briefly for the listener to close.
	time.Sleep(50 * time.Millisecond)
	if p.IsRunning() {
		t.Fatal("proxy should not be running after Stop")
	}
}

func TestNonToolHostPassThrough(t *testing.T) {
	// Start a backend echo server.
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello from backend"))
	}))
	defer backend.Close()

	dir := t.TempDir()
	p, err := NewProxy(dir, 0)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	if err := p.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer p.Stop()

	// Extract the proxy address.
	proxyAddr := p.Addr()
	backendHost := strings.TrimPrefix(backend.URL, "http://")

	// Dial proxy and send CONNECT.
	conn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", backendHost, backendHost)
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("read connect response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("connect status = %d, want 200", resp.StatusCode)
	}

	// Send raw HTTP request through tunnel.
	fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: %s\r\n\r\n", backendHost)
	resp, err = http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("read tunneled response: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "hello from backend" {
		t.Fatalf("tunneled body = %q, want hello from backend", string(body))
	}
}

func TestToolHostMITM(t *testing.T) {
	// Start a target server that the proxy will forward tool requests to.
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("intercepted"))
	}))
	defer target.Close()

	dir := t.TempDir()
	p, err := NewProxy(dir, 0)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	p.SetTarget(target.URL)

	if err := p.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer p.Stop()

	proxyAddr := p.Addr()
	toolHost := "api.cursor.sh:443"

	conn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", toolHost, toolHost)
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("read connect response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("connect status = %d, want 200", resp.StatusCode)
	}

	// Upgrade to TLS using the leaf cert (we need to trust the CA).
	caPool := p.CAPool()
	tlsConn := tls.Client(conn, &tls.Config{RootCAs: caPool, ServerName: "api.cursor.sh"})
	if err := tlsConn.Handshake(); err != nil {
		t.Fatalf("tls handshake: %v", err)
	}
	defer tlsConn.Close()

	// Send HTTPS request.
	fmt.Fprintf(tlsConn, "GET /v1/chat/completions HTTP/1.1\r\nHost: api.cursor.sh\r\n\r\n")
	resp, err = http.ReadResponse(bufio.NewReader(tlsConn), nil)
	if err != nil {
		t.Fatalf("read https response: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "intercepted" {
		t.Fatalf("mitm body = %q, want intercepted", string(body))
	}
}
