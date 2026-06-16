package store

import (
	"path/filepath"
	"testing"
	"time"
)

// TestBackfillNoLockout proves that a legacy row written the OLD way (raw
// plaintext in `key`, empty `key_enc`) is migrated to hash+enc by the in-Open
// backfill, and that presenting the ORIGINAL raw key still resolves it — no
// operator lockout after the upgrade.
func TestBackfillNoLockout(t *testing.T) {
	dir := t.TempDir()
	secret, err := LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	path := filepath.Join(dir, "g0router.db")

	st, err := Open(path, secret)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}

	const rawKey = "g0vk-legacyraw"
	id, err := newID()
	if err != nil {
		t.Fatalf("newID: %v", err)
	}
	now := time.Now().Unix()
	// Legacy row: plaintext in key, key_enc defaults to ''.
	if _, err := st.DB().Exec(
		"INSERT INTO virtual_keys (id, key, name, config_json, is_active, team_id, created_at, updated_at) VALUES (?, ?, ?, ?, 1, '', ?, ?)",
		id, rawKey, "legacy", "{}", now, now,
	); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Reopen with the same secret/path: triggers the in-Open backfill.
	st2, err := Open(path, secret)
	if err != nil {
		t.Fatalf("second Open (backfill): %v", err)
	}
	defer st2.Close()

	// At rest: key is now the hash, key_enc is non-empty, raw not in either.
	var keyCol, keyEnc string
	if err := st2.DB().QueryRow(
		"SELECT key, key_enc FROM virtual_keys WHERE id = ?", id,
	).Scan(&keyCol, &keyEnc); err != nil {
		t.Fatalf("scan migrated row: %v", err)
	}
	if keyCol != sha256hex(rawKey) {
		t.Fatalf("key column = %q, want sha256hex(rawKey) = %q", keyCol, sha256hex(rawKey))
	}
	if keyEnc == "" {
		t.Fatal("key_enc still empty after backfill")
	}
	if keyCol == rawKey || keyEnc == rawKey {
		t.Fatalf("raw plaintext %q still present after backfill", rawKey)
	}

	// No lockout: the original raw key still resolves the row.
	got, err := st2.GetVirtualKeyByKey(rawKey)
	if err != nil {
		t.Fatalf("GetVirtualKeyByKey(rawKey) after backfill: %v", err)
	}
	if got.ID != id {
		t.Fatalf("resolved ID = %q, want %q", got.ID, id)
	}
	// DTO/display: the resolved .Key is the original raw key.
	if got.Key != rawKey {
		t.Fatalf("resolved .Key = %q, want original raw %q", got.Key, rawKey)
	}
}

// TestBackfillIdempotent proves a second backfill pass is a no-op: an already
// migrated row (non-empty key_enc) is skipped by the key_enc='' guard, so its
// key and key_enc columns are byte-identical (no re-encrypt, no re-hash).
func TestBackfillIdempotent(t *testing.T) {
	dir := t.TempDir()
	secret, err := LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	path := filepath.Join(dir, "g0router.db")

	st, err := Open(path, secret)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}

	const rawKey = "g0vk-idempotent"
	id, err := newID()
	if err != nil {
		t.Fatalf("newID: %v", err)
	}
	now := time.Now().Unix()
	if _, err := st.DB().Exec(
		"INSERT INTO virtual_keys (id, key, name, config_json, is_active, team_id, created_at, updated_at) VALUES (?, ?, ?, ?, 1, '', ?, ?)",
		id, rawKey, "legacy", "{}", now, now,
	); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Reopen: first backfill migrates the row.
	st2, err := Open(path, secret)
	if err != nil {
		t.Fatalf("second Open (backfill): %v", err)
	}
	defer st2.Close()

	var keyBefore, encBefore string
	if err := st2.DB().QueryRow(
		"SELECT key, key_enc FROM virtual_keys WHERE id = ?", id,
	).Scan(&keyBefore, &encBefore); err != nil {
		t.Fatalf("scan after first backfill: %v", err)
	}

	// Second backfill pass must skip the now-migrated row.
	if err := st2.backfillVirtualKeyEncryption(); err != nil {
		t.Fatalf("second backfillVirtualKeyEncryption: %v", err)
	}

	var keyAfter, encAfter string
	if err := st2.DB().QueryRow(
		"SELECT key, key_enc FROM virtual_keys WHERE id = ?", id,
	).Scan(&keyAfter, &encAfter); err != nil {
		t.Fatalf("scan after second backfill: %v", err)
	}
	if keyAfter != keyBefore {
		t.Fatalf("key column changed: %q -> %q", keyBefore, keyAfter)
	}
	if encAfter != encBefore {
		t.Fatalf("key_enc changed (re-encrypted): %q -> %q", encBefore, encAfter)
	}
}
