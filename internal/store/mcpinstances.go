package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// MCPClient is a stored MCP client registration. The free-form Config map is
// JSON-encoded into the config_json column (ESC-SCHEMA). Mirrors
// schemas.MCPClient with persistence timestamps.
type MCPClient struct {
	ID        string
	Name      string
	Type      string // 'default' | 'custom' (registration origin)
	Config    map[string]any
	CreatedAt int64
	UpdatedAt int64
}

// MCPInstance is a stored MCP server instance. Args/Env are JSON-encoded into
// args_json/env_json (ESC-SCHEMA). Mirrors schemas.MCPInstance with timestamps.
type MCPInstance struct {
	ID        string
	ClientID  string
	Name      string
	Transport string // 'stdio' | 'http' | 'sse'
	URL       string
	Command   string
	Args      []string
	Env       map[string]string
	Status    string // 'stopped' | 'starting' | 'running' | 'error'
	CreatedAt int64
	UpdatedAt int64
}

// CreateMCPClient inserts a client and returns the stored record with its
// generated ID and timestamps.
func (s *Store) CreateMCPClient(c *MCPClient) (*MCPClient, error) {
	id, err := newID()
	if err != nil {
		return nil, err
	}
	normalizeMCPClientConfig(c)
	configJSON, err := marshalJSONObject(c.Config)
	if err != nil {
		return nil, fmt.Errorf("marshal mcp client config: %w", err)
	}
	now := time.Now().Unix()
	_, err = s.db.Exec(
		`INSERT INTO mcp_clients (id, name, type, config_json, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, c.Name, c.Type, configJSON, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert mcp client %s: %w", c.Name, err)
	}
	return s.GetMCPClient(id)
}

// GetMCPClient returns the client with the given id.
func (s *Store) GetMCPClient(id string) (*MCPClient, error) {
	return s.scanMCPClient(s.db.QueryRow(
		`SELECT id, name, type, config_json, created_at, updated_at FROM mcp_clients WHERE id = ?`, id))
}

// ListMCPClients returns all clients ordered by creation time.
func (s *Store) ListMCPClients() ([]*MCPClient, error) {
	rows, err := s.db.Query(
		`SELECT id, name, type, config_json, created_at, updated_at FROM mcp_clients ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("query mcp clients: %w", err)
	}
	defer rows.Close()

	var out []*MCPClient
	for rows.Next() {
		c, err := s.scanMCPClient(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mcp clients: %w", err)
	}
	return out, nil
}

// UpsertMCPClient inserts or updates a client by id (custom-plugin persistence —
// PAR-MCP-006). The caller supplies the id (e.g. from a prior CreateMCPClient).
func (s *Store) UpsertMCPClient(c *MCPClient) error {
	normalizeMCPClientConfig(c)
	configJSON, err := marshalJSONObject(c.Config)
	if err != nil {
		return fmt.Errorf("marshal mcp client config: %w", err)
	}
	now := time.Now().Unix()
	_, err = s.db.Exec(
		`INSERT INTO mcp_clients (id, name, type, config_json, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET name = excluded.name, type = excluded.type,
		   config_json = excluded.config_json, updated_at = excluded.updated_at`,
		c.ID, c.Name, c.Type, configJSON, now, now,
	)
	if err != nil {
		return fmt.Errorf("upsert mcp client %s: %w", c.ID, err)
	}
	return nil
}

// DeleteMCPClient removes the client with the given id.
func (s *Store) DeleteMCPClient(id string) error {
	res, err := s.db.Exec("DELETE FROM mcp_clients WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete mcp client %s: %w", id, err)
	}
	return requireRowAffected(res)
}

// CreateMCPInstance inserts an instance and returns the stored record.
func (s *Store) CreateMCPInstance(in *MCPInstance) (*MCPInstance, error) {
	id, err := newID()
	if err != nil {
		return nil, err
	}
	argsJSON, err := marshalJSONArray(in.Args)
	if err != nil {
		return nil, fmt.Errorf("marshal mcp instance args: %w", err)
	}
	envJSON, err := marshalJSONStringMap(in.Env)
	if err != nil {
		return nil, fmt.Errorf("marshal mcp instance env: %w", err)
	}
	// Env carries secrets (stdio MCP server process env vars). Encrypt at rest in
	// env_json_enc and drain the legacy plaintext env_json column to '{}' so the
	// secret never persists in plaintext (bf-mcp-3, mirrors bf-gov-5).
	envEnc, err := s.cipher.Encrypt(envJSON)
	if err != nil {
		return nil, fmt.Errorf("encrypt mcp instance env: %w", err)
	}
	transport := in.Transport
	if transport == "" {
		transport = "stdio"
	}
	status := in.Status
	if status == "" {
		status = "stopped"
	}
	now := time.Now().Unix()
	_, err = s.db.Exec(
		`INSERT INTO mcp_instances
		 (id, client_id, name, transport, url, command, args_json, env_json, env_json_enc, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, in.ClientID, in.Name, transport, in.URL, in.Command, argsJSON, "{}", envEnc, status, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert mcp instance %s: %w", in.Name, err)
	}
	return s.GetMCPInstance(id)
}

// GetMCPInstance returns the instance with the given id.
func (s *Store) GetMCPInstance(id string) (*MCPInstance, error) {
	return s.scanMCPInstance(s.db.QueryRow(
		`SELECT id, client_id, name, transport, url, command, args_json, env_json_enc, status, created_at, updated_at
		 FROM mcp_instances WHERE id = ?`, id))
}

// ListMCPInstances returns all instances ordered by creation time.
func (s *Store) ListMCPInstances() ([]*MCPInstance, error) {
	rows, err := s.db.Query(
		`SELECT id, client_id, name, transport, url, command, args_json, env_json_enc, status, created_at, updated_at
		 FROM mcp_instances ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("query mcp instances: %w", err)
	}
	defer rows.Close()

	var out []*MCPInstance
	for rows.Next() {
		in, err := s.scanMCPInstance(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, in)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mcp instances: %w", err)
	}
	return out, nil
}

