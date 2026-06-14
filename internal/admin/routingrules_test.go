package admin

import (
	"testing"
	"time"

	"github.com/valyala/fasthttp"
)

func TestRoutingRuleCRUDHandlers(t *testing.T) {
	env := newTestEnv(t)
	admin, err := env.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	actor := map[string]any{userKey: admin}

	// Create.
	status, envl := call(t, env.handlers.CreateRoutingRule, "POST", "/api/routing-rules",
		`{"name":"Route GPT-4 to OpenAI","priority":1,"cond_field":"model","cond_operator":"equals","cond_value":"gpt-4o","target_provider":"openai"}`,
		actor, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d err = %q", status, errMessage(t, envl))
	}
	created := dataField[map[string]any](t, envl)
	id, _ := created["id"].(string)
	if id == "" || created["name"] != "Route GPT-4 to OpenAI" || created["target_provider"] != "openai" {
		t.Fatalf("created = %v", created)
	}
	if created["is_active"] != true {
		t.Fatalf("is_active default = %v, want true", created["is_active"])
	}
	// created_at must be an RFC3339 ISO string (ESC-CREATED-AT).
	createdAt, _ := created["created_at"].(string)
	if _, err := time.Parse(time.RFC3339, createdAt); err != nil {
		t.Fatalf("created_at = %q not RFC3339: %v", createdAt, err)
	}

	// Empty name → 400.
	status, _ = call(t, env.handlers.CreateRoutingRule, "POST", "/api/routing-rules", `{"name":""}`, actor, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("empty name status = %d, want 400", status)
	}

	// List.
	status, envl = call(t, env.handlers.ListRoutingRules, "GET", "/api/routing-rules", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	if len(dataField[[]map[string]any](t, envl)) != 1 {
		t.Fatalf("list len = %d, want 1", len(dataField[[]map[string]any](t, envl)))
	}

	// Get.
	status, envl = call(t, env.handlers.GetRoutingRule, "GET", "/api/routing-rules/"+id, "", map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("get status = %d", status)
	}

	// Get missing → 404.
	status, _ = call(t, env.handlers.GetRoutingRule, "GET", "/api/routing-rules/missing", "", map[string]any{"id": "missing"}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("get missing status = %d", status)
	}

	// Update.
	status, envl = call(t, env.handlers.UpdateRoutingRule, "PUT", "/api/routing-rules/"+id,
		`{"name":"renamed","priority":5,"cond_field":"model","cond_operator":"equals","cond_value":"gpt-4o","target_provider":"openai","is_active":false}`,
		map[string]any{"id": id, userKey: admin}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("update status = %d err = %q", status, errMessage(t, envl))
	}
	updated := dataField[map[string]any](t, envl)
	if updated["name"] != "renamed" || updated["is_active"] != false {
		t.Fatalf("updated = %v", updated)
	}

	// Update missing → 404.
	status, _ = call(t, env.handlers.UpdateRoutingRule, "PUT", "/api/routing-rules/missing",
		`{"name":"x"}`, map[string]any{"id": "missing"}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("update missing status = %d", status)
	}

	// Delete.
	status, _ = call(t, env.handlers.DeleteRoutingRule, "DELETE", "/api/routing-rules/"+id, "", map[string]any{"id": id, userKey: admin}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete status = %d", status)
	}
	status, _ = call(t, env.handlers.DeleteRoutingRule, "DELETE", "/api/routing-rules/"+id, "", map[string]any{"id": id}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("delete missing status = %d", status)
	}

	// Audit recorded for the create mutation.
	status, envl = call(t, env.handlers.GetAudit, "GET", "/api/audit", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("audit status = %d", status)
	}
	if items, _ := dataField[map[string]any](t, envl)["items"].([]any); len(items) == 0 {
		t.Fatalf("expected audit entries after routing-rule mutations")
	}
}
