package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ModelLock represents an active timed lock on a model for a connection.
type ModelLock struct {
	ConnID    string
	Model     string
	ExpiresAt int64
}

// LockModel creates or replaces a timed lock on the given model for a connection.
func (s *Store) LockModel(connID, model string, expiresAt int64) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO connection_model_locks (connection_id, model, expires_at) VALUES (?, ?, ?)`,
		connID, model, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("lock model %s/%s: %w", connID, model, err)
	}
	return nil
}

// LockAccount locks the entire account for a connection using the "__all" sentinel.
func (s *Store) LockAccount(connID string, expiresAt int64) error {
	return s.LockModel(connID, "__all", expiresAt)
}

// ClearLocks removes all locks for the given connection.
func (s *Store) ClearLocks(connID string) error {
	_, err := s.db.Exec("DELETE FROM connection_model_locks WHERE connection_id = ?", connID)
	if err != nil {
		return fmt.Errorf("clear locks %s: %w", connID, err)
	}
	return nil
}

// ActiveLocks returns all non-expired locks for the given connection.
// now is the current Unix timestamp used as the expiry threshold.
func (s *Store) ActiveLocks(connID string, now int64) ([]*ModelLock, error) {
	rows, err := s.db.Query(
		`SELECT connection_id, model, expires_at FROM connection_model_locks
		 WHERE connection_id = ? AND expires_at > ?`,
		connID, now,
	)
	if err != nil {
		return nil, fmt.Errorf("query active locks %s: %w", connID, err)
	}
	defer rows.Close()

	var out []*ModelLock
	for rows.Next() {
		var l ModelLock
		if err := rows.Scan(&l.ConnID, &l.Model, &l.ExpiresAt); err != nil {
			return nil, fmt.Errorf("scan model lock: %w", err)
		}
		out = append(out, &l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate model locks: %w", err)
	}
	return out, nil
}

// EarliestExpiry returns the earliest active lock expiry across all connections
// for the given model. Returns (0, false, nil) when no active locks exist.
func (s *Store) EarliestExpiry(model string, now int64) (int64, bool, error) {
	var exp sql.NullInt64
	err := s.db.QueryRow(
		`SELECT MIN(expires_at) FROM connection_model_locks WHERE model = ? AND expires_at > ?`,
		model, now,
	).Scan(&exp)
	if err != nil {
		return 0, false, fmt.Errorf("earliest expiry %s: %w", model, err)
	}
	if !exp.Valid {
		return 0, false, nil
	}
	return exp.Int64, true, nil
}

// SetBackoffLevel updates the backoff_level column for the given connection.
func (s *Store) SetBackoffLevel(connID string, level int) error {
	_, err := s.db.Exec(
		"UPDATE connections SET backoff_level = ?, updated_at = ? WHERE id = ?",
		level, time.Now().Unix(), connID,
	)
	if err != nil {
		return fmt.Errorf("set backoff level %s: %w", connID, err)
	}
	return nil
}

// GetBackoffLevel returns the backoff_level for the given connection.
// Returns 0 if the connection does not exist.
func (s *Store) GetBackoffLevel(connID string) (int, error) {
	var level int
	err := s.db.QueryRow("SELECT backoff_level FROM connections WHERE id = ?", connID).Scan(&level)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get backoff level %s: %w", connID, err)
	}
	return level, nil
}
