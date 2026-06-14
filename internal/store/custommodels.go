package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// CustomModel is a user-defined model entry managed via the dashboard. Config
// carries misc metadata (cost/context) as a JSON blob; no secret fields.
type CustomModel struct {
	ID        string
	Provider  string
	ModelID   string
	Name      string
	Config    map[string]any
	CreatedAt int64
}

func marshalCustomModelConfig(cfg map[string]any) (string, error) {
	if cfg == nil {
		return "{}", nil
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshal custom model config: %w", err)
	}
	return string(b), nil
}

func unmarshalCustomModelConfig(raw string, cm *CustomModel) error {
	if raw == "" {
		cm.Config = map[string]any{}
		return nil
	}
	var cfg map[string]any
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return fmt.Errorf("unmarshal custom model config: %w", err)
	}
	cm.Config = cfg
	return nil
}

// CreateCustomModel inserts a new custom-model row and returns it with a
// generated id and creation timestamp.
func (s *Store) CreateCustomModel(in *CustomModel) (*CustomModel, error) {
	id, err := newID()
	if err != nil {
		return nil, err
	}
	cfgJSON, err := marshalCustomModelConfig(in.Config)
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()

	cm := &CustomModel{
		ID:        id,
		Provider:  in.Provider,
		ModelID:   in.ModelID,
		Name:      in.Name,
		Config:    in.Config,
		CreatedAt: now,
	}
	if cm.Config == nil {
		cm.Config = map[string]any{}
	}

	_, err = s.db.Exec(
		"INSERT INTO custom_models (id, provider, model_id, name, config_json, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		cm.ID, cm.Provider, cm.ModelID, cm.Name, cfgJSON, cm.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert custom model %s: %w", cm.ModelID, err)
	}
	return cm, nil
}

// ListCustomModels returns all custom models ordered by creation time (newest first).
func (s *Store) ListCustomModels() ([]*CustomModel, error) {
	rows, err := s.db.Query(
		"SELECT id, provider, model_id, name, config_json, created_at FROM custom_models ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("query custom models: %w", err)
	}
	defer rows.Close()

	var out []*CustomModel
	for rows.Next() {
		cm, err := scanCustomModel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, cm)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate custom models: %w", err)
	}
	return out, nil
}

// DeleteCustomModel removes the custom model with the given id.
func (s *Store) DeleteCustomModel(id string) error {
	res, err := s.db.Exec("DELETE FROM custom_models WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete custom model %s: %w", id, err)
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

func scanCustomModel(row interface {
	Scan(dest ...any) error
}) (*CustomModel, error) {
	var cm CustomModel
	var cfgJSON string
	err := row.Scan(&cm.ID, &cm.Provider, &cm.ModelID, &cm.Name, &cfgJSON, &cm.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan custom model: %w", err)
	}
	if err := unmarshalCustomModelConfig(cfgJSON, &cm); err != nil {
		return nil, err
	}
	return &cm, nil
}
