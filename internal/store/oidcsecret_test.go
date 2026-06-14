package store

import (
	"testing"
)

func TestOIDCSecretRoundTrip(t *testing.T) {
	st := newTestStore(t)

	// Empty by default.
	got, err := st.GetOIDCSecret()
	if err != nil {
		t.Fatalf("GetOIDCSecret (empty): %v", err)
	}
	if got != "" {
		t.Fatalf("default secret = %q, want empty", got)
	}

	// Set then get round-trips through encrypt/decrypt.
	if err := st.SetOIDCSecret("super-secret-value"); err != nil {
		t.Fatalf("SetOIDCSecret: %v", err)
	}
	got, err = st.GetOIDCSecret()
	if err != nil {
		t.Fatalf("GetOIDCSecret: %v", err)
	}
	if got != "super-secret-value" {
		t.Fatalf("round-trip secret = %q, want %q", got, "super-secret-value")
	}

	// Update overwrites.
	if err := st.SetOIDCSecret("rotated"); err != nil {
		t.Fatalf("SetOIDCSecret (rotate): %v", err)
	}
	got, _ = st.GetOIDCSecret()
	if got != "rotated" {
		t.Fatalf("rotated secret = %q, want %q", got, "rotated")
	}
}

func TestOIDCSecretStoredEncrypted(t *testing.T) {
	st := newTestStore(t)
	if err := st.SetOIDCSecret("plaintext-marker"); err != nil {
		t.Fatalf("SetOIDCSecret: %v", err)
	}

	var raw string
	if err := st.DB().QueryRow("SELECT oidc_secret_enc FROM oidc_secret WHERE id = 1").Scan(&raw); err != nil {
		t.Fatalf("read raw column: %v", err)
	}
	if raw == "" {
		t.Fatalf("raw oidc_secret_enc is empty after set")
	}
	if raw == "plaintext-marker" {
		t.Fatalf("raw oidc_secret_enc is plaintext: %q", raw)
	}
}

func TestOIDCSecretMigratesLegacyPlaintext(t *testing.T) {
	st := newTestStore(t)

	// Simulate a pre-existing plaintext secret in the flat settings table.
	if err := st.SetSettings(map[string]string{"oidc_client_secret": "legacy-plain"}); err != nil {
		t.Fatalf("seed legacy plaintext: %v", err)
	}

	// First read encrypt-moves it.
	got, err := st.GetOIDCSecret()
	if err != nil {
		t.Fatalf("GetOIDCSecret (migrate): %v", err)
	}
	if got != "legacy-plain" {
		t.Fatalf("migrated secret = %q, want %q", got, "legacy-plain")
	}

	// The plaintext settings key is now blanked.
	settings, err := st.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if settings["oidc_client_secret"] != "" {
		t.Fatalf("legacy plaintext not blanked: %q", settings["oidc_client_secret"])
	}

	// The encrypted column is populated and not plaintext.
	var raw string
	if err := st.DB().QueryRow("SELECT oidc_secret_enc FROM oidc_secret WHERE id = 1").Scan(&raw); err != nil {
		t.Fatalf("read raw column: %v", err)
	}
	if raw == "" || raw == "legacy-plain" {
		t.Fatalf("oidc_secret_enc after migration = %q (want non-empty, non-plaintext)", raw)
	}

	// Idempotent: a second read still returns the secret and stays migrated.
	got, err = st.GetOIDCSecret()
	if err != nil {
		t.Fatalf("GetOIDCSecret (second): %v", err)
	}
	if got != "legacy-plain" {
		t.Fatalf("second read secret = %q, want %q", got, "legacy-plain")
	}
}
