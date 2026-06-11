package store

import (
	"errors"
	"path/filepath"
	"regexp"
	"testing"
)

func TestAPIKeyCRUD(t *testing.T) {
	st := newTestStore(t)

	key1, err := st.CreateAPIKey("test-key-1")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if key1.ID == "" {
		t.Fatal("key ID empty")
	}
	if key1.Name != "test-key-1" {
		t.Fatalf("Name = %q, want %q", key1.Name, "test-key-1")
	}
	if key1.MachineID == "" {
		t.Fatal("MachineID empty")
	}
	if key1.Key == "" {
		t.Fatal("Key empty")
	}
	if !key1.IsActive {
		t.Fatal("new key should be active")
	}
	if matched, _ := regexp.MatchString(`^sk-[0-9a-f]{16}-[a-z0-9]{6}-[0-9a-f]{8}$`, key1.Key); !matched {
		t.Fatalf("created key %q is not a valid API key", key1.Key)
	}

	key2, err := st.CreateAPIKey("test-key-2")
	if err != nil {
		t.Fatalf("CreateAPIKey second: %v", err)
	}

	list, err := st.ListAPIKeys()
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}

	if err := st.SetAPIKeyActive(key1.ID, false); err != nil {
		t.Fatalf("SetAPIKeyActive: %v", err)
	}
	got, err := st.GetAPIKeyByID(key1.ID)
	if err != nil {
		t.Fatalf("GetAPIKeyByID: %v", err)
	}
	if got.IsActive {
		t.Fatal("key should be inactive after toggle")
	}

	if err := st.SetAPIKeyActive(key1.ID, true); err != nil {
		t.Fatalf("SetAPIKeyActive true: %v", err)
	}
	got, err = st.GetAPIKeyByID(key1.ID)
	if err != nil {
		t.Fatalf("GetAPIKeyByID after reactivate: %v", err)
	}
	if !got.IsActive {
		t.Fatal("key should be active after reactivate")
	}

	if err := st.DeleteAPIKey(key2.ID); err != nil {
		t.Fatalf("DeleteAPIKey: %v", err)
	}
	if _, err := st.GetAPIKeyByID(key2.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted key err = %v, want ErrNotFound", err)
	}

	list, err = st.ListAPIKeys()
	if err != nil {
		t.Fatalf("ListAPIKeys after delete: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) after delete = %d, want 1", len(list))
	}

	// Duplicate key value rejected.
	id, err := newID()
	if err != nil {
		t.Fatalf("newID: %v", err)
	}
	_, err = st.DB().Exec(
		"INSERT INTO api_keys (id, key, name, machine_id, is_active, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		id, key1.Key, "duplicate", key1.MachineID, 1, key1.CreatedAt,
	)
	if err == nil {
		t.Fatal("duplicate key value accepted")
	}
}

func TestAPIKeyLookupByKey(t *testing.T) {
	st := newTestStore(t)

	created, err := st.CreateAPIKey("lookup")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	got, err := st.GetAPIKeyByKey(created.Key)
	if err != nil {
		t.Fatalf("GetAPIKeyByKey: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("lookup ID = %q, want %q", got.ID, created.ID)
	}

	if _, err := st.GetAPIKeyByKey("sk-nonexistent"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing key err = %v, want ErrNotFound", err)
	}
}

func TestMigrationAdditive(t *testing.T) {
	dir := t.TempDir()
	secret, err := LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	path := filepath.Join(dir, "g0router.db")

	st, err := Open(path, secret)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// The api_keys table and index must exist after the first migration.
	var count int
	if err := st.DB().QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'api_keys'").Scan(&count); err != nil {
		st.Close()
		t.Fatalf("count tables: %v", err)
	}
	if count != 1 {
		st.Close()
		t.Fatalf("api_keys table count = %d, want 1", count)
	}

	if err := st.DB().QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_api_keys_key'").Scan(&count); err != nil {
		st.Close()
		t.Fatalf("count indexes: %v", err)
	}
	if count != 1 {
		st.Close()
		t.Fatalf("api_keys key index count = %d, want 1", count)
	}

	// Insert a row, then close and re-open the same database.
	created, err := st.CreateAPIKey("migration")
	if err != nil {
		st.Close()
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if err := st.SetAPIKeyActive(created.ID, false); err != nil {
		st.Close()
		t.Fatalf("SetAPIKeyActive: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	st, err = Open(path, secret)
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer st.Close()

	// Table and row must survive the re-run migrations.
	if err := st.DB().QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'api_keys'").Scan(&count); err != nil {
		t.Fatalf("count tables after re-open: %v", err)
	}
	if count != 1 {
		t.Fatalf("api_keys table count after re-open = %d, want 1", count)
	}

	if err := st.DB().QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_api_keys_key'").Scan(&count); err != nil {
		t.Fatalf("count indexes after re-open: %v", err)
	}
	if count != 1 {
		t.Fatalf("api_keys key index count after re-open = %d, want 1", count)
	}

	row, err := st.GetAPIKeyByID(created.ID)
	if err != nil {
		t.Fatalf("GetAPIKeyByID after re-open: %v", err)
	}
	if row.Key != created.Key {
		t.Fatalf("key after re-open = %q, want %q", row.Key, created.Key)
	}
	if row.IsActive {
		t.Fatal("row should still be inactive after re-open")
	}
}
