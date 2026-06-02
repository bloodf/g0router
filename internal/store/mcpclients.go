package store

import (
	"database/sql"
	"fmt"

	"github.com/bloodf/g0router/internal/mcp"
)

type MCPClient struct {
	ID                string
	Name              string
	Transport         mcp.Transport
	Command           *string
	Args              []string
	URL               *string
	Env               map[string]string
	IsActive          bool
	HealthStatus      string
	LastHealthCheck   *string
	ToolManifest      *mcp.Manifest
	ManifestUpdatedAt *string
	CreatedAt         string
}

func (s *Store) CreateMCPClient(client *MCPClient) error {
	args, err := encodeJSON(client.Args)
	if err != nil {
		return fmt.Errorf("encode mcp args: %w", err)
	}
	env, err := encodeJSON(client.Env)
	if err != nil {
		return fmt.Errorf("encode mcp env: %w", err)
	}

	row := s.db.QueryRow(
		`INSERT INTO mcp_clients (
			name, transport, command, args, url, env, is_active
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id, health_status, created_at`,
		client.Name,
		string(client.Transport),
		client.Command,
		args,
		client.URL,
		env,
		boolToInt(client.IsActive),
	)
	if err := row.Scan(&client.ID, &client.HealthStatus, &client.CreatedAt); err != nil {
		return fmt.Errorf("insert mcp client: %w", err)
	}

	return nil
}

func (s *Store) GetMCPClient(id string) (*MCPClient, error) {
	client, err := scanMCPClient(s.db.QueryRow(mcpClientSelectSQL()+" WHERE id = ?", id))
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (s *Store) ListMCPClients() ([]*MCPClient, error) {
	rows, err := s.db.Query(mcpClientSelectSQL() + " ORDER BY rowid")
	if err != nil {
		return nil, fmt.Errorf("query mcp clients: %w", err)
	}
	defer rows.Close()

	var clients []*MCPClient
	for rows.Next() {
		client, err := scanMCPClient(rows)
		if err != nil {
			return nil, err
		}
		clients = append(clients, client)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mcp clients: %w", err)
	}

	return clients, nil
}

func (s *Store) UpdateMCPClientManifest(id string, manifest mcp.Manifest) error {
	encoded, err := encodeJSON(manifest)
	if err != nil {
		return fmt.Errorf("encode mcp manifest: %w", err)
	}

	result, err := s.db.Exec(
		`UPDATE mcp_clients SET
			tool_manifest = ?,
			manifest_updated_at = datetime('now')
		WHERE id = ?`,
		encoded,
		id,
	)
	if err != nil {
		return fmt.Errorf("update mcp manifest: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}

	return nil
}

func (s *Store) UpdateMCPClientHealth(id, status string) error {
	result, err := s.db.Exec(
		`UPDATE mcp_clients SET
			health_status = ?,
			last_health_check = datetime('now')
		WHERE id = ?`,
		status,
		id,
	)
	if err != nil {
		return fmt.Errorf("update mcp health: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}

	return nil
}

func (s *Store) DeleteMCPClient(id string) error {
	result, err := s.db.Exec("DELETE FROM mcp_clients WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete mcp client: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}

	return nil
}

func (c *MCPClient) ClientConfig() mcp.ClientConfig {
	cfg := mcp.ClientConfig{
		ID:        c.ID,
		Name:      c.Name,
		Transport: c.Transport,
		Args:      append([]string(nil), c.Args...),
		Env:       copyStringMap(c.Env),
	}
	if c.Command != nil {
		cfg.Command = *c.Command
	}
	if c.URL != nil {
		cfg.URL = *c.URL
	}
	return cfg
}

func scanMCPClient(scanner connectionScanner) (*MCPClient, error) {
	var client MCPClient
	var command, args, url, env sql.NullString
	var lastHealthCheck, manifest, manifestUpdatedAt sql.NullString
	var isActive int

	err := scanner.Scan(
		&client.ID,
		&client.Name,
		&client.Transport,
		&command,
		&args,
		&url,
		&env,
		&isActive,
		&client.HealthStatus,
		&lastHealthCheck,
		&manifest,
		&manifestUpdatedAt,
		&client.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan mcp client: %w", err)
	}

	client.Command = stringPtrFromNull(command)
	client.URL = stringPtrFromNull(url)
	client.LastHealthCheck = stringPtrFromNull(lastHealthCheck)
	client.ManifestUpdatedAt = stringPtrFromNull(manifestUpdatedAt)
	client.IsActive = isActive != 0
	if err := decodeJSON(args, &client.Args); err != nil {
		return nil, fmt.Errorf("decode mcp args: %w", err)
	}
	if err := decodeJSON(env, &client.Env); err != nil {
		return nil, fmt.Errorf("decode mcp env: %w", err)
	}
	if manifest.Valid && manifest.String != "" {
		var decoded mcp.Manifest
		if err := decodeJSON(manifest, &decoded); err != nil {
			return nil, fmt.Errorf("decode mcp manifest: %w", err)
		}
		client.ToolManifest = &decoded
	}

	return &client, nil
}

func mcpClientSelectSQL() string {
	return `SELECT
		id, name, transport, command, args, url, env, is_active,
		health_status, last_health_check, tool_manifest, manifest_updated_at, created_at
		FROM mcp_clients`
}

func copyStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}
