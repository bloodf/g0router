package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Tunnel is a configured tunnel (cloudflare or tailscale). Token is plaintext in
// memory and encrypted at rest in the token_enc column.
type Tunnel struct {
	Type      string
	IsEnabled bool
	Status    string // inactive|starting|active|error
	URL       string
	Token     string // cloudflared named-tunnel token; encrypted at rest
	Mode      string // cloudflare: named|quick; tailscale: funnel|serve|''
	LastError string
	UpdatedAt int64
}

// GetTunnel returns the tunnel row for typ, token decrypted. It returns
// ErrNotFound when no row exists for that type.
func (s *Store) GetTunnel(typ string) (Tunnel, error) {
	return s.scanTunnel(s.db.QueryRow(
		`SELECT type, is_enabled, status, url, token_enc, mode, last_error, updated_at
		 FROM tunnels WHERE type = ?`, typ))
}

// ListTunnels returns all tunnel rows in deterministic order (cloudflare,
// tailscale before any other type, then alphabetical).
func (s *Store) ListTunnels() ([]Tunnel, error) {
	rows, err := s.db.Query(
		`SELECT type, is_enabled, status, url, token_enc, mode, last_error, updated_at
		 FROM tunnels
		 ORDER BY CASE type WHEN 'cloudflare' THEN 0 WHEN 'tailscale' THEN 1 ELSE 2 END, type`)
	if err != nil {
		return nil, fmt.Errorf("query tunnels: %w", err)
	}
	defer rows.Close()

	var out []Tunnel
	for rows.Next() {
		t, err := s.scanTunnel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tunnels: %w", err)
	}
	return out, nil
}

// UpsertTunnel inserts or updates the tunnel keyed by type, encrypting the token
// at rest. On conflict it overwrites every mutable field (including the token).
func (s *Store) UpsertTunnel(t Tunnel) error {
	tokenEnc, err := s.cipher.Encrypt(t.Token)
	if err != nil {
		return fmt.Errorf("encrypt tunnel token: %w", err)
	}
	status := t.Status
	if status == "" {
		status = "inactive"
	}
	_, err = s.db.Exec(
		`INSERT INTO tunnels (type, is_enabled, status, url, token_enc, mode, last_error, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(type) DO UPDATE SET
		   is_enabled = excluded.is_enabled,
		   status     = excluded.status,
		   url        = excluded.url,
		   token_enc  = excluded.token_enc,
		   mode       = excluded.mode,
		   last_error = excluded.last_error,
		   updated_at = excluded.updated_at`,
		t.Type, boolToInt(t.IsEnabled), status, t.URL, tokenEnc, t.Mode, t.LastError, time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("upsert tunnel %s: %w", t.Type, err)
	}
	return nil
}

// SetTunnelState writes the runtime state of a tunnel after a runner
// Start/Stop/poll, preserving the stored token and mode (which the state
// transition must not clobber).
func (s *Store) SetTunnelState(typ, status, url, lastErr string, enabled bool) error {
	res, err := s.db.Exec(
		`UPDATE tunnels SET is_enabled = ?, status = ?, url = ?, last_error = ?, updated_at = ?
		 WHERE type = ?`,
		boolToInt(enabled), status, url, lastErr, time.Now().Unix(), typ,
	)
	if err != nil {
		return fmt.Errorf("set tunnel state %s: %w", typ, err)
	}
	return requireRowAffected(res)
}

func (s *Store) scanTunnel(row rowScanner) (Tunnel, error) {
	var t Tunnel
	var tokenEnc string
	var isEnabled int
	err := row.Scan(&t.Type, &isEnabled, &t.Status, &t.URL, &tokenEnc, &t.Mode, &t.LastError, &t.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Tunnel{}, ErrNotFound
	}
	if err != nil {
		return Tunnel{}, fmt.Errorf("scan tunnel: %w", err)
	}
	t.IsEnabled = isEnabled != 0
	if t.Token, err = s.cipher.Decrypt(tokenEnc); err != nil {
		return Tunnel{}, fmt.Errorf("decrypt tunnel token: %w", err)
	}
	return t, nil
}
