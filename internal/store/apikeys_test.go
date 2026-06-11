package store

import (
	"errors"
	"testing"
)

func TestAPIKeyCRUD(t *testing.T) {
	st := newTestStore(t)
	machineID := "deadbeefdeadbeef"

	key1, err := st.CreateAPIKey("test-key-1", "sk-test-key-1", machineID)
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if key1.ID == "" {
		t.Fatal("key ID empty")
	}
	if key1.Name != "test-key-1" {
		t.Fatalf("Name = %q, want %q", key1.Name, "test-key-1")
	}
	if key1.MachineID != machineID {
		t.Fatalf("MachineID = %q, want %q", key1.MachineID, machineID)
	}
	if !key1.IsActive {
		t.Fatal("new key should be active")
	}

	key2, err := st.CreateAPIKey("test-key-2", "sk-test-key-2", machineID)
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
	if _, err := st.CreateAPIKey("duplicate", "sk-test-key-1", machineID); err == nil {
		t.Fatal("duplicate key accepted")
	}
}

func TestAPIKeyLookupByKey(t *testing.T) {
	st := newTestStore(t)
	machineID := "deadbeefdeadbeef"

	created, err := st.CreateAPIKey("lookup", "sk-lookup-key", machineID)
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
	st := newTestStore(t)

	// The api_keys table and index must exist after migration.
	var count int
	if err := st.DB().QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'api_keys'").Scan(&count); err != nil {
		t.Fatalf("count tables: %v", err)
	}
	if count != 1 {
		t.Fatalf("api_keys table count = %d, want 1", count)
	}

	if err := st.DB().QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_api_keys_key'").Scan(&count); err != nil {
		t.Fatalf("count indexes: %v", err)
	}
	if count != 1 {
		t.Fatalf("api_keys key index count = %d, want 1", count)
	}

	// Re-opening (which re-runs migrations) must remain a no-op.
	created, err := st.CreateAPIKey("migration", "sk-migration-key", "deadbeefdeadbeef")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if err := st.SetAPIKeyActive(created.ID, false); err != nil {
		t.Fatalf("SetAPIKeyActive: %v", err)
	}
}
