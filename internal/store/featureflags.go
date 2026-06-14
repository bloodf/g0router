package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// FeatureFlag is a named runtime toggle with a human-readable description.
type FeatureFlag struct {
	ID          int64
	Key         string
	Enabled     bool
	Description string
	CreatedAt   string // ISO-8601 (RFC3339)
}

// ListFeatureFlags returns all feature flags ordered by id ascending.
func (s *Store) ListFeatureFlags() ([]*FeatureFlag, error) {
	rows, err := s.db.Query(
		"SELECT id, key, enabled, description, created_at FROM feature_flags ORDER BY id ASC",
	)
	if err != nil {
		return nil, fmt.Errorf("query feature flags: %w", err)
	}
	defer rows.Close()

	var out []*FeatureFlag
	for rows.Next() {
		f, err := scanFeatureFlag(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feature flags: %w", err)
	}
	return out, nil
}

// GetFeatureFlagByID returns the feature flag with the given id.
func (s *Store) GetFeatureFlagByID(id int64) (*FeatureFlag, error) {
	return scanFeatureFlag(s.db.QueryRow(
		"SELECT id, key, enabled, description, created_at FROM feature_flags WHERE id = ?", id))
}

// SetFeatureFlagEnabled toggles the enabled state of a flag and returns the
// updated record. It returns ErrNotFound when no flag has the given id.
func (s *Store) SetFeatureFlagEnabled(id int64, enabled bool) (*FeatureFlag, error) {
	res, err := s.db.Exec("UPDATE feature_flags SET enabled = ? WHERE id = ?", boolToInt(enabled), id)
	if err != nil {
		return nil, fmt.Errorf("update feature flag %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return nil, ErrNotFound
	}
	return s.GetFeatureFlagByID(id)
}

func scanFeatureFlag(row interface {
	Scan(dest ...any) error
}) (*FeatureFlag, error) {
	var f FeatureFlag
	var enabled int
	err := row.Scan(&f.ID, &f.Key, &enabled, &f.Description, &f.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan feature flag: %w", err)
	}
	f.Enabled = enabled != 0
	return &f, nil
}
