package admin

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestGetAuditReadEndpoint(t *testing.T) {
	env := newTestEnv(t)

	// Seed audit entries through the domain service so the read endpoint has data.
	for i := 0; i < 5; i++ {
		if err := env.handlers.auditService().WriteAudit("admin", "create_key", "key-1", "Created key"); err != nil {
			t.Fatalf("WriteAudit %d: %v", i, err)
		}
	}

	// Default read.
	status, envl := call(t, env.handlers.GetAudit, "GET", "/api/audit", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("audit status = %d err = %q", status, errMessage(t, envl))
	}
	page := dataField[struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}](t, envl)
	if page.Total != 5 {
		t.Fatalf("total = %d, want 5", page.Total)
	}
	if len(page.Items) != 5 {
		t.Fatalf("len(items) = %d, want 5", len(page.Items))
	}
	if page.Items[0]["action"] != "create_key" || page.Items[0]["actor"] != "admin" {
		t.Fatalf("item shape = %v", page.Items[0])
	}

	// Limit clamps the items but total still reflects all.
	status, envl = call(t, env.handlers.GetAudit, "GET", "/api/audit?limit=2", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("audit limit status = %d", status)
	}
	page = dataField[struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}](t, envl)
	if len(page.Items) != 2 {
		t.Fatalf("limited len(items) = %d, want 2", len(page.Items))
	}
	if page.Total != 5 {
		t.Fatalf("limited total = %d, want 5", page.Total)
	}
}

func TestTeamCreateWritesAudit(t *testing.T) {
	env := newTestEnv(t)

	admin, err := env.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}

	status, _ := call(t, env.handlers.CreateTeam, "POST", "/api/teams",
		`{"name":"Engineering","budget_usd":1000,"budget_period":"monthly"}`,
		map[string]any{userKey: admin}, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create team status = %d", status)
	}

	status, envl := call(t, env.handlers.GetAudit, "GET", "/api/audit", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("audit status = %d", status)
	}
	page := dataField[struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}](t, envl)
	if page.Total < 1 {
		t.Fatalf("expected an audit entry after team create, got total=%d", page.Total)
	}
	entry := page.Items[0]
	if entry["actor"] != "admin" {
		t.Fatalf("audit actor = %v, want admin", entry["actor"])
	}
	if entry["target"] != "Engineering" {
		t.Fatalf("audit target = %v, want Engineering", entry["target"])
	}
}
