package mitm

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"sync"
)

// ToolHosts is the list of known AI-tool hostnames that the proxy intercepts.
var ToolHosts = []string{
	"api.githubcopilot.com",
	"api.cursor.sh",
	"api.kiro.dev",
	"antigravity.ai",
}

// IsToolHost reports whether host (with or without port) is a known tool host.
func IsToolHost(host string) bool {
	h, _, err := net.SplitHostPort(host)
	if err != nil {
		h = host
	}
	for _, th := range ToolHosts {
		if h == th {
			return true
		}
	}
	return false
}

// Proxy is an HTTPS CONNECT proxy with optional MITM for tool hosts.
type Proxy struct {
	mu          sync.RWMutex
	ca          *CA
	port        int
	target      string
	listener    net.Listener
	server      *http.Server
	running     bool
	toolEnabled map[string]bool
}

// NewProxy creates a proxy that loads or generates a CA in dataDir.
func NewProxy(dataDir string, port int) (*Proxy, error) {
	ca, err := LoadOrGenerateCA(dataDir)
	if err != nil {
		return nil, err
	}
	return &Proxy{
		ca:          ca,
		port:        port,
		target:      "http://localhost:8080",
		toolEnabled: make(map[string]bool),
	}, nil
}

// SetTarget sets the local inference endpoint that tool requests are forwarded to.
func (p *Proxy) SetTarget(target string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.target = target
}

// Target returns the current target URL.
func (p *Proxy) Target() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.target
}

// Addr returns the listener address (empty if not started).
func (p *Proxy) Addr() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.listener == nil {
		return ""
	}
	return p.listener.Addr().String()
}

// CAPool returns a cert pool containing the CA certificate.
func (p *Proxy) CAPool() *x509.CertPool {
	pool := x509.NewCertPool()
	pool.AddCert(p.ca.Cert)
	return pool
}

// CACertPEM returns the CA certificate in PEM format.
func (p *Proxy) CACertPEM() []byte {
	return p.ca.CertPEM()
}

// ToolEnabled reports whether a specific tool is enabled for interception.
func (p *Proxy) ToolEnabled(name string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.toolEnabled[name]
}

// SetToolEnabled enables or disables interception for a specific tool.
func (p *Proxy) SetToolEnabled(name string, enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.toolEnabled[name] = enabled
}

// Start begins listening for CONNECT requests.
func (p *Proxy) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.running {
		return nil
	}
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(p.port))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	p.listener = ln
	p.running = true
	p.server = &http.Server{Handler: http.HandlerFunc(p.handleConnect)}
	go p.server.Serve(ln)
	return nil
}

// Stop closes the listener and stops accepting new connections.
func (p *Proxy) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.running {
		return nil
	}
	err := p.server.Close()
	p.running = false
	return err
}

// IsRunning reports whether the proxy is currently listening.
func (p *Proxy) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

func (p *Proxy) handleConnect(w http.ResponseWriter, r *http.Request) {
	if !IsToolHost(r.Host) {
		p.tunnel(w, r)
		return
	}
	p.mitm(w, r)
}

func (p *Proxy) tunnel(w http.ResponseWriter, r *http.Request) {
	destConn, err := net.Dial("tcp", r.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer destConn.Close()

	w.WriteHeader(http.StatusOK)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		return
	}
	defer clientConn.Close()

	go func() { _, _ = io.Copy(destConn, clientConn) }()
	_, _ = io.Copy(clientConn, destConn)
}

func (p *Proxy) mitm(w http.ResponseWriter, r *http.Request) {
	cert, err := MintLeafCert(p.ca, r.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{*cert},
	}

	w.WriteHeader(http.StatusOK)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}
	conn, _, err := hijacker.Hijack()
	if err != nil {
		return
	}

	tlsConn := tls.Server(conn, config)
	if err := tlsConn.Handshake(); err != nil {
		tlsConn.Close()
		return
	}

	targetURL, err := url.Parse(p.Target())
	if err != nil {
		tlsConn.Close()
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	oldDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		oldDirector(req)
		req.Host = targetURL.Host
	}

	ln := &oneShotListener{conn: tlsConn}
	srv := &http.Server{Handler: proxy}
	srv.Serve(ln)
}

// oneShotListener yields a single connection and then returns io.EOF so that
// http.Server.Serve exits cleanly when the connection is done.
type oneShotListener struct {
	conn net.Conn
	once sync.Once
}

func (l *oneShotListener) Accept() (net.Conn, error) {
	var c net.Conn
	l.once.Do(func() { c = l.conn })
	if c != nil {
		return c, nil
	}
	return nil, io.EOF
}

func (l *oneShotListener) Close() error   { return nil }
func (l *oneShotListener) Addr() net.Addr { return l.conn.LocalAddr() }

// ToolInstructions returns per-tool proxy setup instructions.
func ToolInstructions(proxyAddr string) []map[string]interface{} {
	var out []map[string]interface{}
	for _, name := range []string{"Copilot", "Cursor", "Kiro", "Antigravity"} {
		out = append(out, map[string]interface{}{
			"name":        name,
			"proxy_env":   fmt.Sprintf("HTTPS_PROXY=http://%s", proxyAddr),
			"hosts_line":  fmt.Sprintf("# Add to /etc/hosts for %s interception", name),
		})
	}
	return out
}
