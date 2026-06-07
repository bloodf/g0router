package mitm

import (
	"encoding/pem"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCertPEM(t *testing.T) {
	dir := t.TempDir()
	ca, err := LoadOrGenerateCA(dir)
	if err != nil {
		t.Fatalf("LoadOrGenerateCA: %v", err)
	}
	pemBytes := ca.CertPEM()
	if len(pemBytes) == 0 {
		t.Fatal("expected non-empty PEM")
	}
}

func TestProxyNewProxyError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mitm"), []byte{}, 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	_, err := NewProxy(dir, 0)
	if err == nil {
		t.Fatal("expected error when mitm is a file")
	}
}

func TestProxyCACertPEM(t *testing.T) {
	dir := t.TempDir()
	p, err := NewProxy(dir, 0)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	pem := p.CACertPEM()
	if len(pem) == 0 {
		t.Fatal("expected non-empty PEM")
	}
}

func TestProxyToolEnabled(t *testing.T) {
	dir := t.TempDir()
	p, err := NewProxy(dir, 0)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	if p.ToolEnabled("cursor") {
		t.Fatal("expected cursor disabled")
	}
	p.SetToolEnabled("cursor", true)
	if !p.ToolEnabled("cursor") {
		t.Fatal("expected cursor enabled")
	}
}

func TestProxyAddrNilListener(t *testing.T) {
	dir := t.TempDir()
	p, err := NewProxy(dir, 0)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	if p.Addr() != "" {
		t.Fatalf("expected empty addr, got %q", p.Addr())
	}
}

func TestProxyStartListenError(t *testing.T) {
	dir := t.TempDir()
	p, err := NewProxy(dir, 99999)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	err = p.Start()
	if err == nil {
		t.Fatal("expected listen error")
	}
}

func TestProxyStopNotRunning(t *testing.T) {
	dir := t.TempDir()
	p, err := NewProxy(dir, 0)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	if err := p.Stop(); err != nil {
		t.Fatalf("Stop when not running: %v", err)
	}
}

func TestToolInstructions(t *testing.T) {
	inst := ToolInstructions("127.0.0.1:8080")
	if len(inst) != 4 {
		t.Fatalf("expected 4 instructions, got %d", len(inst))
	}
}

func TestOneShotListenerAddr(t *testing.T) {
	l, r := net.Pipe()
	defer l.Close()
	defer r.Close()

	osl := &oneShotListener{conn: l}
	addr := osl.Addr()
	if addr == nil {
		t.Fatal("expected non-nil addr")
	}
}

func TestGenerateCAMkdirAllError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mitm"), []byte{}, 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	_, err := generateCA(filepath.Join(dir, "mitm", "ca.crt"), filepath.Join(dir, "mitm", "ca.key"))
	if err == nil {
		t.Fatal("expected MkdirAll error")
	}
}

func TestGenerateCACertFileError(t *testing.T) {
	dir := t.TempDir()
	mitmDir := filepath.Join(dir, "mitm")
	if err := os.MkdirAll(mitmDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Mkdir(filepath.Join(mitmDir, "ca.crt"), 0755); err != nil {
		t.Fatalf("mkdir ca.crt: %v", err)
	}
	_, err := generateCA(filepath.Join(mitmDir, "ca.crt"), filepath.Join(mitmDir, "ca.key"))
	if err == nil {
		t.Fatal("expected cert file error")
	}
}

func TestGenerateCAKeyFileError(t *testing.T) {
	dir := t.TempDir()
	mitmDir := filepath.Join(dir, "mitm")
	if err := os.MkdirAll(mitmDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Mkdir(filepath.Join(mitmDir, "ca.key"), 0755); err != nil {
		t.Fatalf("mkdir ca.key: %v", err)
	}
	_, err := generateCA(filepath.Join(mitmDir, "ca.crt"), filepath.Join(mitmDir, "ca.key"))
	if err == nil {
		t.Fatal("expected key file error")
	}
}

func TestLoadCABadCertPEM(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "ca.crt")
	keyPath := filepath.Join(dir, "ca.key")
	if err := os.WriteFile(certPath, []byte("not pem"), 0644); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte("not pem"), 0644); err != nil {
		t.Fatalf("write key: %v", err)
	}
	_, err := loadCA(certPath, keyPath)
	if err == nil {
		t.Fatal("expected error for bad cert PEM")
	}
}

