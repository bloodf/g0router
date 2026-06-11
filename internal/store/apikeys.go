package store

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"time"
)

// APIKey is a persisted dashboard API key.
type APIKey struct {
	ID        string
	Key       string
	Name      string
	MachineID string
	IsActive  bool
	CreatedAt int64
}

// DataDir returns the directory containing the SQLite database, derived from
// the database file path. It is used by callers that need to read persisted
// machine-id and CLI-secret files.
func (s *Store) DataDir() string {
	var seq int
	var name, path string
	if err := s.db.QueryRow("PRAGMA database_list").Scan(&seq, &name, &path); err == nil && path != "" {
		return filepath.Dir(path)
	}
	return ""
}

// CreateAPIKey inserts a new API key record. The caller must supply a
// formatted key value and the machine identifier used to derive it.
func (s *Store) CreateAPIKey(name, key, machineID string) (*APIKey, error) {
	now := time.Now().Unix()
	id, err := newID()
	if err != nil {
		return nil, err
	}

	k := &APIKey{
		ID:        id,
		Key:       key,
		Name:      name,
		MachineID: machineID,
		IsActive:  true,
		CreatedAt: now,
	}
	_, err = s.db.Exec(
		"INSERT INTO api_keys (id, key, name, machine_id, is_active, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		k.ID, k.Key, k.Name, k.MachineID, boolToInt(k.IsActive), k.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert api key %s: %w", name, err)
	}
	return k, nil
}

// ListAPIKeys returns all API keys ordered by creation time (newest first).
func (s *Store) ListAPIKeys() ([]*APIKey, error) {
	rows, err := s.db.Query(
		"SELECT id, key, name, machine_id, is_active, created_at FROM api_keys ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("query api keys: %w", err)
	}
	defer rows.Close()

	var out []*APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate api keys: %w", err)
	}
	return out, nil
}

// GetAPIKeyByID returns the API key with the given id.
func (s *Store) GetAPIKeyByID(id string) (*APIKey, error) {
	return scanAPIKey(s.db.QueryRow(
		"SELECT id, key, name, machine_id, is_active, created_at FROM api_keys WHERE id = ?", id))
}

// GetAPIKeyByKey returns the API key with the exact key value.
func (s *Store) GetAPIKeyByKey(key string) (*APIKey, error) {
	return scanAPIKey(s.db.QueryRow(
		"SELECT id, key, name, machine_id, is_active, created_at FROM api_keys WHERE key = ?", key))
}

// SetAPIKeyActive updates the is_active flag for the given key id.
func (s *Store) SetAPIKeyActive(id string, active bool) error {
	res, err := s.db.Exec(
		"UPDATE api_keys SET is_active = ? WHERE id = ?",
		boolToInt(active), id,
	)
	if err != nil {
		return fmt.Errorf("update api key active %s: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteAPIKey removes the API key with the given id.
func (s *Store) DeleteAPIKey(id string) error {
	res, err := s.db.Exec("DELETE FROM api_keys WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete api key %s: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func scanAPIKey(row interface {
	Scan(dest ...any) error
}) (*APIKey, error) {
	var k APIKey
	var isActive int
	err := row.Scan(&k.ID, &k.Key, &k.Name, &k.MachineID, &isActive, &k.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan api key: %w", err)
	}
	k.IsActive = isActive != 0
	return &k, nil
}

