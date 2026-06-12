package admin

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestDisabledModelsCRUD(t *testing.T) {
	env := newTestEnv(t)

	// Initially empty — list returns empty disabled map.
	status, envl := call(t, env.handlers.GetDisabledModels, "GET", "/api/models/disabled", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("GET all status = %d, err = %q", status, errMessage(t, envl))
	}
	got := dataField[map[string]any](t, envl)
	if _, ok := got["disabled"]; !ok {
		t.Fatalf("expected 'disabled' key in response, got %v", got)
	}

	// POST to disable two models for openai.
	status, envl = call(t, env.handlers.PostDisabledModels, "POST", "/api/models/disabled",
		`{"provider_alias":"openai","ids":["gpt-4","gpt-3.5-turbo"]}`, nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("POST status = %d, err = %q", status, errMessage(t, envl))
	}

	// GET with provider_alias query param returns the two disabled IDs.
	status, envl = call(t, env.handlers.GetDisabledModels, "GET",
		"/api/models/disabled?provider_alias=openai", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("GET filtered status = %d, err = %q", status, errMessage(t, envl))
	}
	filtered := dataField[map[string]any](t, envl)
	rawIDs, ok := filtered["ids"]
	if !ok {
		t.Fatalf("expected 'ids' key in filtered response, got %v", filtered)
	}
	ids, _ := rawIDs.([]any)
	if len(ids) != 2 {
		t.Fatalf("disabled ids count = %d, want 2: %v", len(ids), ids)
	}

	// DELETE single model: remove gpt-3.5-turbo.
	status, envl = call(t, env.handlers.DeleteDisabledModels, "DELETE",
		"/api/models/disabled?provider_alias=openai&id=gpt-3.5-turbo", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("DELETE single status = %d, err = %q", status, errMessage(t, envl))
	}

	// Verify only gpt-4 remains.
	status, envl = call(t, env.handlers.GetDisabledModels, "GET",
		"/api/models/disabled?provider_alias=openai", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("GET after delete status = %d", status)
	}
	after := dataField[map[string]any](t, envl)
	afterIDs, _ := after["ids"].([]any)
	if len(afterIDs) != 1 {
		t.Fatalf("after single delete, ids count = %d, want 1: %v", len(afterIDs), afterIDs)
	}
	if afterIDs[0] != "gpt-4" {
		t.Fatalf("remaining id = %q, want gpt-4", afterIDs[0])
	}

	// DELETE all models for openai (no id param).
	status, envl = call(t, env.handlers.DeleteDisabledModels, "DELETE",
		"/api/models/disabled?provider_alias=openai", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("DELETE all status = %d, err = %q", status, errMessage(t, envl))
	}

	// Verify openai is now empty.
	status, envl = call(t, env.handlers.GetDisabledModels, "GET",
		"/api/models/disabled?provider_alias=openai", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("GET final status = %d", status)
	}
	final := dataField[map[string]any](t, envl)
	finalIDs, _ := final["ids"].([]any)
	if len(finalIDs) != 0 {
		t.Fatalf("after delete all, ids = %v, want empty", finalIDs)
	}
}

func TestDisabledModelsPostValidation(t *testing.T) {
	env := newTestEnv(t)

	// Missing provider_alias → 400.
	status, _ := call(t, env.handlers.PostDisabledModels, "POST", "/api/models/disabled",
		`{"ids":["gpt-4"]}`, nil, nil)
	if status != fasthttp.StatusBadRequest {
		t.Errorf("missing provider_alias status = %d, want 400", status)
	}

	// Invalid JSON → 400.
	status, _ = call(t, env.handlers.PostDisabledModels, "POST", "/api/models/disabled",
		`not-json`, nil, nil)
	if status != fasthttp.StatusBadRequest {
		t.Errorf("invalid JSON status = %d, want 400", status)
	}
}

func TestDisabledModelsDeleteRequiresAlias(t *testing.T) {
	env := newTestEnv(t)

	// DELETE without provider_alias → 400.
	status, _ := call(t, env.handlers.DeleteDisabledModels, "DELETE", "/api/models/disabled", "", nil, nil)
	if status != fasthttp.StatusBadRequest {
		t.Errorf("missing provider_alias delete status = %d, want 400", status)
	}
}
