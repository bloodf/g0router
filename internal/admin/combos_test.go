package admin

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestComboCRUDHandlers(t *testing.T) {
	env := newTestEnv(t)

	// Create.
	status, envl := call(t, env.handlers.CreateCombo, "POST", "/api/combos",
		`{"name":"fast-combo","models":["gpt-4","claude-3-haiku"]}`, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, err = %q", status, errMessage(t, envl))
	}
	created := dataField[map[string]any](t, envl)
	if created["name"] != "fast-combo" {
		t.Fatalf("created = %v", created)
	}
	models, _ := created["models"].([]any)
	if len(models) != 2 {
		t.Fatalf("created models = %v, want 2 entries", models)
	}

	// List.
	status, envl = call(t, env.handlers.ListCombos, "GET", "/api/combos", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	list := dataField[[]any](t, envl)
	if len(list) != 1 {
		t.Fatalf("list count = %d, want 1", len(list))
	}

	// Update.
	status, envl = call(t, env.handlers.UpdateCombo, "PUT", "/api/combos/fast-combo",
		`{"models":["claude-3-opus"]}`, map[string]any{"name": "fast-combo"}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("update status = %d, err = %q", status, errMessage(t, envl))
	}
	updated := dataField[map[string]any](t, envl)
	updatedModels, _ := updated["models"].([]any)
	if len(updatedModels) != 1 || updatedModels[0] != "claude-3-opus" {
		t.Fatalf("updated models = %v, want [claude-3-opus]", updatedModels)
	}

	// Delete.
	status, _ = call(t, env.handlers.DeleteCombo, "DELETE", "/api/combos/fast-combo",
		"", map[string]any{"name": "fast-combo"}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete status = %d", status)
	}

	// Delete missing → 404.
	status, _ = call(t, env.handlers.DeleteCombo, "DELETE", "/api/combos/fast-combo",
		"", map[string]any{"name": "fast-combo"}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("delete missing status = %d, want 404", status)
	}
}

func TestComboNameValidation(t *testing.T) {
	env := newTestEnv(t)

	valid := []string{"mycombo", "my-combo", "combo.1", "combo_2", "COMBO123", "a-b.c_d"}
	for _, name := range valid {
		body := `{"name":"` + name + `","models":["gpt-4"]}`
		status, _ := call(t, env.handlers.CreateCombo, "POST", "/api/combos", body, nil, nil)
		if status != fasthttp.StatusCreated {
			t.Errorf("valid name %q: status = %d, want 201", name, status)
		}
		call(t, env.handlers.DeleteCombo, "DELETE", "/api/combos/"+name, "", map[string]any{"name": name}, nil)
	}

	invalid := []string{"", "my combo", "combo/1", "combo@1", "combo#1", `combo\1`}
	for _, name := range invalid {
		body := `{"name":"` + name + `","models":["gpt-4"]}`
		status, _ := call(t, env.handlers.CreateCombo, "POST", "/api/combos", body, nil, nil)
		if status != fasthttp.StatusBadRequest {
			t.Errorf("invalid name %q: status = %d, want 400", name, status)
		}
	}
}
