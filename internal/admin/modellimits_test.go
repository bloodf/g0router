package admin

import (
	"strconv"
	"testing"

	"github.com/valyala/fasthttp"
)

func TestModelLimitCRUDHandlers(t *testing.T) {
	env := newTestEnv(t)
	admin, err := env.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	actor := map[string]any{userKey: admin}

	// Create.
	status, envl := call(t, env.handlers.CreateModelLimit, "POST", "/api/model-limits",
		`{"model":"gpt-4o","max_tokens":128000,"max_rpm":1000,"allowed_key_ids":["key-1"]}`, actor, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d err = %q", status, errMessage(t, envl))
	}
	created := dataField[map[string]any](t, envl)
	idF, ok := created["id"].(float64)
	if !ok || idF == 0 {
		t.Fatalf("id is not numeric: %v", created["id"])
	}
	if created["model"] != "gpt-4o" {
		t.Fatalf("created = %v", created)
	}
	keyIDs, _ := created["allowed_key_ids"].([]any)
	if len(keyIDs) != 1 || keyIDs[0] != "key-1" {
		t.Fatalf("allowed_key_ids = %v", created["allowed_key_ids"])
	}
	id := strconv.FormatInt(int64(idF), 10)

	// Empty model → 400.
	status, _ = call(t, env.handlers.CreateModelLimit, "POST", "/api/model-limits", `{"model":""}`, actor, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("empty model status = %d, want 400", status)
	}

	// Non-numeric {id} → 400.
	status, _ = call(t, env.handlers.GetModelLimit, "GET", "/api/model-limits/abc", "", map[string]any{"id": "abc"}, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("non-numeric id status = %d, want 400", status)
	}

	// List.
	status, envl = call(t, env.handlers.ListModelLimits, "GET", "/api/model-limits", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	if len(dataField[[]map[string]any](t, envl)) != 1 {
		t.Fatalf("list len = %d, want 1", len(dataField[[]map[string]any](t, envl)))
	}

	// Get by numeric id.
	status, envl = call(t, env.handlers.GetModelLimit, "GET", "/api/model-limits/"+id, "", map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("get status = %d", status)
	}
	if dataField[map[string]any](t, envl)["model"] != "gpt-4o" {
		t.Fatalf("get = %v", dataField[map[string]any](t, envl))
	}

	// Get missing → 404.
	status, _ = call(t, env.handlers.GetModelLimit, "GET", "/api/model-limits/99999", "", map[string]any{"id": "99999"}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("get missing status = %d", status)
	}

	// Update.
	status, envl = call(t, env.handlers.UpdateModelLimit, "PUT", "/api/model-limits/"+id,
		`{"model":"gpt-4o-mini","max_tokens":64000,"max_rpm":500,"allowed_key_ids":["key-2"]}`,
		map[string]any{"id": id, userKey: admin}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("update status = %d err = %q", status, errMessage(t, envl))
	}
	if dataField[map[string]any](t, envl)["model"] != "gpt-4o-mini" {
		t.Fatalf("updated = %v", dataField[map[string]any](t, envl))
	}

	// Update missing → 404.
	status, _ = call(t, env.handlers.UpdateModelLimit, "PUT", "/api/model-limits/99999",
		`{"model":"x"}`, map[string]any{"id": "99999"}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("update missing status = %d", status)
	}

	// Delete.
	status, _ = call(t, env.handlers.DeleteModelLimit, "DELETE", "/api/model-limits/"+id, "", map[string]any{"id": id, userKey: admin}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete status = %d", status)
	}
	status, _ = call(t, env.handlers.DeleteModelLimit, "DELETE", "/api/model-limits/"+id, "", map[string]any{"id": id}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("delete missing status = %d", status)
	}
}
