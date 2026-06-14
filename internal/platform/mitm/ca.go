// Package mitm implements the HTTPS man-in-the-middle subsystem: a self-signed
// root CA that mints per-host leaf certificates for TLS interception, a leaf-cert
// cache, an injectable proxy listener, and the enable/disable/per-tool state
// machine. The CA generation, leaf minting, CA signing, chain verification, and
// PEM encoding are PURE crypto (crypto/x509, crypto/tls, crypto/ecdsa) and are
// fully unit-tested with no port binding, no network, and no real TLS handshake.
// The live reverse-proxy listener (proxy.go) is integration-only behind the
// MitmProxy interface — the test suite injects a deterministic fake. The CA
// PRIVATE KEY is SECRET: it is never serialized into any response/DTO/log; only
// the public CertPEM is served (the raw-PEM /api/mitm/ca-cert endpoint body).
package mitm

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"sync"
)

const (
	caKeyFileName  = "mitm-ca.key"
	caCertFileName = "mitm-ca.crt"
)

// CAOpts configures GenerateCA. A zero value yields sensible defaults.
type CAOpts struct {
	CommonName   string
	Organization string
}

// CA holds the self-signed root CA used to mint per-host leaf certificates for
// MITM interception. The private key is SECRET — never serialized into any
// response/DTO/log; only CertPEM() (the public cert) is served.
type CA struct {
	cert    *x509.Certificate
	key     *ecdsa.PrivateKey
	certPEM []byte

	mu    sync.RWMutex
	cache map[string]tls.Certificate
}

var errNotImplemented = errors.New("mitm: not implemented")

// GenerateCA creates a fresh self-signed root CA. PURE — no I/O.
func GenerateCA(opts CAOpts) (*CA, error) { return nil, errNotImplemented }

// LoadOrCreateCA loads the CA from dataDir, or generates+persists one on first use.
func LoadOrCreateCA(dataDir string) (*CA, error) { return nil, errNotImplemented }

// CertPEM returns the PUBLIC root CA cert as a PEM CERTIFICATE block. PURE.
func (c *CA) CertPEM() []byte { return nil }

// MintLeaf mints a CA-signed leaf certificate for the given SNI host. PURE.
func (c *CA) MintLeaf(host string) (tls.Certificate, error) {
	return tls.Certificate{}, errNotImplemented
}

// getLeaf returns the cached leaf for host, minting and caching it on a miss.
func (c *CA) getLeaf(host string) (tls.Certificate, error) {
	return tls.Certificate{}, errNotImplemented
}
