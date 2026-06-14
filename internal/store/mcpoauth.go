package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// MCPOAuthAccount is a persisted OAuth account for an MCP server. AccessToken and
// RefreshToken are plaintext in memory and encrypted at rest in the
// access_token_enc / refresh_token_enc columns (mirrors connections.go /
// oauthsessions.go). They are decrypted on read for the OAuth engine's internal
// use ONLY and must never be echoed into a handler response.
type MCPOAuthAccount struct {
	ID           string
	InstanceID   string
	ServerURL    string
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
	Scope        string
	Status       string // 'connected' | 'expired' | 'error' (no secrets)
	CreatedAt    int64
	UpdatedAt    int64
}

// MCPOAuthFlow is an in-flight MCP OAuth authorization. The PKCE Verifier is
// plaintext in memory and encrypted at rest in verifier_enc (mirrors
// oauth_sessions.verifier_enc).
type MCPOAuthFlow struct {
	State       string
	InstanceID  string
	ServerURL   string
	Verifier    string
	RedirectURI string
	ExpiresAt   int64
	CreatedAt   int64
}

// UpsertMCPOAuthAccount inserts a new account (generating an ID) or updates the
// existing account for the same instance_id, encrypting both tokens at rest. It
// returns the stored record (with the generated/existing ID).
func (s *Store) UpsertMCPOAuthAccount(a *MCPOAuthAccount) (*MCPOAuthAccount, error) {
	accessEnc, err := s.cipher.Encrypt(a.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("encrypt access token: %w", err)
	}
	refreshEnc, err := s.cipher.Encrypt(a.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("encrypt refresh token: %w", err)
	}

	id := a.ID
	if id == "" {
		// Reuse the existing account for this instance if present.
		if a.InstanceID != "" {
			var existing string
			err := s.db.QueryRow(
				"SELECT id FROM mcp_oauth_accounts WHERE instance_id = ?", a.InstanceID).Scan(&existing)
			if err == nil {
				id = existing
			} else if !errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("lookup existing account: %w", err)
			}
		}
		if id == "" {
			if id, err = newID(); err != nil {
				return nil, err
			}
		}
	}

	now := time.Now().Unix()
	_, err = s.db.Exec(
		`INSERT INTO mcp_oauth_accounts
		 (id, instance_id, server_url, access_token_enc, refresh_token_enc, expires_at, scope, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET instance_id = excluded.instance_id, server_url = excluded.server_url,
		   access_token_enc = excluded.access_token_enc, refresh_token_enc = excluded.refresh_token_enc,
		   expires_at = excluded.expires_at, scope = excluded.scope, status = excluded.status,
		   updated_at = excluded.updated_at`,
		id, a.InstanceID, a.ServerURL, accessEnc, refreshEnc, a.ExpiresAt, a.Scope, a.Status, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert mcp oauth account: %w", err)
	}
	return s.GetMCPOAuthAccount(id)
}

// GetMCPOAuthAccount returns the account with the given id, tokens decrypted.
func (s *Store) GetMCPOAuthAccount(id string) (*MCPOAuthAccount, error) {
	return s.scanMCPOAuthAccount(s.db.QueryRow(
		`SELECT id, instance_id, server_url, access_token_enc, refresh_token_enc, expires_at, scope, status, created_at, updated_at
		 FROM mcp_oauth_accounts WHERE id = ?`, id))
}

// GetMCPOAuthAccountByInstance returns the account bound to instanceID.
func (s *Store) GetMCPOAuthAccountByInstance(instanceID string) (*MCPOAuthAccount, error) {
	return s.scanMCPOAuthAccount(s.db.QueryRow(
		`SELECT id, instance_id, server_url, access_token_enc, refresh_token_enc, expires_at, scope, status, created_at, updated_at
		 FROM mcp_oauth_accounts WHERE instance_id = ?`, instanceID))
}

// ListMCPOAuthAccounts returns all accounts ordered by creation time.
func (s *Store) ListMCPOAuthAccounts() ([]*MCPOAuthAccount, error) {
	rows, err := s.db.Query(
		`SELECT id, instance_id, server_url, access_token_enc, refresh_token_enc, expires_at, scope, status, created_at, updated_at
		 FROM mcp_oauth_accounts ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("query mcp oauth accounts: %w", err)
	}
	defer rows.Close()

	var out []*MCPOAuthAccount
	for rows.Next() {
		a, err := s.scanMCPOAuthAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mcp oauth accounts: %w", err)
	}
	return out, nil
}

// DeleteMCPOAuthAccount removes the account with the given id.
func (s *Store) DeleteMCPOAuthAccount(id string) error {
	res, err := s.db.Exec("DELETE FROM mcp_oauth_accounts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete mcp oauth account %s: %w", id, err)
	}
	return requireRowAffected(res)
}

// CreateMCPOAuthFlow stores an in-flight OAuth flow with its encrypted verifier.
func (s *Store) CreateMCPOAuthFlow(f *MCPOAuthFlow) error {
	verifierEnc, err := s.cipher.Encrypt(f.Verifier)
	if err != nil {
		return fmt.Errorf("encrypt verifier: %w", err)
	}
	_, err = s.db.Exec(
		`INSERT INTO mcp_oauth_flows (state, instance_id, server_url, verifier_enc, redirect_uri, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		f.State, f.InstanceID, f.ServerURL, verifierEnc, f.RedirectURI, f.ExpiresAt, time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("insert mcp oauth flow: %w", err)
	}
	return nil
}

// ConsumeMCPOAuthFlow returns and deletes the flow for state, decrypting the
// verifier. An expired or already-consumed state returns ErrNotFound. Mirrors
// ConsumeOAuthSession (oauthsessions.go:37).
func (s *Store) ConsumeMCPOAuthFlow(state string) (*MCPOAuthFlow, error) {
	var f MCPOAuthFlow
	var verifierEnc string
	err := s.db.QueryRow(
		`SELECT state, instance_id, server_url, verifier_enc, redirect_uri, expires_at, created_at
		 FROM mcp_oauth_flows WHERE state = ?`, state,
	).Scan(&f.State, &f.InstanceID, &f.ServerURL, &verifierEnc, &f.RedirectURI, &f.ExpiresAt, &f.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan mcp oauth flow: %w", err)
	}

	if _, err := s.db.Exec("DELETE FROM mcp_oauth_flows WHERE state = ?", state); err != nil {
		return nil, fmt.Errorf("delete mcp oauth flow: %w", err)
	}

	if f.ExpiresAt <= time.Now().Unix() {
		return nil, ErrNotFound
	}

	if f.Verifier, err = s.cipher.Decrypt(verifierEnc); err != nil {
		return nil, fmt.Errorf("decrypt verifier: %w", err)
	}
	return &f, nil
}

func (s *Store) scanMCPOAuthAccount(row rowScanner) (*MCPOAuthAccount, error) {
	var a MCPOAuthAccount
	var accessEnc, refreshEnc string
	err := row.Scan(&a.ID, &a.InstanceID, &a.ServerURL, &accessEnc, &refreshEnc,
		&a.ExpiresAt, &a.Scope, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan mcp oauth account: %w", err)
	}
	if a.AccessToken, err = s.cipher.Decrypt(accessEnc); err != nil {
		return nil, fmt.Errorf("decrypt access token: %w", err)
	}
	if a.RefreshToken, err = s.cipher.Decrypt(refreshEnc); err != nil {
		return nil, fmt.Errorf("decrypt refresh token: %w", err)
	}
	return &a, nil
}
