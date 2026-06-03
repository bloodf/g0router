package store

import (
	"database/sql"
	"fmt"
	"time"
)

type OAuthSession struct {
	ID           string
	State        string
	Provider     string
	CodeVerifier string
	RedirectURI  string
	AccountLabel string
	ExpiresAt    time.Time
	CreatedAt    string
}

func (s *Store) CreateOAuthSession(session *OAuthSession) error {
	row := s.db.QueryRow(
		`INSERT INTO oauth_sessions (
			provider, state_hash, code_verifier, redirect_uri, account_label, expires_at
		) VALUES (?, ?, ?, ?, ?, ?)
		RETURNING id, created_at`,
		session.Provider,
		hashOAuthState(session.State),
		emptyStringNil(session.CodeVerifier),
		emptyStringNil(session.RedirectURI),
		emptyStringNil(session.AccountLabel),
		session.ExpiresAt.Format(time.RFC3339),
	)
	if err := row.Scan(&session.ID, &session.CreatedAt); err != nil {
		return fmt.Errorf("insert oauth session: %w", err)
	}
	return nil
}

func (s *Store) ConsumeOAuthSession(state string) (*OAuthSession, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin consume oauth session: %w", err)
	}
	defer tx.Rollback()

	session, err := scanOAuthSession(tx.QueryRow(oauthSessionSelectSQL()+" WHERE state_hash = ?", hashOAuthState(state)))
	if err != nil {
		return nil, err
	}
	if !session.ExpiresAt.IsZero() && time.Now().After(session.ExpiresAt) {
		_, _ = tx.Exec("DELETE FROM oauth_sessions WHERE id = ?", session.ID)
		return nil, ErrNotFound
	}
	if _, err := tx.Exec("DELETE FROM oauth_sessions WHERE id = ?", session.ID); err != nil {
		return nil, fmt.Errorf("delete oauth session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit consume oauth session: %w", err)
	}
	session.State = state
	return session, nil
}

func scanOAuthSession(scanner connectionScanner) (*OAuthSession, error) {
	var session OAuthSession
	var codeVerifier, redirectURI, accountLabel sql.NullString
	var expiresAt string

	err := scanner.Scan(
		&session.ID,
		&session.Provider,
		&codeVerifier,
		&redirectURI,
		&accountLabel,
		&expiresAt,
		&session.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan oauth session: %w", err)
	}

	session.CodeVerifier = stringValueFromNull(codeVerifier)
	session.RedirectURI = stringValueFromNull(redirectURI)
	session.AccountLabel = stringValueFromNull(accountLabel)
	if expiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, expiresAt)
		if err != nil {
			return nil, fmt.Errorf("parse oauth session expiry: %w", err)
		}
		session.ExpiresAt = parsed
	}
	return &session, nil
}

func oauthSessionSelectSQL() string {
	return `SELECT id, provider, code_verifier, redirect_uri, account_label, expires_at, created_at FROM oauth_sessions`
}

func emptyStringNil(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func stringValueFromNull(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
