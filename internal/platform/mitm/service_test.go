package mitm

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"path/filepath"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

// fakeProxy is a deterministic MitmProxy that records Start/Stop WITHOUT binding
// any port or performing a real TLS handshake.
type fakeProxy struct {
	started  bool
	stopped  bool
	startErr error
}

func (f *fakeProxy) Start(addr string) error {
	if f.startErr != nil {
		return f.startErr
	}
	f.started = true
	f.stopped = false
	return nil
}

func (f *fakeProxy) Stop() error {
	f.stopped = true
	f.started = false
	return nil
}

func (f *fakeProxy) Running() bool { return f.started }

func newMitmServiceStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	secret, err := store.LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	st, err := store.Open(filepath.Join(dir, "test.db"), secret)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.EnsureMitmTools(); err != nil {
		t.Fatalf("EnsureMitmTools: %v", err)
	}
	return st
}

func TestServiceStatusReturnsFlagAndTools(t *testing.T) {
	st := newMitmServiceStore(t)
	svc := NewService(st)
	svc.SetProxy(&fakeProxy{})

	enabled, tools, err := svc.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if enabled {
		t.Fatalf("default enabled = true, want false")
	}
	if len(tools) != 2 {
		t.Fatalf("Status tools len = %d, want 2", len(tools))
	}
}

func TestServiceToggleFlipsAndPersists(t *testing.T) {
	st := newMitmServiceStore(t)
	svc := NewService(st)
	fp := &fakeProxy{}
	svc.SetProxy(fp)

	enabled, err := svc.Toggle()
	if err != nil {
		t.Fatalf("Toggle: %v", err)
	}
	if !enabled {
		t.Fatalf("first Toggle = false, want true")
	}
	if got, _ := st.GetMitmEnabled(); !got {
		t.Fatalf("global flag not persisted after enable")
	}
	if !fp.started {
		t.Fatalf("proxy not started on enable")
	}

	enabled, err = svc.Toggle()
	if err != nil {
		t.Fatalf("Toggle (disable): %v", err)
	}
	if enabled {
		t.Fatalf("second Toggle = true, want false")
	}
	if got, _ := st.GetMitmEnabled(); got {
		t.Fatalf("global flag not persisted after disable")
	}
	if !fp.stopped {
		t.Fatalf("proxy not stopped on disable")
	}
}

func TestServiceToggleToolDerivesStatus(t *testing.T) {
	st := newMitmServiceStore(t)
	svc := NewService(st)
	svc.SetProxy(&fakeProxy{})

	// mitm-2 starts disabled; toggling enables it (status active).
	tool, err := svc.ToggleTool("mitm-2")
	if err != nil {
		t.Fatalf("ToggleTool: %v", err)
	}
	if !tool.Enabled || tool.Status != "active" {
		t.Fatalf("toggled tool = %+v, want enabled+active", tool)
	}
}

func TestServiceCACertPEMIsPublicCertOnly(t *testing.T) {
	st := newMitmServiceStore(t)
	svc := NewService(st)
	svc.SetProxy(&fakeProxy{})

	pemBytes, err := svc.CACertPEM()
	if err != nil {
		t.Fatalf("CACertPEM: %v", err)
	}
	if bytes.Contains(pemBytes, []byte("PRIVATE KEY")) {
		t.Fatalf("CACertPEM leaked a PRIVATE KEY block")
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatalf("CACertPEM is not a CERTIFICATE block")
	}
	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		t.Fatalf("CACertPEM does not parse: %v", err)
	}
}

func TestGetCertificateClosureVerifiesAgainstCA(t *testing.T) {
	ca, err := GenerateCA(CAOpts{})
	if err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}
	proxy := newListenerProxy(ca)

	const host = "intercept.example.com"
	cert, err := proxy.certForClientHello(&tls.ClientHelloInfo{ServerName: host})
	if err != nil {
		t.Fatalf("certForClientHello: %v", err)
	}
	if cert == nil || cert.Leaf == nil {
		t.Fatalf("nil leaf certificate")
	}

	roots := x509.NewCertPool()
	roots.AddCert(ca.cert)
	if _, err := cert.Leaf.Verify(x509.VerifyOptions{DNSName: host, Roots: roots}); err != nil {
		t.Fatalf("intercept leaf does not verify against CA: %v", err)
	}
}

func TestNextBackoffDoublesAndCaps(t *testing.T) {
	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
		{5, 30 * time.Second}, // 32s capped at 30s
		{10, 30 * time.Second},
	}
	for _, c := range cases {
		if got := nextBackoff(c.attempt); got != c.want {
			t.Fatalf("nextBackoff(%d) = %s, want %s", c.attempt, got, c.want)
		}
	}
}
