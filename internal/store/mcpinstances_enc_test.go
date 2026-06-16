package store

import (
	"strings"
	"testing"
)

const mcpEnvSecretLiteral = "sk-secret-abc123"

// TestMCPInstanceEnvEncColumnExists is the bf-mcp-3 Step 1 RED guard: it asserts
// the additive env_json_enc column exists on mcp_instances. The column holds the
// reversible AES ciphertext of the instance env map (mirrors bf-gov-5's key_enc).
func TestMCPInstanceEnvEncColumnExists(t *testing.T) {
	st := newMCPTestStore(t)

	rows, err := st.DB().Query("PRAGMA table_info(mcp_instances)")
	if err != nil {
		t.Fatalf("PRAGMA table_info: %v", err)
	}
	defer rows.Close()

	found := false
	for rows.Next() {
		var (
			cid       int
			name      string
			ctype     string
			notnull   int
			dfltValue any
			pk        int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			t.Fatalf("scan table_info row: %v", err)
		}
		if name == "env_json_enc" {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table_info: %v", err)
	}
	if !found {
		t.Fatal("mcp_instances.env_json_enc column missing")
	}
}

// rawMCPInstanceEnvColumns reads the two raw env storage columns straight from
// the row, bypassing the store scan, so tests can assert what is physically at
// rest.
func rawMCPInstanceEnvColumns(t *testing.T, st *Store, id string) (envJSON, envEnc string) {
	t.Helper()
	if err := st.DB().QueryRow(
		"SELECT env_json, env_json_enc FROM mcp_instances WHERE id = ?", id,
	).Scan(&envJSON, &envEnc); err != nil {
		t.Fatalf("read raw env columns: %v", err)
	}
	return envJSON, envEnc
}

// TestMCPInstanceEnvEncryptedAtRest proves the secret env is AES-encrypted at
// rest: after Create, env_json_enc is non-empty, the legacy env_json column is
// drained to '{}', and the secret literal appears in NEITHER column.
func TestMCPInstanceEnvEncryptedAtRest(t *testing.T) {
	st := newMCPTestStore(t)

	created, err := st.CreateMCPInstance(&MCPInstance{
		Name:      "enc-at-rest",
		Transport: "stdio",
		Command:   "npx",
		Env:       map[string]string{"API_TOKEN": mcpEnvSecretLiteral},
		Status:    "stopped",
	})
	if err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}

	envJSON, envEnc := rawMCPInstanceEnvColumns(t, st, created.ID)
	if envEnc == "" {
		t.Fatal("env_json_enc empty after Create (env not encrypted at rest)")
	}
	if envJSON != "{}" {
		t.Fatalf("legacy env_json = %q, want drained to %q", envJSON, "{}")
	}
	if strings.Contains(envJSON, mcpEnvSecretLiteral) {
		t.Fatalf("secret literal present in plaintext env_json column: %q", envJSON)
	}
	if strings.Contains(envEnc, mcpEnvSecretLiteral) {
		t.Fatalf("secret literal present in env_json_enc column (not encrypted): %q", envEnc)
	}
}

// TestMCPInstanceEnvRoundTrip proves Create→Get/List return the original env map
// (decrypted) so the MCP server still launches with the right secrets.
func TestMCPInstanceEnvRoundTrip(t *testing.T) {
	st := newMCPTestStore(t)

	created, err := st.CreateMCPInstance(&MCPInstance{
		Name:      "round-trip",
		Transport: "stdio",
		Command:   "npx",
		Env:       map[string]string{"API_TOKEN": mcpEnvSecretLiteral, "NODE_ENV": "production"},
		Status:    "stopped",
	})
	if err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}

	got, err := st.GetMCPInstance(created.ID)
	if err != nil {
		t.Fatalf("GetMCPInstance: %v", err)
	}
	if got.Env["API_TOKEN"] != mcpEnvSecretLiteral {
		t.Fatalf("Get env API_TOKEN = %q, want %q", got.Env["API_TOKEN"], mcpEnvSecretLiteral)
	}
	if got.Env["NODE_ENV"] != "production" {
		t.Fatalf("Get env NODE_ENV = %q, want production", got.Env["NODE_ENV"])
	}

	list, err := st.ListMCPInstances()
	if err != nil {
		t.Fatalf("ListMCPInstances: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListMCPInstances len = %d, want 1", len(list))
	}
	if list[0].Env["API_TOKEN"] != mcpEnvSecretLiteral {
		t.Fatalf("List env API_TOKEN = %q, want %q", list[0].Env["API_TOKEN"], mcpEnvSecretLiteral)
	}
}

// TestMCPInstanceEnvUpdateNoPlaintext proves UpdateMCPInstance re-encrypts the
// rotated env and never leaves plaintext behind: the rotated secret round-trips
// via Get, env_json stays drained, and neither raw column holds the literal.
func TestMCPInstanceEnvUpdateNoPlaintext(t *testing.T) {
	st := newMCPTestStore(t)

	created, err := st.CreateMCPInstance(&MCPInstance{
		Name:      "update-no-leak",
		Transport: "stdio",
		Command:   "npx",
		Env:       map[string]string{"API_TOKEN": "sk-initial"},
		Status:    "stopped",
	})
	if err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}

	const rotated = "sk-rotated-xyz789"
	created.Env = map[string]string{"API_TOKEN": rotated}
	if err := st.UpdateMCPInstance(created); err != nil {
		t.Fatalf("UpdateMCPInstance: %v", err)
	}

	got, err := st.GetMCPInstance(created.ID)
	if err != nil {
		t.Fatalf("GetMCPInstance after update: %v", err)
	}
	if got.Env["API_TOKEN"] != rotated {
		t.Fatalf("rotated env not round-tripped: %q, want %q", got.Env["API_TOKEN"], rotated)
	}

	envJSON, envEnc := rawMCPInstanceEnvColumns(t, st, created.ID)
	if envJSON != "{}" {
		t.Fatalf("env_json after update = %q, want drained %q", envJSON, "{}")
	}
	if strings.Contains(envJSON, rotated) || strings.Contains(envEnc, rotated) {
		t.Fatalf("rotated secret leaked into a raw column (env_json=%q env_json_enc=%q)", envJSON, envEnc)
	}
	if envEnc == "" {
		t.Fatal("env_json_enc empty after update")
	}
}
