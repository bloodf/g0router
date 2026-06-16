package store

import (
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

// TestVirtualKeyKeyEncColumnExists asserts the additive key_enc column is
// present on the virtual_keys table (the at-rest AES column for the VK value).
func TestVirtualKeyKeyEncColumnExists(t *testing.T) {
	st := newTestStore(t)

	rows, err := st.DB().Query("PRAGMA table_info(virtual_keys)")
	if err != nil {
		t.Fatalf("PRAGMA table_info: %v", err)
	}
	defer rows.Close()

	found := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan table_info: %v", err)
		}
		if name == "key_enc" {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table_info: %v", err)
	}
	if !found {
		t.Fatal("virtual_keys.key_enc column is absent")
	}
}

// TestCreateVirtualKeyStoredAtRest proves the raw VK value is never persisted
// in plaintext: after create, the `key` column holds sha256hex(raw) and the
// `key_enc` column holds non-empty AES ciphertext.
func TestCreateVirtualKeyStoredAtRest(t *testing.T) {
	st := newTestStore(t)

	created, err := st.CreateVirtualKey(&VirtualKey{
		VirtualKey: schemas.VirtualKey{Name: "vk-at-rest"},
	})
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	raw := created.Key
	if raw == "" {
		t.Fatal("created Key empty")
	}

	var keyCol, keyEnc string
	if err := st.DB().QueryRow(
		"SELECT key, key_enc FROM virtual_keys WHERE id = ?", created.ID,
	).Scan(&keyCol, &keyEnc); err != nil {
		t.Fatalf("scan raw row: %v", err)
	}

	if keyCol != sha256hex(raw) {
		t.Fatalf("key column = %q, want sha256hex(raw) = %q", keyCol, sha256hex(raw))
	}
	if len(keyCol) != 64 {
		t.Fatalf("key column length = %d, want 64 hex chars", len(keyCol))
	}
	if keyCol == raw {
		t.Fatal("key column equals raw plaintext")
	}
	if keyEnc == "" {
		t.Fatal("key_enc is empty")
	}
	if strings.Contains(keyCol, raw) || strings.Contains(keyEnc, raw) {
		t.Fatalf("raw plaintext %q leaked: key=%q key_enc=%q", raw, keyCol, keyEnc)
	}
}

// TestVirtualKeyRoundTripPlaintext proves every read path decrypts key_enc back
// into .Key so consumers (DTO display, spend attribution, gate) see plaintext.
func TestVirtualKeyRoundTripPlaintext(t *testing.T) {
	st := newTestStore(t)

	created, err := st.CreateVirtualKey(&VirtualKey{
		VirtualKey: schemas.VirtualKey{Name: "vk-roundtrip"},
	})
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	raw := created.Key

	byID, err := st.GetVirtualKeyByID(created.ID)
	if err != nil {
		t.Fatalf("GetVirtualKeyByID: %v", err)
	}
	if byID.Key != raw {
		t.Fatalf("GetVirtualKeyByID .Key = %q, want raw %q", byID.Key, raw)
	}

	list, err := st.ListVirtualKeys()
	if err != nil {
		t.Fatalf("ListVirtualKeys: %v", err)
	}
	var found bool
	for _, vk := range list {
		if vk.ID == created.ID {
			found = true
			if vk.Key != raw {
				t.Fatalf("ListVirtualKeys .Key = %q, want raw %q", vk.Key, raw)
			}
		}
	}
	if !found {
		t.Fatal("created VK missing from ListVirtualKeys")
	}
}

// TestGetVirtualKeyByKeyResolvesRaw proves a lookup with the raw key still
// resolves the row (the method hashes the input before the WHERE clause).
func TestGetVirtualKeyByKeyResolvesRaw(t *testing.T) {
	st := newTestStore(t)

	created, err := st.CreateVirtualKey(&VirtualKey{
		VirtualKey: schemas.VirtualKey{Name: "vk-lookup"},
	})
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}

	got, err := st.GetVirtualKeyByKey(created.Key)
	if err != nil {
		t.Fatalf("GetVirtualKeyByKey(raw): %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("resolved ID = %q, want %q", got.ID, created.ID)
	}
	if got.Key != created.Key {
		t.Fatalf("resolved .Key = %q, want raw %q", got.Key, created.Key)
	}
}

// TestSumCostByTeamSurvivesEncryption is the load-bearing proof that team
// budgets still enforce after the VK value is hashed in the `key` column: a
// request_log row logged under the raw VK key must still be summed by
// SumCostByTeam via the Go-mediated decrypt-then-IN correlation.
func TestSumCostByTeamSurvivesEncryption(t *testing.T) {
	st := newTestStore(t)

	created, err := st.CreateVirtualKey(&VirtualKey{
		VirtualKey: schemas.VirtualKey{Name: "vk-teamcost"},
		TeamID:     "T",
	})
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	raw := created.Key

	const since = "2026-01-01T00:00:00Z"
	const wantCost = 1.25
	// request_log.api_key is written as the raw VK value (see
	// internal/api/usage_glue_test.go:713 — entry.APIKey == vkKey).
	if _, err := st.DB().Exec(
		"INSERT INTO request_log (timestamp, api_key, cost, status) VALUES (?, ?, ?, ?)",
		"2026-06-01T00:00:00Z", raw, wantCost, "ok",
	); err != nil {
		t.Fatalf("insert request_log: %v", err)
	}

	total, err := st.SumCostByTeam("T", since)
	if err != nil {
		t.Fatalf("SumCostByTeam: %v", err)
	}
	if total != wantCost {
		t.Fatalf("SumCostByTeam = %v, want %v (team budget would silently zero)", total, wantCost)
	}
}
