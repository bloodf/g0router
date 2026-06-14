package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// oidcSecretSettingsKey is the legacy flat-settings key that held the OIDC
// client secret in plaintext before secret-at-rest. GetOIDCSecret migrates any
// value found here into the encrypted oidc_secret table, then blanks it.
const oidcSecretSettingsKey = "oidc_client_secret"

// GetOIDCSecret returns the decrypted OIDC client secret, or the empty string
// when unset. On first read it migrates any legacy plaintext value from the
// settings table into the encrypted oidc_secret row (encrypt-move-blank,
// idempotent).
func (s *Store) GetOIDCSecret() (string, error) {
	var enc string
	err := s.db.QueryRow("SELECT oidc_secret_enc FROM oidc_secret WHERE id = 1").Scan(&enc)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("read oidc secret: %w", err)
	}

	if enc != "" {
		secret, err := s.cipher.Decrypt(enc)
		if err != nil {
			return "", fmt.Errorf("decrypt oidc secret: %w", err)
		}
		return secret, nil
	}

	// No encrypted value yet — migrate the legacy plaintext if present.
	var legacy string
	err = s.db.QueryRow("SELECT value FROM settings WHERE key = ?", oidcSecretSettingsKey).Scan(&legacy)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read legacy oidc secret: %w", err)
	}
	if legacy == "" {
		return "", nil
	}

	if err := s.SetOIDCSecret(legacy); err != nil {
		return "", fmt.Errorf("migrate legacy oidc secret: %w", err)
	}
	if err := s.SetSettings(map[string]string{oidcSecretSettingsKey: ""}); err != nil {
		return "", fmt.Errorf("blank legacy oidc secret: %w", err)
	}
	return legacy, nil
}

// SetOIDCSecret encrypts and upserts the OIDC client secret into the singleton
// oidc_secret row.
func (s *Store) SetOIDCSecret(secret string) error {
	enc, err := s.cipher.Encrypt(secret)
	if err != nil {
		return fmt.Errorf("encrypt oidc secret: %w", err)
	}
	_, err = s.db.Exec(
		`INSERT INTO oidc_secret (id, oidc_secret_enc) VALUES (1, ?)
		 ON CONFLICT(id) DO UPDATE SET oidc_secret_enc = excluded.oidc_secret_enc`,
		enc,
	)
	if err != nil {
		return fmt.Errorf("upsert oidc secret: %w", err)
	}
	return nil
}
