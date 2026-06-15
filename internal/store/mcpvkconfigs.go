package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// VKMCPConfig is one VK↔MCP assignment row: a virtual key's per-client tool
// scope (bf-mcp-2 / PAR-BF-MCP-033, D2). ToolsToExecute holds the executeOnlyTools
// wildcard patterns (017/019) and ToolsToAutoExecute their subset (018/049), each
// JSON-encoded into its column. ConfigHash is the deterministic write-time hash
// exposed for drift-detection (079). The numeric id mirrors the mcp_tool_groups
// INTEGER-PK additive-table precedent.
type VKMCPConfig struct {
	ID                 int64
	VirtualKeyID       string
	MCPClientID        string
	ToolsToExecute     []string
	ToolsToAutoExecute []string
	ConfigHash         string
	CreatedAt          int64
	UpdatedAt          int64
}

// CreateVKMCPConfig inserts an assignment and returns the stored record with its
// generated numeric id and timestamps.
func (s *Store) CreateVKMCPConfig(c *VKMCPConfig) (*VKMCPConfig, error) {
	execJSON, err := marshalJSONArray(c.ToolsToExecute)
	if err != nil {
		return nil, fmt.Errorf("marshal vk mcp config tools_to_execute: %w", err)
	}
	autoJSON, err := marshalJSONArray(c.ToolsToAutoExecute)
	if err != nil {
		return nil, fmt.Errorf("marshal vk mcp config tools_to_auto_execute: %w", err)
	}
	now := time.Now().Unix()
	res, err := s.db.Exec(
		`INSERT INTO virtual_key_mcp_configs
		 (virtual_key_id, mcp_client_id, tools_to_execute_json, tools_to_auto_execute_json, config_hash, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.VirtualKeyID, c.MCPClientID, execJSON, autoJSON, c.ConfigHash, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert vk mcp config %s/%s: %w", c.VirtualKeyID, c.MCPClientID, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}
	return s.GetVKMCPConfig(id)
}

// GetVKMCPConfig returns the assignment with the given id.
func (s *Store) GetVKMCPConfig(id int64) (*VKMCPConfig, error) {
	return s.scanVKMCPConfig(s.db.QueryRow(
		`SELECT id, virtual_key_id, mcp_client_id, tools_to_execute_json, tools_to_auto_execute_json, config_hash, created_at, updated_at
		 FROM virtual_key_mcp_configs WHERE id = ?`, id))
}

// ListVKMCPConfigsByVK returns all assignments for a virtual key, ordered by id.
// This is the filter's request-time live source (D2/D3).
func (s *Store) ListVKMCPConfigsByVK(virtualKeyID string) ([]*VKMCPConfig, error) {
	rows, err := s.db.Query(
		`SELECT id, virtual_key_id, mcp_client_id, tools_to_execute_json, tools_to_auto_execute_json, config_hash, created_at, updated_at
		 FROM virtual_key_mcp_configs WHERE virtual_key_id = ? ORDER BY id ASC`, virtualKeyID)
	if err != nil {
		return nil, fmt.Errorf("query vk mcp configs for %s: %w", virtualKeyID, err)
	}
	defer rows.Close()

	var out []*VKMCPConfig
	for rows.Next() {
		c, err := s.scanVKMCPConfig(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate vk mcp configs: %w", err)
	}
	return out, nil
}

// UpdateVKMCPConfig persists the mutable fields of an assignment and returns the
// updated record. It returns ErrNotFound when no row has the given id.
func (s *Store) UpdateVKMCPConfig(id int64, c *VKMCPConfig) (*VKMCPConfig, error) {
	execJSON, err := marshalJSONArray(c.ToolsToExecute)
	if err != nil {
		return nil, fmt.Errorf("marshal vk mcp config tools_to_execute: %w", err)
	}
	autoJSON, err := marshalJSONArray(c.ToolsToAutoExecute)
	if err != nil {
		return nil, fmt.Errorf("marshal vk mcp config tools_to_auto_execute: %w", err)
	}
	res, err := s.db.Exec(
		`UPDATE virtual_key_mcp_configs
		 SET virtual_key_id = ?, mcp_client_id = ?, tools_to_execute_json = ?, tools_to_auto_execute_json = ?, config_hash = ?, updated_at = ?
		 WHERE id = ?`,
		c.VirtualKeyID, c.MCPClientID, execJSON, autoJSON, c.ConfigHash, time.Now().Unix(), id,
	)
	if err != nil {
		return nil, fmt.Errorf("update vk mcp config %d: %w", id, err)
	}
	if err := requireRowAffected(res); err != nil {
		return nil, err
	}
	return s.GetVKMCPConfig(id)
}

// DeleteVKMCPConfig removes the assignment with the given id.
func (s *Store) DeleteVKMCPConfig(id int64) error {
	res, err := s.db.Exec("DELETE FROM virtual_key_mcp_configs WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete vk mcp config %d: %w", id, err)
	}
	return requireRowAffected(res)
}

func (s *Store) scanVKMCPConfig(row rowScanner) (*VKMCPConfig, error) {
	var c VKMCPConfig
	var execJSON, autoJSON string
	err := row.Scan(&c.ID, &c.VirtualKeyID, &c.MCPClientID, &execJSON, &autoJSON, &c.ConfigHash, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan vk mcp config: %w", err)
	}
	if c.ToolsToExecute, err = unmarshalJSONArray(execJSON); err != nil {
		return nil, fmt.Errorf("unmarshal vk mcp config tools_to_execute: %w", err)
	}
	if c.ToolsToExecute == nil {
		c.ToolsToExecute = []string{}
	}
	if c.ToolsToAutoExecute, err = unmarshalJSONArray(autoJSON); err != nil {
		return nil, fmt.Errorf("unmarshal vk mcp config tools_to_auto_execute: %w", err)
	}
	if c.ToolsToAutoExecute == nil {
		c.ToolsToAutoExecute = []string{}
	}
	return &c, nil
}
