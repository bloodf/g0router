package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestGeneratePKCE(t *testing.T) {
	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE: %v", err)
	}
	if verifier == "" || challenge == "" {
		t.Fatalf("empty verifier/challenge: %q %q", verifier, challenge)
	}
	// The challenge must be the S256 of the verifier (RFC 7636), matching the
	// in-tree pkceChallenge primitive this helper wraps.
	sum := sha256.Sum256([]byte(verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if challenge != want {
		t.Fatalf("challenge = %q, want %q", challenge, want)
	}
}

func TestGeneratePKCEUnique(t *testing.T) {
	v1, _, _ := GeneratePKCE()
	v2, _, _ := GeneratePKCE()
	if v1 == v2 {
		t.Fatalf("verifiers should differ between calls")
	}
}
