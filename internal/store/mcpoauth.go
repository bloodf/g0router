package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bloodf/g0router/internal/mcp"
)

type MCPOAuthFlow struct {
	ID                 string
	InstanceID         string
	State              string
	CodeVerifierSecret string
	RedirectURI        string
	AuthorizationURL   string
	ResourceURI        string
	ExpiresAt          time.Time
	CreatedAt          string
}

type MCPOAuthAccount struct {
	ID           string
	InstanceID   string
	AccountLabel string
	Subject      string
	Email        string
	Issuer       string
	ResourceURI  string
	Scopes       []string
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	AuthMetadata map[string]string
	CreatedAt    string
	UpdatedAt    string
}

func (s *Store) CreateMCPOAuthFlow(flow *MCPOAuthFlow) error {
	row := s.db.QueryRow(
		`INSERT INTO mcp_oauth_flows (
			instance_id, state_hash, code_verifier_secret, redirect_uri,
			authorization_url, resource_uri, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id, created_at`,
		flow.InstanceID,
		hashOAuthState(flow.State),
		flow.CodeVerifierSecret,
		flow.RedirectURI,
		flow.AuthorizationURL,
		flow.ResourceURI,
		flow.ExpiresAt.Format(time.RFC3339),
	)
	if err := row.Scan(&flow.ID, &flow.CreatedAt); err != nil {
		return fmt.Errorf("insert mcp oauth flow: %w", err)
	}
	return nil
}

func (s *Store) ConsumeMCPOAuthFlow(instanceID, state string) (*MCPOAuthFlow, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin consume mcp oauth flow: %w", err)
	}
	defer tx.Rollback()

	flow, err := scanMCPOAuthFlow(tx.QueryRow(mcpOAuthFlowSelectSQL()+" WHERE instance_id = ? AND state_hash = ?", instanceID, hashOAuthState(state)))
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec("DELETE FROM mcp_oauth_flows WHERE id = ?", flow.ID); err != nil {
		return nil, fmt.Errorf("delete mcp oauth flow: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit consume mcp oauth flow: %w", err)
	}
	flow.State = state
	return flow, nil
}

func (s *Store) UpsertMCPOAuthAccount(account *MCPOAuthAccount) error {
	scopes, err := encodeJSON(account.Scopes)
	if err != nil {
		return fmt.Errorf("encode mcp oauth scopes: %w", err)
	}
	metadata, err := encodeJSON(account.AuthMetadata)
	if err != nil {
		return fmt.Errorf("encode mcp oauth metadata: %w", err)
	}
	row := s.db.QueryRow(
		`INSERT INTO mcp_oauth_accounts (
			instance_id, account_label, subject, email, issuer, resource_uri,
			scopes, access_token, refresh_token, expires_at, auth_metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(instance_id, account_label) DO UPDATE SET
			subject = excluded.subject,
			email = excluded.email,
			issuer = excluded.issuer,
			resource_uri = excluded.resource_uri,
			scopes = excluded.scopes,
			access_token = excluded.access_token,
			refresh_token = excluded.refresh_token,
			expires_at = excluded.expires_at,
			auth_metadata = excluded.auth_metadata,
			updated_at = datetime('now')
		RETURNING id, created_at, updated_at`,
		account.InstanceID,
		account.AccountLabel,
		account.Subject,
		account.Email,
		account.Issuer,
		account.ResourceURI,
		scopes,
		account.AccessToken,
		account.RefreshToken,
		account.ExpiresAt.Format(time.RFC3339),
		metadata,
	)
	if err := row.Scan(&account.ID, &account.CreatedAt, &account.UpdatedAt); err != nil {
		return fmt.Errorf("upsert mcp oauth account: %w", err)
	}
	return nil
}

func (s *Store) ListMCPOAuthAccounts(instanceID string) ([]*MCPOAuthAccount, error) {
	rows, err := s.db.Query(mcpOAuthAccountSelectSQL()+" WHERE instance_id = ? ORDER BY account_label", instanceID)
	if err != nil {
		return nil, fmt.Errorf("query mcp oauth accounts: %w", err)
	}
	defer rows.Close()

	var accounts []*MCPOAuthAccount
	for rows.Next() {
		account, err := scanMCPOAuthAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mcp oauth accounts: %w", err)
	}
	return accounts, nil
}

