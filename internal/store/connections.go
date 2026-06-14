package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Connection links a provider to a credential: an API key or an OAuth
// token pair. Secret fields are plaintext in memory and encrypted at rest
// in the *_enc columns.
type Connection struct {
	ID           string
	ProviderID   string
	Name         string
	Kind         string // "api_key" | "oauth"
	Secret       string
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
	Metadata     string
	ProxyPoolID  string
	CreatedAt    int64
	UpdatedAt    int64
}

// CreateConnection inserts a connection, encrypting its secrets.
func (s *Store) CreateConnection(c *Connection) error {
	enc, err := s.encryptConnection(c)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	id, err := newID()
	if err != nil {
		return err
	}
	c.ID = id
	c.CreatedAt = now
	c.UpdatedAt = now
	_, err = s.db.Exec(
		`INSERT INTO connections
		 (id, provider_id, name, kind, secret_enc, access_token_enc, refresh_token_enc, expires_at, metadata, proxy_pool_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.ProviderID, c.Name, c.Kind, enc.secret, enc.access, enc.refresh, c.ExpiresAt, c.Metadata, c.ProxyPoolID, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert connection %s: %w", c.Name, err)
	}
	return nil
}

// ListConnections returns all connections with secrets decrypted.
func (s *Store) ListConnections() ([]*Connection, error) {
	rows, err := s.db.Query(
		`SELECT id, provider_id, name, kind, secret_enc, access_token_enc, refresh_token_enc, expires_at, metadata, proxy_pool_id, created_at, updated_at
		 FROM connections ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("query connections: %w", err)
	}
	defer rows.Close()

	var out []*Connection
	for rows.Next() {
		c, err := s.scanConnection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate connections: %w", err)
	}
	return out, nil
}

// GetConnection returns the connection with the given ID, secrets decrypted.
func (s *Store) GetConnection(id string) (*Connection, error) {
	row := s.db.QueryRow(
		`SELECT id, provider_id, name, kind, secret_enc, access_token_enc, refresh_token_enc, expires_at, metadata, proxy_pool_id, created_at, updated_at
		 FROM connections WHERE id = ?`, id)
	return s.scanConnection(row)
}

// UpdateConnection persists mutable fields, re-encrypting secrets.
func (s *Store) UpdateConnection(c *Connection) error {
	enc, err := s.encryptConnection(c)
	if err != nil {
		return err
	}
	res, err := s.db.Exec(
		`UPDATE connections SET provider_id = ?, name = ?, kind = ?, secret_enc = ?, access_token_enc = ?,
		 refresh_token_enc = ?, expires_at = ?, metadata = ?, proxy_pool_id = ?, updated_at = ? WHERE id = ?`,
		c.ProviderID, c.Name, c.Kind, enc.secret, enc.access, enc.refresh, c.ExpiresAt, c.Metadata, c.ProxyPoolID, time.Now().Unix(), c.ID,
	)
	if err != nil {
		return fmt.Errorf("update connection %s: %w", c.ID, err)
	}
	return requireRowAffected(res)
}

// DeleteConnection removes the connection with the given ID.
func (s *Store) DeleteConnection(id string) error {
	res, err := s.db.Exec("DELETE FROM connections WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete connection %s: %w", id, err)
	}
	return requireRowAffected(res)
}

type encryptedSecrets struct {
	secret, access, refresh string
}

func (s *Store) encryptConnection(c *Connection) (encryptedSecrets, error) {
	var enc encryptedSecrets
	var err error
	if enc.secret, err = s.cipher.Encrypt(c.Secret); err != nil {
		return enc, fmt.Errorf("encrypt secret: %w", err)
	}
	if enc.access, err = s.cipher.Encrypt(c.AccessToken); err != nil {
		return enc, fmt.Errorf("encrypt access token: %w", err)
	}
	if enc.refresh, err = s.cipher.Encrypt(c.RefreshToken); err != nil {
		return enc, fmt.Errorf("encrypt refresh token: %w", err)
	}
	return enc, nil
}

func (s *Store) scanConnection(row rowScanner) (*Connection, error) {
	var c Connection
	var secretEnc, accessEnc, refreshEnc string
	err := row.Scan(&c.ID, &c.ProviderID, &c.Name, &c.Kind, &secretEnc, &accessEnc, &refreshEnc,
		&c.ExpiresAt, &c.Metadata, &c.ProxyPoolID, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan connection: %w", err)
	}
	if c.Secret, err = s.cipher.Decrypt(secretEnc); err != nil {
		return nil, fmt.Errorf("decrypt secret: %w", err)
	}
	if c.AccessToken, err = s.cipher.Decrypt(accessEnc); err != nil {
		return nil, fmt.Errorf("decrypt access token: %w", err)
	}
	if c.RefreshToken, err = s.cipher.Decrypt(refreshEnc); err != nil {
		return nil, fmt.Errorf("decrypt refresh token: %w", err)
	}
	return &c, nil
}
