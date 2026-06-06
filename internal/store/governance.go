package store

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

type Team struct {
	ID            int64
	Name          string
	BudgetUSD     *float64
	BudgetPeriod  string
	BudgetUsedUSD float64
	BudgetResetAt *time.Time
	RateLimitRPM  *int
	CreatedAt     time.Time
}

type VirtualKey struct {
	ID            int64
	Name          string
	KeyPrefix     string
	KeyHash       string
	BudgetUSD     *float64
	BudgetPeriod  string
	BudgetUsedUSD float64
	BudgetResetAt *time.Time
	RateLimitRPM  *int
	RateLimitTPM  *int
	TeamID        *int64
	IsActive      bool
	MCPToolGroup  string
	CreatedAt     time.Time
}

func (s *Store) CreateTeam(name string, budgetUSD *float64, budgetPeriod string, rateLimitRPM *int) (*Team, error) {
	if budgetPeriod == "" {
		budgetPeriod = "monthly"
	}
	res, err := s.db.Exec(
		`INSERT INTO teams (name, budget_usd, budget_period, rate_limit_rpm)
		VALUES (?, ?, ?, ?)`,
		name, budgetUSD, budgetPeriod, rateLimitRPM,
	)
	if err != nil {
		return nil, fmt.Errorf("create team: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("create team id: %w", err)
	}
	return s.GetTeam(id)
}

func (s *Store) GetTeam(id int64) (*Team, error) {
	row := s.db.QueryRow(
		`SELECT id, name, budget_usd, budget_period, budget_used_usd, budget_reset_at, rate_limit_rpm, created_at
		FROM teams WHERE id = ?`,
		id,
	)
	team, err := scanTeam(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("get team: %w", ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get team: %w", err)
	}
	return team, nil
}

func (s *Store) ListTeams() ([]Team, error) {
	rows, err := s.db.Query(
		`SELECT id, name, budget_usd, budget_period, budget_used_usd, budget_reset_at, rate_limit_rpm, created_at
		FROM teams ORDER BY created_at, name`,
	)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	defer rows.Close()

	var teams []Team
	for rows.Next() {
		team, err := scanTeam(rows)
		if err != nil {
			return nil, fmt.Errorf("scan team: %w", err)
		}
		teams = append(teams, *team)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate teams: %w", err)
	}
	return teams, nil
}

func (s *Store) UpdateTeam(id int64, name string, budgetUSD *float64, budgetPeriod string, rateLimitRPM *int) error {
	res, err := s.db.Exec(
		`UPDATE teams SET name = ?, budget_usd = ?, budget_period = ?, rate_limit_rpm = ? WHERE id = ?`,
		name, budgetUSD, budgetPeriod, rateLimitRPM, id,
	)
	if err != nil {
		return fmt.Errorf("update team: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update team rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("update team: %w", ErrNotFound)
	}
	return nil
}

func (s *Store) DeleteTeam(id int64) error {
	if _, err := s.db.Exec(`DELETE FROM teams WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete team: %w", err)
	}
	return nil
}

func (s *Store) AddTeamBudgetUsed(id int64, deltaUSD float64) error {
	if deltaUSD <= 0 {
		return nil
	}
	res, err := s.db.Exec(
		`UPDATE teams SET budget_used_usd = budget_used_usd + ? WHERE id = ?`,
		deltaUSD, id,
	)
	if err != nil {
		return fmt.Errorf("add team budget used: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("add team budget used rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("add team budget used: %d not found", id)
	}
	return nil
}

func (s *Store) ResetTeamBudget(id int64, resetAt time.Time) error {
	res, err := s.db.Exec(
		`UPDATE teams SET budget_used_usd = 0, budget_reset_at = ? WHERE id = ?`,
		resetAt.Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("reset team budget: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("reset team budget rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("reset team budget: %d not found", id)
	}
	return nil
}

func scanTeam(scanner interface {
	Scan(dest ...any) error
}) (*Team, error) {
	var team Team
	var budgetUSD sql.NullFloat64
	var budgetResetAt sql.NullString
	var rateLimitRPM sql.NullInt64
	var createdAt string

	err := scanner.Scan(
		&team.ID, &team.Name, &budgetUSD, &team.BudgetPeriod,
		&team.BudgetUsedUSD, &budgetResetAt, &rateLimitRPM, &createdAt,
	)
	if err != nil {
		return nil, err
	}
	if budgetUSD.Valid {
		team.BudgetUSD = &budgetUSD.Float64
	}
	if budgetResetAt.Valid && budgetResetAt.String != "" {
		parsed, err := time.Parse(time.RFC3339, budgetResetAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse team budget_reset_at: %w", err)
		}
		team.BudgetResetAt = &parsed
	}
	if rateLimitRPM.Valid {
		v := int(rateLimitRPM.Int64)
		team.RateLimitRPM = &v
	}
	team.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse team created_at: %w", err)
	}
	return &team, nil
}

func (s *Store) CreateVirtualKey(name string, teamID *int64, budgetUSD *float64, budgetPeriod string, rateLimitRPM, rateLimitTPM *int, mcpToolGroup string) (*VirtualKey, string, error) {
	raw, err := generateVirtualKey()
	if err != nil {
		return nil, "", err
	}
	if budgetPeriod == "" {
		budgetPeriod = "monthly"
	}
	keyHash := hashVirtualKey(raw)
	prefix := raw[:8]

	res, err := s.db.Exec(
		`INSERT INTO virtual_keys (name, key_prefix, key_hash, budget_usd, budget_period, rate_limit_rpm, rate_limit_tpm, team_id, mcp_tool_group)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		name, prefix, keyHash, budgetUSD, budgetPeriod, rateLimitRPM, rateLimitTPM, teamID, mcpToolGroup,
	)
	if err != nil {
		return nil, "", fmt.Errorf("create virtual key: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, "", fmt.Errorf("create virtual key id: %w", err)
	}
	key, err := s.GetVirtualKey(id)
	if err != nil {
		return nil, "", fmt.Errorf("reload virtual key: %w", err)
	}
	return key, raw, nil
}

func (s *Store) GetVirtualKey(id int64) (*VirtualKey, error) {
	row := s.db.QueryRow(
		`SELECT id, name, key_prefix, key_hash, budget_usd, budget_period, budget_used_usd, budget_reset_at, rate_limit_rpm, rate_limit_tpm, team_id, is_active, mcp_tool_group, created_at
		FROM virtual_keys WHERE id = ?`,
		id,
	)
	key, err := scanVirtualKey(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("get virtual key: %w", ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get virtual key: %w", err)
	}
	return key, nil
}

func (s *Store) ListVirtualKeys() ([]VirtualKey, error) {
	rows, err := s.db.Query(
		`SELECT id, name, key_prefix, budget_usd, budget_period, budget_used_usd, budget_reset_at, rate_limit_rpm, rate_limit_tpm, team_id, is_active, mcp_tool_group, created_at
		FROM virtual_keys ORDER BY created_at, name`,
	)
	if err != nil {
		return nil, fmt.Errorf("list virtual keys: %w", err)
	}
	defer rows.Close()

	var keys []VirtualKey
	for rows.Next() {
		key, err := scanVirtualKeyList(rows)
		if err != nil {
			return nil, fmt.Errorf("scan virtual key: %w", err)
		}
		keys = append(keys, *key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate virtual keys: %w", err)
	}
	return keys, nil
}

func (s *Store) ValidateVirtualKey(raw string) (*VirtualKey, bool, error) {
	keyHash := hashVirtualKey(raw)

	row := s.db.QueryRow(
		`SELECT id, name, key_prefix, key_hash, budget_usd, budget_period, budget_used_usd, budget_reset_at, rate_limit_rpm, rate_limit_tpm, team_id, is_active, mcp_tool_group, created_at
		FROM virtual_keys
		WHERE key_hash = ?`,
		keyHash,
	)
	key, err := scanVirtualKey(row)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("validate virtual key: %w", err)
	}
	return key, true, nil
}

func (s *Store) UpdateVirtualKey(id int64, name string, teamID *int64, budgetUSD *float64, budgetPeriod string, rateLimitRPM, rateLimitTPM *int, isActive bool, mcpToolGroup string) error {
	res, err := s.db.Exec(
		`UPDATE virtual_keys SET name = ?, team_id = ?, budget_usd = ?, budget_period = ?, rate_limit_rpm = ?, rate_limit_tpm = ?, is_active = ?, mcp_tool_group = ? WHERE id = ?`,
		name, teamID, budgetUSD, budgetPeriod, rateLimitRPM, rateLimitTPM, isActive, mcpToolGroup, id,
	)
	if err != nil {
		return fmt.Errorf("update virtual key: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update virtual key rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("update virtual key: %w", ErrNotFound)
	}
	return nil
}

func (s *Store) DeleteVirtualKey(id int64) error {
	if _, err := s.db.Exec(`DELETE FROM virtual_keys WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete virtual key: %w", err)
	}
	return nil
}

func (s *Store) AddVirtualKeyBudgetUsed(id int64, deltaUSD float64) error {
	if deltaUSD <= 0 {
		return nil
	}
	res, err := s.db.Exec(
		`UPDATE virtual_keys SET budget_used_usd = budget_used_usd + ? WHERE id = ?`,
		deltaUSD, id,
	)
	if err != nil {
		return fmt.Errorf("add virtual key budget used: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("add virtual key budget used rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("add virtual key budget used: %d not found", id)
	}
	return nil
}

func (s *Store) ResetVirtualKeyBudget(id int64, resetAt time.Time) error {
	res, err := s.db.Exec(
		`UPDATE virtual_keys SET budget_used_usd = 0, budget_reset_at = ? WHERE id = ?`,
		resetAt.Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("reset virtual key budget: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("reset virtual key budget rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("reset virtual key budget: %d not found", id)
	}
	return nil
}

func generateVirtualKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate virtual key: %w", err)
	}
	return "gvk-" + hex.EncodeToString(buf), nil
}

func hashVirtualKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func scanVirtualKey(scanner interface {
	Scan(dest ...any) error
}) (*VirtualKey, error) {
	var key VirtualKey
	var budgetUSD sql.NullFloat64
	var budgetResetAt sql.NullString
	var rateLimitRPM, rateLimitTPM sql.NullInt64
	var teamID sql.NullInt64
	var isActive int
	var mcpToolGroup sql.NullString
	var createdAt string

	err := scanner.Scan(
		&key.ID, &key.Name, &key.KeyPrefix, &key.KeyHash, &budgetUSD, &key.BudgetPeriod,
		&key.BudgetUsedUSD, &budgetResetAt, &rateLimitRPM, &rateLimitTPM, &teamID, &isActive, &mcpToolGroup, &createdAt,
	)
	if err != nil {
		return nil, err
	}
	if budgetUSD.Valid {
		key.BudgetUSD = &budgetUSD.Float64
	}
	if budgetResetAt.Valid && budgetResetAt.String != "" {
		parsed, err := time.Parse(time.RFC3339, budgetResetAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse virtual key budget_reset_at: %w", err)
		}
		key.BudgetResetAt = &parsed
	}
	if rateLimitRPM.Valid {
		v := int(rateLimitRPM.Int64)
		key.RateLimitRPM = &v
	}
	if rateLimitTPM.Valid {
		v := int(rateLimitTPM.Int64)
		key.RateLimitTPM = &v
	}
	if teamID.Valid {
		key.TeamID = &teamID.Int64
	}
	key.IsActive = isActive != 0
	if mcpToolGroup.Valid {
		key.MCPToolGroup = mcpToolGroup.String
	}
	key.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse virtual key created_at: %w", err)
	}
	return &key, nil
}

func scanVirtualKeyList(scanner interface {
	Scan(dest ...any) error
}) (*VirtualKey, error) {
	var key VirtualKey
	var budgetUSD sql.NullFloat64
	var budgetResetAt sql.NullString
	var rateLimitRPM, rateLimitTPM sql.NullInt64
	var teamID sql.NullInt64
	var isActive int
	var mcpToolGroup sql.NullString
	var createdAt string

	err := scanner.Scan(
		&key.ID, &key.Name, &key.KeyPrefix, &budgetUSD, &key.BudgetPeriod,
		&key.BudgetUsedUSD, &budgetResetAt, &rateLimitRPM, &rateLimitTPM, &teamID, &isActive, &mcpToolGroup, &createdAt,
	)
	if err != nil {
		return nil, err
	}
	if budgetUSD.Valid {
		key.BudgetUSD = &budgetUSD.Float64
	}
	if budgetResetAt.Valid && budgetResetAt.String != "" {
		parsed, err := time.Parse(time.RFC3339, budgetResetAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse virtual key budget_reset_at: %w", err)
		}
		key.BudgetResetAt = &parsed
	}
	if rateLimitRPM.Valid {
		v := int(rateLimitRPM.Int64)
		key.RateLimitRPM = &v
	}
	if rateLimitTPM.Valid {
		v := int(rateLimitTPM.Int64)
		key.RateLimitTPM = &v
	}
	if teamID.Valid {
		key.TeamID = &teamID.Int64
	}
	key.IsActive = isActive != 0
	if mcpToolGroup.Valid {
		key.MCPToolGroup = mcpToolGroup.String
	}
	key.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse virtual key created_at: %w", err)
	}
	return &key, nil
}
