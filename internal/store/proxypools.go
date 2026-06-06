package store

import (
	"database/sql"
	"fmt"
)

// ProxyPool represents a proxy pool entry.
type ProxyPool struct {
	ID              string
	Name            string
	Protocol        string
	Host            string
	Port            int
	Username        string
	Password        string // plaintext, NOT stored; only for encryption/decryption
	IsActive        bool
	LastCheckAt     string
	LastCheckStatus string
	CreatedAt       string
}

// ListProxyPools returns all proxy pools with decrypted passwords.
func (s *Store) ListProxyPools() ([]ProxyPool, error) {
	rows, err := s.db.Query(proxyPoolSelectSQL() + " ORDER BY created_at, id")
	if err != nil {
		return nil, fmt.Errorf("query proxy pools: %w", err)
	}
	defer rows.Close()

	var pools []ProxyPool
	for rows.Next() {
		pool, err := s.scanProxyPool(rows)
		if err != nil {
			return nil, err
		}
		pools = append(pools, pool)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate proxy pools: %w", err)
	}
	return pools, nil
}

// GetProxyPool returns a single proxy pool by id with decrypted password.
func (s *Store) GetProxyPool(id string) (*ProxyPool, error) {
	pool, err := s.scanProxyPool(s.db.QueryRow(proxyPoolSelectSQL()+" WHERE id = ?", id))
	if err != nil {
		return nil, err
	}
	return &pool, nil
}

// CreateProxyPool inserts a new proxy pool, encrypting the Password field.
func (s *Store) CreateProxyPool(pool ProxyPool) (*ProxyPool, error) {
	passwordEnc, err := s.encryptString(pool.Password)
	if err != nil {
		return nil, fmt.Errorf("encrypt password: %w", err)
	}

	row := s.db.QueryRow(
		`INSERT INTO proxy_pools (name, protocol, host, port, username, password_enc, is_active, last_check_at, last_check_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, created_at`,
		pool.Name,
		pool.Protocol,
		pool.Host,
		pool.Port,
		emptyStringNil(pool.Username),
		emptyStringNil(passwordEnc),
		boolToInt(pool.IsActive),
		emptyStringNil(pool.LastCheckAt),
		emptyStringNil(pool.LastCheckStatus),
	)
	if err := row.Scan(&pool.ID, &pool.CreatedAt); err != nil {
		return nil, fmt.Errorf("insert proxy pool: %w", err)
	}
	pool.Password = ""
	return &pool, nil
}

// UpdateProxyPool updates an existing proxy pool. If Password is non-empty it
// is encrypted and stored; otherwise the existing password_enc is preserved.
func (s *Store) UpdateProxyPool(id string, pool ProxyPool) error {
	var passwordEnc any
	if pool.Password != "" {
		enc, err := s.encryptString(pool.Password)
		if err != nil {
			return fmt.Errorf("encrypt password: %w", err)
		}
		passwordEnc = enc
	} else {
		passwordEnc = nil
	}

	result, err := s.db.Exec(
		`UPDATE proxy_pools SET
			name = ?,
			protocol = ?,
			host = ?,
			port = ?,
			username = ?,
			password_enc = COALESCE(?, password_enc),
			is_active = ?,
			last_check_at = ?,
			last_check_status = ?
		WHERE id = ?`,
		pool.Name,
		pool.Protocol,
		pool.Host,
		pool.Port,
		emptyStringNil(pool.Username),
		passwordEnc,
		boolToInt(pool.IsActive),
		emptyStringNil(pool.LastCheckAt),
		emptyStringNil(pool.LastCheckStatus),
		id,
	)
	if err != nil {
		return fmt.Errorf("update proxy pool: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}
	return nil
}

// DeleteProxyPool removes a proxy pool by id.
func (s *Store) DeleteProxyPool(id string) error {
	result, err := s.db.Exec("DELETE FROM proxy_pools WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete proxy pool: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}
	return nil
}

// UpdateProxyPoolStatus updates only the last_check_at and last_check_status fields for a proxy pool.
func (s *Store) UpdateProxyPoolStatus(id string, status, lastError string) error {
	_, err := s.db.Exec(
		"UPDATE proxy_pools SET last_check_at = CURRENT_TIMESTAMP, last_check_status = ? WHERE id = ?",
		status, id,
	)
	if err != nil {
		return fmt.Errorf("update proxy pool status: %w", err)
	}
	return nil
}

// TestProxyPool is a placeholder for connectivity testing.
func (s *Store) TestProxyPool(id string) (ok bool, latencyMs int, err error) {
	return true, 0, nil
}

func proxyPoolSelectSQL() string {
	return `SELECT id, name, protocol, host, port, username, password_enc, is_active, last_check_at, last_check_status, created_at FROM proxy_pools`
}

func (s *Store) scanProxyPool(scanner connectionScanner) (ProxyPool, error) {
	var pool ProxyPool
	var username, passwordEnc, lastCheckAt, lastCheckStatus sql.NullString
	var isActive int

	err := scanner.Scan(
		&pool.ID,
		&pool.Name,
		&pool.Protocol,
		&pool.Host,
		&pool.Port,
		&username,
		&passwordEnc,
		&isActive,
		&lastCheckAt,
		&lastCheckStatus,
		&pool.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return ProxyPool{}, ErrNotFound
	}
	if err != nil {
		return ProxyPool{}, fmt.Errorf("scan proxy pool: %w", err)
	}

	pool.Username = stringValueFromNull(username)
	pool.IsActive = isActive != 0
	pool.LastCheckAt = stringValueFromNull(lastCheckAt)
	pool.LastCheckStatus = stringValueFromNull(lastCheckStatus)

	if passwordEnc.Valid && passwordEnc.String != "" {
		decrypted, err := s.decryptString(passwordEnc.String)
		if err != nil {
			return ProxyPool{}, fmt.Errorf("decrypt password: %w", err)
		}
		pool.Password = decrypted
	}

	return pool, nil
}
