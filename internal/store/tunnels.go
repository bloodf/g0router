package store

import (
	"database/sql"
	"fmt"
)

// TunnelConfig represents a tunnel configuration entry.
type TunnelConfig struct {
	ID        string
	Type      string // 'cloudflare' | 'tailscale'
	IsEnabled bool
	Config    string // plaintext in memory; encrypted at rest
	URL       string
	Status    string // 'inactive' | 'starting' | 'active' | 'error'
	LastError string
	CreatedAt string
}

// ListTunnelConfigs returns all tunnel configs with decrypted config values.
func (s *Store) ListTunnelConfigs() ([]TunnelConfig, error) {
	rows, err := s.db.Query(tunnelConfigSelectSQL() + " ORDER BY created_at, id")
	if err != nil {
		return nil, fmt.Errorf("query tunnel configs: %w", err)
	}
	defer rows.Close()

	var configs []TunnelConfig
	for rows.Next() {
		cfg, err := s.scanTunnelConfig(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, cfg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tunnel configs: %w", err)
	}
	return configs, nil
}

// GetTunnelConfig returns a single tunnel config by type with decrypted config.
func (s *Store) GetTunnelConfig(tunnelType string) (*TunnelConfig, error) {
	cfg, err := s.scanTunnelConfig(s.db.QueryRow(tunnelConfigSelectSQL()+" WHERE type = ?", tunnelType))
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// UpsertTunnelConfig inserts a new tunnel config or updates an existing one
// (matched by type), encrypting the Config field into config_enc.
func (s *Store) UpsertTunnelConfig(cfg TunnelConfig) error {
	configEnc, err := s.encryptString(cfg.Config)
	if err != nil {
		return fmt.Errorf("encrypt config: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT INTO tunnel_config (type, is_enabled, config_enc, url, status, last_error)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(type) DO UPDATE SET
			is_enabled = excluded.is_enabled,
			config_enc = excluded.config_enc,
			url = excluded.url,
			status = excluded.status,
			last_error = excluded.last_error`,
		cfg.Type,
		boolToInt(cfg.IsEnabled),
		emptyStringNil(configEnc),
		emptyStringNil(cfg.URL),
		emptyStringNil(cfg.Status),
		emptyStringNil(cfg.LastError),
	)
	if err != nil {
		return fmt.Errorf("upsert tunnel config: %w", err)
	}
	return nil
}

// UpdateTunnelStatus updates only the status and last_error fields for a tunnel.
func (s *Store) UpdateTunnelStatus(tunnelType, status, lastError string) error {
	result, err := s.db.Exec(
		"UPDATE tunnel_config SET status = ?, last_error = ? WHERE type = ?",
		emptyStringNil(status),
		emptyStringNil(lastError),
		tunnelType,
	)
	if err != nil {
		return fmt.Errorf("update tunnel status: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}
	return nil
}

func tunnelConfigSelectSQL() string {
	return `SELECT id, type, is_enabled, config_enc, url, status, last_error, created_at FROM tunnel_config`
}

func (s *Store) scanTunnelConfig(scanner connectionScanner) (TunnelConfig, error) {
	var cfg TunnelConfig
	var id int
	var configEnc, url, status, lastError sql.NullString
	var isEnabled int

	err := scanner.Scan(
		&id,
		&cfg.Type,
		&isEnabled,
		&configEnc,
		&url,
		&status,
		&lastError,
		&cfg.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return TunnelConfig{}, ErrNotFound
	}
	if err != nil {
		return TunnelConfig{}, fmt.Errorf("scan tunnel config: %w", err)
	}

	cfg.ID = fmt.Sprintf("%d", id)
	cfg.IsEnabled = isEnabled != 0
	cfg.URL = stringValueFromNull(url)
	cfg.Status = stringValueFromNull(status)
	cfg.LastError = stringValueFromNull(lastError)

	if configEnc.Valid && configEnc.String != "" {
		decrypted, err := s.decryptString(configEnc.String)
		if err != nil {
			return TunnelConfig{}, fmt.Errorf("decrypt config: %w", err)
		}
		cfg.Config = decrypted
	}

	return cfg, nil
}
