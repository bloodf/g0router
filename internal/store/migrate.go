package store

import (
	"database/sql"
	"fmt"
	"regexp"
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
		{"model_aliases", `CREATE TABLE IF NOT EXISTS model_aliases (
			name TEXT PRIMARY KEY,
			target TEXT NOT NULL,
			created_at INTEGER NOT NULL
		)`},
		{"connection_model_locks", `CREATE TABLE IF NOT EXISTS connection_model_locks (
			connection_id TEXT NOT NULL,
			model TEXT NOT NULL,
			expires_at INTEGER NOT NULL,
			PRIMARY KEY (connection_id, model)
		)`},
		{"disabled_models", `CREATE TABLE IF NOT EXISTS disabled_models (
			provider_alias TEXT NOT NULL,
			model_id TEXT NOT NULL,
			PRIMARY KEY (provider_alias, model_id)
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
