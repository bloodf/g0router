package store

import (
	"os"
	"path/filepath"
	"testing"
)

// TestEnsureColumnQueryErrorOnClosedDB exercises the "read columns" error branch
// in ensureColumn (line 281): when the DB is closed, the PRAGMA query fails.
func TestEnsureColumnQueryErrorOnClosedDB(t *testing.T) {
	s := openTestStore(t)
	// Close immediately so the PRAGMA query inside ensureColumn will fail.
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// Should return error from the closed-DB query.
	if err := s.ensureColumn("connections", "some_new_col", "TEXT"); err == nil {
		t.Fatal("ensureColumn on closed DB should return error")
	}
}

// TestMigrateErrorOnClosedDB exercises the DDL exec error branch in migrate
// (the for-range loop over ddl): a closed DB causes the first Exec to fail and
// migrate returns the wrapped error.
func TestMigrateErrorOnClosedDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// Calling migrate on a closed DB should trigger the DDL Exec error path.
	if err := s.migrate(); err == nil {
		t.Fatal("migrate on closed DB should return error")
	}
}

// TestNewStoreParentIsFile exercises the MkdirAll error path in NewStore when
// a file exists where a directory is expected.
func TestNewStoreParentIsFile(t *testing.T) {
	dir := t.TempDir()
	// Create a regular file at a path NewStore would treat as a directory.
	blockingPath := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(blockingPath, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	// NewStore tries to MkdirAll(blockingPath, …) which will fail because
	// blockingPath is an existing file, not a directory.
	_, err := NewStore(filepath.Join(blockingPath, "test.db"))
	if err == nil {
		t.Fatal("NewStore with file-as-parent should return error")
	}
}
