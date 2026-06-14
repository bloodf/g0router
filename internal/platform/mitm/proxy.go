package mitm

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"sync"
)

// MitmProxy abstracts the lifecycle of the live MITM TLS reverse-proxy listener.
// The REAL impl (listenerProxy) binds a port and performs real TLS interception
// and is INTEGRATION-ONLY (§1.9) — never exercised by a unit test. Tests inject a
// deterministic fake so the enable/disable state machine + admin API run WITHOUT
// binding a port or performing a real TLS handshake. Mirrors the tunnel.Runner
// injection seam.
type MitmProxy interface {
	// Start binds the MITM listener on addr and begins intercepting. INTEGRATION-ONLY.
	Start(addr string) error
	// Stop closes the listener. Idempotent.
	Stop() error
	// Running reports whether the listener is currently bound.
	Running() bool
}

// listenerProxy is the REAL MitmProxy: a TLS listener whose GetCertificate
// closure mints/returns the per-SNI leaf via the CA + leaf cache. The
// GetCertificate logic is factored into certForClientHello so it is unit-testable
// without a bind; Start/Stop (the actual net.Listen + tls.NewListener + the
// intercept-and-forward loop) are INTEGRATION-ONLY (§1.9).
type listenerProxy struct {
	ca *CA

	mu       sync.Mutex
	listener net.Listener
	running  bool
}

// newListenerProxy constructs the real MITM proxy over a CA. It does NOT bind any
// port (mirrors tunnel.NewService constructing runners without spawning).
func newListenerProxy(ca *CA) *listenerProxy {
	return &listenerProxy{ca: ca}
}

// certForClientHello is the GetCertificate closure body: given a ClientHelloInfo
// it returns the leaf certificate for the requested SNI host (minted+cached). It
// is PURE — no port bind, no handshake — so it is unit-testable with a synthetic
// *tls.ClientHelloInfo.
func (p *listenerProxy) certForClientHello(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	host := hello.ServerName
	if host == "" {
		host = "localhost"
	}
	leaf, err := p.ca.getLeaf(host)
	if err != nil {
		return nil, err
	}
	return &leaf, nil
}

// tlsConfig builds the interception tls.Config wired to certForClientHello.
func (p *listenerProxy) tlsConfig() *tls.Config {
	return &tls.Config{GetCertificate: p.certForClientHello}
}

// Start binds the MITM listener and serves intercepted connections.
// INTEGRATION-ONLY (§1.9): it performs a real net.Listen + tls.NewListener and is
// never invoked by a unit test.
func (p *listenerProxy) Start(addr string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.running {
		return nil
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	tlsLn := tls.NewListener(ln, p.tlsConfig())
	p.listener = tlsLn
	p.running = true
	go p.serve(tlsLn)
	return nil
}

// serve accepts intercepted TLS connections and forwards them. INTEGRATION-ONLY.
func (p *listenerProxy) serve(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go handleIntercepted(conn)
	}
}

// handleIntercepted is the per-connection intercept-and-forward body.
// INTEGRATION-ONLY: it operates on a live TLS-terminated connection.
func handleIntercepted(conn net.Conn) {
	defer conn.Close()
	// Forwarding is part of the live listener path (integration-only). Draining
	// keeps the connection well-behaved until the forward target is wired in the
	// desktop/agent escalation (ESC-OS-PRIV).
	_, _ = io.Copy(io.Discard, conn)
}

// Stop closes the listener. Idempotent. INTEGRATION-ONLY.
func (p *listenerProxy) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.running {
		return nil
	}
	p.running = false
	if p.listener != nil {
		err := p.listener.Close()
		p.listener = nil
		if err != nil && !errors.Is(err, net.ErrClosed) {
			return err
		}
	}
	return nil
}

// Running reports whether the listener is currently bound.
func (p *listenerProxy) Running() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}
