package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// PromptTemplate is a reusable system-prompt template with target models.
type PromptTemplate struct {
	ID           int64
	Name         string
	SystemPrompt string
	Models       []string
	IsActive     bool
	CreatedAt    string // ISO-8601 (RFC3339)
	UpdatedAt    string // ISO-8601 (RFC3339)
}

func marshalModels(models []string) (string, error) {
	if models == nil {
		models = []string{}
	}
	b, err := json.Marshal(models)
	if err != nil {
		return "", fmt.Errorf("marshal models: %w", err)
	}
	return string(b), nil
}

func unmarshalModels(raw string) ([]string, error) {
	var models []string
	if err := json.Unmarshal([]byte(raw), &models); err != nil {
		return nil, fmt.Errorf("unmarshal models: %w", err)
	}
	return models, nil
}

// CreatePromptTemplate inserts a new prompt template and returns it with its id.
func (s *Store) CreatePromptTemplate(in *PromptTemplate) (*PromptTemplate, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	modelsJSON, err := marshalModels(in.Models)
	if err != nil {
		return nil, err
	}
	res, err := s.db.Exec(
		"INSERT INTO prompt_templates (name, system_prompt, models_json, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		in.Name, in.SystemPrompt, modelsJSON, boolToInt(in.IsActive), now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert prompt template %s: %w", in.Name, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}
	return s.GetPromptTemplateByID(id)
}

// ListPromptTemplates returns all prompt templates ordered by id ascending.
func (s *Store) ListPromptTemplates() ([]*PromptTemplate, error) {
	rows, err := s.db.Query(
		"SELECT id, name, system_prompt, models_json, is_active, created_at, updated_at FROM prompt_templates ORDER BY id ASC",
	)
	if err != nil {
		return nil, fmt.Errorf("query prompt templates: %w", err)
	}
	defer rows.Close()

	var out []*PromptTemplate
	for rows.Next() {
		p, err := scanPromptTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate prompt templates: %w", err)
	}
	return out, nil
}

// GetPromptTemplateByID returns the prompt template with the given id.
func (s *Store) GetPromptTemplateByID(id int64) (*PromptTemplate, error) {
	return scanPromptTemplate(s.db.QueryRow(
		"SELECT id, name, system_prompt, models_json, is_active, created_at, updated_at FROM prompt_templates WHERE id = ?", id))
}

// UpdatePromptTemplate updates all mutable fields and bumps updated_at.
func (s *Store) UpdatePromptTemplate(id int64, in *PromptTemplate) (*PromptTemplate, error) {
	modelsJSON, err := marshalModels(in.Models)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := s.db.Exec(
		"UPDATE prompt_templates SET name = ?, system_prompt = ?, models_json = ?, is_active = ?, updated_at = ? WHERE id = ?",
		in.Name, in.SystemPrompt, modelsJSON, boolToInt(in.IsActive), now, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update prompt template %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return nil, ErrNotFound
	}
	return s.GetPromptTemplateByID(id)
}

// DeletePromptTemplate removes the prompt template with the given id.
func (s *Store) DeletePromptTemplate(id int64) error {
	res, err := s.db.Exec("DELETE FROM prompt_templates WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete prompt template %d: %w", id, err)
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

func scanPromptTemplate(row interface {
	Scan(dest ...any) error
}) (*PromptTemplate, error) {
	var p PromptTemplate
	var isActive int
	var modelsJSON string
	err := row.Scan(&p.ID, &p.Name, &p.SystemPrompt, &modelsJSON, &isActive, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan prompt template: %w", err)
	}
	p.IsActive = isActive != 0
	models, err := unmarshalModels(modelsJSON)
	if err != nil {
		return nil, err
	}
	p.Models = models
	return &p, nil
}
