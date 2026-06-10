package store

import "testing"

func TestEnsureColumnRejectsBadNames(t *testing.T) {
	st := newTestStore(t)

	badNames := []struct{ table, column string }{
		{"settings;", "extra"},
		{"settings", "extra;drop table users--"},
		{"SETTINGS", "extra"},
		{"1settings", "extra"},
		{"settings", "1extra"},
		{"settings", "extra col"},
	}

	for _, tc := range badNames {
		err := ensureColumn(st.DB(), tc.table, tc.column, "TEXT NOT NULL DEFAULT ''")
		if err == nil {
			t.Fatalf("ensureColumn(%q, %q) should reject bad name", tc.table, tc.column)
		}
	}
}
