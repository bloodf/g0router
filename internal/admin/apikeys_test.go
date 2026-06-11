package admin

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/bloodf/g0router/internal/auth"
	"github.com/valyala/fasthttp"
)

func TestKeysCRUDEndpoints(t *testing.T) {
	env := newTestEnv(t)
	token := loginToken(t, env)
	authHeader := map[string]string{"Authorization": "Bearer " + token}

	// Create.
	status, envl := call(t, env.handlers.CreateAPIKey, "POST", "/api/keys",
		`{"name":"production"}`, nil, authHeader)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d err = %q", status, errMessage(t, envl))
	}
	created := dataField[map[string]any](t, envl)
	if created["name"] != "production" {
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
	re := regexp.MustCompile(`^sk-[0-9a-f]{16}-[a-z0-9]{6}-[0-9a-f]{8}$`)
	if !re.MatchString(key) {
		t.Fatalf("created key %q does not match expected format", key)
	}
	machineID, _ := created["machine_id"].(string)
	if matched, _ := regexp.MatchString(`^[0-9a-f]{16}$`, machineID); !matched {
		t.Fatalf("created machine_id %q is not 16 hex chars", machineID)
	}

	// Create without name -> 400.
	status, _ = call(t, env.handlers.CreateAPIKey, "POST", "/api/keys",
		`{"name":""}`, nil, authHeader)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("empty name status = %d, want 400", status)
	}

	// List.
	status, envl = call(t, env.handlers.ListAPIKeys, "GET", "/api/keys", "", nil, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	listPayload := dataField[map[string]any](t, envl)
	keys, _ := listPayload["keys"].([]any)
	if len(keys) != 1 {
		t.Fatalf("list keys length = %d, want 1", len(keys))
	}

	// Get by id.
	status, envl = call(t, env.handlers.GetAPIKey, "GET", "/api/keys/"+id, "",
		map[string]any{"id": id}, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("get status = %d err = %q", status, errMessage(t, envl))
	}
	getPayload := dataField[map[string]any](t, envl)
	gotKey, _ := getPayload["key"].(map[string]any)
	if gotKey["id"] != id {
		t.Fatalf("get key id = %v, want %v", gotKey["id"], id)
	}

	// Update active flag.
	status, envl = call(t, env.handlers.UpdateAPIKey, "PUT", "/api/keys/"+id,
		`{"is_active":false}`, map[string]any{"id": id}, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("update status = %d err = %q", status, errMessage(t, envl))
	}
	updatePayload := dataField[map[string]any](t, envl)
	updated, _ := updatePayload["key"].(map[string]any)
	if updated["is_active"] != false {
		t.Fatalf("updated is_active = %v, want false", updated["is_active"])
	}

	// Update missing -> 404.
	status, _ = call(t, env.handlers.UpdateAPIKey, "PUT", "/api/keys/missing",
		`{"is_active":false}`, map[string]any{"id": "missing"}, authHeader)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("update missing status = %d, want 404", status)
	}

	// Delete.
	status, envl = call(t, env.handlers.DeleteAPIKey, "DELETE", "/api/keys/"+id, "",
		map[string]any{"id": id}, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete status = %d err = %q", status, errMessage(t, envl))
	}
	msg := dataField[map[string]any](t, envl)
	if msg["message"] != "Key deleted successfully" {
		t.Fatalf("delete message = %v", msg)
	}

	status, _ = call(t, env.handlers.DeleteAPIKey, "DELETE", "/api/keys/"+id, "",
		map[string]any{"id": id}, authHeader)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("delete missing status = %d, want 404", status)
	}
}

func TestKeysEndpointsRequireSession(t *testing.T) {
	env := newTestEnv(t)

	status, _ := call(t, env.handlers.RequireSession(env.handlers.ListAPIKeys), "GET", "/api/keys", "", nil, nil)
	if status != fasthttp.StatusUnauthorized {
		t.Fatalf("unauthenticated list status = %d, want 401", status)
	}

	status, _ = call(t, env.handlers.RequireSession(env.handlers.CreateAPIKey), "POST", "/api/keys",
		`{"name":"x"}`, nil, nil)
	if status != fasthttp.StatusUnauthorized {
		t.Fatalf("unauthenticated create status = %d, want 401", status)
	}
}

func TestCreateAPIKeyMachineIdMatchesServer(t *testing.T) {
	env := newTestEnv(t)
	token := loginToken(t, env)
	authHeader := map[string]string{"Authorization": "Bearer " + token}

	status, envl := call(t, env.handlers.CreateAPIKey, "POST", "/api/keys",
		`{"name":"server-id"}`, nil, authHeader)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d err = %q", status, errMessage(t, envl))
	}
	created := dataField[map[string]any](t, envl)
	machineID, _ := created["machine_id"].(string)

	expected, err := auth.MachineID(env.store.DataDir(), "")
	if err != nil {
		t.Fatalf("MachineID: %v", err)
	}
	if machineID != expected {
		t.Fatalf("machine_id = %q, want %q", machineID, expected)
	}

	_ = fmt.Sprint(machineID)
}
