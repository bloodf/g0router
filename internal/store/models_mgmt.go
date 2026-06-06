package store

import (
	"database/sql"
	"errors"
	"fmt"
)

var ErrDisabledModelExists = errors.New("disabled model already exists")
var ErrCustomModelExists = errors.New("custom model already exists")

type DisabledModel struct {
	ID        string
	Provider  string
	Model     string
	CreatedAt string
}

type CustomModel struct {
	ID          string
	Provider    string
	Model       string
	DisplayName string
	CreatedAt   string
}

func (s *Store) ListDisabledModels() ([]DisabledModel, error) {
	rows, err := s.db.Query(`SELECT id, provider, model, created_at FROM disabled_models ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("query disabled models: %w", err)
	}
	defer rows.Close()

	var models []DisabledModel
	for rows.Next() {
		var m DisabledModel
		if err := rows.Scan(&m.ID, &m.Provider, &m.Model, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan disabled model: %w", err)
		}
		models = append(models, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate disabled models: %w", err)
	}
	return models, nil
}

func (s *Store) CreateDisabledModel(provider, model string) (*DisabledModel, error) {
	var m DisabledModel
	err := s.db.QueryRow(
		`INSERT INTO disabled_models (provider, model) VALUES (?, ?) RETURNING id, provider, model, created_at`,
		provider, model,
	).Scan(&m.ID, &m.Provider, &m.Model, &m.CreatedAt)
	if err != nil {
		if isUniqueConstraintError(err, "disabled_models") {
			return nil, fmt.Errorf("create disabled model: %w", ErrDisabledModelExists)
		}
		return nil, fmt.Errorf("create disabled model: %w", err)
	}
	return &m, nil
}

func (s *Store) DeleteDisabledModel(provider, model string) error {
	result, err := s.db.Exec(`DELETE FROM disabled_models WHERE provider = ? AND model = ?`, provider, model)
	if err != nil {
		return fmt.Errorf("delete disabled model: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}
	return nil
}

func (s *Store) IsModelDisabled(provider, model string) (bool, error) {
	var id int64
	err := s.db.QueryRow(`SELECT id FROM disabled_models WHERE provider = ? AND model = ? LIMIT 1`, provider, model).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check model disabled: %w", err)
	}
	return true, nil
}

func (s *Store) ListCustomModels() ([]CustomModel, error) {
	rows, err := s.db.Query(`SELECT id, provider, model, display_name, created_at FROM custom_models ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("query custom models: %w", err)
	}
	defer rows.Close()

	var models []CustomModel
	for rows.Next() {
		var m CustomModel
		var displayName sql.NullString
		if err := rows.Scan(&m.ID, &m.Provider, &m.Model, &displayName, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan custom model: %w", err)
		}
		m.DisplayName = stringValueFromNull(displayName)
		models = append(models, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate custom models: %w", err)
	}
	return models, nil
}

func (s *Store) CreateCustomModel(provider, model, displayName string) (*CustomModel, error) {
	var m CustomModel
	var dn sql.NullString
	err := s.db.QueryRow(
		`INSERT INTO custom_models (provider, model, display_name) VALUES (?, ?, ?) RETURNING id, provider, model, display_name, created_at`,
		provider, model, emptyStringNil(displayName),
	).Scan(&m.ID, &m.Provider, &m.Model, &dn, &m.CreatedAt)
	if err != nil {
		if isUniqueConstraintError(err, "custom_models") {
			return nil, fmt.Errorf("create custom model: %w", ErrCustomModelExists)
		}
		return nil, fmt.Errorf("create custom model: %w", err)
	}
	m.DisplayName = stringValueFromNull(dn)
	return &m, nil
}

func (s *Store) GetCustomModel(id string) (*CustomModel, error) {
	var m CustomModel
	var displayName sql.NullString
	err := s.db.QueryRow(
		`SELECT id, provider, model, display_name, created_at FROM custom_models WHERE id = ?`,
		id,
	).Scan(&m.ID, &m.Provider, &m.Model, &displayName, &m.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get custom model: %w", err)
	}
	m.DisplayName = stringValueFromNull(displayName)
	return &m, nil
}

func (s *Store) DeleteCustomModel(id string) error {
	result, err := s.db.Exec(`DELETE FROM custom_models WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete custom model: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}
	return nil
}
