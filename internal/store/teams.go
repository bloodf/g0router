package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Team is a governance grouping with a shared budget and rate limit.
type Team struct {
	ID            string
	Name          string
	BudgetUSD     float64
	BudgetUsedUSD float64
	BudgetPeriod  string
	RateLimitRPM  int
	CreatedAt     int64
	UpdatedAt     int64
}

// CreateTeam inserts a new team and returns it with its generated ID.
func (s *Store) CreateTeam(in *Team) (*Team, error) {
	now := time.Now().Unix()
	id, err := newID()
	if err != nil {
		return nil, err
	}
	period := in.BudgetPeriod
	if period == "" {
		period = "monthly"
	}
	t := &Team{
		ID:            id,
		Name:          in.Name,
		BudgetUSD:     in.BudgetUSD,
		BudgetUsedUSD: in.BudgetUsedUSD,
		BudgetPeriod:  period,
		RateLimitRPM:  in.RateLimitRPM,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	_, err = s.db.Exec(
		"INSERT INTO teams (id, name, budget_usd, budget_used_usd, budget_period, rate_limit_rpm, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		t.ID, t.Name, t.BudgetUSD, t.BudgetUsedUSD, t.BudgetPeriod, t.RateLimitRPM, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert team %s: %w", t.Name, err)
	}
	return t, nil
}

// ListTeams returns all teams ordered by creation time (newest first).
func (s *Store) ListTeams() ([]*Team, error) {
	rows, err := s.db.Query(
		"SELECT id, name, budget_usd, budget_used_usd, budget_period, rate_limit_rpm, created_at, updated_at FROM teams ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("query teams: %w", err)
	}
	defer rows.Close()

	var out []*Team
	for rows.Next() {
		t, err := scanTeam(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate teams: %w", err)
	}
	return out, nil
}

// GetTeamByID returns the team with the given id.
func (s *Store) GetTeamByID(id string) (*Team, error) {
	return scanTeam(s.db.QueryRow(
		"SELECT id, name, budget_usd, budget_used_usd, budget_period, rate_limit_rpm, created_at, updated_at FROM teams WHERE id = ?", id))
}

// UpdateTeam updates all mutable fields of the team with the given id.
func (s *Store) UpdateTeam(in *Team) error {
	period := in.BudgetPeriod
	if period == "" {
		period = "monthly"
	}
	res, err := s.db.Exec(
		"UPDATE teams SET name = ?, budget_usd = ?, budget_used_usd = ?, budget_period = ?, rate_limit_rpm = ?, updated_at = ? WHERE id = ?",
		in.Name, in.BudgetUSD, in.BudgetUsedUSD, period, in.RateLimitRPM, time.Now().Unix(), in.ID,
	)
	if err != nil {
		return fmt.Errorf("update team %s: %w", in.ID, err)
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

// DeleteTeam removes the team with the given id.
func (s *Store) DeleteTeam(id string) error {
	res, err := s.db.Exec("DELETE FROM teams WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete team %s: %w", id, err)
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

func scanTeam(row interface {
	Scan(dest ...any) error
}) (*Team, error) {
	var t Team
	err := row.Scan(&t.ID, &t.Name, &t.BudgetUSD, &t.BudgetUsedUSD, &t.BudgetPeriod, &t.RateLimitRPM, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan team: %w", err)
	}
	return &t, nil
}
