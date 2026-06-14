package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// MCPToolGroup is a named, toggleable grouping of MCP tool names. ToolIDs is
// JSON-encoded into the tool_ids column; timestamps are ISO-8601 strings to
// mirror the w6-l mock contract (seed/mcp.ts created_at). The numeric id mirrors
// the feature_flags / alert_channels INTEGER-PK precedent (ESC-IDTYPE).
type MCPToolGroup struct {
	ID        int64
	Name      string
	ToolIDs   []string
	IsActive  bool
	CreatedAt string // ISO-8601 (RFC3339)
	UpdatedAt string // ISO-8601 (RFC3339), empty until first update
}

// CreateMCPToolGroup inserts a tool group and returns the stored record with its
// generated numeric id and ISO-8601 created_at.
func (s *Store) CreateMCPToolGroup(g *MCPToolGroup) (*MCPToolGroup, error) {
	toolIDsJSON, err := marshalJSONArray(g.ToolIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal mcp tool group tool_ids: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := s.db.Exec(
		`INSERT INTO mcp_tool_groups (name, tool_ids, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)`,
		g.Name, toolIDsJSON, boolToInt(g.IsActive), now, "",
	)
	if err != nil {
		return nil, fmt.Errorf("insert mcp tool group %s: %w", g.Name, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}
	return s.GetMCPToolGroup(id)
}

// GetMCPToolGroup returns the tool group with the given id.
func (s *Store) GetMCPToolGroup(id int64) (*MCPToolGroup, error) {
	return s.scanMCPToolGroup(s.db.QueryRow(
		`SELECT id, name, tool_ids, is_active, created_at, updated_at FROM mcp_tool_groups WHERE id = ?`, id))
}

// ListMCPToolGroups returns all tool groups ordered by id ascending.
func (s *Store) ListMCPToolGroups() ([]*MCPToolGroup, error) {
	rows, err := s.db.Query(
		`SELECT id, name, tool_ids, is_active, created_at, updated_at FROM mcp_tool_groups ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("query mcp tool groups: %w", err)
	}
	defer rows.Close()

	var out []*MCPToolGroup
	for rows.Next() {
		g, err := s.scanMCPToolGroup(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mcp tool groups: %w", err)
	}
	return out, nil
}

// UpdateMCPToolGroup persists the mutable fields of a tool group and returns the
// updated record. It returns ErrNotFound when no group has the given id.
func (s *Store) UpdateMCPToolGroup(id int64, g *MCPToolGroup) (*MCPToolGroup, error) {
	toolIDsJSON, err := marshalJSONArray(g.ToolIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal mcp tool group tool_ids: %w", err)
	}
	res, err := s.db.Exec(
		`UPDATE mcp_tool_groups SET name = ?, tool_ids = ?, is_active = ?, updated_at = ? WHERE id = ?`,
		g.Name, toolIDsJSON, boolToInt(g.IsActive), time.Now().UTC().Format(time.RFC3339Nano), id,
	)
	if err != nil {
		return nil, fmt.Errorf("update mcp tool group %d: %w", id, err)
	}
	if err := requireRowAffected(res); err != nil {
		return nil, err
	}
	return s.GetMCPToolGroup(id)
}

// DeleteMCPToolGroup removes the tool group with the given id.
func (s *Store) DeleteMCPToolGroup(id int64) error {
	res, err := s.db.Exec("DELETE FROM mcp_tool_groups WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete mcp tool group %d: %w", id, err)
	}
	return requireRowAffected(res)
}

func (s *Store) scanMCPToolGroup(row rowScanner) (*MCPToolGroup, error) {
	var g MCPToolGroup
	var toolIDsJSON string
	var isActive int
	err := row.Scan(&g.ID, &g.Name, &toolIDsJSON, &isActive, &g.CreatedAt, &g.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan mcp tool group: %w", err)
	}
	if g.ToolIDs, err = unmarshalJSONArray(toolIDsJSON); err != nil {
		return nil, fmt.Errorf("unmarshal mcp tool group tool_ids: %w", err)
	}
	if g.ToolIDs == nil {
		g.ToolIDs = []string{}
	}
	g.IsActive = isActive != 0
	return &g, nil
}
