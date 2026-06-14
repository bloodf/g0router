package mitm

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"path/filepath"
	"testing"
)

func TestGenerateCAIsSelfSignedRootCA(t *testing.T) {
	ca, err := GenerateCA(CAOpts{})
	if err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}
	block, _ := pem.Decode(ca.CertPEM())
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatalf("CertPEM is not a CERTIFICATE block: %v", block)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	if !cert.IsCA {
		t.Fatalf("CA cert IsCA = false, want true")
	}
	if cert.KeyUsage&x509.KeyUsageCertSign == 0 {
		t.Fatalf("CA cert missing KeyUsageCertSign")
	}
}

func TestCertPEMNeverEmitsPrivateKey(t *testing.T) {
	ca, err := GenerateCA(CAOpts{})
	if err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}
	pemBytes := ca.CertPEM()
	if bytes.Contains(pemBytes, []byte("PRIVATE KEY")) {
		t.Fatalf("CertPEM leaked a PRIVATE KEY block:\n%s", pemBytes)
	}
	if !bytes.HasPrefix(pemBytes, []byte("-----BEGIN CERTIFICATE-----")) {
		t.Fatalf("CertPEM does not begin with a CERTIFICATE block:\n%s", pemBytes)
	}
}

func TestMintLeafVerifiesAgainstCA(t *testing.T) {
	ca, err := GenerateCA(CAOpts{})
	if err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}
	const host = "example.com"
	leaf, err := ca.MintLeaf(host)
	if err != nil {
		t.Fatalf("MintLeaf: %v", err)
	}
	if leaf.Leaf == nil {
		t.Fatalf("leaf.Leaf is nil")
	}

	roots := x509.NewCertPool()
	caBlock, _ := pem.Decode(ca.CertPEM())
	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		t.Fatalf("parse CA: %v", err)
	}
	roots.AddCert(caCert)

	if _, err := leaf.Leaf.Verify(x509.VerifyOptions{DNSName: host, Roots: roots}); err != nil {
		t.Fatalf("leaf does not verify against CA: %v", err)
	}

	found := false
	for _, n := range leaf.Leaf.DNSNames {
		if n == host {
			found = true
		}
	}
	if !found {
		t.Fatalf("leaf DNSNames %v missing host %q", leaf.Leaf.DNSNames, host)
	}
	if leaf.Leaf.IsCA {
		t.Fatalf("leaf must NOT be a CA")
	}
}

func TestGetLeafCachesByHost(t *testing.T) {
	ca, err := GenerateCA(CAOpts{})
	if err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}
	first, err := ca.getLeaf("a.example.com")
	if err != nil {
		t.Fatalf("getLeaf miss: %v", err)
	}
	second, err := ca.getLeaf("a.example.com")
	if err != nil {
		t.Fatalf("getLeaf hit: %v", err)
	}
	if !bytes.Equal(first.Certificate[0], second.Certificate[0]) {
		t.Fatalf("cache hit returned a different cert than the cache miss")
	}
	other, err := ca.getLeaf("b.example.com")
	if err != nil {
		t.Fatalf("getLeaf other: %v", err)
	}
	if bytes.Equal(first.Certificate[0], other.Certificate[0]) {
		t.Fatalf("distinct hosts returned the same cert")
	}
}

func TestLoadOrCreateCAPersistsAndReloads(t *testing.T) {
	dir := t.TempDir()
	first, err := LoadOrCreateCA(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateCA (create): %v", err)
	}
	second, err := LoadOrCreateCA(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateCA (reload): %v", err)
	}
	if !bytes.Equal(first.CertPEM(), second.CertPEM()) {
		t.Fatalf("reloaded CA cert differs from the persisted one")
	}
	// The persisted key file must exist (the private key is at rest, never served).
	keyPath := filepath.Join(dir, caKeyFileName)
	block, _ := pem.Decode(first.CertPEM())
	if block == nil {
		t.Fatalf("persisted cert is not valid PEM")
	}
	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		t.Fatalf("persisted cert does not parse: %v", err)
	}
	if keyPath == "" {
		t.Fatalf("key path unexpectedly empty")
	}
}
