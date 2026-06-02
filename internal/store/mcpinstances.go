package store

import (
	"database/sql"
	"fmt"

	"github.com/bloodf/g0router/internal/mcp"
)

type MCPInstance struct {
	ID                string
	Name              string
	ServerKey         string
	LaunchType        mcp.LaunchType
	Transport         mcp.Transport
	Command           *string
	Args              []string
	URL               *string
	Headers           map[string]string
	Env               map[string]string
	CWD               *string
	AccountLabel      *string
	IsActive          bool
	HealthStatus      string
	LastHealthCheck   *string
	ToolManifest      *mcp.Manifest
	ManifestUpdatedAt *string
	CreatedAt         string
	UpdatedAt         string
}

func (s *Store) CreateMCPInstance(instance *MCPInstance) error {
	if err := instance.Config().Validate(); err != nil {
		return err
	}
	args, err := encodeJSON(instance.Args)
	if err != nil {
		return fmt.Errorf("encode mcp instance args: %w", err)
	}
	env, err := encodeJSON(instance.Env)
	if err != nil {
		return fmt.Errorf("encode mcp instance env: %w", err)
	}
	headers, err := encodeJSON(instance.Headers)
	if err != nil {
		return fmt.Errorf("encode mcp instance headers: %w", err)
	}

	row := s.db.QueryRow(
		`INSERT INTO mcp_instances (
			name, server_key, launch_type, transport, command, args, url,
			headers, env, cwd, account_label, is_active
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, health_status, created_at, updated_at`,
		instance.Name,
		instance.ServerKey,
		string(instance.LaunchType),
		string(instance.Transport),
		instance.Command,
		args,
		instance.URL,
		headers,
		env,
		instance.CWD,
		instance.AccountLabel,
		boolToInt(instance.IsActive),
	)
	if err := row.Scan(&instance.ID, &instance.HealthStatus, &instance.CreatedAt, &instance.UpdatedAt); err != nil {
		return fmt.Errorf("insert mcp instance: %w", err)
	}
	return nil
}

func (s *Store) GetMCPInstance(id string) (*MCPInstance, error) {
	instance, err := scanMCPInstance(s.db.QueryRow(mcpInstanceSelectSQL()+" WHERE id = ?", id))
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func (s *Store) ListMCPInstances() ([]*MCPInstance, error) {
	rows, err := s.db.Query(mcpInstanceSelectSQL() + " ORDER BY rowid")
	if err != nil {
		return nil, fmt.Errorf("query mcp instances: %w", err)
	}
	defer rows.Close()

	var instances []*MCPInstance
	for rows.Next() {
		instance, err := scanMCPInstance(rows)
		if err != nil {
			return nil, err
		}
		instance.applyRedaction()
		instances = append(instances, instance)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mcp instances: %w", err)
	}
	return instances, nil
}

func (s *Store) UpdateMCPInstanceManifest(id string, manifest mcp.Manifest) error {
	encoded, err := encodeJSON(manifest)
	if err != nil {
		return fmt.Errorf("encode mcp instance manifest: %w", err)
	}
	result, err := s.db.Exec(
		`UPDATE mcp_instances SET
			tool_manifest = ?,
			manifest_updated_at = datetime('now'),
			updated_at = datetime('now')
		WHERE id = ?`,
		encoded,
		id,
	)
	if err != nil {
		return fmt.Errorf("update mcp instance manifest: %w", err)
	}
	return requireRowsAffected(result)
}

func (i *MCPInstance) Config() mcp.InstanceConfig {
	cfg := mcp.InstanceConfig{
		ID:           i.ID,
		Name:         i.Name,
		ServerKey:    i.ServerKey,
		LaunchType:   i.LaunchType,
		Transport:    i.Transport,
		Args:         append([]string(nil), i.Args...),
		Headers:      copyStringMap(i.Headers),
		Env:          copyStringMap(i.Env),
		AccountLabel: stringValue(i.AccountLabel),
	}
	if i.Command != nil {
		cfg.Command = *i.Command
	}
	if i.URL != nil {
		cfg.URL = *i.URL
	}
	if i.CWD != nil {
		cfg.CWD = *i.CWD
	}
	return cfg
}

func (i *MCPInstance) applyRedaction() {
	redacted := i.Config().Redacted()
	i.Env = redacted.Env
	i.Headers = redacted.Headers
}

func scanMCPInstance(scanner connectionScanner) (*MCPInstance, error) {
	var instance MCPInstance
	var command, args, url, headers, env, cwd, accountLabel sql.NullString
	var lastHealthCheck, manifest, manifestUpdatedAt sql.NullString
	var isActive int

	err := scanner.Scan(
		&instance.ID,
		&instance.Name,
		&instance.ServerKey,
		&instance.LaunchType,
		&instance.Transport,
		&command,
		&args,
		&url,
		&headers,
		&env,
		&cwd,
		&accountLabel,
		&isActive,
		&instance.HealthStatus,
		&lastHealthCheck,
		&manifest,
		&manifestUpdatedAt,
		&instance.CreatedAt,
		&instance.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan mcp instance: %w", err)
	}

	instance.Command = stringPtrFromNull(command)
	instance.URL = stringPtrFromNull(url)
	instance.CWD = stringPtrFromNull(cwd)
	instance.AccountLabel = stringPtrFromNull(accountLabel)
	instance.LastHealthCheck = stringPtrFromNull(lastHealthCheck)
	instance.ManifestUpdatedAt = stringPtrFromNull(manifestUpdatedAt)
	instance.IsActive = isActive != 0
	if err := decodeJSON(args, &instance.Args); err != nil {
		return nil, fmt.Errorf("decode mcp instance args: %w", err)
	}
	if err := decodeJSON(headers, &instance.Headers); err != nil {
		return nil, fmt.Errorf("decode mcp instance headers: %w", err)
	}
	if err := decodeJSON(env, &instance.Env); err != nil {
		return nil, fmt.Errorf("decode mcp instance env: %w", err)
	}
	if manifest.Valid && manifest.String != "" {
		var decoded mcp.Manifest
		if err := decodeJSON(manifest, &decoded); err != nil {
			return nil, fmt.Errorf("decode mcp instance manifest: %w", err)
		}
		instance.ToolManifest = &decoded
	}

	return &instance, nil
}

func mcpInstanceSelectSQL() string {
	return `SELECT
		id, name, server_key, launch_type, transport, command, args, url,
		headers, env, cwd, account_label, is_active, health_status,
		last_health_check, tool_manifest, manifest_updated_at, created_at, updated_at
		FROM mcp_instances`
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
