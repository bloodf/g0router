package store

import (
	"database/sql"
	"fmt"
	"regexp"
	"time"
)

var identifierRe = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// migrate runs all schema migrations. Migrations are additive-only:
// tables are created if missing and columns are appended via ensureColumn.
// Existing tables and columns are never modified or dropped.
func migrate(db *sql.DB) error {
	tables := []struct {
		name   string
		create string
	}{
		{"users", `CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`},
		{"sessions", `CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			expires_at INTEGER NOT NULL,
			created_at INTEGER NOT NULL
		)`},
		{"settings", `CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at INTEGER NOT NULL
		)`},
		{"providers", `CREATE TABLE IF NOT EXISTS providers (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			base_url TEXT NOT NULL DEFAULT '',
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`},
		{"connections", `CREATE TABLE IF NOT EXISTS connections (
			id TEXT PRIMARY KEY,
			provider_id TEXT NOT NULL,
			name TEXT NOT NULL,
			kind TEXT NOT NULL,
			secret_enc TEXT NOT NULL DEFAULT '',
			access_token_enc TEXT NOT NULL DEFAULT '',
			refresh_token_enc TEXT NOT NULL DEFAULT '',
			expires_at INTEGER NOT NULL DEFAULT 0,
			metadata TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`},
		{"oauth_sessions", `CREATE TABLE IF NOT EXISTS oauth_sessions (
			state TEXT PRIMARY KEY,
			provider TEXT NOT NULL,
			verifier_enc TEXT NOT NULL DEFAULT '',
			expires_at INTEGER NOT NULL,
			created_at INTEGER NOT NULL
		)`},
		{"api_keys", `CREATE TABLE IF NOT EXISTS api_keys (
			id TEXT PRIMARY KEY,
			key TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			machine_id TEXT NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at INTEGER NOT NULL
		)`},
		{"virtual_keys", `CREATE TABLE IF NOT EXISTS virtual_keys (
			id TEXT PRIMARY KEY,
			key TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			config_json TEXT NOT NULL DEFAULT '{}',
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`},
		{"model_aliases", `CREATE TABLE IF NOT EXISTS model_aliases (
			name TEXT PRIMARY KEY,
			target TEXT NOT NULL,
			created_at INTEGER NOT NULL
		)`},
		{"connection_model_locks", `CREATE TABLE IF NOT EXISTS connection_model_locks (
			connection_id TEXT NOT NULL,
			provider_id TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL,
			expires_at INTEGER NOT NULL,
			PRIMARY KEY (connection_id, model)
		)`},
		{"disabled_models", `CREATE TABLE IF NOT EXISTS disabled_models (
			provider_alias TEXT NOT NULL,
			model_id TEXT NOT NULL,
			PRIMARY KEY (provider_alias, model_id)
		)`},
		{"combos", `CREATE TABLE IF NOT EXISTS combos (
			name TEXT PRIMARY KEY,
			models_json TEXT NOT NULL DEFAULT '[]',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`},
		{"request_log", `CREATE TABLE IF NOT EXISTS request_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TEXT NOT NULL,
			provider TEXT,
			model TEXT,
			connection_id TEXT,
			api_key TEXT,
			endpoint TEXT,
			prompt_tokens INTEGER NOT NULL DEFAULT 0,
			completion_tokens INTEGER NOT NULL DEFAULT 0,
			cost REAL NOT NULL DEFAULT 0,
			status TEXT,
			tokens TEXT NOT NULL DEFAULT '{}',
			meta TEXT NOT NULL DEFAULT '{}'
		)`},
		{"usage_daily", `CREATE TABLE IF NOT EXISTS usage_daily (
			date_key TEXT PRIMARY KEY,
			data TEXT NOT NULL
		)`},
		{"request_details", `CREATE TABLE IF NOT EXISTS request_details (
			id TEXT PRIMARY KEY,
			timestamp TEXT NOT NULL,
			provider TEXT,
			model TEXT,
			connection_id TEXT,
			status TEXT,
			data TEXT NOT NULL
		)`},
		{"kv", `CREATE TABLE IF NOT EXISTS kv (
			scope TEXT NOT NULL,
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			PRIMARY KEY (scope, key)
		)`},
		{"teams", `CREATE TABLE IF NOT EXISTS teams (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			budget_usd REAL NOT NULL DEFAULT 0,
			budget_used_usd REAL NOT NULL DEFAULT 0,
			budget_period TEXT NOT NULL DEFAULT 'monthly',
			rate_limit_rpm INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`},
		{"audit_log", `CREATE TABLE IF NOT EXISTS audit_log (
			id TEXT PRIMARY KEY,
			timestamp TEXT NOT NULL,
			actor TEXT NOT NULL,
			action TEXT NOT NULL,
			target TEXT NOT NULL DEFAULT '',
			details TEXT NOT NULL DEFAULT ''
		)`},
		{"feature_flags", `CREATE TABLE IF NOT EXISTS feature_flags (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			enabled INTEGER NOT NULL DEFAULT 0,
			description TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`},
		{"prompt_templates", `CREATE TABLE IF NOT EXISTS prompt_templates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			system_prompt TEXT NOT NULL DEFAULT '',
			models_json TEXT NOT NULL DEFAULT '[]',
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`},
		{"guardrails", `CREATE TABLE IF NOT EXISTS guardrails (
			id INTEGER PRIMARY KEY,
			guardrails_enabled INTEGER NOT NULL DEFAULT 0,
			guardrails_blocklist_json TEXT NOT NULL DEFAULT '[]',
			pii_redaction_enabled INTEGER NOT NULL DEFAULT 0,
			pii_redaction_types_json TEXT NOT NULL DEFAULT '[]',
			updated_at INTEGER NOT NULL DEFAULT 0
		)`},
		{"alert_channels", `CREATE TABLE IF NOT EXISTS alert_channels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			channel_type TEXT NOT NULL DEFAULT 'webhook',
			config_enc TEXT NOT NULL DEFAULT '',
			events_json TEXT NOT NULL DEFAULT '[]',
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL
		)`},
		{"proxy_pools", `CREATE TABLE IF NOT EXISTS proxy_pools (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			protocol TEXT NOT NULL DEFAULT 'http',
			host TEXT NOT NULL,
			port INTEGER NOT NULL DEFAULT 0,
			username TEXT NOT NULL DEFAULT '',
			password_enc TEXT NOT NULL DEFAULT '',
			is_active INTEGER NOT NULL DEFAULT 1,
			last_check_status TEXT NOT NULL DEFAULT '',
			last_check_at TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`},
		{"tunnels", `CREATE TABLE IF NOT EXISTS tunnels (
			type TEXT PRIMARY KEY,
			is_enabled INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'inactive',
			url TEXT NOT NULL DEFAULT '',
			token_enc TEXT NOT NULL DEFAULT '',
			mode TEXT NOT NULL DEFAULT '',
			last_error TEXT NOT NULL DEFAULT '',
			updated_at INTEGER NOT NULL DEFAULT 0
		)`},
		{"mitm_tools", `CREATE TABLE IF NOT EXISTS mitm_tools (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 0,
			dns_override TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'inactive',
			updated_at INTEGER NOT NULL DEFAULT 0
		)`},
		{"mcp_clients", `CREATE TABLE IF NOT EXISTS mcp_clients (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT '',
			config_json TEXT NOT NULL DEFAULT '{}',
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`},
		{"mcp_instances", `CREATE TABLE IF NOT EXISTS mcp_instances (
			id TEXT PRIMARY KEY,
			client_id TEXT NOT NULL DEFAULT '',
			name TEXT NOT NULL,
			transport TEXT NOT NULL DEFAULT 'stdio',
			url TEXT NOT NULL DEFAULT '',
			command TEXT NOT NULL DEFAULT '',
			args_json TEXT NOT NULL DEFAULT '[]',
			env_json TEXT NOT NULL DEFAULT '{}',
			status TEXT NOT NULL DEFAULT 'stopped',
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`},
		{"mcp_oauth_accounts", `CREATE TABLE IF NOT EXISTS mcp_oauth_accounts (
			id TEXT PRIMARY KEY,
			instance_id TEXT NOT NULL DEFAULT '',
			server_url TEXT NOT NULL DEFAULT '',
			access_token_enc TEXT NOT NULL DEFAULT '',
			refresh_token_enc TEXT NOT NULL DEFAULT '',
			expires_at INTEGER NOT NULL DEFAULT 0,
			scope TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`},
		{"mcp_oauth_flows", `CREATE TABLE IF NOT EXISTS mcp_oauth_flows (
			state TEXT PRIMARY KEY,
			instance_id TEXT NOT NULL DEFAULT '',
			server_url TEXT NOT NULL DEFAULT '',
			verifier_enc TEXT NOT NULL DEFAULT '',
			redirect_uri TEXT NOT NULL DEFAULT '',
			expires_at INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL DEFAULT 0
		)`},
		{"mcp_tool_groups", `CREATE TABLE IF NOT EXISTS mcp_tool_groups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			tool_ids TEXT NOT NULL DEFAULT '[]',
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL DEFAULT ''
		)`},
		// virtual_key_mcp_configs is the additive VK↔MCP assignment junction
		// (bf-mcp-2 / PAR-BF-MCP-033): a many-to-many scope binding read at
		// request time to narrow a VK's server-mode tools/list+tools/call. Mirrors
		// the mcp_tool_groups INTEGER-PK additive-table shape.
		{"virtual_key_mcp_configs", `CREATE TABLE IF NOT EXISTS virtual_key_mcp_configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			virtual_key_id TEXT NOT NULL DEFAULT '',
			mcp_client_id TEXT NOT NULL DEFAULT '',
			tools_to_execute_json TEXT NOT NULL DEFAULT '[]',
			tools_to_auto_execute_json TEXT NOT NULL DEFAULT '[]',
			config_hash TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`},
		// Aliases admin table (w7-route-a, ESC-ALIAS-SHAPE). Distinct from the
		// gateway model_aliases resolver table: this carries the id-keyed UI
		// {id,alias,provider,model} shape the frozen /aliases page reads.
		{"aliases", `CREATE TABLE IF NOT EXISTS aliases (
			id TEXT PRIMARY KEY,
			alias TEXT NOT NULL,
			provider TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`},
		// Routing-rules admin table (w7-route-a). Admin CRUD only — rules are not
		// yet applied to live inference (tracked follow-up).
		{"routing_rules", `CREATE TABLE IF NOT EXISTS routing_rules (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			priority INTEGER NOT NULL DEFAULT 0,
			cond_field TEXT NOT NULL DEFAULT '',
			cond_operator TEXT NOT NULL DEFAULT '',
			cond_value TEXT NOT NULL DEFAULT '',
			target_provider TEXT NOT NULL DEFAULT '',
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`},
		// Model-limits admin table (w7-route-a, ESC-IDTYPE). INTEGER PK to mirror
		// the numeric UI ModelLimit.id; allowed_key_ids []string is a JSON blob.
		{"model_limits", `CREATE TABLE IF NOT EXISTS model_limits (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			model TEXT NOT NULL,
			max_tokens INTEGER NOT NULL DEFAULT 0,
			max_rpm INTEGER NOT NULL DEFAULT 0,
			key_ids_json TEXT NOT NULL DEFAULT '[]',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`},
		// Combos admin table (w7-route-a, ESC-COMBOS). Separate from the engine
		// combos table: this carries the id-keyed UI shape
		// {id,name,strategy,steps[{provider,model}],is_active} the frozen /combos
		// page reads; steps are a JSON blob. The engine combos table + /v1/models
		// lister stay intact, fed by a best-effort mirror-write.
		{"combos_admin", `CREATE TABLE IF NOT EXISTS combos_admin (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			strategy TEXT NOT NULL DEFAULT 'fallback',
			steps_json TEXT NOT NULL DEFAULT '[]',
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`},
		// Custom-model metadata (w7-misc). User-defined model rows the
		// ModelSelectModal manages; config_json carries misc fields (cost/context).
		// No secret fields — custom-model metadata is not sensitive.
		{"custom_models", `CREATE TABLE IF NOT EXISTS custom_models (
			id TEXT PRIMARY KEY,
			provider TEXT NOT NULL DEFAULT '',
			model_id TEXT NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			config_json TEXT NOT NULL DEFAULT '{}',
			created_at INTEGER NOT NULL
		)`},
		// OIDC client secret at rest (w7-misc). A dedicated single-row table (id=1)
		// holding the encrypted secret in oidc_secret_enc (the *_enc precedent),
		// keeping the secret out of the plaintext settings flat map.
		{"oidc_secret", `CREATE TABLE IF NOT EXISTS oidc_secret (
			id INTEGER PRIMARY KEY,
			oidc_secret_enc TEXT NOT NULL DEFAULT ''
		)`},
		// Semantic cache (bf-core-2 / phase-19:34-46). Backs the exact-key-hash
		// read-through/write-through cache. cache_key = sha256(normalized prompt +
		// model). embedding_json is retained for forward-compatibility (the
		// deferred semantic-similarity half populates it; the hash-only cache
		// writes "[]"). DDL is verbatim from phase-19 — no invented columns.
		{"semantic_cache", `CREATE TABLE IF NOT EXISTS semantic_cache (
			id INTEGER PRIMARY KEY,
			cache_key TEXT NOT NULL,
			embedding_json TEXT NOT NULL,
			model TEXT NOT NULL,
			response_json TEXT NOT NULL,
			expires_at DATETIME,
			hit_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`},
	}

	for _, t := range tables {
		if _, err := db.Exec(t.create); err != nil {
			return fmt.Errorf("create table %s: %w", t.name, err)
		}
	}

	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_api_keys_key ON api_keys(key)"); err != nil {
		return fmt.Errorf("create api_keys key index: %w", err)
	}

	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_virtual_keys_key ON virtual_keys(key)"); err != nil {
		return fmt.Errorf("create virtual_keys key index: %w", err)
	}

	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_audit_log_timestamp ON audit_log(timestamp DESC)"); err != nil {
		return fmt.Errorf("create audit_log timestamp index: %w", err)
	}

	// Semantic-cache indexes (bf-core-2 / phase-19:45-46). Additive, IF NOT EXISTS.
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_semantic_cache_model ON semantic_cache(model)"); err != nil {
		return fmt.Errorf("create semantic_cache model index: %w", err)
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_semantic_cache_expires ON semantic_cache(expires_at)"); err != nil {
		return fmt.Errorf("create semantic_cache expires index: %w", err)
	}

	usageIndexes := []struct{ name, table, columns string }{
		{"idx_request_log_timestamp", "request_log", "timestamp DESC"},
		{"idx_request_log_provider", "request_log", "provider"},
		{"idx_request_log_model", "request_log", "model"},
		{"idx_request_log_connection_id", "request_log", "connection_id"},
		{"idx_request_details_timestamp", "request_details", "timestamp DESC"},
		{"idx_request_details_provider", "request_details", "provider"},
		{"idx_request_details_model", "request_details", "model"},
		{"idx_request_details_connection_id", "request_details", "connection_id"},
	}
	for _, idx := range usageIndexes {
		stmt := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s)", idx.name, idx.table, idx.columns)
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("create index %s: %w", idx.name, err)
		}
	}

	// Seed feature flags (bf-core-2 / D8). Idempotent INSERT OR IGNORE on the
	// UNIQUE key: the row is created OFF by default on a fresh store and never
	// overwritten on subsequent migrations, so an admin toggle persists.
	for _, f := range []struct{ key, description string }{
		{"semantic_cache", "Exact-key-hash semantic response cache for non-streaming chat"},
		{"vk_mandatory", "Reject requests with no resolved virtual key (mandatory-VK mode)"},
	} {
		if _, err := db.Exec(
			`INSERT OR IGNORE INTO feature_flags (key, enabled, description, created_at)
			 VALUES (?, 0, ?, ?)`,
			f.key, f.description, time.Now().UTC().Format(time.RFC3339),
		); err != nil {
			return fmt.Errorf("seed feature flag %s: %w", f.key, err)
		}
	}

	// FK migrations must run before ensureColumn so the recreate+copy step
	// sees only the base schema columns, not any newly appended ones.
	if err := migrateForeignKeys(db); err != nil {
		return fmt.Errorf("migrate foreign keys: %w", err)
	}

	// Additive column migrations (w4-c, PAR-ROUTE-014/015).
	for _, col := range []struct{ table, column, decl string }{
		{"connections", "backoff_level", "INTEGER NOT NULL DEFAULT 0"},
		{"connections", "rate_limited_until", "INTEGER NOT NULL DEFAULT 0"},
		{"connections", "last_error", "TEXT NOT NULL DEFAULT ''"},
		{"connections", "proxy_pool_id", "TEXT NOT NULL DEFAULT ''"},
		{"users", "display_name", "TEXT NOT NULL DEFAULT ''"},
		{"users", "role", "TEXT NOT NULL DEFAULT 'user'"},
		// Provider-node prefix-routing columns (w7-platnodes, PAR-PLAT-014). A
		// provider node is a providers row carrying a routing prefix and an API
		// type (openai/anthropic); non-node providers keep the '' defaults.
		{"providers", "prefix", "TEXT NOT NULL DEFAULT ''"},
		{"providers", "api_type", "TEXT NOT NULL DEFAULT ''"},
		// VK→Team governance link (bf-gov-1, PAR-BF-GOV-001/D4). Empty string is
		// the "un-teamed" sentinel; the hierarchy check skips the Team tier when
		// team_id == ''.
		{"virtual_keys", "team_id", "TEXT NOT NULL DEFAULT ''"},
		// VK bearer value encrypted at rest (bf-gov-5, PAR-BF-GOV-006). The
		// reversible AES ciphertext of the raw VK value; the `key` column is
		// repurposed to hold sha256hex(raw) for lookup. Empty string default
		// marks a legacy un-migrated row for the one-time backfill on Open().
		{"virtual_keys", "key_enc", "TEXT NOT NULL DEFAULT ''"},
		// MCP instance env secrets encrypted at rest (bf-mcp-3, PAR-BF-MCP-080).
		// The reversible AES ciphertext of the instance env map (stdio MCP server
		// process env vars routinely carry API tokens/keys). Empty string default
		// marks a legacy un-migrated row for the one-time backfill on Open(); the
		// legacy plaintext env_json column is drained to '{}' on write/backfill.
		{"mcp_instances", "env_json_enc", "TEXT NOT NULL DEFAULT ''"},
	} {
		if err := ensureColumn(db, col.table, col.column, col.decl); err != nil {
			return fmt.Errorf("ensure column %s.%s: %w", col.table, col.column, err)
		}
	}

	return nil
}

