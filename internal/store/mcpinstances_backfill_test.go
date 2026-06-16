package store

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMCPInstanceBackfillNoLockout proves that an instance row written the OLD
// way (plaintext env_json, empty env_json_enc) is migrated by the in-Open
// backfill, and that afterwards GetMCPInstance(id).Env returns the ORIGINAL
// secret map (decrypted) so the MCP server still launches with the right env —
// no launch-survival regression after the upgrade.
func TestMCPInstanceBackfillNoLockout(t *testing.T) {
	dir := t.TempDir()
	secret, err := LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	dbPath := filepath.Join(dir, "test.db")

	st, err := Open(dbPath, secret)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}

	id, err := newID()
	if err != nil {
		t.Fatalf("newID: %v", err)
	}
	const legacySecret = "sk-legacy-secret"
	rawEnvJSON := `{"API_TOKEN":"` + legacySecret + `"}`
	now := time.Now().Unix()
	// Simulate a legacy row: plaintext in env_json, env_json_enc defaults to ''.
	if _, err := st.DB().Exec(
		`INSERT INTO mcp_instances
		 (id, client_id, name, transport, url, command, args_json, env_json, env_json_enc, status, created_at, updated_at)
		 VALUES (?, '', 'legacy', 'stdio', '', 'npx', '[]', ?, '', 'stopped', ?, ?)`,
		id, rawEnvJSON, now, now,
	); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Reopen with the same secret/path: triggers the in-Open backfill.
	st2, err := Open(dbPath, secret)
	if err != nil {
		t.Fatalf("second Open (backfill): %v", err)
	}
	defer st2.Close()

	// At rest: env_json_enc non-empty, env_json drained to '{}', secret absent.
	var envJSON, envEnc string
	if err := st2.DB().QueryRow(
		"SELECT env_json, env_json_enc FROM mcp_instances WHERE id = ?", id,
	).Scan(&envJSON, &envEnc); err != nil {
		t.Fatalf("scan migrated row: %v", err)
	}
	if envEnc == "" {
		t.Fatal("env_json_enc still empty after backfill")
	}
	if envJSON != "{}" {
		t.Fatalf("env_json = %q, want drained to %q", envJSON, "{}")
	}
	if strings.Contains(envJSON, legacySecret) || strings.Contains(envEnc, legacySecret) {
		t.Fatalf("legacy secret %q still present after backfill (env_json=%q env_json_enc=%q)",
			legacySecret, envJSON, envEnc)
	}

	// Launch-survival: the decrypted env still carries the original secret.
	got, err := st2.GetMCPInstance(id)
	if err != nil {
		t.Fatalf("GetMCPInstance after backfill: %v", err)
	}
	if got.Env["API_TOKEN"] != legacySecret {
		t.Fatalf("backfilled env API_TOKEN = %q, want original %q", got.Env["API_TOKEN"], legacySecret)
	}
}

// TestMCPInstanceBackfillIdempotent proves a second backfill pass is a no-op: an
// already-migrated row (non-empty env_json_enc) is skipped by the env_json_enc=''
// guard, so its env_json and env_json_enc columns are byte-identical (no
// re-encrypt). Mirrors TestBackfillIdempotent (bf-gov-5).
func TestMCPInstanceBackfillIdempotent(t *testing.T) {
	dir := t.TempDir()
	secret, err := LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	dbPath := filepath.Join(dir, "test.db")

	st, err := Open(dbPath, secret)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}

	id, err := newID()
	if err != nil {
		t.Fatalf("newID: %v", err)
	}
	now := time.Now().Unix()
	if _, err := st.DB().Exec(
		`INSERT INTO mcp_instances
		 (id, client_id, name, transport, url, command, args_json, env_json, env_json_enc, status, created_at, updated_at)
		 VALUES (?, '', 'legacy', 'stdio', '', 'npx', '[]', ?, '', 'stopped', ?, ?)`,
		id, `{"API_TOKEN":"sk-idempotent"}`, now, now,
	); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Reopen: first backfill migrates the row.
	st2, err := Open(dbPath, secret)
	if err != nil {
		t.Fatalf("second Open (backfill): %v", err)
	}
	defer st2.Close()

	var envBefore, encBefore string
	if err := st2.DB().QueryRow(
		"SELECT env_json, env_json_enc FROM mcp_instances WHERE id = ?", id,
	).Scan(&envBefore, &encBefore); err != nil {
		t.Fatalf("scan after first backfill: %v", err)
	}

	// Second backfill pass must skip the now-migrated row.
	if err := st2.backfillMCPInstanceEnvEncryption(); err != nil {
		t.Fatalf("second backfillMCPInstanceEnvEncryption: %v", err)
	}

	var envAfter, encAfter string
	if err := st2.DB().QueryRow(
		"SELECT env_json, env_json_enc FROM mcp_instances WHERE id = ?", id,
	).Scan(&envAfter, &encAfter); err != nil {
		t.Fatalf("scan after second backfill: %v", err)
	}
	if envAfter != envBefore {
		t.Fatalf("env_json changed: %q -> %q", envBefore, envAfter)
	}
	if encAfter != encBefore {
		t.Fatalf("env_json_enc changed (re-encrypted): %q -> %q", encBefore, encAfter)
	}
}
