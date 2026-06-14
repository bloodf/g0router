package admin

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestAliasAdminCRUDHandlers(t *testing.T) {
	env := newTestEnv(t)

	// Create.
	status, envl := call(t, env.handlers.CreateAlias, "POST", "/api/aliases",
		`{"alias":"gpt4","provider":"openai","model":"gpt-4o"}`, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d err = %q", status, errMessage(t, envl))
	}
	created := dataField[map[string]any](t, envl)
	id, _ := created["id"].(string)
	if id == "" || created["alias"] != "gpt4" || created["provider"] != "openai" || created["model"] != "gpt-4o" {
		t.Fatalf("created = %v", created)
	}

	// Empty alias → 400.
	status, _ = call(t, env.handlers.CreateAlias, "POST", "/api/aliases", `{"alias":""}`, nil, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("empty alias status = %d, want 400", status)
	}

	// Create two more so list == 3.
	for _, b := range []string{
		`{"alias":"claude","provider":"anthropic","model":"claude-sonnet-4"}`,
		`{"alias":"gemini","provider":"google","model":"gemini-2.5-pro"}`,
	} {
		if s, _ := call(t, env.handlers.CreateAlias, "POST", "/api/aliases", b, nil, nil); s != fasthttp.StatusCreated {
			t.Fatalf("seed create status = %d", s)
		}
	}

	// List.
	status, envl = call(t, env.handlers.ListAliases, "GET", "/api/aliases", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	list := dataField[[]map[string]any](t, envl)
	if len(list) != 3 {
		t.Fatalf("list len = %d, want 3", len(list))
	}

	// Get.
	status, envl = call(t, env.handlers.GetAlias, "GET", "/api/aliases/"+id, "", map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("get status = %d", status)
	}
	if dataField[map[string]any](t, envl)["alias"] != "gpt4" {
		t.Fatalf("get = %v", dataField[map[string]any](t, envl))
	}

	// Get missing → 404.
	status, _ = call(t, env.handlers.GetAlias, "GET", "/api/aliases/missing", "", map[string]any{"id": "missing"}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("get missing status = %d", status)
	}

	// Update.
	status, envl = call(t, env.handlers.UpdateAlias, "PUT", "/api/aliases/"+id,
		`{"alias":"gpt4-fast","provider":"openai","model":"gpt-4o-mini"}`, map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("update status = %d err = %q", status, errMessage(t, envl))
	}
	if dataField[map[string]any](t, envl)["alias"] != "gpt4-fast" {
		t.Fatalf("updated = %v", dataField[map[string]any](t, envl))
	}

	// Update missing → 404.
	status, _ = call(t, env.handlers.UpdateAlias, "PUT", "/api/aliases/missing",
		`{"alias":"x"}`, map[string]any{"id": "missing"}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("update missing status = %d", status)
	}

	// Delete.
	status, _ = call(t, env.handlers.DeleteAlias, "DELETE", "/api/aliases/"+id, "", map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete status = %d", status)
	}
	status, _ = call(t, env.handlers.DeleteAlias, "DELETE", "/api/aliases/"+id, "", map[string]any{"id": id}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("delete missing status = %d", status)
	}
}

func TestAliasAdminCreateWritesAudit(t *testing.T) {
	env := newTestEnv(t)
	admin, err := env.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}

	status, _ := call(t, env.handlers.CreateAlias, "POST", "/api/aliases",
		`{"alias":"gpt4","provider":"openai","model":"gpt-4o"}`,
		map[string]any{userKey: admin}, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d", status)
	}

	status, envl := call(t, env.handlers.GetAudit, "GET", "/api/audit", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("audit status = %d", status)
	}
	audit := dataField[map[string]any](t, envl)
	items, _ := audit["items"].([]any)
	if len(items) == 0 {
		t.Fatalf("expected an audit entry after alias create, got none")
	}
}
