package admin

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func adminUser(t *testing.T, e *testEnv) map[string]any {
	t.Helper()
	u, err := e.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	return map[string]any{userKey: u}
}

func createPool(t *testing.T, e *testEnv, body string) map[string]json.RawMessage {
	t.Helper()
	status, env := call(t, e.handlers.CreateProxyPool, "POST", "/api/proxy-pools", body, adminUser(t, e), nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create pool status = %d, body %v", status, env)
	}
	return env
}

func TestProxyPoolCreateListGetUpdateDelete(t *testing.T) {
	e := newTestEnv(t)

	env := createPool(t, e, `{"name":"US East","protocol":"https","host":"us-east.proxy.example.com","port":8080,"username":"u","password":"p"}`)
	created := dataField[map[string]any](t, env)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatalf("created pool missing id: %v", created)
	}
	if created["protocol"] != "https" || created["host"] != "us-east.proxy.example.com" {
		t.Fatalf("unexpected created DTO: %v", created)
	}
	if _, present := created["password"]; present {
		t.Fatalf("create response leaked password: %v", created)
	}

	// List returns a bare array under {data}.
	status, listEnv := call(t, e.handlers.ListProxyPools, "GET", "/api/proxy-pools", "", adminUser(t, e), nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	list := dataField[[]map[string]any](t, listEnv)
	if len(list) != 1 {
		t.Fatalf("expected 1 pool, got %d", len(list))
	}

	// Get.
	status, getEnv := call(t, e.handlers.GetProxyPool, "GET", "/api/proxy-pools/"+id, "", map[string]any{"id": id, userKey: mustAdmin(t, e)}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("get status = %d", status)
	}
	got := dataField[map[string]any](t, getEnv)
	if got["name"] != "US East" {
		t.Fatalf("unexpected get DTO: %v", got)
	}

	// Update.
	status, updEnv := call(t, e.handlers.UpdateProxyPool, "PUT", "/api/proxy-pools/"+id,
		`{"name":"US East 2","protocol":"https","host":"us-east.proxy.example.com","port":8080,"is_active":false}`,
		map[string]any{"id": id, userKey: mustAdmin(t, e)}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("update status = %d, body %v", status, updEnv)
	}
	upd := dataField[map[string]any](t, updEnv)
	if upd["name"] != "US East 2" || upd["is_active"] != false {
		t.Fatalf("update DTO not applied: %v", upd)
	}

	// Delete.
	status, _ = call(t, e.handlers.DeleteProxyPool, "DELETE", "/api/proxy-pools/"+id, "", map[string]any{"id": id, userKey: mustAdmin(t, e)}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete status = %d", status)
	}

	// Get after delete = 404.
	status, _ = call(t, e.handlers.GetProxyPool, "GET", "/api/proxy-pools/"+id, "", map[string]any{"id": id, userKey: mustAdmin(t, e)}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("get after delete status = %d, want 404", status)
	}
}

func mustAdmin(t *testing.T, e *testEnv) any {
	t.Helper()
	u, err := e.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	return u
}

func TestProxyPoolCreateValidation(t *testing.T) {
	e := newTestEnv(t)

	// Empty name.
	status, _ := call(t, e.handlers.CreateProxyPool, "POST", "/api/proxy-pools", `{"host":"h.example.com"}`, adminUser(t, e), nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("empty name status = %d, want 400", status)
	}
	// Empty host.
	status, _ = call(t, e.handlers.CreateProxyPool, "POST", "/api/proxy-pools", `{"name":"x"}`, adminUser(t, e), nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("empty host status = %d, want 400", status)
	}
	// Invalid port.
	status, _ = call(t, e.handlers.CreateProxyPool, "POST", "/api/proxy-pools", `{"name":"x","host":"h.example.com","port":70000}`, adminUser(t, e), nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("bad port status = %d, want 400", status)
	}
}

