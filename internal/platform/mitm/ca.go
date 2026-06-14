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
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	caKeyFileName  = "mitm-ca.key"
	caCertFileName = "mitm-ca.crt"

	caValidity   = 10 * 365 * 24 * time.Hour // 10 years for the root CA
	leafValidity = 365 * 24 * time.Hour      // 1 year for minted leaves
)

// CAOpts configures GenerateCA. A zero value yields sensible defaults.
type CAOpts struct {
	// CommonName is the root CA subject common name. Defaults to "g0router MITM CA".
	CommonName string
	// Organization is the root CA subject organization. Defaults to "g0router".
	Organization string
}

// CA holds the self-signed root CA used to mint per-host leaf certificates for
// MITM interception. The private key is SECRET — never serialized into any
// response/DTO/log; only CertPEM() (the public cert) is served. Leaf certs are
// cached by SNI host under a mutex.
type CA struct {
	cert    *x509.Certificate
	key     *ecdsa.PrivateKey
	certPEM []byte

	mu    sync.RWMutex
	cache map[string]tls.Certificate
}

// GenerateCA creates a fresh self-signed root CA (IsCA=true, KeyUsageCertSign).
// It is PURE — no I/O. The key type is ECDSA P-256 (ESC-KEYTYPE default).
func GenerateCA(opts CAOpts) (*CA, error) {
	cn := opts.CommonName
	if cn == "" {
		cn = "g0router MITM CA"
	}
	org := opts.Organization
	if org == "" {
		org = "g0router"
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate CA key: %w", err)
	}

	serial, err := randomSerial()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   cn,
			Organization: []string{org},
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(caValidity),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("create CA certificate: %w", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("parse CA certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	return &CA{
		cert:    cert,
		key:     key,
		certPEM: certPEM,
		cache:   make(map[string]tls.Certificate),
	}, nil
}

// LoadOrCreateCA loads the root CA from dataDir/mitm-ca.{key,crt}, or generates
// and persists one on first use, mirroring store.LoadOrCreateSecret: the key is
// written 0o600 and the cert 0o644 under a 0o700 data dir. The FILE I/O here is
// the only I/O in the CA core; GenerateCA is the pure unit-tested generator.
func LoadOrCreateCA(dataDir string) (*CA, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir %s: %w", dataDir, err)
	}
	keyPath := filepath.Join(dataDir, caKeyFileName)
	certPath := filepath.Join(dataDir, caCertFileName)

	keyBytes, keyErr := os.ReadFile(keyPath)
	certBytes, certErr := os.ReadFile(certPath)
	if keyErr == nil && certErr == nil {
		return loadCA(keyBytes, certBytes)
	}
	if keyErr != nil && !os.IsNotExist(keyErr) {
		return nil, fmt.Errorf("read CA key %s: %w", keyPath, keyErr)
	}
	if certErr != nil && !os.IsNotExist(certErr) {
		return nil, fmt.Errorf("read CA cert %s: %w", certPath, certErr)
	}

	ca, err := GenerateCA(CAOpts{})
	if err != nil {
		return nil, err
	}
	keyPEM, err := ca.keyPEM()
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return nil, fmt.Errorf("write CA key %s: %w", keyPath, err)
	}
	if err := os.WriteFile(certPath, ca.certPEM, 0o644); err != nil {
		return nil, fmt.Errorf("write CA cert %s: %w", certPath, err)
	}
	return ca, nil
}

// loadCA reconstructs a CA from PEM-encoded key + cert bytes read from disk.
func loadCA(keyPEM, certPEM []byte) (*CA, error) {
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, errors.New("decode CA key PEM")
	}
	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CA key: %w", err)
	}
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, errors.New("decode CA cert PEM")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CA cert: %w", err)
	}
	return &CA{
		cert:    cert,
		key:     key,
		certPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBlock.Bytes}),
		cache:   make(map[string]tls.Certificate),
	}, nil
}

// keyPEM serializes the CA private key as a PEM EC PRIVATE KEY block. It is used
// ONLY by the persist-key path (LoadOrCreateCA write); it is NEVER served, logged,
// or placed in any DTO.
func (c *CA) keyPEM() ([]byte, error) {
	der, err := x509.MarshalECPrivateKey(c.key)
	if err != nil {
		return nil, fmt.Errorf("marshal CA key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}), nil
}

// CertPEM returns the PUBLIC root CA cert as a PEM CERTIFICATE block (the
// application/x-pem-file body served by GET /api/mitm/ca-cert). It is PURE and
// NEVER returns any key material.
func (c *CA) CertPEM() []byte {
	out := make([]byte, len(c.certPEM))
	copy(out, c.certPEM)
	return out
}

// MintLeaf mints a leaf certificate for the given SNI host, signed by the root
// CA. It is PURE crypto — no I/O, no listener. The leaf verifies against the CA
// (x509.Verify with the CA in the roots pool) and carries host in its DNSNames.
func (c *CA) MintLeaf(host string) (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate leaf key: %w", err)
	}
	serial, err := randomSerial()
	if err != nil {
		return tls.Certificate{}, err
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: host},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(leafValidity),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{host},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, c.cert, &key.PublicKey, c.key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create leaf certificate: %w", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("parse leaf certificate: %w", err)
	}
	return tls.Certificate{
		Certificate: [][]byte{der, c.cert.Raw},
		PrivateKey:  key,
		Leaf:        leaf,
	}, nil
}

// getLeaf returns the cached leaf for host, minting and caching it on a miss.
// Concurrent-safe. Used by the proxy's GetCertificate closure.
func (c *CA) getLeaf(host string) (tls.Certificate, error) {
	c.mu.RLock()
	cert, ok := c.cache[host]
	c.mu.RUnlock()
	if ok {
		return cert, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if cert, ok := c.cache[host]; ok {
		return cert, nil
	}
	cert, err := c.MintLeaf(host)
	if err != nil {
		return tls.Certificate{}, err
	}
	c.cache[host] = cert
	return cert, nil
}

func randomSerial() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return nil, fmt.Errorf("generate serial: %w", err)
	}
	return serial, nil
}
