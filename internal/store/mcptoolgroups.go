package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// MCPToolGroup holds a named set of MCP tools that can be injected as a unit.
type MCPToolGroup struct {
	ID        int64
	Name      string
	ToolIDs   []string
	IsActive  bool
	CreatedAt string
	UpdatedAt string
}

// ListMCPToolGroups returns all MCP tool groups ordered by name.
func (s *Store) ListMCPToolGroups() ([]MCPToolGroup, error) {
	rows, err := s.db.Query(`SELECT id, name, tool_ids_json, is_active, created_at, updated_at FROM mcp_tool_groups ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("query mcp tool groups: %w", err)
	}
	defer rows.Close()

	var groups []MCPToolGroup
	for rows.Next() {
		g, err := scanMCPToolGroup(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mcp tool groups: %w", err)
	}
	return groups, nil
}

// GetMCPToolGroup returns a single MCP tool group by ID.
func (s *Store) GetMCPToolGroup(id int64) (*MCPToolGroup, error) {
	g, err := scanMCPToolGroup(s.db.QueryRow(
		`SELECT id, name, tool_ids_json, is_active, created_at, updated_at FROM mcp_tool_groups WHERE id = ?`,
		id,
	))
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// GetMCPToolGroupByName returns a single active MCP tool group by name.
func (s *Store) GetMCPToolGroupByName(name string) (*MCPToolGroup, error) {
	g, err := scanMCPToolGroup(s.db.QueryRow(
		`SELECT id, name, tool_ids_json, is_active, created_at, updated_at FROM mcp_tool_groups WHERE name = ? AND is_active = 1`,
		name,
	))
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// CreateMCPToolGroup inserts a new MCP tool group.
func (s *Store) CreateMCPToolGroup(name string, toolIDs []string, isActive bool) (*MCPToolGroup, error) {
	toolIDsJSON, err := json.Marshal(toolIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal tool ids: %w", err)
	}

	var g MCPToolGroup
	var toolIDsRaw string
	err = s.db.QueryRow(
		`INSERT INTO mcp_tool_groups (name, tool_ids_json, is_active) VALUES (?, ?, ?)
		RETURNING id, name, tool_ids_json, is_active, created_at, updated_at`,
		name, string(toolIDsJSON), boolInt(isActive),
	).Scan(&g.ID, &g.Name, &toolIDsRaw, &g.IsActive, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		if isUniqueConstraintError(err, "mcp_tool_groups") {
			return nil, fmt.Errorf("create mcp tool group: %w", ErrDuplicateName)
		}
		return nil, fmt.Errorf("create mcp tool group: %w", err)
	}
	g.ToolIDs = unmarshalToolIDsJSON(toolIDsRaw)
	return &g, nil
}

// UpdateMCPToolGroup modifies an existing MCP tool group by ID.
func (s *Store) UpdateMCPToolGroup(id int64, name string, toolIDs []string, isActive bool) error {
	toolIDsJSON, err := json.Marshal(toolIDs)
	if err != nil {
		return fmt.Errorf("marshal tool ids: %w", err)
	}

	result, err := s.db.Exec(
		`UPDATE mcp_tool_groups SET name = ?, tool_ids_json = ?, is_active = ?, updated_at = datetime('now') WHERE id = ?`,
		name, string(toolIDsJSON), boolInt(isActive), id,
	)
	if err != nil {
		if isUniqueConstraintError(err, "mcp_tool_groups") {
			return fmt.Errorf("update mcp tool group: %w", ErrDuplicateName)
		}
		return fmt.Errorf("update mcp tool group: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}
	return nil
}

// DeleteMCPToolGroup removes an MCP tool group by ID.
func (s *Store) DeleteMCPToolGroup(id int64) error {
	result, err := s.db.Exec(`DELETE FROM mcp_tool_groups WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete mcp tool group: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}
	return nil
}

func scanMCPToolGroup(scanner interface {
	Scan(dest ...any) error
}) (MCPToolGroup, error) {
	var g MCPToolGroup
	var toolIDsRaw string
	var isActive int
	err := scanner.Scan(&g.ID, &g.Name, &toolIDsRaw, &isActive, &g.CreatedAt, &g.UpdatedAt)
	if err == sql.ErrNoRows {
		return MCPToolGroup{}, ErrNotFound
	}
	if err != nil {
		return MCPToolGroup{}, fmt.Errorf("scan mcp tool group: %w", err)
	}
	g.IsActive = isActive == 1
	g.ToolIDs = unmarshalToolIDsJSON(toolIDsRaw)
	return g, nil
}

func unmarshalToolIDsJSON(raw string) []string {
	var ids []string
	_ = json.Unmarshal([]byte(raw), &ids)
	return ids
}
