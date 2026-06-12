package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// DisableModels adds the given model IDs to the disabled set for a provider alias.
// Calling DisableModels with an already-disabled model is idempotent.
func (s *Store) DisableModels(providerAlias string, modelIDs []string) error {
	if len(modelIDs) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin disable tx: %w", err)
	}
	defer tx.Rollback()

	for _, id := range modelIDs {
		if _, err := tx.Exec(
			"INSERT OR IGNORE INTO disabled_models (provider_alias, model_id) VALUES (?, ?)",
			providerAlias, id,
		); err != nil {
			return fmt.Errorf("disable model %s/%s: %w", providerAlias, id, err)
		}
	}
	return tx.Commit()
}

// EnableModels removes the given model IDs from the disabled set for a provider alias.
// If modelIDs is empty, ALL models for that alias are enabled (cleared).
func (s *Store) EnableModels(providerAlias string, modelIDs []string) error {
	if len(modelIDs) == 0 {
		_, err := s.db.Exec("DELETE FROM disabled_models WHERE provider_alias = ?", providerAlias)
		if err != nil {
			return fmt.Errorf("enable all models %s: %w", providerAlias, err)
		}
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin enable tx: %w", err)
	}
	defer tx.Rollback()

	for _, id := range modelIDs {
		if _, err := tx.Exec(
			"DELETE FROM disabled_models WHERE provider_alias = ? AND model_id = ?",
			providerAlias, id,
		); err != nil {
			return fmt.Errorf("enable model %s/%s: %w", providerAlias, id, err)
		}
	}
	return tx.Commit()
}

// ListDisabledModels returns all disabled models as a map of providerAlias → []modelID.
func (s *Store) ListDisabledModels() (map[string][]string, error) {
	rows, err := s.db.Query(
		"SELECT provider_alias, model_id FROM disabled_models ORDER BY provider_alias, model_id",
	)
	if err != nil {
		return nil, fmt.Errorf("query disabled models: %w", err)
	}
	defer rows.Close()

	out := make(map[string][]string)
	for rows.Next() {
		var alias, id string
		if err := rows.Scan(&alias, &id); err != nil {
			return nil, fmt.Errorf("scan disabled model: %w", err)
		}
		out[alias] = append(out[alias], id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate disabled models: %w", err)
	}
	return out, nil
}

// IsDisabled reports whether the given model is disabled for the given provider alias.
func (s *Store) IsDisabled(providerAlias, modelID string) (bool, error) {
	var dummy int
	err := s.db.QueryRow(
		"SELECT 1 FROM disabled_models WHERE provider_alias = ? AND model_id = ?",
		providerAlias, modelID,
	).Scan(&dummy)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check disabled %s/%s: %w", providerAlias, modelID, err)
	}
	return true, nil
}
