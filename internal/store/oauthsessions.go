package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// OAuthSession is an in-flight OAuth authorization: the state parameter
// plus the PKCE verifier, which is encrypted at rest (verifier_enc).
type OAuthSession struct {
	State     string
	Provider  string
	Verifier  string
	ExpiresAt int64
}

// CreateOAuthSession stores an in-flight OAuth state with its encrypted verifier.
func (s *Store) CreateOAuthSession(o *OAuthSession) error {
	verifierEnc, err := s.cipher.Encrypt(o.Verifier)
	if err != nil {
		return fmt.Errorf("encrypt verifier: %w", err)
	}
	_, err = s.db.Exec(
		"INSERT INTO oauth_sessions (state, provider, verifier_enc, expires_at, created_at) VALUES (?, ?, ?, ?, ?)",
		o.State, o.Provider, verifierEnc, o.ExpiresAt, time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("insert oauth session: %w", err)
	}
	return nil
}

// ConsumeOAuthSession returns and deletes the OAuth session for state.
// Expired or already-consumed states return ErrNotFound.
func (s *Store) ConsumeOAuthSession(state string) (*OAuthSession, error) {
	var o OAuthSession
	var verifierEnc string
	err := s.db.QueryRow(
		"SELECT state, provider, verifier_enc, expires_at FROM oauth_sessions WHERE state = ?", state,
	).Scan(&o.State, &o.Provider, &verifierEnc, &o.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan oauth session: %w", err)
	}

	if _, err := s.db.Exec("DELETE FROM oauth_sessions WHERE state = ?", state); err != nil {
		return nil, fmt.Errorf("delete oauth session: %w", err)
	}

	if o.ExpiresAt <= time.Now().Unix() {
		return nil, ErrNotFound
	}

	if o.Verifier, err = s.cipher.Decrypt(verifierEnc); err != nil {
		return nil, fmt.Errorf("decrypt verifier: %w", err)
	}
	return &o, nil
}
