package admin

import (
	"testing"

	"github.com/valyala/fasthttp"
)

// fakeModelProber returns a deterministic probe result (no network).
type fakeModelProber struct {
	ok        bool
	latencyMS int
	err       error
}

func (f fakeModelProber) Probe(provider, modelID string) (bool, int, error) {
	return f.ok, f.latencyMS, f.err
}

func TestTestModelReturnsProbeResult(t *testing.T) {
	env := newTestEnv(t)
	env.handlers.SetModelProber(fakeModelProber{ok: true, latencyMS: 42})
	token := loginToken(t, env)

	status, envl := call(t, env.handlers.RequireSession(env.handlers.TestModel),
		"POST", "/api/models/test", `{"provider":"deepseek","model_id":"deepseek-chat"}`, nil,
		map[string]string{"Authorization": "Bearer " + token})
	if status != fasthttp.StatusOK {
		t.Fatalf("test status = %d, err = %q", status, errMessage(t, envl))
	}
	data := dataField[map[string]any](t, envl)
	if data["ok"] != true {
		t.Fatalf("test ok = %v, want true", data["ok"])
	}
	if got, _ := data["latency_ms"].(float64); got != 42 {
		t.Fatalf("test latency_ms = %v, want 42", data["latency_ms"])
	}
}

func TestTestModelEmptyModelID400(t *testing.T) {
	env := newTestEnv(t)
	env.handlers.SetModelProber(fakeModelProber{ok: true, latencyMS: 1})
	token := loginToken(t, env)

	status, _ := call(t, env.handlers.RequireSession(env.handlers.TestModel),
		"POST", "/api/models/test", `{"provider":"deepseek"}`, nil,
		map[string]string{"Authorization": "Bearer " + token})
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("empty model_id status = %d, want 400", status)
	}
}

func TestModelAvailabilityReturnsList(t *testing.T) {
	env := newTestEnv(t)
	token := loginToken(t, env)

	status, envl := call(t, env.handlers.RequireSession(env.handlers.ModelAvailability),
		"GET", "/api/models/availability", "", nil,
		map[string]string{"Authorization": "Bearer " + token})
	if status != fasthttp.StatusOK {
		t.Fatalf("availability status = %d, err = %q", status, errMessage(t, envl))
	}
	data := dataField[map[string]any](t, envl)
	available, ok := data["available"].([]any)
	if !ok {
		t.Fatalf("availability data missing available array: %v", data)
	}
	if len(available) == 0 {
		t.Fatalf("availability list empty")
	}
	first, _ := available[0].(map[string]any)
	if _, ok := first["id"]; !ok {
		t.Fatalf("availability entry missing id: %v", first)
	}
	if _, ok := first["available"]; !ok {
		t.Fatalf("availability entry missing available flag: %v", first)
	}
}

func TestCustomModelAdminCRUDAndAudit(t *testing.T) {
	env := newTestEnv(t)
	admin, err := env.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}

	// Create.
	status, envl := call(t, env.handlers.CreateCustomModel, "POST", "/api/models/custom",
		`{"provider":"openai","model_id":"my-model","name":"My Model"}`,
		map[string]any{userKey: admin}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("create status = %d, err = %q", status, errMessage(t, envl))
	}
	created := dataField[map[string]any](t, envl)
	if created["is_custom"] != true {
		t.Fatalf("created is_custom = %v, want true", created["is_custom"])
	}
	if created["model_id"] != "my-model" {
		t.Fatalf("created model_id = %v", created["model_id"])
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatalf("created id empty")
	}

	// List ≥ 1.
	status, envl = call(t, env.handlers.ListCustomModels, "GET", "/api/models/custom", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	list := dataField[[]map[string]any](t, envl)
	if len(list) < 1 {
		t.Fatalf("list len = %d, want >= 1", len(list))
	}

	// Audit entry on create.
	status, envl = call(t, env.handlers.GetAudit, "GET", "/api/audit", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("audit status = %d", status)
	}
	audit := dataField[map[string]any](t, envl)
	items, _ := audit["items"].([]any)
	if len(items) == 0 {
		t.Fatalf("expected an audit entry after custom model create, got none")
	}

	// Delete.
	status, _ = call(t, env.handlers.DeleteCustomModel, "DELETE", "/api/models/custom/"+id, "",
		map[string]any{"id": id, userKey: admin}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete status = %d", status)
	}

	// Delete unknown → 404.
	status, _ = call(t, env.handlers.DeleteCustomModel, "DELETE", "/api/models/custom/nope", "",
		map[string]any{"id": "nope", userKey: admin}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("delete unknown status = %d, want 404", status)
	}
}

func TestCreateCustomModelEmptyModelID400(t *testing.T) {
	env := newTestEnv(t)
	admin, err := env.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	status, _ := call(t, env.handlers.CreateCustomModel, "POST", "/api/models/custom",
		`{"provider":"openai"}`, map[string]any{userKey: admin}, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("empty model_id status = %d, want 400", status)
	}
}
