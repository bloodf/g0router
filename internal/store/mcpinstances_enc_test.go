package store

import "testing"

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
