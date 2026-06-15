package admin

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestVirtualKeysAdminCRUD(t *testing.T) {
	env := newTestEnv(t)
	token := loginToken(t, env)
	authHeader := map[string]string{"Authorization": "Bearer " + token}

	createBody := `{
		"name":"production-vk",
		"provider_configs":[{"provider":"openai","allowed_models":["gpt-4o"],"key_ids":["conn-1"],"weight":1}],
		"budget":{"limit":100,"period":"daily","used":0},
		"rate_limit_rpm":60
	}`

	// Create.
	status, envl := call(t, env.handlers.RequireSession(env.handlers.CreateVirtualKey), "POST", "/api/virtual-keys",
		createBody, nil, authHeader)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d err = %q", status, errMessage(t, envl))
	}
	createdWrap := dataField[map[string]any](t, envl)
	created, _ := createdWrap["virtual_key"].(map[string]any)
	if created["name"] != "production-vk" {
		t.Fatalf("created name = %v", created["name"])
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatalf("created id empty: %v", created)
	}
	key, _ := created["key"].(string)
	if key == "" {
		t.Fatalf("created key empty: %v", created)
	}

	// Validation: missing name.
	status, _ = call(t, env.handlers.RequireSession(env.handlers.CreateVirtualKey), "POST", "/api/virtual-keys",
		`{"provider_configs":[{"provider":"openai","allowed_models":["gpt-4o"],"key_ids":["conn-1"]}]}`, nil, authHeader)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("missing name status = %d, want 400", status)
	}

	// Validation: empty provider_configs.
	status, _ = call(t, env.handlers.RequireSession(env.handlers.CreateVirtualKey), "POST", "/api/virtual-keys",
		`{"name":"x","provider_configs":[]}`, nil, authHeader)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("empty provider_configs status = %d, want 400", status)
	}

	// Validation: negative budget limit.
	status, _ = call(t, env.handlers.RequireSession(env.handlers.CreateVirtualKey), "POST", "/api/virtual-keys",
		`{"name":"x","provider_configs":[{"provider":"openai","allowed_models":["gpt-4o"],"key_ids":["conn-1"]}],"budget":{"limit":-1,"period":"daily"}}`, nil, authHeader)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("negative budget status = %d, want 400", status)
	}

	// List.
	status, envl = call(t, env.handlers.RequireSession(env.handlers.ListVirtualKeys), "GET", "/api/virtual-keys", "", nil, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	listPayload := dataField[map[string]any](t, envl)
	vks, _ := listPayload["virtual_keys"].([]any)
	if len(vks) != 1 {
		t.Fatalf("list virtual_keys length = %d, want 1", len(vks))
	}

	// Get by id.
	status, envl = call(t, env.handlers.RequireSession(env.handlers.GetVirtualKey), "GET", "/api/virtual-keys/"+id, "",
		map[string]any{"id": id}, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("get status = %d err = %q", status, errMessage(t, envl))
	}
	getPayload := dataField[map[string]any](t, envl)
	gotVK, _ := getPayload["virtual_key"].(map[string]any)
	if gotVK["id"] != id {
		t.Fatalf("get virtual_key id = %v, want %v", gotVK["id"], id)
	}

	// Update.
	updateBody := `{
		"name":"production-vk-renamed",
		"provider_configs":[{"provider":"anthropic","allowed_models":["claude-3-opus"],"key_ids":["conn-2"]}],
		"budget":{"limit":200,"period":"monthly","used":10},
		"rate_limit_rpm":120,
		"is_active":false
	}`
	status, envl = call(t, env.handlers.RequireSession(env.handlers.UpdateVirtualKey), "PUT", "/api/virtual-keys/"+id,
		updateBody, map[string]any{"id": id}, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("update status = %d err = %q", status, errMessage(t, envl))
	}
	updatePayload := dataField[map[string]any](t, envl)
	updated, _ := updatePayload["virtual_key"].(map[string]any)
	if updated["name"] != "production-vk-renamed" {
		t.Fatalf("updated name = %v, want production-vk-renamed", updated["name"])
	}
	if updated["is_active"] != false {
		t.Fatalf("updated is_active = %v, want false", updated["is_active"])
	}

	// Update missing → 404.
	status, _ = call(t, env.handlers.RequireSession(env.handlers.UpdateVirtualKey), "PUT", "/api/virtual-keys/missing",
		updateBody, map[string]any{"id": "missing"}, authHeader)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("update missing status = %d, want 404", status)
	}

	// Delete.
	status, envl = call(t, env.handlers.RequireSession(env.handlers.DeleteVirtualKey), "DELETE", "/api/virtual-keys/"+id, "",
		map[string]any{"id": id}, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete status = %d err = %q", status, errMessage(t, envl))
	}
	msg := dataField[map[string]any](t, envl)
	if msg["message"] != "Virtual key deleted successfully" {
		t.Fatalf("delete message = %v", msg)
	}

	status, _ = call(t, env.handlers.RequireSession(env.handlers.DeleteVirtualKey), "DELETE", "/api/virtual-keys/"+id, "",
		map[string]any{"id": id}, authHeader)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("delete missing status = %d, want 404", status)
	}
}

// TestVirtualKeyTeamIDAssignment verifies the VK→Team link is accepted, persisted,
// and surfaced by the admin API (bf-gov-1, D2/D4): a create/update carrying
// team_id round-trips, and the budget-owner assignment passes ValidateBudgetOwner.
func TestVirtualKeyTeamIDAssignment(t *testing.T) {
	env := newTestEnv(t)
	token := loginToken(t, env)
	authHeader := map[string]string{"Authorization": "Bearer " + token}

	createBody := `{
		"name":"teamed-vk",
		"team_id":"team-abc",
		"provider_configs":[{"provider":"openai","allowed_models":["gpt-4o"],"key_ids":["conn-1"]}],
		"budget":{"limit":100,"period":"daily","used":0}
	}`
	status, envl := call(t, env.handlers.RequireSession(env.handlers.CreateVirtualKey), "POST", "/api/virtual-keys",
		createBody, nil, authHeader)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create teamed vk status = %d err = %q", status, errMessage(t, envl))
	}
	createdWrap := dataField[map[string]any](t, envl)
	created, _ := createdWrap["virtual_key"].(map[string]any)
	if created["team_id"] != "team-abc" {
		t.Fatalf("created team_id = %v, want team-abc", created["team_id"])
	}
	id, _ := created["id"].(string)

	// Re-assign the team via update.
	updateBody := `{
		"name":"teamed-vk",
		"team_id":"team-xyz",
		"provider_configs":[{"provider":"openai","allowed_models":["gpt-4o"],"key_ids":["conn-1"]}]
	}`
	status, envl = call(t, env.handlers.RequireSession(env.handlers.UpdateVirtualKey), "PUT", "/api/virtual-keys/"+id,
		updateBody, map[string]any{"id": id}, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("update team_id status = %d err = %q", status, errMessage(t, envl))
	}
	updatePayload := dataField[map[string]any](t, envl)
	updated, _ := updatePayload["virtual_key"].(map[string]any)
	if updated["team_id"] != "team-xyz" {
		t.Fatalf("updated team_id = %v, want team-xyz", updated["team_id"])
	}
}
