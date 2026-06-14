package admin

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestComboAdminCRUDHandlers(t *testing.T) {
	env := newTestEnv(t)
	admin, err := env.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	actor := map[string]any{userKey: admin}

	// Create with structured steps.
	status, envl := call(t, env.handlers.CreateComboAdmin, "POST", "/api/combos",
		`{"name":"Fast + Cheap","strategy":"fallback","steps":[{"provider":"groq","model":"llama-3-70b"},{"provider":"openai","model":"gpt-4o-mini"}]}`,
		actor, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d err = %q", status, errMessage(t, envl))
	}
	created := dataField[map[string]any](t, envl)
	id, _ := created["id"].(string)
	if id == "" || created["name"] != "Fast + Cheap" || created["strategy"] != "fallback" {
		t.Fatalf("created = %v", created)
	}
	if created["is_active"] != true {
		t.Fatalf("is_active default = %v, want true", created["is_active"])
	}
	steps, _ := created["steps"].([]any)
	if len(steps) != 2 {
		t.Fatalf("steps len = %d, want 2", len(steps))
	}
	step0, _ := steps[0].(map[string]any)
	if step0["provider"] != "groq" || step0["model"] != "llama-3-70b" {
		t.Fatalf("step0 = %v", step0)
	}

	// Empty name → 400.
	status, _ = call(t, env.handlers.CreateComboAdmin, "POST", "/api/combos", `{"name":""}`, actor, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("empty name status = %d, want 400", status)
	}

	// List.
	status, envl = call(t, env.handlers.ListCombosAdmin, "GET", "/api/combos", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	if len(dataField[[]map[string]any](t, envl)) != 1 {
		t.Fatalf("list len = %d, want 1", len(dataField[[]map[string]any](t, envl)))
	}

	// Get.
	status, envl = call(t, env.handlers.GetComboAdmin, "GET", "/api/combos/"+id, "", map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("get status = %d", status)
	}

	// Get missing → 404.
	status, _ = call(t, env.handlers.GetComboAdmin, "GET", "/api/combos/missing", "", map[string]any{"id": "missing"}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("get missing status = %d", status)
	}

	// PUT with the steps-order body the frozen combos.spec.ts sends
	// ({steps:[{provider,model}]}); the persisted step order must round-trip.
	status, envl = call(t, env.handlers.UpdateComboAdmin, "PUT", "/api/combos/"+id,
		`{"steps":[{"provider":"openai","model":"gpt-4o-mini"},{"provider":"groq","model":"llama-3-70b"}]}`,
		map[string]any{"id": id, userKey: admin}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("update status = %d err = %q", status, errMessage(t, envl))
	}
	updated := dataField[map[string]any](t, envl)
	upSteps, _ := updated["steps"].([]any)
	if len(upSteps) != 2 {
		t.Fatalf("updated steps len = %d, want 2", len(upSteps))
	}
	first, _ := upSteps[0].(map[string]any)
	second, _ := upSteps[1].(map[string]any)
	if first["model"] != "gpt-4o-mini" || second["model"] != "llama-3-70b" {
		t.Fatalf("updated step order = %v, %v", first["model"], second["model"])
	}

	// Update missing → 404.
	status, _ = call(t, env.handlers.UpdateComboAdmin, "PUT", "/api/combos/missing",
		`{"name":"x"}`, map[string]any{"id": "missing"}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("update missing status = %d", status)
	}

	// Delete.
	status, _ = call(t, env.handlers.DeleteComboAdmin, "DELETE", "/api/combos/"+id, "", map[string]any{"id": id, userKey: admin}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete status = %d", status)
	}
	status, _ = call(t, env.handlers.DeleteComboAdmin, "DELETE", "/api/combos/"+id, "", map[string]any{"id": id}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("delete missing status = %d", status)
	}
}

func TestComboAdminMirrorsEngineStore(t *testing.T) {
	env := newTestEnv(t)

	status, _ := call(t, env.handlers.CreateComboAdmin, "POST", "/api/combos",
		`{"name":"mirror-combo","strategy":"fallback","steps":[{"provider":"groq","model":"llama-3-70b"},{"provider":"openai","model":"gpt-4o-mini"}]}`,
		nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d", status)
	}

	// Best-effort mirror-write keeps the engine combos table populated so
	// /v1/models still lists the combo by name.
	combo, err := env.store.GetCombo("mirror-combo")
	if err != nil {
		t.Fatalf("engine GetCombo: %v", err)
	}
	if len(combo.Models) != 2 || combo.Models[0] != "llama-3-70b" {
		t.Fatalf("mirrored engine combo models = %v", combo.Models)
	}
}
