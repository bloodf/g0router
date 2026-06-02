package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	path string
	db   *sql.DB
}

func NewStore(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create data dir: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	for _, pragma := range []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA foreign_keys = ON",
		"PRAGMA synchronous = NORMAL",
	} {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("pragma %q: %w", pragma, err)
		}
	}

	s := &Store{path: path, db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	ddl := []string{
		`CREATE TABLE IF NOT EXISTS connections (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			provider TEXT NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			auth_type TEXT NOT NULL CHECK (auth_type IN ('oauth', 'api_key', 'noauth')),
			access_token TEXT,
			refresh_token TEXT,
			expires_at INTEGER,
			api_key TEXT,
			is_active INTEGER NOT NULL DEFAULT 1,
			provider_specific_data TEXT,
			account_id TEXT,
			email TEXT,
			unavailable_until INTEGER,
			backoff_level INTEGER NOT NULL DEFAULT 0,
			model_locks TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_connections_provider ON connections(provider)`,
		`CREATE INDEX IF NOT EXISTS idx_connections_active ON connections(provider, is_active)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			name TEXT NOT NULL UNIQUE,
			key_hash TEXT NOT NULL,
			prefix TEXT NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 1,
			last_used_at TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS combos (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			name TEXT NOT NULL UNIQUE,
			steps TEXT NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS model_aliases (
			alias TEXT PRIMARY KEY,
			provider TEXT NOT NULL,
			model TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS pricing_overrides (
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			input_cost_per_token REAL,
			output_cost_per_token REAL,
			PRIMARY KEY (provider, model)
		)`,
		`CREATE TABLE IF NOT EXISTS request_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			request_id TEXT NOT NULL,
			timestamp TEXT NOT NULL DEFAULT (datetime('now')),
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			connection_id TEXT,
			auth_type TEXT NOT NULL,
			input_tokens INTEGER,
			output_tokens INTEGER,
			cache_read_tokens INTEGER,
			cache_write_tokens INTEGER,
			total_tokens INTEGER,
			cost_usd REAL,
			latency_ms INTEGER,
			status_code INTEGER,
			error TEXT,
			source_format TEXT,
			target_format TEXT,
			rtk_enabled INTEGER,
			rtk_bytes_saved INTEGER,
			caveman_enabled INTEGER,
			combo_name TEXT,
			api_key_id TEXT,
			client_tool TEXT,
			FOREIGN KEY (connection_id) REFERENCES connections(id),
			FOREIGN KEY (api_key_id) REFERENCES api_keys(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_request_log_timestamp ON request_log(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_request_log_provider ON request_log(provider)`,
		`CREATE INDEX IF NOT EXISTS idx_request_log_model ON request_log(model)`,
		`CREATE INDEX IF NOT EXISTS idx_request_log_auth ON request_log(auth_type)`,
		`CREATE TABLE IF NOT EXISTS mcp_clients (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			name TEXT NOT NULL UNIQUE,
			transport TEXT NOT NULL CHECK (transport IN ('stdio', 'sse', 'streamable-http')),
			command TEXT,
			args TEXT,
			url TEXT,
			env TEXT,
			is_active INTEGER NOT NULL DEFAULT 1,
			health_status TEXT DEFAULT 'unknown',
			last_health_check TEXT,
			tool_manifest TEXT,
			manifest_updated_at TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS mcp_instances (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			name TEXT NOT NULL UNIQUE,
			server_key TEXT NOT NULL,
			launch_type TEXT NOT NULL CHECK (launch_type IN ('command', 'npx', 'docker', 'http')),
			transport TEXT NOT NULL CHECK (transport IN ('stdio', 'sse', 'streamable-http')),
			command TEXT,
			args TEXT,
			url TEXT,
			headers TEXT,
			env TEXT,
			cwd TEXT,
			account_label TEXT,
			is_active INTEGER NOT NULL DEFAULT 1,
			health_status TEXT DEFAULT 'unknown',
			last_health_check TEXT,
			tool_manifest TEXT,
			manifest_updated_at TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_mcp_instances_server_key ON mcp_instances(server_key)`,
		`CREATE TABLE IF NOT EXISTS mcp_oauth_accounts (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			instance_id TEXT NOT NULL,
			account_label TEXT NOT NULL DEFAULT 'default',
			subject TEXT,
			email TEXT,
			issuer TEXT,
			resource_uri TEXT,
			scopes TEXT,
			access_token TEXT NOT NULL,
			refresh_token TEXT,
			expires_at TEXT,
			auth_metadata TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now')),
			UNIQUE(instance_id, account_label),
			FOREIGN KEY (instance_id) REFERENCES mcp_instances(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_mcp_oauth_accounts_instance ON mcp_oauth_accounts(instance_id)`,
		`CREATE TABLE IF NOT EXISTS mcp_oauth_flows (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			instance_id TEXT NOT NULL,
			state_hash TEXT NOT NULL,
			code_verifier_secret TEXT NOT NULL,
			redirect_uri TEXT,
			authorization_url TEXT,
			resource_uri TEXT,
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			UNIQUE(instance_id, state_hash),
			FOREIGN KEY (instance_id) REFERENCES mcp_instances(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_mcp_oauth_flows_instance ON mcp_oauth_flows(instance_id)`,
	}

	for _, stmt := range ddl {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("exec ddl: %w", err)
		}
	}

	return nil
}
