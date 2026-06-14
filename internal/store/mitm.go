package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// MitmTool is a configured MITM tool row (the 5-field UI MitmTool shape). The
// table holds NO secret: dns_override is config, not a credential. The only
// secret in the MITM domain is the root CA private key, handled out-of-band in
// internal/platform/mitm (a 0o600 file under the data dir), never in this table.
type MitmTool struct {
	ID          string
	Name        string
	Enabled     bool
	DNSOverride string
	Status      string // active|inactive
	UpdatedAt   int64
}

// mitmEnabledSettingKey is where the single global MITM enable flag lives, in the
// shared key-value settings surface (mirroring tunnelDashboardAccess/tunnelUrl).
const mitmEnabledSettingKey = "mitmEnabled"

// mitmSeedTools are the two named domain tools surfaced by the UI. EnsureMitmTools
// seeds them once on first run so ListMitmTools always returns >=2 entries.
var mitmSeedTools = []MitmTool{
	{ID: "mitm-1", Name: "Request Inspector", Enabled: true, DNSOverride: "localhost", Status: "active"},
	{ID: "mitm-2", Name: "Response Modifier", Enabled: false, DNSOverride: "", Status: "inactive"},
}

// ListMitmTools returns all MITM tool rows in deterministic order (by id).
func (s *Store) ListMitmTools() ([]MitmTool, error) {
	rows, err := s.db.Query(
		`SELECT id, name, enabled, dns_override, status, updated_at
		 FROM mitm_tools ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("query mitm_tools: %w", err)
	}
	defer rows.Close()

	var out []MitmTool
	for rows.Next() {
		t, err := scanMitmTool(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mitm_tools: %w", err)
	}
	return out, nil
}

// GetMitmTool returns the MITM tool row for id, or ErrNotFound.
func (s *Store) GetMitmTool(id string) (MitmTool, error) {
	return scanMitmTool(s.db.QueryRow(
		`SELECT id, name, enabled, dns_override, status, updated_at
		 FROM mitm_tools WHERE id = ?`, id))
}

// UpsertMitmTool inserts or updates a MITM tool keyed by id, overwriting every
// mutable field on conflict.
func (s *Store) UpsertMitmTool(t MitmTool) error {
	status := t.Status
	if status == "" {
		status = "inactive"
	}
	_, err := s.db.Exec(
		`INSERT INTO mitm_tools (id, name, enabled, dns_override, status, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   name         = excluded.name,
		   enabled      = excluded.enabled,
		   dns_override = excluded.dns_override,
		   status       = excluded.status,
		   updated_at   = excluded.updated_at`,
		t.ID, t.Name, boolToInt(t.Enabled), t.DNSOverride, status, time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("upsert mitm tool %s: %w", t.ID, err)
	}
	return nil
}

// SetMitmToolEnabled flips a tool's enabled flag, derives its status
// (enabled→active, disabled→inactive), persists, and returns the updated row.
// ErrNotFound on an unknown id.
func (s *Store) SetMitmToolEnabled(id string, enabled bool) (MitmTool, error) {
	status := "inactive"
	if enabled {
		status = "active"
	}
	res, err := s.db.Exec(
		`UPDATE mitm_tools SET enabled = ?, status = ?, updated_at = ? WHERE id = ?`,
		boolToInt(enabled), status, time.Now().Unix(), id,
	)
	if err != nil {
		return MitmTool{}, fmt.Errorf("set mitm tool enabled %s: %w", id, err)
	}
	if err := requireRowAffected(res); err != nil {
		return MitmTool{}, err
	}
	return s.GetMitmTool(id)
}

// GetMitmEnabled reports the global MITM enable flag. An unset key reads as false.
func (s *Store) GetMitmEnabled() (bool, error) {
	v, err := s.GetSetting(mitmEnabledSettingKey)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return v == "true", nil
}

// SetMitmEnabled persists the global MITM enable flag.
func (s *Store) SetMitmEnabled(enabled bool) error {
	v := "false"
	if enabled {
		v = "true"
	}
	return s.SetSetting(mitmEnabledSettingKey, v)
}

// EnsureMitmTools seeds the two named MITM tools (Request Inspector, Response
// Modifier) if they are absent. Idempotent: existing rows are left untouched.
func (s *Store) EnsureMitmTools() error {
	for _, t := range mitmSeedTools {
		_, err := s.db.Exec(
			`INSERT INTO mitm_tools (id, name, enabled, dns_override, status, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?)
			 ON CONFLICT(id) DO NOTHING`,
			t.ID, t.Name, boolToInt(t.Enabled), t.DNSOverride, t.Status, time.Now().Unix(),
		)
		if err != nil {
			return fmt.Errorf("seed mitm tool %s: %w", t.ID, err)
		}
	}
	return nil
}

func scanMitmTool(row rowScanner) (MitmTool, error) {
	var t MitmTool
	var enabled int
	err := row.Scan(&t.ID, &t.Name, &enabled, &t.DNSOverride, &t.Status, &t.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return MitmTool{}, ErrNotFound
	}
	if err != nil {
		return MitmTool{}, fmt.Errorf("scan mitm tool: %w", err)
	}
	t.Enabled = enabled != 0
	return t, nil
}
