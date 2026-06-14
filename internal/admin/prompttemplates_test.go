package admin

import (
	"strconv"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func TestPromptTemplateCRUDHandlers(t *testing.T) {
	env := newTestEnv(t)

	admin, err := env.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	actor := map[string]any{userKey: admin}

	// Create.
	status, envl := call(t, env.handlers.CreatePromptTemplate, "POST", "/api/prompt-templates",
		`{"name":"Code Review","system_prompt":"Be concise.","models":["gpt-4o","claude-sonnet-4"],"is_active":true}`,
		actor, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d err = %q", status, errMessage(t, envl))
	}
	created := dataField[map[string]any](t, envl)
	idF, ok := created["id"].(float64)
	if !ok {
		t.Fatalf("id is not numeric: %v", created["id"])
	}
	if created["name"] != "Code Review" || created["system_prompt"] != "Be concise." {
		t.Fatalf("created = %v", created)
	}
	if created["is_active"] != true {
		t.Fatalf("is_active = %v, want true", created["is_active"])
	}
	models, _ := created["models"].([]any)
	if len(models) != 2 || models[0] != "gpt-4o" {
		t.Fatalf("models = %v", created["models"])
	}
	if _, leaked := created["updated_at"]; leaked {
		t.Fatalf("DTO must not surface updated_at: %v", created)
	}
	id := "1"
	_ = idF

	// Empty name → 400.
	status, _ = call(t, env.handlers.CreatePromptTemplate, "POST", "/api/prompt-templates",
		`{"name":"","system_prompt":"x"}`, actor, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("empty-name create status = %d, want 400", status)
	}

	// List.
	status, envl = call(t, env.handlers.ListPromptTemplates, "GET", "/api/prompt-templates", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	list := dataField[[]map[string]any](t, envl)
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}

	// Get.
	status, envl = call(t, env.handlers.GetPromptTemplate, "GET", "/api/prompt-templates/"+id, "",
		map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("get status = %d", status)
	}
	if dataField[map[string]any](t, envl)["name"] != "Code Review" {
		t.Fatalf("get name mismatch")
	}

	// Get missing → 404.
	status, _ = call(t, env.handlers.GetPromptTemplate, "GET", "/api/prompt-templates/99", "",
		map[string]any{"id": "99"}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("get missing status = %d, want 404", status)
	}

	// Update.
	status, envl = call(t, env.handlers.UpdatePromptTemplate, "PUT", "/api/prompt-templates/"+id,
		`{"name":"Code Review v2","system_prompt":"Shorter.","models":["gpt-4o"],"is_active":false}`,
		map[string]any{"id": id, userKey: admin}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("update status = %d err = %q", status, errMessage(t, envl))
	}
	updated := dataField[map[string]any](t, envl)
	if updated["name"] != "Code Review v2" || updated["is_active"] != false {
		t.Fatalf("updated = %v", updated)
	}

	// Update missing → 404.
	status, _ = call(t, env.handlers.UpdatePromptTemplate, "PUT", "/api/prompt-templates/99",
		`{"name":"x"}`, map[string]any{"id": "99", userKey: admin}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("update missing status = %d, want 404", status)
	}

	// Delete.
	status, _ = call(t, env.handlers.DeletePromptTemplate, "DELETE", "/api/prompt-templates/"+id, "",
		map[string]any{"id": id, userKey: admin}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete status = %d", status)
	}
	status, _ = call(t, env.handlers.DeletePromptTemplate, "DELETE", "/api/prompt-templates/"+id, "",
		map[string]any{"id": id, userKey: admin}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("delete missing status = %d, want 404", status)
	}
}

func TestCreatePromptTemplateWritesAudit(t *testing.T) {
	env := newTestEnv(t)
	admin, err := env.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}

	status, _ := call(t, env.handlers.CreatePromptTemplate, "POST", "/api/prompt-templates",
		`{"name":"Audited","system_prompt":"x","is_active":true}`,
		map[string]any{userKey: admin}, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d", status)
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
		t.Fatalf("expected an audit entry after prompt create, total=%d", page.Total)
	}
	if page.Items[0]["actor"] != "admin" || page.Items[0]["target"] != "Audited" {
		t.Fatalf("audit entry = %v", page.Items[0])
	}
}

func TestTestPromptTemplateEndpoint(t *testing.T) {
	env := newTestEnv(t)

	// Inline system_prompt + sample.
	status, envl := call(t, env.handlers.TestPromptTemplate, "POST", "/api/prompt-templates/test",
		`{"system_prompt":"You are helpful.","sample":"Hello"}`, nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("test status = %d err = %q", status, errMessage(t, envl))
	}
	out := dataField[map[string]any](t, envl)
	rendered, ok := out["rendered"].(string)
	if !ok || rendered == "" {
		t.Fatalf("rendered = %v", out["rendered"])
	}

	// Resolve from a stored template by prompt_id.
	created, err := env.store.CreatePromptTemplate(&store.PromptTemplate{
		Name:         "Stored",
		SystemPrompt: "STORED-PROMPT",
		IsActive:     true,
	})
	if err != nil {
		t.Fatalf("CreatePromptTemplate: %v", err)
	}
	body := `{"prompt_id":` + strconv.FormatInt(created.ID, 10) + `,"sample":"World"}`
	status, envl = call(t, env.handlers.TestPromptTemplate, "POST", "/api/prompt-templates/test", body, nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("test by id status = %d err = %q", status, errMessage(t, envl))
	}
	rendered, _ = dataField[map[string]any](t, envl)["rendered"].(string)
	if rendered == "" {
		t.Fatalf("rendered from stored template is empty")
	}
}
