package store

import (
	"database/sql"
	"fmt"
)

// SetKV upserts a single key→value pair within a scope.
func (s *Store) SetKV(scope, key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO kv (scope, key, value) VALUES (?, ?, ?)
		 ON CONFLICT(scope, key) DO UPDATE SET value = excluded.value`,
		scope, key, value,
	)
	if err != nil {
		return fmt.Errorf("set kv %s/%s: %w", scope, key, err)
	}
	return nil
}

// GetKV returns a single kv value by scope and key.
// A missing key returns ("", nil), mirroring the settings convention.
func (s *Store) GetKV(scope, key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM kv WHERE scope = ? AND key = ?", scope, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("get kv %s/%s: %w", scope, key, err)
	}
	return value, nil
}

// ListKV returns all key→value pairs for a scope.
func (s *Store) ListKV(scope string) (map[string]string, error) {
	rows, err := s.db.Query("SELECT key, value FROM kv WHERE scope = ?", scope)
	if err != nil {
		return nil, fmt.Errorf("list kv %s: %w", scope, err)
	}
	defer rows.Close()

	out := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, fmt.Errorf("scan kv %s: %w", scope, err)
		}
		out[k] = v
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate kv %s: %w", scope, err)
	}
	return out, nil
}
