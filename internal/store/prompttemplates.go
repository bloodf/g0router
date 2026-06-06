package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// PromptTemplate holds a reusable system prompt tied to one or more models.
type PromptTemplate struct {
	ID           int64
	Name         string
	SystemPrompt string
	Models       []string
	IsActive     bool
	CreatedAt    string
	UpdatedAt    string
}

// ListPromptTemplates returns all prompt templates ordered by name.
func (s *Store) ListPromptTemplates() ([]PromptTemplate, error) {
	rows, err := s.db.Query(`SELECT id, name, system_prompt, models_json, is_active, created_at, updated_at FROM prompt_templates ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("query prompt templates: %w", err)
	}
	defer rows.Close()

	var templates []PromptTemplate
	for rows.Next() {
		t, err := scanPromptTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate prompt templates: %w", err)
	}
	return templates, nil
}

// GetPromptTemplate returns a single prompt template by ID.
func (s *Store) GetPromptTemplate(id int64) (*PromptTemplate, error) {
	t, err := scanPromptTemplate(s.db.QueryRow(
		`SELECT id, name, system_prompt, models_json, is_active, created_at, updated_at FROM prompt_templates WHERE id = ?`,
		id,
	))
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// CreatePromptTemplate inserts a new prompt template.
func (s *Store) CreatePromptTemplate(name, systemPrompt string, models []string, isActive bool) (*PromptTemplate, error) {
	modelsJSON, err := json.Marshal(models)
	if err != nil {
		return nil, fmt.Errorf("marshal models: %w", err)
	}

	var t PromptTemplate
	var modelsRaw string
	err = s.db.QueryRow(
		`INSERT INTO prompt_templates (name, system_prompt, models_json, is_active) VALUES (?, ?, ?, ?)
		RETURNING id, name, system_prompt, models_json, is_active, created_at, updated_at`,
		name, systemPrompt, string(modelsJSON), boolInt(isActive),
	).Scan(&t.ID, &t.Name, &t.SystemPrompt, &modelsRaw, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if isUniqueConstraintError(err, "prompt_templates") {
			return nil, fmt.Errorf("create prompt template: %w", ErrDuplicateName)
		}
		return nil, fmt.Errorf("create prompt template: %w", err)
	}
	t.Models = unmarshalModelsJSON(modelsRaw)
	return &t, nil
}

// UpdatePromptTemplate modifies an existing prompt template by ID.
func (s *Store) UpdatePromptTemplate(id int64, name, systemPrompt string, models []string, isActive bool) error {
	modelsJSON, err := json.Marshal(models)
	if err != nil {
		return fmt.Errorf("marshal models: %w", err)
	}

	result, err := s.db.Exec(
		`UPDATE prompt_templates SET name = ?, system_prompt = ?, models_json = ?, is_active = ?, updated_at = datetime('now') WHERE id = ?`,
		name, systemPrompt, string(modelsJSON), boolInt(isActive), id,
	)
	if err != nil {
		if isUniqueConstraintError(err, "prompt_templates") {
			return fmt.Errorf("update prompt template: %w", ErrDuplicateName)
		}
		return fmt.Errorf("update prompt template: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}
	return nil
}

// DeletePromptTemplate removes a prompt template by ID.
func (s *Store) DeletePromptTemplate(id int64) error {
	result, err := s.db.Exec(`DELETE FROM prompt_templates WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete prompt template: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}
	return nil
}

func scanPromptTemplate(scanner interface {
	Scan(dest ...any) error
}) (PromptTemplate, error) {
	var t PromptTemplate
	var modelsRaw string
	var isActive int
	err := scanner.Scan(&t.ID, &t.Name, &t.SystemPrompt, &modelsRaw, &isActive, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return PromptTemplate{}, ErrNotFound
	}
	if err != nil {
		return PromptTemplate{}, fmt.Errorf("scan prompt template: %w", err)
	}
	t.IsActive = isActive == 1
	t.Models = unmarshalModelsJSON(modelsRaw)
	return t, nil
}

func unmarshalModelsJSON(raw string) []string {
	var models []string
	_ = json.Unmarshal([]byte(raw), &models)
	return models
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