func TestLoadCABadCertParse(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "ca.crt")
	keyPath := filepath.Join(dir, "ca.key")
	block := &pem.Block{Type: "CERTIFICATE", Bytes: []byte("bad")}
	if err := os.WriteFile(certPath, pem.EncodeToMemory(block), 0644); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte("not pem"), 0644); err != nil {
		t.Fatalf("write key: %v", err)
	}
	_, err := loadCA(certPath, keyPath)
	if err == nil {
		t.Fatal("expected error for bad cert parse")
	}
}

func TestLoadCABadKeyPEM(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "ca.crt")
	keyPath := filepath.Join(dir, "ca.key")

	// Generate valid CA first so cert parsing succeeds
	_, _ = generateCA(certPath, keyPath)

	// Overwrite key with bad PEM
	if err := os.WriteFile(keyPath, []byte("not pem"), 0644); err != nil {
		t.Fatalf("write key: %v", err)
	}
	_, err := loadCA(certPath, keyPath)
	if err == nil {
		t.Fatal("expected error for bad key PEM")
	}
}

func TestLoadCABadKeyParse(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "ca.crt")
	keyPath := filepath.Join(dir, "ca.key")

	// Generate valid CA first so cert parsing succeeds
	_, _ = generateCA(certPath, keyPath)

	// Overwrite key with valid PEM but bad content
	block := &pem.Block{Type: "EC PRIVATE KEY", Bytes: []byte("bad")}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(block), 0644); err != nil {
		t.Fatalf("write key: %v", err)
	}
	_, err := loadCA(certPath, keyPath)
	if err == nil {
		t.Fatal("expected error for bad key parse")
	}
}

func TestLoadCAReadCertError(t *testing.T) {
	dir := t.TempDir()
	_, _ = LoadOrGenerateCA(dir)
	certPath := filepath.Join(dir, "mitm", "ca.crt")
	os.Remove(certPath)
	_, err := loadCA(certPath, filepath.Join(dir, "mitm", "ca.key"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadCAReadKeyError(t *testing.T) {
	dir := t.TempDir()
	_, _ = LoadOrGenerateCA(dir)
	keyPath := filepath.Join(dir, "mitm", "ca.key")
	os.Remove(keyPath)
	_, err := loadCA(filepath.Join(dir, "mitm", "ca.crt"), keyPath)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIsToolHostNoPort(t *testing.T) {
	if !IsToolHost("api.cursor.sh") {
		t.Fatal("expected api.cursor.sh to be a tool host")
	}
	if IsToolHost("example.com") {
		t.Fatal("expected example.com to not be a tool host")
	}
}

func TestTunnelDialError(t *testing.T) {
	dir := t.TempDir()
	p, _ := NewProxy(dir, 0)

	// Use a closed listener address to get a fast connection refused.
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()

	req := httptest.NewRequest("CONNECT", addr, nil)
	w := httptest.NewRecorder()
	p.tunnel(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestTunnelHijackerNotSupported(t *testing.T) {
	dir := t.TempDir()
	p, _ := NewProxy(dir, 0)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	backendHost := strings.TrimPrefix(backend.URL, "http://")
	req := httptest.NewRequest("CONNECT", backendHost, nil)
	w := httptest.NewRecorder()
	p.tunnel(w, req)
	// ResponseRecorder doesn't support hijacking; code records first WriteHeader (200)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestMITMHijackerNotSupported(t *testing.T) {
	dir := t.TempDir()
	p, _ := NewProxy(dir, 0)
	p.SetTarget("http://localhost:8080")

	req := httptest.NewRequest("CONNECT", "fake-tool.example.com:443", nil)
	w := httptest.NewRecorder()
	p.mitm(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}
