package store

import (
	"testing"
)

// TestScanAPIKeyBadScopesJSON exercises the decodeJSON error branch in scanAPIKey
// (line 137-138): a key with malformed JSON in the scopes column returns an error.
func TestScanAPIKeyBadScopesJSON(t *testing.T) {
	s := openTestStore(t)

	// Insert a key with invalid JSON in scopes column directly.
	if _, err := s.db.Exec(`
		INSERT INTO api_keys (id, name, key_hash, prefix, is_active, scopes)
		VALUES ('bad-scope-key', 'bad-scope', 'hash', 'g0r_bad', 1, 'not-valid-json')
	`); err != nil {
		t.Fatalf("INSERT malformed key: %v", err)
	}

	// GetAPIKey should fail when scanning the malformed scopes.
	_, err := s.GetAPIKey("bad-scope-key")
	if err == nil {
		t.Fatal("GetAPIKey with bad scopes JSON should return error")
	}
}

// TestListAPIKeysScanBadScopesJSON exercises the scan error branch in ListAPIKeys
// (line 210-211): when a key has malformed scopes JSON, scan returns an error.
func TestListAPIKeysScanBadScopesJSON(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.db.Exec(`
		INSERT INTO api_keys (id, name, key_hash, prefix, is_active, scopes)
		VALUES ('bad-scope-2', 'bad-scope-2', 'hash2', 'g0r_b', 1, '{invalid}')
	`); err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	// ListAPIKeys should fail when scanning the malformed key.
	_, err := s.ListAPIKeys()
	if err == nil {
		t.Fatal("ListAPIKeys with bad scopes should return error")
	}
}