// UpdateMCPInstance persists the mutable fields of an instance.
func (s *Store) UpdateMCPInstance(in *MCPInstance) error {
	argsJSON, err := marshalJSONArray(in.Args)
	if err != nil {
		return fmt.Errorf("marshal mcp instance args: %w", err)
	}
	envJSON, err := marshalJSONStringMap(in.Env)
	if err != nil {
		return fmt.Errorf("marshal mcp instance env: %w", err)
	}
	// Re-encrypt the (possibly rotated) env and drain plaintext env_json to '{}'
	// so an update never leaves a plaintext secret behind (bf-mcp-3).
	envEnc, err := s.cipher.Encrypt(envJSON)
	if err != nil {
		return fmt.Errorf("encrypt mcp instance env: %w", err)
	}
	res, err := s.db.Exec(
		`UPDATE mcp_instances SET client_id = ?, name = ?, transport = ?, url = ?, command = ?,
		 args_json = ?, env_json = '{}', env_json_enc = ?, status = ?, updated_at = ? WHERE id = ?`,
		in.ClientID, in.Name, in.Transport, in.URL, in.Command, argsJSON, envEnc, in.Status,
		time.Now().Unix(), in.ID,
	)
	if err != nil {
		return fmt.Errorf("update mcp instance %s: %w", in.ID, err)
	}
	return requireRowAffected(res)
}

// SetMCPInstanceStatus updates only the status of an instance.
func (s *Store) SetMCPInstanceStatus(id, status string) error {
	res, err := s.db.Exec(
		"UPDATE mcp_instances SET status = ?, updated_at = ? WHERE id = ?",
		status, time.Now().Unix(), id,
	)
	if err != nil {
		return fmt.Errorf("set mcp instance status %s: %w", id, err)
	}
	return requireRowAffected(res)
}

// DeleteMCPInstance removes the instance with the given id.
func (s *Store) DeleteMCPInstance(id string) error {
	res, err := s.db.Exec("DELETE FROM mcp_instances WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete mcp instance %s: %w", id, err)
	}
	return requireRowAffected(res)
}

func (s *Store) scanMCPClient(row rowScanner) (*MCPClient, error) {
	var c MCPClient
	var configJSON string
	err := row.Scan(&c.ID, &c.Name, &c.Type, &configJSON, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan mcp client: %w", err)
	}
	if c.Config, err = unmarshalJSONObject(configJSON); err != nil {
		return nil, fmt.Errorf("unmarshal mcp client config: %w", err)
	}
	return &c, nil
}

func (s *Store) scanMCPInstance(row rowScanner) (*MCPInstance, error) {
	var in MCPInstance
	var argsJSON, envEnc string
	err := row.Scan(&in.ID, &in.ClientID, &in.Name, &in.Transport, &in.URL, &in.Command,
		&argsJSON, &envEnc, &in.Status, &in.CreatedAt, &in.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan mcp instance: %w", err)
	}
	if in.Args, err = unmarshalJSONArray(argsJSON); err != nil {
		return nil, fmt.Errorf("unmarshal mcp instance args: %w", err)
	}
	// Empty-string env_json_enc means "no env" (degenerate pre-backfill / never
	// saved); treat as empty rather than calling Decrypt("") which would error.
	// Create/Update always encrypt and the backfill always seals legacy rows, so
	// '' only ever means empty. Otherwise decrypt is fail-closed (mirror mcpoauth.go).
	if envEnc == "" {
		return &in, nil
	}
	envJSON, err := s.cipher.Decrypt(envEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt mcp instance env: %w", err)
	}
	if in.Env, err = unmarshalJSONStringMap(envJSON); err != nil {
		return nil, fmt.Errorf("unmarshal mcp instance env: %w", err)
	}
	return &in, nil
}

// --- JSON column helpers (config_json / args_json / env_json) ---

func marshalJSONObject(m map[string]any) (string, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func unmarshalJSONObject(s string) (map[string]any, error) {
	if s == "" {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

func marshalJSONArray(a []string) (string, error) {
	if a == nil {
		return "[]", nil
	}
	b, err := json.Marshal(a)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func unmarshalJSONArray(s string) ([]string, error) {
	if s == "" {
		return nil, nil
	}
	var a []string
	if err := json.Unmarshal([]byte(s), &a); err != nil {
		return nil, err
	}
	return a, nil
}

func marshalJSONStringMap(m map[string]string) (string, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func unmarshalJSONStringMap(s string) (map[string]string, error) {
	if s == "" {
		return nil, nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	return m, nil
}
