package store

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureColumnRejectsBadNames(t *testing.T) {
	st := newTestStore(t)

	badNames := []struct{ table, column string }{
		{"settings;", "extra"},
		{"settings", "extra;drop table users--"},
		{"SETTINGS", "extra"},
		{"1settings", "extra"},
		{"settings", "1extra"},
		{"settings", "extra col"},
	}

	for _, tc := range badNames {
		err := ensureColumn(st.DB(), tc.table, tc.column, "TEXT NOT NULL DEFAULT ''")
		if err == nil {
			t.Fatalf("ensureColumn(%q, %q) should reject bad name", tc.table, tc.column)
		}
	}
}

func TestForeignKeysExistOnFreshDB(t *testing.T) {
	st := newTestStore(t)

	var count int
	if err := st.DB().QueryRow("SELECT COUNT(*) FROM pragma_foreign_key_list('connections')").Scan(&count); err != nil {
		t.Fatalf("pragma foreign_key_list connections: %v", err)
	}
	if count == 0 {
		t.Fatal("connections table has no foreign keys")
	}

	if err := st.DB().QueryRow("SELECT COUNT(*) FROM pragma_foreign_key_list('sessions')").Scan(&count); err != nil {
		t.Fatalf("pragma foreign_key_list sessions: %v", err)
	}
	if count == 0 {
		t.Fatal("sessions table has no foreign keys")
	}
}

func TestForeignKeyMigrationPreservesData(t *testing.T) {
	dir := t.TempDir()
	secret, err := LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	path := filepath.Join(dir, "test.db")

	// Create old schema (no FKs) manually and seed data.
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		t.Fatalf("enable wal: %v", err)
	}

	oldSchema := []string{
		`CREATE TABLE providers (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			base_url TEXT NOT NULL DEFAULT '',
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE TABLE connections (
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
		)`,
	}
	for _, stmt := range oldSchema {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("create old table: %v", err)
		}
	}
	if _, err := db.Exec("INSERT INTO providers VALUES ('p1', 'OpenAI', 'openai', '', 1, 1, 1)"); err != nil {
		t.Fatalf("insert provider: %v", err)
	}
	if _, err := db.Exec("INSERT INTO connections VALUES ('c1', 'p1', 'main', 'api_key', '', '', '', 0, '', 1, 1)"); err != nil {
		t.Fatalf("insert connection: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close raw db: %v", err)
	}

	// Re-open via store.Open to trigger migration.
	st, err := Open(path, secret)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	var count int
	if err := st.DB().QueryRow("SELECT COUNT(*) FROM providers").Scan(&count); err != nil || count != 1 {
		t.Fatalf("providers count = %d, err = %v", count, err)
	}
	if err := st.DB().QueryRow("SELECT COUNT(*) FROM connections").Scan(&count); err != nil || count != 1 {
		t.Fatalf("connections count = %d, err = %v", count, err)
	}
	if err := st.DB().QueryRow("SELECT COUNT(*) FROM pragma_foreign_key_list('connections')").Scan(&count); err != nil || count == 0 {
		t.Fatalf("connections fk missing: count = %d, err = %v", count, err)
	}
}

func TestDeleteProviderCascadesConnections(t *testing.T) {
	st := newTestStore(t)

	p := &ProviderRecord{Name: "OpenAI", Type: "openai", Enabled: true}
	if err := st.CreateProvider(p); err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}
	c := &Connection{ProviderID: p.ID, Name: "main", Kind: "api_key"}
	if err := st.CreateConnection(c); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if err := st.DeleteProvider(p.ID); err != nil {
		t.Fatalf("DeleteProvider: %v", err)
	}
	if _, err := st.GetConnection(c.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected connection deleted, got err = %v", err)
	}
}

func TestForeignKeysEnabled(t *testing.T) {
	st := newTestStore(t)

	var on int
	if err := st.DB().QueryRow("PRAGMA foreign_keys").Scan(&on); err != nil {
		t.Fatalf("pragma foreign_keys: %v", err)
	}
	if on != 1 {
		t.Fatalf("foreign_keys = %d, want 1", on)
	}
}

func TestMigrateUsageTables(t *testing.T) {
	st := newTestStore(t)

	wantTables := map[string][]string{
		"request_log": {
			"id", "timestamp", "provider", "model", "connection_id",
			"api_key", "endpoint", "prompt_tokens", "completion_tokens",
			"cost", "status", "tokens", "meta",
		},
		"usage_daily": {
			"date_key", "data",
		},
		"request_details": {
			"id", "timestamp", "provider", "model", "connection_id",
			"status", "data",
		},
		"kv": {
			"scope", "key", "value",
		},
	}

	for table, wantCols := range wantTables {
		rows, err := st.DB().Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
		if err != nil {
			t.Fatalf("table_info %s: %v", table, err)
		}
		cols := map[string]bool{}
		for rows.Next() {
			var cid, notNull, primaryKey int
			var name, typ string
			var dflt sql.NullString
			if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &primaryKey); err != nil {
				t.Fatalf("scan table_info %s: %v", table, err)
			}
			cols[name] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			t.Fatalf("iterate table_info %s: %v", table, err)
		}
		for _, c := range wantCols {
			if !cols[c] {
				t.Errorf("table %s missing column %q", table, c)
			}
		}
	}

	wantIndexes := map[string][]string{
		"request_log": {
			"idx_request_log_timestamp",
			"idx_request_log_provider",
			"idx_request_log_model",
			"idx_request_log_connection_id",
		},
		"request_details": {
			"idx_request_details_timestamp",
			"idx_request_details_provider",
			"idx_request_details_model",
			"idx_request_details_connection_id",
		},
	}

	for table, wantIdxs := range wantIndexes {
		rows, err := st.DB().Query(fmt.Sprintf("PRAGMA index_list(%s)", table))
		if err != nil {
			t.Fatalf("index_list %s: %v", table, err)
		}
		idxs := map[string]bool{}
		for rows.Next() {
			var seq int
			var name, origin string
			var partial, unique int
			if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
				t.Fatalf("scan index_list %s: %v", table, err)
			}
			idxs[name] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			t.Fatalf("iterate index_list %s: %v", table, err)
		}
		for _, idx := range wantIdxs {
			if !idxs[idx] {
				t.Errorf("table %s missing index %q", table, idx)
			}
		}
	}

	// Verify DESC ordering on the timestamp indexes via sqlite_master.
	for _, idx := range []string{"idx_request_log_timestamp", "idx_request_details_timestamp"} {
		var sql string
		if err := st.DB().QueryRow(
			"SELECT sql FROM sqlite_master WHERE type = 'index' AND name = ?", idx,
		).Scan(&sql); err != nil {
			t.Fatalf("lookup sqlite_master for %s: %v", idx, err)
		}
		if !strings.Contains(strings.ToLower(sql), "desc") {
			t.Errorf("index %s is not DESC: %s", idx, sql)
		}
	}
}