func TestProxyPoolNoPasswordLeak(t *testing.T) {
	e := newTestEnv(t)
	env := createPool(t, e, `{"name":"x","host":"h.example.com","password":"supersecret"}`)
	id := dataField[map[string]any](t, env)["id"].(string)

	// Marshal every response shape and assert it never contains the cleartext.
	checks := []func() map[string]json.RawMessage{
		func() map[string]json.RawMessage {
			_, ev := call(t, e.handlers.ListProxyPools, "GET", "/api/proxy-pools", "", adminUser(t, e), nil)
			return ev
		},
		func() map[string]json.RawMessage {
			_, ev := call(t, e.handlers.GetProxyPool, "GET", "/api/proxy-pools/"+id, "", map[string]any{"id": id, userKey: mustAdmin(t, e)}, nil)
			return ev
		},
	}
	for i, fn := range checks {
		ev := fn()
		raw, _ := json.Marshal(ev)
		if strings.Contains(string(raw), "supersecret") {
			t.Fatalf("response %d leaked cleartext password: %s", i, raw)
		}
		if strings.Contains(string(raw), `"password"`) {
			t.Fatalf("response %d contains a password field: %s", i, raw)
		}
	}
}

func TestProxyPoolDeleteBoundReturns409(t *testing.T) {
	e := newTestEnv(t)
	env := createPool(t, e, `{"name":"x","host":"h.example.com"}`)
	id := dataField[map[string]any](t, env)["id"].(string)

	// Bind a connection to the pool.
	if err := e.store.CreateProvider(&store.ProviderRecord{Name: "p", Type: "openai"}); err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	provs, err := e.store.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders: %v", err)
	}
	conn := &store.Connection{ProviderID: provs[0].ID, Name: "c", Kind: "api_key", Secret: "k", ProxyPoolID: id}
	if err := e.store.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	status, errEnv := call(t, e.handlers.DeleteProxyPool, "DELETE", "/api/proxy-pools/"+id, "", map[string]any{"id": id, userKey: mustAdmin(t, e)}, nil)
	if status != fasthttp.StatusConflict {
		t.Fatalf("delete bound pool status = %d, want 409", status)
	}
	if msg := errMessage(t, errEnv); msg == "" {
		t.Fatalf("expected an error message on 409")
	}

	// Pool still exists.
	if _, err := e.store.GetProxyPoolByID(id); err != nil {
		t.Fatalf("pool should still exist after blocked delete: %v", err)
	}
}

func TestProxyPoolTestEndpointPersistsStatus(t *testing.T) {
	e := newTestEnv(t)
	env := createPool(t, e, `{"name":"x","protocol":"http","host":"proxy.example.com","port":8080}`)
	id := dataField[map[string]any](t, env)["id"].(string)

	// Force a deterministic reachable probe through the service prober.
	e.handlers.SetProxyProber(func(proxyURL, target string) (int, error) { return 7, nil })

	status, testEnv := call(t, e.handlers.TestProxyPool, "POST", "/api/proxy-pools/"+id+"/test", "",
		map[string]any{"id": id, userKey: mustAdmin(t, e)}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("test status = %d, body %v", status, testEnv)
	}
	result := dataField[map[string]any](t, testEnv)
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got %v", result)
	}

	// last_check_status persisted on the pool.
	got, err := e.store.GetProxyPoolByID(id)
	if err != nil {
		t.Fatalf("GetProxyPoolByID: %v", err)
	}
	if got.LastCheckStatus != "ok" || got.LastCheckAt == "" {
		t.Fatalf("test did not persist check: %+v", got)
	}
}

func TestProxyPoolCreateWritesAudit(t *testing.T) {
	e := newTestEnv(t)
	createPool(t, e, `{"name":"audited","host":"h.example.com"}`)
	items, err := e.store.ListAuditEntries(100)
	if err != nil {
		t.Fatalf("ListAuditEntries: %v", err)
	}
	found := false
	for _, it := range items {
		if strings.Contains(it.Action, "proxy_pool") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a proxy_pool audit entry, got %d entries", len(items))
	}
}
