package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	_ "modernc.org/sqlite"
)

type Store struct {
	path   string
	db     *sql.DB
	encKey []byte
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
		`CREATE INDEX IF NOT EXISTS idx_request_log_status_code ON request_log(status_code)`,
		`CREATE INDEX IF NOT EXISTS idx_request_log_source_format ON request_log(source_format)`,
		`CREATE INDEX IF NOT EXISTS idx_request_log_provider_model_ts ON request_log(provider, model, timestamp)`,
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
			client_id TEXT,
			client_secret TEXT,
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			UNIQUE(instance_id, state_hash),
			FOREIGN KEY (instance_id) REFERENCES mcp_instances(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_mcp_oauth_flows_instance ON mcp_oauth_flows(instance_id)`,
		`CREATE TABLE IF NOT EXISTS oauth_sessions (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			provider TEXT NOT NULL,
			state_hash TEXT NOT NULL UNIQUE,
			code_verifier TEXT,
			redirect_uri TEXT,
			account_label TEXT,
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TEXT NOT NULL DEFAULT (datetime('now')),
			actor_api_key_id TEXT,
			action TEXT NOT NULL,
			target TEXT,
			details TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_log_timestamp ON audit_log(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_log_actor ON audit_log(actor_api_key_id)`,
		`CREATE TABLE IF NOT EXISTS dashboard_users (
			id INTEGER PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			display_name TEXT,
			role TEXT NOT NULL DEFAULT 'user',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_dashboard_users_created_at ON dashboard_users(created_at)`,
		`CREATE TABLE IF NOT EXISTS dashboard_sessions (
			token_hash TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_seen_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			user_agent TEXT,
			ip TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_dashboard_sessions_user_id ON dashboard_sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_dashboard_sessions_expires_at ON dashboard_sessions(expires_at)`,
		`CREATE TABLE IF NOT EXISTS proxy_pools (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			protocol TEXT NOT NULL,
			host TEXT NOT NULL,
			port INTEGER NOT NULL,
			username TEXT,
			password_enc TEXT,
			is_active INTEGER DEFAULT 1,
			last_check_at DATETIME,
			last_check_status TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS disabled_models (
			id INTEGER PRIMARY KEY,
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(provider, model)
		)`,
		`CREATE TABLE IF NOT EXISTS custom_models (
			id INTEGER PRIMARY KEY,
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			display_name TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(provider, model)
		)`,
		`CREATE TABLE IF NOT EXISTS tunnel_config (
			id INTEGER PRIMARY KEY,
			type TEXT NOT NULL UNIQUE,
			is_enabled INTEGER DEFAULT 0,
			config_enc TEXT,
			url TEXT,
			status TEXT DEFAULT 'inactive',
			last_error TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS chat_sessions (
			id INTEGER PRIMARY KEY,
			title TEXT,
			model TEXT,
			provider TEXT,
			messages_json TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS teams (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			budget_usd REAL,
			budget_period TEXT DEFAULT 'monthly',
			budget_used_usd REAL DEFAULT 0,
			budget_reset_at DATETIME,
			rate_limit_rpm INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS virtual_keys (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			key_prefix TEXT NOT NULL,
			key_hash TEXT NOT NULL,
			budget_usd REAL,
			budget_period TEXT DEFAULT 'monthly',
			budget_used_usd REAL DEFAULT 0,
			budget_reset_at DATETIME,
			rate_limit_rpm INTEGER,
			rate_limit_tpm INTEGER,
			team_id INTEGER,
			is_active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, stmt := range ddl {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("exec ddl: %w", err)
		}
	}

	if err := s.ensureColumn("combos", "strategy", "TEXT NOT NULL DEFAULT 'fallback'"); err != nil {
		return err
	}
	if err := s.ensureColumn("mcp_oauth_flows", "client_id", "TEXT"); err != nil {
		return err
	}
	if err := s.ensureColumn("mcp_oauth_flows", "client_secret", "TEXT"); err != nil {
		return err
	}
	if err := s.ensureColumn("connections", "needs_reauth", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.ensureColumn("connections", "last_refresh_error", "TEXT"); err != nil {
		return err
	}
	if err := s.ensureColumn("api_keys", "expires_at", "INTEGER"); err != nil {
		return err
	}
	if err := s.ensureColumn("api_keys", "scopes", "TEXT"); err != nil {
		return err
	}
	if err := s.ensureColumn("api_keys", "rate_limit_rpm", "INTEGER"); err != nil {
		return err
	}
	if err := s.ensureColumn("api_keys", "rate_limit_tpm", "INTEGER"); err != nil {
		return err
	}
	if err := s.ensureColumn("api_keys", "daily_spend_cap_usd", "REAL"); err != nil {
		return err
	}
	if err := s.ensureColumn("connections", "proxy_pool_id", "INTEGER"); err != nil {
		return err
	}
	if err := s.ensureColumn("connections", "quota_limit", "REAL"); err != nil {
		return err
	}
	if err := s.ensureColumn("connections", "quota_remaining", "REAL"); err != nil {
		return err
	}
	if err := s.ensureColumn("request_log", "virtual_key_id", "TEXT"); err != nil {
		return err
	}

	return nil
}

var validIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func (s *Store) ensureColumn(table, column, definition string) error {
	if !validIdentifier.MatchString(table) {
		return fmt.Errorf("ensureColumn: invalid table identifier %q", table)
	}
	if !validIdentifier.MatchString(column) {
		return fmt.Errorf("ensureColumn: invalid column identifier %q", column)
	}
	rows, err := s.db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return fmt.Errorf("read %s columns: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &primaryKey); err != nil {
			return fmt.Errorf("scan %s columns: %w", table, err)
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate %s columns: %w", table, err)
	}
	if _, err := s.db.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + definition); err != nil {
		return fmt.Errorf("add %s.%s column: %w", table, column, err)
	}
	return nil
}
