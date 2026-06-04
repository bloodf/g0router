package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")
	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	return s
}

func TestNewStoreCreatesDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer s.Close()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("db file missing: %v", err)
	}
}

func TestNewStoreCreatesParentDirs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "deep", "test.db")

	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer s.Close()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("db file missing: %v", err)
	}
}

func TestMigrateCreatesTables(t *testing.T) {
	s := openTestStore(t)

	expected := []string{
		"connections",
		"settings",
		"api_keys",
		"combos",
		"model_aliases",
		"pricing_overrides",
		"request_log",
		"mcp_clients",
	}

	for _, table := range expected {
		var name string
		err := s.db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?",
			table,
		).Scan(&name)
		if err == sql.ErrNoRows {
			t.Errorf("table %q not created", table)
			continue
		}
		if err != nil {
			t.Errorf("table %q query: %v", table, err)
			continue
		}
		if name != table {
			t.Errorf("table name = %q, want %q", name, table)
		}
	}
}

func TestMigrateCreatesIndexes(t *testing.T) {
	s := openTestStore(t)

	expected := []string{
		"idx_connections_provider",
		"idx_connections_active",
		"idx_request_log_timestamp",
		"idx_request_log_provider",
		"idx_request_log_model",
		"idx_request_log_auth",
	}

	for _, index := range expected {
		var name string
		err := s.db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type = 'index' AND name = ?",
			index,
		).Scan(&name)
		if err == sql.ErrNoRows {
			t.Errorf("index %q not created", index)
			continue
		}
		if err != nil {
			t.Errorf("index %q query: %v", index, err)
			continue
		}
		if name != index {
			t.Errorf("index name = %q, want %q", name, index)
		}
	}
}

func TestMigrateAddsMCPOAuthFlowClientCredentialColumns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE mcp_oauth_flows (
		id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
		instance_id TEXT NOT NULL,
		state_hash TEXT NOT NULL,
		code_verifier_secret TEXT NOT NULL,
		redirect_uri TEXT,
		authorization_url TEXT,
		resource_uri TEXT,
		expires_at TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(instance_id, state_hash)
	)`); err != nil {
		t.Fatalf("create old mcp_oauth_flows: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close setup db: %v", err)
	}

	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	instance := createOAuthTestInstance(t, s, "legacy-linear")

	if err := s.CreateMCPOAuthFlow(&MCPOAuthFlow{
		InstanceID:         instance.ID,
		State:              "state-1",
		CodeVerifierSecret: "verifier",
		RedirectURI:        "http://localhost/callback",
		AuthorizationURL:   "https://auth.example/authorize",
		ResourceURI:        "https://mcp.example",
		ClientID:           "client-123",
		ClientSecret:       "client-secret",
		ExpiresAt:          time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateMCPOAuthFlow after migration: %v", err)
	}
	flow, err := s.ConsumeMCPOAuthFlow(instance.ID, "state-1")
	if err != nil {
		t.Fatalf("ConsumeMCPOAuthFlow after migration: %v", err)
	}
	if flow.ClientID != "client-123" || flow.ClientSecret != "client-secret" {
		t.Fatal("client credentials did not survive migrated flow table")
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	s := openTestStore(t)

	if err := s.migrate(); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table'").Scan(&count); err != nil {
		t.Fatalf("count tables: %v", err)
	}
	if count < 8 {
		t.Errorf("table count = %d, want at least 8", count)
	}
}

func TestNewStoreEnablesWAL(t *testing.T) {
	s := openTestStore(t)

	var mode string
	if err := s.db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}
}

func TestStoreImplementsCloser(t *testing.T) {
	var _ interface{ Close() error } = (*Store)(nil)
}
