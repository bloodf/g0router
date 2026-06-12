package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Combo is a named, ordered chain of model references.
type Combo struct {
	Name   string
	Models []string
}

// CreateCombo inserts a new combo. Returns an error if the name already exists.
func (s *Store) CreateCombo(c *Combo) error {
	raw, err := json.Marshal(c.Models)
	if err != nil {
		return fmt.Errorf("marshal combo models: %w", err)
	}
	now := time.Now().Unix()
	_, err = s.db.Exec(
		`INSERT INTO combos (name, models_json, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		c.Name, string(raw), now, now,
	)
	if err != nil {
		return fmt.Errorf("create combo %s: %w", c.Name, err)
	}
	return nil
}

// GetCombo returns the combo by name. Returns ErrNotFound if absent.
func (s *Store) GetCombo(name string) (*Combo, error) {
	var gotName, raw string
	err := s.db.QueryRow("SELECT name, models_json FROM combos WHERE name = ?", name).Scan(&gotName, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get combo %s: %w", name, err)
	}
	var models []string
	if err := json.Unmarshal([]byte(raw), &models); err != nil {
		return nil, fmt.Errorf("unmarshal combo models %s: %w", name, err)
	}
	return &Combo{Name: gotName, Models: models}, nil
}

// ListCombos returns all combos ordered by name.
func (s *Store) ListCombos() ([]*Combo, error) {
	rows, err := s.db.Query("SELECT name, models_json FROM combos ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("list combos: %w", err)
	}
	defer rows.Close()

	var out []*Combo
	for rows.Next() {
		var name, raw string
		if err := rows.Scan(&name, &raw); err != nil {
			return nil, fmt.Errorf("scan combo: %w", err)
		}
		var models []string
		if err := json.Unmarshal([]byte(raw), &models); err != nil {
			return nil, fmt.Errorf("unmarshal combo models %s: %w", name, err)
		}
		out = append(out, &Combo{Name: name, Models: models})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate combos: %w", err)
	}
	return out, nil
}

// UpdateCombo replaces the models list for an existing combo. Returns ErrNotFound if absent.
func (s *Store) UpdateCombo(name string, models []string) error {
	raw, err := json.Marshal(models)
	if err != nil {
		return fmt.Errorf("marshal combo models: %w", err)
	}
	res, err := s.db.Exec(
		"UPDATE combos SET models_json = ?, updated_at = ? WHERE name = ?",
		string(raw), time.Now().Unix(), name,
	)
	if err != nil {
		return fmt.Errorf("update combo %s: %w", name, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteCombo removes the combo by name. Returns ErrNotFound if absent.
func (s *Store) DeleteCombo(name string) error {
	res, err := s.db.Exec("DELETE FROM combos WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("delete combo %s: %w", name, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListComboNames returns all combo names in alphabetical order.
// Used by the /v1/models handler (api layer) without importing store types.
func (s *Store) ListComboNames() ([]string, error) {
	combos, err := s.ListCombos()
	if err != nil {
		return nil, err
	}
	names := make([]string, len(combos))
	for i, c := range combos {
		names[i] = c.Name
	}
	return names, nil
}
