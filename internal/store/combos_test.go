package store

import (
	"database/sql"
	"errors"
	"testing"
)

func TestComboCRUD(t *testing.T) {
	st := newTestStore(t)

	// Create.
	c := &Combo{Name: "my-combo", Models: []string{"gpt-4", "claude-3-opus"}}
	if err := st.CreateCombo(c); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	// Get.
	got, err := st.GetCombo("my-combo")
	if err != nil {
		t.Fatalf("GetCombo: %v", err)
	}
	if got.Name != "my-combo" || len(got.Models) != 2 || got.Models[0] != "gpt-4" || got.Models[1] != "claude-3-opus" {
		t.Fatalf("GetCombo = %+v", got)
	}

	// List.
	list, err := st.ListCombos()
	if err != nil {
		t.Fatalf("ListCombos: %v", err)
	}
	if len(list) != 1 || list[0].Name != "my-combo" {
		t.Fatalf("ListCombos = %+v", list)
	}

	// Update models.
	if err := st.UpdateCombo("my-combo", []string{"claude-3-haiku"}); err != nil {
		t.Fatalf("UpdateCombo: %v", err)
	}
	got, err = st.GetCombo("my-combo")
	if err != nil {
		t.Fatalf("GetCombo after update: %v", err)
	}
	if len(got.Models) != 1 || got.Models[0] != "claude-3-haiku" {
		t.Fatalf("after update models = %v", got.Models)
	}

	// Delete.
	if err := st.DeleteCombo("my-combo"); err != nil {
		t.Fatalf("DeleteCombo: %v", err)
	}
	_, err = st.GetCombo("my-combo")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete GetCombo = %v, want ErrNotFound", err)
	}

	// Delete non-existent → ErrNotFound.
	if err := st.DeleteCombo("nonexistent"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete nonexistent = %v, want ErrNotFound", err)
	}
}

func TestMigrationAdditiveRerun2(t *testing.T) {
	st := newTestStore(t)

	// Running migrate a second time must not error.
	if err := migrate(st.db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	// combos table must exist with name and models_json columns.
	if _, err := st.db.Exec("SELECT name, models_json FROM combos LIMIT 0"); err != nil {
		t.Fatalf("combos table missing or wrong schema: %v", err)
	}

	// No strategy column — strategy lives in settings, not the combos row.
	rows, err := st.db.Query("PRAGMA table_info(combos)")
	if err != nil {
		t.Fatalf("PRAGMA table_info: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if name == "strategy" {
			t.Fatalf("combos table must NOT have a strategy column")
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate: %v", err)
	}
}