func migrateForeignKeys(db *sql.DB) error {
	if err := ensureForeignKey(db, "sessions", `CREATE TABLE sessions (
		token TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		expires_at INTEGER NOT NULL,
		created_at INTEGER NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
	)`); err != nil {
		return fmt.Errorf("sessions fk: %w", err)
	}

	if err := ensureForeignKey(db, "connections", `CREATE TABLE connections (
		id TEXT PRIMARY KEY,
		provider_id TEXT NOT NULL,
		name TEXT NOT NULL,
		kind TEXT NOT NULL,
		secret_enc TEXT NOT NULL DEFAULT '',
		access_token_enc TEXT NOT NULL DEFAULT '',
		refresh_token_enc TEXT NOT NULL DEFAULT '',
		expires_at INTEGER NOT NULL DEFAULT 0,
		metadata TEXT NOT NULL DEFAULT '',
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE
	)`); err != nil {
		return fmt.Errorf("connections fk: %w", err)
	}

	return nil
}

func ensureForeignKey(db *sql.DB, table, newSchema string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA foreign_key_list(%s)", table))
	if err != nil {
		return fmt.Errorf("check fk list %s: %w", table, err)
	}
	defer rows.Close()
	if rows.Next() {
		return nil
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate fk list %s: %w", table, err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx for %s: %w", table, err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO _%s_old", table, table)); err != nil {
		return fmt.Errorf("rename %s: %w", table, err)
	}
	if _, err := tx.Exec(newSchema); err != nil {
		return fmt.Errorf("create new %s: %w", table, err)
	}
	if _, err := tx.Exec(fmt.Sprintf("INSERT INTO %s SELECT * FROM _%s_old", table, table)); err != nil {
		return fmt.Errorf("copy %s data: %w", table, err)
	}
	if _, err := tx.Exec(fmt.Sprintf("DROP TABLE _%s_old", table)); err != nil {
		return fmt.Errorf("drop old %s: %w", table, err)
	}

	return tx.Commit()
}

// ensureColumn appends column to table if it does not exist yet.
// It never alters or drops existing columns (additive-only policy).
func ensureColumn(db *sql.DB, table, column, decl string) error {
	if !identifierRe.MatchString(table) {
		return fmt.Errorf("invalid table name %q", table)
	}
	if !identifierRe.MatchString(column) {
		return fmt.Errorf("invalid column name %q", column)
	}

	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return fmt.Errorf("table_info %s: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name, typ  string
			notNull    int
			dfltValue  sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dfltValue, &primaryKey); err != nil {
			return fmt.Errorf("scan table_info %s: %w", table, err)
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate table_info %s: %w", table, err)
	}

	if _, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, decl)); err != nil {
		return fmt.Errorf("add column %s.%s: %w", table, column, err)
	}
	return nil
}
