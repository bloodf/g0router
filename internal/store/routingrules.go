package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// RoutingRule is an admin-defined routing rule mirroring the dashboard UI shape.
// Rules are admin CRUD only and are not yet applied to live inference traffic
// (tracked follow-up).
type RoutingRule struct {
	ID             string
	Name           string
	Priority       int
	CondField      string
	CondOperator   string
	CondValue      string
	TargetProvider string
	IsActive       bool
	CreatedAt      int64
	UpdatedAt      int64
}

// CreateRoutingRule inserts a new routing rule with a generated id.
func (s *Store) CreateRoutingRule(in *RoutingRule) (*RoutingRule, error) {
	now := time.Now().Unix()
	id, err := newID()
	if err != nil {
		return nil, err
	}
	rule := &RoutingRule{
		ID:             id,
		Name:           in.Name,
		Priority:       in.Priority,
		CondField:      in.CondField,
		CondOperator:   in.CondOperator,
		CondValue:      in.CondValue,
		TargetProvider: in.TargetProvider,
		IsActive:       in.IsActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	_, err = s.db.Exec(
		`INSERT INTO routing_rules
		 (id, name, priority, cond_field, cond_operator, cond_value, target_provider, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.ID, rule.Name, rule.Priority, rule.CondField, rule.CondOperator, rule.CondValue,
		rule.TargetProvider, boolToInt(rule.IsActive), rule.CreatedAt, rule.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert routing rule %s: %w", rule.Name, err)
	}
	return rule, nil
}

// ListRoutingRules returns all routing rules ordered by priority then creation.
func (s *Store) ListRoutingRules() ([]*RoutingRule, error) {
	rows, err := s.db.Query(
		`SELECT id, name, priority, cond_field, cond_operator, cond_value, target_provider, is_active, created_at, updated_at
		 FROM routing_rules ORDER BY priority ASC, created_at ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("query routing rules: %w", err)
	}
	defer rows.Close()

	var out []*RoutingRule
	for rows.Next() {
		rule, err := scanRoutingRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate routing rules: %w", err)
	}
	return out, nil
}

// GetRoutingRuleByID returns the routing rule with the given id.
func (s *Store) GetRoutingRuleByID(id string) (*RoutingRule, error) {
	return scanRoutingRule(s.db.QueryRow(
		`SELECT id, name, priority, cond_field, cond_operator, cond_value, target_provider, is_active, created_at, updated_at
		 FROM routing_rules WHERE id = ?`, id))
}

// UpdateRoutingRule persists the mutable fields of the routing rule.
func (s *Store) UpdateRoutingRule(in *RoutingRule) error {
	res, err := s.db.Exec(
		`UPDATE routing_rules SET name = ?, priority = ?, cond_field = ?, cond_operator = ?, cond_value = ?,
		 target_provider = ?, is_active = ?, updated_at = ? WHERE id = ?`,
		in.Name, in.Priority, in.CondField, in.CondOperator, in.CondValue,
		in.TargetProvider, boolToInt(in.IsActive), time.Now().Unix(), in.ID,
	)
	if err != nil {
		return fmt.Errorf("update routing rule %s: %w", in.ID, err)
	}
	return requireRowAffected(res)
}

// DeleteRoutingRule removes the routing rule with the given id.
func (s *Store) DeleteRoutingRule(id string) error {
	res, err := s.db.Exec("DELETE FROM routing_rules WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete routing rule %s: %w", id, err)
	}
	return requireRowAffected(res)
}

func scanRoutingRule(row rowScanner) (*RoutingRule, error) {
	var rule RoutingRule
	var isActive int
	err := row.Scan(&rule.ID, &rule.Name, &rule.Priority, &rule.CondField, &rule.CondOperator,
		&rule.CondValue, &rule.TargetProvider, &isActive, &rule.CreatedAt, &rule.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan routing rule: %w", err)
	}
	rule.IsActive = isActive != 0
	return &rule, nil
}
