package admin

import (
	"strconv"
	"testing"

	"github.com/valyala/fasthttp"
)

// seedFeatureFlag inserts a flag row directly through the store DB (the surface
// is list + get + toggle only — there is no create handler/store method).
func seedFeatureFlag(t *testing.T, env *testEnv, key, description string, enabled bool) int64 {
	t.Helper()
	on := 0
	if enabled {
		on = 1
	}
	res, err := env.store.DB().Exec(
		"INSERT INTO feature_flags (key, enabled, description, created_at) VALUES (?, ?, ?, ?)",
		key, on, description, "2026-06-14T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("seed feature flag: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	return id
}

func TestListFeatureFlags(t *testing.T) {
	env := newTestEnv(t)
	mcpID := seedFeatureFlag(t, env, "mcp_gateway", "Enable MCP gateway", true)
	seedFeatureFlag(t, env, "rtk_compression", "Enable RTK compression", false)

	status, envl := call(t, env.handlers.ListFeatureFlags, "GET", "/api/feature-flags", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d err = %q", status, errMessage(t, envl))
	}
	list := dataField[[]map[string]any](t, envl)
	// 2 inserted here + the migration-seeded semantic_cache flag (bf-core-2, D8).
	if len(list) != 3 {
		t.Fatalf("len(list) = %d, want 3", len(list))
	}
	// Locate the seeded mcp_gateway row by id rather than assuming list order.
	var mcp map[string]any
	for _, f := range list {
		if id, ok := f["id"].(float64); ok && int64(id) == mcpID {
			mcp = f
			break
		}
	}
	if mcp == nil {
		t.Fatalf("mcp_gateway flag (id=%d) not in list: %v", mcpID, list)
	}
	if mcp["key"] != "mcp_gateway" || mcp["description"] != "Enable MCP gateway" {
		t.Fatalf("mcp = %v", mcp)
	}
	if mcp["enabled"] != true {
		t.Fatalf("mcp enabled = %v, want true", mcp["enabled"])
	}
	if _, ok := mcp["created_at"]; !ok {
		t.Fatalf("mcp missing created_at: %v", mcp)
	}
}

func TestGetFeatureFlag(t *testing.T) {
	env := newTestEnv(t)
	id := seedFeatureFlag(t, env, "new_dashboard", "New React dashboard", false)
	idStr := strconv.FormatInt(id, 10)

	status, envl := call(t, env.handlers.GetFeatureFlag, "GET", "/api/feature-flags/"+idStr, "",
		map[string]any{"id": idStr}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("get status = %d err = %q", status, errMessage(t, envl))
	}
	flag := dataField[map[string]any](t, envl)
	if flag["key"] != "new_dashboard" {
		t.Fatalf("flag = %v", flag)
	}

	// Missing → 404.
	status, _ = call(t, env.handlers.GetFeatureFlag, "GET", "/api/feature-flags/99", "",
		map[string]any{"id": "99"}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("get missing status = %d, want 404", status)
	}
}

func TestToggleFeatureFlag(t *testing.T) {
	env := newTestEnv(t)
	id := seedFeatureFlag(t, env, "mcp_gateway", "Enable MCP gateway", false)
	idStr := strconv.FormatInt(id, 10)

	admin, err := env.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}

	status, envl := call(t, env.handlers.ToggleFeatureFlag, "PUT", "/api/feature-flags/"+idStr,
		`{"enabled":true}`, map[string]any{"id": idStr, userKey: admin}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("toggle status = %d err = %q", status, errMessage(t, envl))
	}
	flag := dataField[map[string]any](t, envl)
	if flag["enabled"] != true {
		t.Fatalf("toggled enabled = %v, want true", flag["enabled"])
	}

	// Persisted.
	status, envl = call(t, env.handlers.GetFeatureFlag, "GET", "/api/feature-flags/"+idStr, "",
		map[string]any{"id": idStr}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("get after toggle status = %d", status)
	}
	if dataField[map[string]any](t, envl)["enabled"] != true {
		t.Fatalf("toggle not persisted")
	}

	// Audit entry written on toggle.
	status, envl = call(t, env.handlers.GetAudit, "GET", "/api/audit", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("audit status = %d", status)
	}
	page := dataField[struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}](t, envl)
	if page.Total < 1 {
		t.Fatalf("expected an audit entry after toggle, total=%d", page.Total)
	}
	if page.Items[0]["actor"] != "admin" || page.Items[0]["target"] != "mcp_gateway" {
		t.Fatalf("audit entry = %v", page.Items[0])
	}
}

func TestToggleFeatureFlagMissing(t *testing.T) {
	env := newTestEnv(t)
	status, _ := call(t, env.handlers.ToggleFeatureFlag, "PUT", "/api/feature-flags/404",
		`{"enabled":true}`, map[string]any{"id": "404"}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("toggle missing status = %d, want 404", status)
	}
}

func TestToggleFeatureFlagBadID(t *testing.T) {
	env := newTestEnv(t)
	status, _ := call(t, env.handlers.ToggleFeatureFlag, "PUT", "/api/feature-flags/abc",
		`{"enabled":true}`, map[string]any{"id": "abc"}, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("toggle bad id status = %d, want 400", status)
	}
}
