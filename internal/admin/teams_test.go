package admin

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestTeamsCRUDHandlers(t *testing.T) {
	env := newTestEnv(t)

	// Create.
	status, envl := call(t, env.handlers.CreateTeam, "POST", "/api/teams",
		`{"name":"Engineering","budget_usd":2000,"budget_period":"monthly","rate_limit_rpm":5000}`, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d err = %q", status, errMessage(t, envl))
	}
	created := dataField[map[string]any](t, envl)
	id, _ := created["id"].(string)
	if id == "" || created["name"] != "Engineering" {
		t.Fatalf("created = %v", created)
	}
	if created["budget_usd"].(float64) != 2000 || created["budget_period"] != "monthly" || created["rate_limit_rpm"].(float64) != 5000 {
		t.Fatalf("created fields = %v", created)
	}
	if created["budget_used_usd"].(float64) != 0 {
		t.Fatalf("budget_used_usd = %v, want 0", created["budget_used_usd"])
	}

	// Empty name → 400.
	status, _ = call(t, env.handlers.CreateTeam, "POST", "/api/teams", `{"name":""}`, nil, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("empty-name create status = %d, want 400", status)
	}

	// List (bare array under data).
	status, envl = call(t, env.handlers.ListTeams, "GET", "/api/teams", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	list := dataField[[]map[string]any](t, envl)
	if len(list) != 1 {
		t.Fatalf("list = %v", list)
	}

	// Get.
	status, envl = call(t, env.handlers.GetTeam, "GET", "/api/teams/"+id, "", map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("get status = %d", status)
	}
	got := dataField[map[string]any](t, envl)
	if got["name"] != "Engineering" {
		t.Fatalf("get = %v", got)
	}

	// Get missing → 404.
	status, _ = call(t, env.handlers.GetTeam, "GET", "/api/teams/missing", "", map[string]any{"id": "missing"}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("get missing status = %d", status)
	}

	// Update.
	status, envl = call(t, env.handlers.UpdateTeam, "PUT", "/api/teams/"+id,
		`{"name":"Engineering EU","budget_usd":2500,"budget_period":"weekly","rate_limit_rpm":4000}`,
		map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("update status = %d err = %q", status, errMessage(t, envl))
	}
	updated := dataField[map[string]any](t, envl)
	if updated["name"] != "Engineering EU" || updated["budget_usd"].(float64) != 2500 || updated["budget_period"] != "weekly" {
		t.Fatalf("updated = %v", updated)
	}

	// Update missing → 404.
	status, _ = call(t, env.handlers.UpdateTeam, "PUT", "/api/teams/missing",
		`{"name":"X"}`, map[string]any{"id": "missing"}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("update missing status = %d", status)
	}

	// Delete.
	status, _ = call(t, env.handlers.DeleteTeam, "DELETE", "/api/teams/"+id, "", map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete status = %d", status)
	}
	status, _ = call(t, env.handlers.DeleteTeam, "DELETE", "/api/teams/"+id, "", map[string]any{"id": id}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("delete missing status = %d", status)
	}
}
