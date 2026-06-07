package mitm

import (
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateCAPersists(t *testing.T) {
	dir := t.TempDir()
	ca, err := LoadOrGenerateCA(dir)
	if err != nil {
		t.Fatalf("LoadOrGenerateCA: %v", err)
	}
	if ca.Cert == nil {
		t.Fatal("CA cert is nil")
	}
	if ca.Key == nil {
		t.Fatal("CA key is nil")
	}

	certPath := filepath.Join(dir, "mitm", "ca.crt")
	keyPath := filepath.Join(dir, "mitm", "ca.key")

	if _, err := os.Stat(certPath); err != nil {
		t.Fatalf("CA cert not persisted: %v", err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("CA key not persisted: %v", err)
	}

	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("CA key mode = %04o, want 0600", info.Mode().Perm())
	}
}

func TestLoadExistingCA(t *testing.T) {
	dir := t.TempDir()
	ca1, err := LoadOrGenerateCA(dir)
	if err != nil {
		t.Fatalf("first LoadOrGenerateCA: %v", err)
	}

	ca2, err := LoadOrGenerateCA(dir)
	if err != nil {
		t.Fatalf("second LoadOrGenerateCA: %v", err)
	}

	if ca1.Cert.SerialNumber.Cmp(ca2.Cert.SerialNumber) != 0 {
		t.Fatal("loaded CA has different serial number")
	}
}

func TestMintLeafCertValidForHost(t *testing.T) {
	dir := t.TempDir()
	ca, err := LoadOrGenerateCA(dir)
	if err != nil {
		t.Fatalf("LoadOrGenerateCA: %v", err)
	}

	leaf, err := MintLeafCert(ca, "example.com")
	if err != nil {
		t.Fatalf("MintLeafCert: %v", err)
	}
	if len(leaf.Certificate) == 0 {
		t.Fatal("leaf cert is empty")
	}

	cert, err := x509.ParseCertificate(leaf.Certificate[0])
	if err != nil {
		t.Fatalf("parse leaf cert: %v", err)
	}

	if err := cert.VerifyHostname("example.com"); err != nil {
		t.Fatalf("verify hostname: %v", err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(ca.Cert)
	opts := x509.VerifyOptions{Roots: pool}
	if _, err := cert.Verify(opts); err != nil {
		t.Fatalf("verify leaf cert against CA: %v", err)
	}
}

func TestMintLeafCertCached(t *testing.T) {
	dir := t.TempDir()
	ca, err := LoadOrGenerateCA(dir)
	if err != nil {
		t.Fatalf("LoadOrGenerateCA: %v", err)
	}

	leaf1, err := MintLeafCert(ca, "cached.example.com")
	if err != nil {
		t.Fatalf("MintLeafCert first: %v", err)
	}
	leaf2, err := MintLeafCert(ca, "cached.example.com")
	if err != nil {
		t.Fatalf("MintLeafCert second: %v", err)
	}

	c1, _ := x509.ParseCertificate(leaf1.Certificate[0])
	c2, _ := x509.ParseCertificate(leaf2.Certificate[0])
	if c1.SerialNumber.Cmp(c2.SerialNumber) != 0 {
		t.Fatal("cached leaf cert has different serial")
	}
}

func TestMintLeafCertExpiresBeforeCA(t *testing.T) {
	dir := t.TempDir()
	ca, err := LoadOrGenerateCA(dir)
	if err != nil {
		t.Fatalf("LoadOrGenerateCA: %v", err)
	}

	leaf, err := MintLeafCert(ca, "short.example.com")
	if err != nil {
		t.Fatalf("MintLeafCert: %v", err)
	}

	cert, _ := x509.ParseCertificate(leaf.Certificate[0])
	if cert.NotAfter.After(ca.Cert.NotAfter) {
		t.Fatal("leaf cert expires after CA")
	}
	if time.Until(cert.NotAfter) > 366*24*time.Hour {
		t.Fatal("leaf cert lifetime > 1 year")
	}
}
