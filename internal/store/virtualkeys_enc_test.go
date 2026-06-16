package store

import (
	"testing"
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
