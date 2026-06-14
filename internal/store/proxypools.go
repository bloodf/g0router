package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ProxyPool is a configured outbound proxy. Password is plaintext in memory and
// encrypted at rest in the password_enc column.
type ProxyPool struct {
	ID              string
	Name            string
	Protocol        string
	Host            string
	Port            int
	Username        string
	Password        string
	IsActive        bool
	LastCheckStatus string
	LastCheckAt     string // ISO-8601 (RFC3339)
	CreatedAt       int64
	UpdatedAt       int64
}

// CreateProxyPool inserts a proxy pool, encrypting its password at rest, and
// returns the stored record with its generated ID and timestamps.
func (s *Store) CreateProxyPool(p *ProxyPool) (*ProxyPool, error) {
	passwordEnc, err := s.cipher.Encrypt(p.Password)
	if err != nil {
		return nil, fmt.Errorf("encrypt proxy password: %w", err)
	}
	id, err := newID()
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	protocol := p.Protocol
	if protocol == "" {
		protocol = "http"
	}
	_, err = s.db.Exec(
		`INSERT INTO proxy_pools
		 (id, name, protocol, host, port, username, password_enc, is_active, last_check_status, last_check_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, p.Name, protocol, p.Host, p.Port, p.Username, passwordEnc, boolToInt(p.IsActive),
		p.LastCheckStatus, p.LastCheckAt, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert proxy pool %s: %w", p.Name, err)
	}
	return s.GetProxyPoolByID(id)
}

// ListProxyPools returns proxy pools ordered by creation time. When filterActive
// is non-nil, only pools whose is_active matches *filterActive are returned.
func (s *Store) ListProxyPools(filterActive *bool) ([]*ProxyPool, error) {
	query := `SELECT id, name, protocol, host, port, username, password_enc, is_active, last_check_status, last_check_at, created_at, updated_at
		 FROM proxy_pools`
	var args []any
	if filterActive != nil {
		query += " WHERE is_active = ?"
		args = append(args, boolToInt(*filterActive))
	}
	query += " ORDER BY created_at, id"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query proxy pools: %w", err)
	}
	defer rows.Close()

	var out []*ProxyPool
	for rows.Next() {
		p, err := s.scanProxyPool(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate proxy pools: %w", err)
	}
	return out, nil
}

// GetProxyPoolByID returns the pool with the given id, password decrypted.
func (s *Store) GetProxyPoolByID(id string) (*ProxyPool, error) {
	return s.scanProxyPool(s.db.QueryRow(
		`SELECT id, name, protocol, host, port, username, password_enc, is_active, last_check_status, last_check_at, created_at, updated_at
		 FROM proxy_pools WHERE id = ?`, id))
}

// UpdateProxyPool persists mutable fields, re-encrypting the password at rest.
func (s *Store) UpdateProxyPool(p *ProxyPool) error {
	passwordEnc, err := s.cipher.Encrypt(p.Password)
	if err != nil {
		return fmt.Errorf("encrypt proxy password: %w", err)
	}
	protocol := p.Protocol
	if protocol == "" {
		protocol = "http"
	}
	res, err := s.db.Exec(
		`UPDATE proxy_pools SET name = ?, protocol = ?, host = ?, port = ?, username = ?, password_enc = ?,
		 is_active = ?, last_check_status = ?, last_check_at = ?, updated_at = ? WHERE id = ?`,
		p.Name, protocol, p.Host, p.Port, p.Username, passwordEnc, boolToInt(p.IsActive),
		p.LastCheckStatus, p.LastCheckAt, time.Now().Unix(), p.ID,
	)
	if err != nil {
		return fmt.Errorf("update proxy pool %s: %w", p.ID, err)
	}
	return requireRowAffected(res)
}

// DeleteProxyPool removes the pool with the given id.
func (s *Store) DeleteProxyPool(id string) error {
	res, err := s.db.Exec("DELETE FROM proxy_pools WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete proxy pool %s: %w", id, err)
	}
	return requireRowAffected(res)
}

// SetProxyPoolCheck records the result of a connectivity test on the pool.
func (s *Store) SetProxyPoolCheck(id, status, atRFC3339 string) error {
	res, err := s.db.Exec(
		"UPDATE proxy_pools SET last_check_status = ?, last_check_at = ?, updated_at = ? WHERE id = ?",
		status, atRFC3339, time.Now().Unix(), id,
	)
	if err != nil {
		return fmt.Errorf("set proxy pool check %s: %w", id, err)
	}
	return requireRowAffected(res)
}

// CountConnectionsUsingProxyPool returns the number of connections bound to the
// pool via connections.proxy_pool_id — the bound-connection delete guard.
func (s *Store) CountConnectionsUsingProxyPool(id string) (int, error) {
	var n int
	err := s.db.QueryRow("SELECT COUNT(*) FROM connections WHERE proxy_pool_id = ?", id).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count connections using proxy pool %s: %w", id, err)
	}
	return n, nil
}

func (s *Store) scanProxyPool(row rowScanner) (*ProxyPool, error) {
	var p ProxyPool
	var passwordEnc string
	var isActive int
	err := row.Scan(&p.ID, &p.Name, &p.Protocol, &p.Host, &p.Port, &p.Username, &passwordEnc,
		&isActive, &p.LastCheckStatus, &p.LastCheckAt, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan proxy pool: %w", err)
	}
	p.IsActive = isActive != 0
	if p.Password, err = s.cipher.Decrypt(passwordEnc); err != nil {
		return nil, fmt.Errorf("decrypt proxy password: %w", err)
	}
	return &p, nil
}
