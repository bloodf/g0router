package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// GetSettings returns all settings as a key→value map.
func (s *Store) GetSettings() (map[string]string, error) {
	rows, err := s.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, fmt.Errorf("query settings: %w", err)
	}
	defer rows.Close()

	out := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, fmt.Errorf("scan setting: %w", err)
		}
		out[k] = v
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate settings: %w", err)
	}
	return out, nil
}

// GetSetting returns a single setting value by key.
func (s *Store) GetSetting(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("get setting %s: %w", key, err)
	}
	return value, nil
}

// SetSetting upserts a single key→value pair.
func (s *Store) SetSetting(key, value string) error {
	return s.SetSettings(map[string]string{key: value})
}

// SetSettings upserts the given key→value pairs.
func (s *Store) SetSettings(values map[string]string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin settings tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().Unix()
	for k, v := range values {
		_, err := tx.Exec(
			`INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
			k, v, now,
		)
		if err != nil {
			return fmt.Errorf("upsert setting %s: %w", k, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit settings tx: %w", err)
	}
	return nil
}
