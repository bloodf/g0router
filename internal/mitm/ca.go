package mitm

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const caKeyPerm = 0600

// CA holds a generated or loaded certificate authority.
type CA struct {
	Cert *x509.Certificate
	Key  *ecdsa.PrivateKey
}

// CertPEM returns the CA certificate encoded as PEM.
func (ca *CA) CertPEM() []byte {
	block := &pem.Block{Type: "CERTIFICATE", Bytes: ca.Cert.Raw}
	return pem.EncodeToMemory(block)
}

var (
	leafCache   = make(map[string]*tls.Certificate)
	leafCacheMu sync.RWMutex
	leafSerial  = big.NewInt(1000)
	leafSerialMu sync.Mutex
)

// LoadOrGenerateCA loads an existing CA from dataDir/mitm/ or generates a new
// ECDSA P-256 CA with a 10-year lifetime.
func LoadOrGenerateCA(dataDir string) (*CA, error) {
	certPath := filepath.Join(dataDir, "mitm", "ca.crt")
	keyPath := filepath.Join(dataDir, "mitm", "ca.key")

	if _, err := os.Stat(certPath); err == nil {
		return loadCA(certPath, keyPath)
	}
	return generateCA(certPath, keyPath)
}

func generateCA(certPath, keyPath string) (*CA, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate CA key: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"g0router MITM CA"},
		},
		NotBefore:             time.Now().UTC(),
		NotAfter:              time.Now().UTC().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("create CA cert: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("parse CA cert: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(certPath), 0750); err != nil {
		return nil, fmt.Errorf("create CA dir: %w", err)
	}

	certFile, err := os.OpenFile(certPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("create CA cert file: %w", err)
	}
	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		certFile.Close()
		return nil, fmt.Errorf("write CA cert: %w", err)
	}
	certFile.Close()

	keyFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, caKeyPerm)
	if err != nil {
		return nil, fmt.Errorf("create CA key file: %w", err)
	}
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		keyFile.Close()
		return nil, fmt.Errorf("marshal CA key: %w", err)
	}
	if err := pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		keyFile.Close()
		return nil, fmt.Errorf("write CA key: %w", err)
	}
	keyFile.Close()

	return &CA{Cert: cert, Key: key}, nil
}

func loadCA(certPath, keyPath string) (*CA, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("decode CA cert PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CA cert: %w", err)
	}

	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read CA key: %w", err)
	}
	block, _ = pem.Decode(keyPEM)
	if block == nil {
		return nil, fmt.Errorf("decode CA key PEM")
	}
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CA key: %w", err)
	}

	return &CA{Cert: cert, Key: key}, nil
}

// MintLeafCert creates a per-host TLS certificate signed by the CA. Results are
// cached in memory so repeated mints for the same host return the same cert.
func MintLeafCert(ca *CA, host string) (*tls.Certificate, error) {
	// Strip port if present; DNSNames must not contain ports.
	hostOnly, _, err := net.SplitHostPort(host)
	if err != nil {
		hostOnly = host
	}

	leafCacheMu.RLock()
	cached, ok := leafCache[hostOnly]
	leafCacheMu.RUnlock()
	if ok {
		return cached, nil
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate leaf key: %w", err)
	}

	leafSerialMu.Lock()
	leafSerial.Add(leafSerial, big.NewInt(1))
	sn := new(big.Int).Set(leafSerial)
	leafSerialMu.Unlock()

	template := &x509.Certificate{
		SerialNumber: sn,
		Subject: pkix.Name{
			Organization: []string{"g0router MITM"},
		},
		DNSNames:    []string{hostOnly},
		NotBefore:   time.Now().UTC().Add(-1 * time.Hour),
		NotAfter:    time.Now().UTC().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.Cert, &key.PublicKey, ca.Key)
	if err != nil {
		return nil, fmt.Errorf("create leaf cert: %w", err)
	}

	cert := &tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}

	leafCacheMu.Lock()
	leafCache[hostOnly] = cert
	leafCacheMu.Unlock()

	return cert, nil
}
