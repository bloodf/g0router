package store

import (
	"database/sql"
	"fmt"
)

// FeatureFlag represents a toggleable feature.
type FeatureFlag struct {
	ID          int64
	Key         string
	Enabled     bool
	Description string
	CreatedAt   string
}

// ListFeatureFlags returns all feature flags ordered by key.
func (s *Store) ListFeatureFlags() ([]FeatureFlag, error) {
	rows, err := s.db.Query(`SELECT id, key, enabled, description, created_at FROM feature_flags ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("list feature flags: %w", err)
	}
	defer rows.Close()

	var flags []FeatureFlag
	for rows.Next() {
		f, err := scanFeatureFlag(rows)
		if err != nil {
			return nil, err
		}
		flags = append(flags, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feature flags: %w", err)
	}
	return flags, nil
}

// GetFeatureFlag returns a single feature flag by id.
func (s *Store) GetFeatureFlag(id int64) (*FeatureFlag, error) {
	f, err := scanFeatureFlag(s.db.QueryRow(
		`SELECT id, key, enabled, description, created_at FROM feature_flags WHERE id = ?`,
		id,
	))
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// GetFeatureFlagByKey returns a single feature flag by key.
func (s *Store) GetFeatureFlagByKey(key string) (*FeatureFlag, error) {
	f, err := scanFeatureFlag(s.db.QueryRow(
		`SELECT id, key, enabled, description, created_at FROM feature_flags WHERE key = ?`,
		key,
	))
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// ToggleFeatureFlag updates only the enabled field of a feature flag.
func (s *Store) ToggleFeatureFlag(id int64, enabled bool) error {
	result, err := s.db.Exec(
		`UPDATE feature_flags SET enabled = ? WHERE id = ?`,
		boolInt(enabled), id,
	)
	if err != nil {
		return fmt.Errorf("toggle feature flag: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}
	return nil
}

func scanFeatureFlag(scanner interface{ Scan(dest ...any) error }) (FeatureFlag, error) {
	var f FeatureFlag
	var description sql.NullString
	var enabled int

	err := scanner.Scan(&f.ID, &f.Key, &enabled, &description, &f.CreatedAt)
	if err == sql.ErrNoRows {
		return FeatureFlag{}, ErrNotFound
	}
	if err != nil {
		return FeatureFlag{}, fmt.Errorf("scan feature flag: %w", err)
	}

	f.Enabled = enabled == 1
	f.Description = stringValueFromNull(description)
	return f, nil
}

// seedFeatureFlags inserts the default flags if they don't exist.
func (s *Store) seedFeatureFlags() error {
	flags := []struct {
		key         string
		description string
	}{
		{"semantic_cache", "Enable semantic caching of inference responses"},
		{"guardrails", "Enable guardrails (blocklist + PII redaction)"},
		{"pii_redaction", "Enable PII redaction in prompts"},
		{"websocket_chat", "Enable WebSocket chat interface"},
		{"mitm_proxy", "Enable MITM proxy for request inspection"},
	}
	for _, f := range flags {
		_, err := s.db.Exec(
			`INSERT OR IGNORE INTO feature_flags (key, enabled, description) VALUES (?, 0, ?)`,
			f.key, f.description,
		)
		if err != nil {
			return fmt.Errorf("seed feature flag %q: %w", f.key, err)
		}
	}
	return nil
}
