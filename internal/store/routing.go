package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// RoutingRule defines a conditional rewrite rule evaluated before alias/combo
// resolution in the proxy engine.
type RoutingRule struct {
	ID             int64
	Name           string
	Priority       int
	CondField      string
	CondOperator   string
	CondValue      string
	TargetProvider string
	TargetModel    *string
	IsActive       bool
	CreatedAt      time.Time
}

// ModelLimit defines per-model usage constraints checked at dispatch time.
type ModelLimit struct {
	ID            int64
	Model         string
	MaxTokens     *int
	MaxRPM        *int
	AllowedKeyIDs []string
	CreatedAt     time.Time
}

// --- RoutingRule store ---

func (s *Store) CreateRoutingRule(name string, priority int, condField, condOperator, condValue, targetProvider string, targetModel *string) (*RoutingRule, error) {
	res, err := s.db.Exec(
		`INSERT INTO routing_rules (name, priority, cond_field, cond_operator, cond_value, target_provider, target_model)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		name, priority, condField, condOperator, condValue, targetProvider, targetModel,
	)
	if err != nil {
		return nil, fmt.Errorf("create routing rule: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("create routing rule id: %w", err)
	}
	return s.GetRoutingRule(id)
}

func (s *Store) GetRoutingRule(id int64) (*RoutingRule, error) {
	row := s.db.QueryRow(
		`SELECT id, name, priority, cond_field, cond_operator, cond_value, target_provider, target_model, is_active, created_at
		FROM routing_rules WHERE id = ?`,
		id,
	)
	rule, err := scanRoutingRule(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("get routing rule: %w", ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get routing rule: %w", err)
	}
	return rule, nil
}

func (s *Store) ListRoutingRules() ([]RoutingRule, error) {
	rows, err := s.db.Query(
		`SELECT id, name, priority, cond_field, cond_operator, cond_value, target_provider, target_model, is_active, created_at
		FROM routing_rules ORDER BY priority DESC, created_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("list routing rules: %w", err)
	}
	defer rows.Close()

	var rules []RoutingRule
	for rows.Next() {
		rule, err := scanRoutingRule(rows)
		if err != nil {
			return nil, fmt.Errorf("scan routing rule: %w", err)
		}
		rules = append(rules, *rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate routing rules: %w", err)
	}
	return rules, nil
}

func (s *Store) UpdateRoutingRule(id int64, name string, priority int, condField, condOperator, condValue, targetProvider string, targetModel *string, isActive bool) error {
	res, err := s.db.Exec(
		`UPDATE routing_rules SET name = ?, priority = ?, cond_field = ?, cond_operator = ?, cond_value = ?, target_provider = ?, target_model = ?, is_active = ? WHERE id = ?`,
		name, priority, condField, condOperator, condValue, targetProvider, targetModel, isActive, id,
	)
	if err != nil {
		return fmt.Errorf("update routing rule: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update routing rule rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("update routing rule: %w", ErrNotFound)
	}
	return nil
}

func (s *Store) DeleteRoutingRule(id int64) error {
	if _, err := s.db.Exec(`DELETE FROM routing_rules WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete routing rule: %w", err)
	}
	return nil
}

func scanRoutingRule(scanner interface {
	Scan(dest ...any) error
}) (*RoutingRule, error) {
	var rule RoutingRule
	var targetModel sql.NullString
	var isActive int
	var createdAt string

	err := scanner.Scan(
		&rule.ID, &rule.Name, &rule.Priority, &rule.CondField, &rule.CondOperator,
		&rule.CondValue, &rule.TargetProvider, &targetModel, &isActive, &createdAt,
	)
	if err != nil {
		return nil, err
	}
	if targetModel.Valid && targetModel.String != "" {
		rule.TargetModel = &targetModel.String
	}
	rule.IsActive = isActive != 0
	rule.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse routing rule created_at: %w", err)
	}
	return &rule, nil
}

// --- ModelLimit store ---

func (s *Store) CreateModelLimit(model string, maxTokens, maxRPM *int, allowedKeyIDs []string) (*ModelLimit, error) {
	allowedJSON, err := json.Marshal(allowedKeyIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal allowed_key_ids: %w", err)
	}
	res, err := s.db.Exec(
		`INSERT INTO model_limits (model, max_tokens, max_rpm, allowed_key_ids)
		VALUES (?, ?, ?, ?)`,
		model, maxTokens, maxRPM, string(allowedJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("create model limit: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("create model limit id: %w", err)
	}
	return s.GetModelLimit(id)
}

func (s *Store) GetModelLimit(id int64) (*ModelLimit, error) {
	row := s.db.QueryRow(
		`SELECT id, model, max_tokens, max_rpm, allowed_key_ids, created_at
		FROM model_limits WHERE id = ?`,
		id,
	)
	limit, err := scanModelLimit(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("get model limit: %w", ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get model limit: %w", err)
	}
	return limit, nil
}

func (s *Store) GetModelLimitByModel(model string) (*ModelLimit, error) {
	row := s.db.QueryRow(
		`SELECT id, model, max_tokens, max_rpm, allowed_key_ids, created_at
		FROM model_limits WHERE model = ?`,
		model,
	)
	limit, err := scanModelLimit(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("get model limit by model: %w", ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get model limit by model: %w", err)
	}
	return limit, nil
}

func (s *Store) ListModelLimits() ([]ModelLimit, error) {
	rows, err := s.db.Query(
		`SELECT id, model, max_tokens, max_rpm, allowed_key_ids, created_at
		FROM model_limits ORDER BY created_at, model`,
	)
	if err != nil {
		return nil, fmt.Errorf("list model limits: %w", err)
	}
	defer rows.Close()

	var limits []ModelLimit
	for rows.Next() {
		limit, err := scanModelLimit(rows)
		if err != nil {
			return nil, fmt.Errorf("scan model limit: %w", err)
		}
		limits = append(limits, *limit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate model limits: %w", err)
	}
	return limits, nil
}

func (s *Store) UpdateModelLimit(id int64, model string, maxTokens, maxRPM *int, allowedKeyIDs []string) error {
	allowedJSON, err := json.Marshal(allowedKeyIDs)
	if err != nil {
		return fmt.Errorf("marshal allowed_key_ids: %w", err)
	}
	res, err := s.db.Exec(
		`UPDATE model_limits SET model = ?, max_tokens = ?, max_rpm = ?, allowed_key_ids = ? WHERE id = ?`,
		model, maxTokens, maxRPM, string(allowedJSON), id,
	)
	if err != nil {
		return fmt.Errorf("update model limit: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update model limit rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("update model limit: %w", ErrNotFound)
	}
	return nil
}

func (s *Store) DeleteModelLimit(id int64) error {
	if _, err := s.db.Exec(`DELETE FROM model_limits WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete model limit: %w", err)
	}
	return nil
}

func scanModelLimit(scanner interface {
	Scan(dest ...any) error
}) (*ModelLimit, error) {
	var limit ModelLimit
	var maxTokens, maxRPM sql.NullInt64
	var allowedJSON sql.NullString
	var createdAt string

	err := scanner.Scan(
		&limit.ID, &limit.Model, &maxTokens, &maxRPM, &allowedJSON, &createdAt,
	)
	if err != nil {
		return nil, err
	}
	if maxTokens.Valid {
		v := int(maxTokens.Int64)
		limit.MaxTokens = &v
	}
	if maxRPM.Valid {
		v := int(maxRPM.Int64)
		limit.MaxRPM = &v
	}
	if allowedJSON.Valid && allowedJSON.String != "" {
		_ = json.Unmarshal([]byte(allowedJSON.String), &limit.AllowedKeyIDs)
	}
	limit.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse model limit created_at: %w", err)
	}
	return &limit, nil
}
