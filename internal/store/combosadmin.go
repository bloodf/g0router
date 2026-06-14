package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ComboStep is a single provider/model step in an admin combo.
type ComboStep struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

// ComboAdmin is an id-keyed admin combo mirroring the dashboard UI shape
// {id, name, strategy, steps:[{provider,model}], is_active}. It is separate from
// the engine Combo ({name, models[]}) in combos.go: the admin surface persists
// structured steps and is keyed by id, while the engine table + /v1/models
// lister stay intact (fed by a best-effort mirror-write from the admin handler).
type ComboAdmin struct {
	ID        string
	Name      string
	Strategy  string
	Steps     []ComboStep
	IsActive  bool
	CreatedAt int64
	UpdatedAt int64
}

func marshalComboSteps(steps []ComboStep) (string, error) {
	if steps == nil {
		steps = []ComboStep{}
	}
	b, err := json.Marshal(steps)
	if err != nil {
		return "", fmt.Errorf("marshal combo steps: %w", err)
	}
	return string(b), nil
}

// CreateComboAdmin inserts a new admin combo with a generated id.
func (s *Store) CreateComboAdmin(in *ComboAdmin) (*ComboAdmin, error) {
	now := time.Now().Unix()
	id, err := newID()
	if err != nil {
		return nil, err
	}
	strategy := in.Strategy
	if strategy == "" {
		strategy = "fallback"
	}
	stepsJSON, err := marshalComboSteps(in.Steps)
	if err != nil {
		return nil, err
	}
	steps := in.Steps
	if steps == nil {
		steps = []ComboStep{}
	}
	rec := &ComboAdmin{
		ID:        id,
		Name:      in.Name,
		Strategy:  strategy,
		Steps:     steps,
		IsActive:  in.IsActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err = s.db.Exec(
		"INSERT INTO combos_admin (id, name, strategy, steps_json, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		rec.ID, rec.Name, rec.Strategy, stepsJSON, boolToInt(rec.IsActive), rec.CreatedAt, rec.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert combo admin %s: %w", rec.Name, err)
	}
	return rec, nil
}

// ListComboAdmins returns all admin combos ordered by creation time.
func (s *Store) ListComboAdmins() ([]*ComboAdmin, error) {
	rows, err := s.db.Query(
		"SELECT id, name, strategy, steps_json, is_active, created_at, updated_at FROM combos_admin ORDER BY created_at, id")
	if err != nil {
		return nil, fmt.Errorf("query combo admins: %w", err)
	}
	defer rows.Close()

	var out []*ComboAdmin
	for rows.Next() {
		rec, err := scanComboAdmin(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate combo admins: %w", err)
	}
	return out, nil
}

// GetComboAdminByID returns the admin combo with the given id.
func (s *Store) GetComboAdminByID(id string) (*ComboAdmin, error) {
	return scanComboAdmin(s.db.QueryRow(
		"SELECT id, name, strategy, steps_json, is_active, created_at, updated_at FROM combos_admin WHERE id = ?", id))
}

// UpdateComboAdmin persists the mutable fields of the admin combo.
func (s *Store) UpdateComboAdmin(in *ComboAdmin) error {
	stepsJSON, err := marshalComboSteps(in.Steps)
	if err != nil {
		return err
	}
	strategy := in.Strategy
	if strategy == "" {
		strategy = "fallback"
	}
	res, err := s.db.Exec(
		"UPDATE combos_admin SET name = ?, strategy = ?, steps_json = ?, is_active = ?, updated_at = ? WHERE id = ?",
		in.Name, strategy, stepsJSON, boolToInt(in.IsActive), time.Now().Unix(), in.ID,
	)
	if err != nil {
		return fmt.Errorf("update combo admin %s: %w", in.ID, err)
	}
	return requireRowAffected(res)
}

// DeleteComboAdmin removes the admin combo with the given id.
func (s *Store) DeleteComboAdmin(id string) error {
	res, err := s.db.Exec("DELETE FROM combos_admin WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete combo admin %s: %w", id, err)
	}
	return requireRowAffected(res)
}

func scanComboAdmin(row rowScanner) (*ComboAdmin, error) {
	var rec ComboAdmin
	var isActive int
	var stepsJSON string
	err := row.Scan(&rec.ID, &rec.Name, &rec.Strategy, &stepsJSON, &isActive, &rec.CreatedAt, &rec.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan combo admin: %w", err)
	}
	rec.IsActive = isActive != 0
	if err := json.Unmarshal([]byte(stepsJSON), &rec.Steps); err != nil {
		return nil, fmt.Errorf("unmarshal combo steps: %w", err)
	}
	if rec.Steps == nil {
		rec.Steps = []ComboStep{}
	}
	return &rec, nil
}