func (s *Store) GetValidMCPOAuthAccount(instanceID, resourceURI string) (*MCPOAuthAccount, error) {
	rows, err := s.db.Query(mcpOAuthAccountSelectSQL()+" WHERE instance_id = ? AND resource_uri = ? ORDER BY updated_at DESC", instanceID, resourceURI)
	if err != nil {
		return nil, fmt.Errorf("query valid mcp oauth account: %w", err)
	}
	defer rows.Close()

	now := time.Now()
	for rows.Next() {
		account, err := scanMCPOAuthAccount(rows)
		if err != nil {
			return nil, err
		}
		if account.ExpiresAt.IsZero() || account.ExpiresAt.After(now) {
			return account, nil
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate valid mcp oauth account: %w", err)
	}
	return nil, ErrNotFound
}

func (s *Store) ConsumeFlow(instanceID, state string) (mcp.OAuthFlow, error) {
	flow, err := s.ConsumeMCPOAuthFlow(instanceID, state)
	if err != nil {
		if err == ErrNotFound {
			return mcp.OAuthFlow{}, mcp.ErrOAuthFlowNotFound
		}
		return mcp.OAuthFlow{}, err
	}
	return mcp.OAuthFlow{
		ID:                 flow.ID,
		InstanceID:         flow.InstanceID,
		State:              flow.State,
		CodeVerifierSecret: flow.CodeVerifierSecret,
		RedirectURI:        flow.RedirectURI,
		AuthorizationURL:   flow.AuthorizationURL,
		ResourceURI:        flow.ResourceURI,
		ExpiresAt:          flow.ExpiresAt,
		CreatedAt:          flow.CreatedAt,
	}, nil
}

func (s *Store) SaveAccount(account mcp.OAuthAccount) error {
	return s.UpsertMCPOAuthAccount(&MCPOAuthAccount{
		InstanceID:   account.InstanceID,
		AccountLabel: account.AccountLabel,
		Subject:      account.Subject,
		Email:        account.Email,
		Issuer:       account.Issuer,
		ResourceURI:  account.ResourceURI,
		Scopes:       account.Scopes,
		AccessToken:  account.AccessToken,
		RefreshToken: account.RefreshToken,
		ExpiresAt:    account.ExpiresAt,
		AuthMetadata: account.AuthMetadata,
	})
}

func scanMCPOAuthFlow(scanner connectionScanner) (*MCPOAuthFlow, error) {
	var flow MCPOAuthFlow
	var expiresAt string
	err := scanner.Scan(&flow.ID, &flow.InstanceID, &flow.CodeVerifierSecret, &flow.RedirectURI, &flow.AuthorizationURL, &flow.ResourceURI, &expiresAt, &flow.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan mcp oauth flow: %w", err)
	}
	parsed, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("parse mcp oauth flow expiry: %w", err)
	}
	flow.ExpiresAt = parsed
	return &flow, nil
}

func scanMCPOAuthAccount(scanner connectionScanner) (*MCPOAuthAccount, error) {
	var account MCPOAuthAccount
	var scopes, metadata sql.NullString
	var expiresAt string
	err := scanner.Scan(
		&account.ID,
		&account.InstanceID,
		&account.AccountLabel,
		&account.Subject,
		&account.Email,
		&account.Issuer,
		&account.ResourceURI,
		&scopes,
		&account.AccessToken,
		&account.RefreshToken,
		&expiresAt,
		&metadata,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan mcp oauth account: %w", err)
	}
	if err := decodeJSON(scopes, &account.Scopes); err != nil {
		return nil, fmt.Errorf("decode mcp oauth scopes: %w", err)
	}
	if err := decodeJSON(metadata, &account.AuthMetadata); err != nil {
		return nil, fmt.Errorf("decode mcp oauth metadata: %w", err)
	}
	if expiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, expiresAt)
		if err != nil {
			return nil, fmt.Errorf("parse mcp oauth account expiry: %w", err)
		}
		account.ExpiresAt = parsed
	}
	return &account, nil
}

func mcpOAuthFlowSelectSQL() string {
	return `SELECT id, instance_id, code_verifier_secret, redirect_uri, authorization_url, resource_uri, expires_at, created_at FROM mcp_oauth_flows`
}

func mcpOAuthAccountSelectSQL() string {
	return `SELECT id, instance_id, account_label, subject, email, issuer, resource_uri, scopes, access_token, refresh_token, expires_at, auth_metadata, created_at, updated_at FROM mcp_oauth_accounts`
}

func hashOAuthState(state string) string {
	sum := sha256.Sum256([]byte(state))
	return hex.EncodeToString(sum[:])
}
