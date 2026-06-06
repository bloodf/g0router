package handlers

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakePromptTemplateStore struct {
	templates []store.PromptTemplate
	nextID    int64
}

func (f *fakePromptTemplateStore) ListPromptTemplates() ([]store.PromptTemplate, error) {
	return f.templates, nil
}

func (f *fakePromptTemplateStore) GetPromptTemplate(id int64) (*store.PromptTemplate, error) {
	for i := range f.templates {
		if f.templates[i].ID == id {
			return &f.templates[i], nil
		}
	}
	return nil, store.ErrNotFound
}

func (f *fakePromptTemplateStore) CreatePromptTemplate(name, systemPrompt string, models []string, isActive bool) (*store.PromptTemplate, error) {
	f.nextID++
	t := store.PromptTemplate{
		ID: f.nextID, Name: name, SystemPrompt: systemPrompt,
		Models: models, IsActive: isActive, CreatedAt: "now", UpdatedAt: "now",
	}
	f.templates = append(f.templates, t)
	return &t, nil
}

func (f *fakePromptTemplateStore) UpdatePromptTemplate(id int64, name, systemPrompt string, models []string, isActive bool) error {
	for i := range f.templates {
		if f.templates[i].ID == id {
			f.templates[i].Name = name
			f.templates[i].SystemPrompt = systemPrompt
			f.templates[i].Models = models
			f.templates[i].IsActive = isActive
			return nil
		}
	}
	return store.ErrNotFound
}

func (f *fakePromptTemplateStore) DeletePromptTemplate(id int64) error {
	for i := range f.templates {
		if f.templates[i].ID == id {
			f.templates = append(f.templates[:i], f.templates[i+1:]...)
			return nil
		}
	}
	return store.ErrNotFound
}

func TestPromptTemplatesList(t *testing.T) {
	s := &fakePromptTemplateStore{
		templates: []store.PromptTemplate{
			{ID: 1, Name: "t1", SystemPrompt: "p1", Models: []string{"m1"}, IsActive: true},
		},
	}
	ctx := newTestCtx(fasthttp.MethodGet, "/api/prompt-templates", nil)
	PromptTemplates(ctx, s, "")
	assertStatus(t, ctx, fasthttp.StatusOK)

	var resp listResponse[promptTemplateResponse]
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("len = %d, want 1", len(resp.Data))
	}
}

func TestPromptTemplatesGet(t *testing.T) {
	s := &fakePromptTemplateStore{
		templates: []store.PromptTemplate{
			{ID: 1, Name: "t1", SystemPrompt: "p1", Models: []string{"m1"}, IsActive: true},
		},
	}
	ctx := newTestCtx(fasthttp.MethodGet, "/api/prompt-templates/1", nil)
	PromptTemplates(ctx, s, "1")
	assertStatus(t, ctx, fasthttp.StatusOK)

	var resp promptTemplateResponse
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ID != 1 {
		t.Errorf("id = %d, want 1", resp.ID)
	}
}

func TestPromptTemplatesGetNotFound(t *testing.T) {
	s := &fakePromptTemplateStore{}
	ctx := newTestCtx(fasthttp.MethodGet, "/api/prompt-templates/1", nil)
	PromptTemplates(ctx, s, "1")
	assertStatus(t, ctx, fasthttp.StatusNotFound)
}

func TestPromptTemplatesCreate(t *testing.T) {
	s := &fakePromptTemplateStore{}
	body, _ := json.Marshal(promptTemplateRequest{
		Name: "t1", SystemPrompt: "p1", Models: []string{"m1"}, IsActive: true,
	})
	ctx := newTestCtx(fasthttp.MethodPost, "/api/prompt-templates", body)
	PromptTemplates(ctx, s, "")
	assertStatus(t, ctx, fasthttp.StatusCreated)

	var resp promptTemplateResponse
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Name != "t1" {
		t.Errorf("name = %q, want t1", resp.Name)
	}
}

func TestPromptTemplatesCreateMissingName(t *testing.T) {
	s := &fakePromptTemplateStore{}
	body, _ := json.Marshal(promptTemplateRequest{
		SystemPrompt: "p1",
	})
	ctx := newTestCtx(fasthttp.MethodPost, "/api/prompt-templates", body)
	PromptTemplates(ctx, s, "")
	assertStatus(t, ctx, fasthttp.StatusBadRequest)
}

func TestPromptTemplatesUpdate(t *testing.T) {
	s := &fakePromptTemplateStore{
		templates: []store.PromptTemplate{
			{ID: 1, Name: "t1", SystemPrompt: "p1", Models: []string{"m1"}, IsActive: true},
		},
	}
	body, _ := json.Marshal(promptTemplateRequest{
		Name: "t1-updated", SystemPrompt: "p1-updated", Models: []string{"m2"}, IsActive: false,
	})
	ctx := newTestCtx(fasthttp.MethodPut, "/api/prompt-templates/1", body)
	PromptTemplates(ctx, s, "1")
	assertStatus(t, ctx, fasthttp.StatusNoContent)
}

func TestPromptTemplatesUpdateNotFound(t *testing.T) {
	s := &fakePromptTemplateStore{}
	body, _ := json.Marshal(promptTemplateRequest{
		Name: "t1", SystemPrompt: "p1", Models: []string{"m1"}, IsActive: true,
	})
	ctx := newTestCtx(fasthttp.MethodPut, "/api/prompt-templates/1", body)
	PromptTemplates(ctx, s, "1")
	assertStatus(t, ctx, fasthttp.StatusNotFound)
}

func TestPromptTemplatesDelete(t *testing.T) {
	s := &fakePromptTemplateStore{
		templates: []store.PromptTemplate{
			{ID: 1, Name: "t1", SystemPrompt: "p1", Models: []string{"m1"}, IsActive: true},
		},
	}
	ctx := newTestCtx(fasthttp.MethodDelete, "/api/prompt-templates/1", nil)
	PromptTemplates(ctx, s, "1")
	assertStatus(t, ctx, fasthttp.StatusNoContent)
	if len(s.templates) != 0 {
		t.Errorf("templates len = %d, want 0", len(s.templates))
	}
}

func TestPromptTemplatesDeleteNotFound(t *testing.T) {
	s := &fakePromptTemplateStore{}
	ctx := newTestCtx(fasthttp.MethodDelete, "/api/prompt-templates/1", nil)
	PromptTemplates(ctx, s, "1")
	assertStatus(t, ctx, fasthttp.StatusNotFound)
}

func TestPromptTemplatesNilStore(t *testing.T) {
	ctx := newTestCtx(fasthttp.MethodGet, "/api/prompt-templates", nil)
	PromptTemplates(ctx, nil, "")
	assertStatus(t, ctx, fasthttp.StatusServiceUnavailable)
}

func TestPromptTemplatesInvalidMethod(t *testing.T) {
	s := &fakePromptTemplateStore{}
	ctx := newTestCtx(fasthttp.MethodPatch, "/api/prompt-templates", nil)
	PromptTemplates(ctx, s, "")
	assertStatus(t, ctx, fasthttp.StatusMethodNotAllowed)
}

func TestPromptTemplatesTest(t *testing.T) {
	s := &fakePromptTemplateStore{
		templates: []store.PromptTemplate{
			{ID: 1, Name: "t1", SystemPrompt: "p1", Models: []string{"gpt-4"}, IsActive: true},
			{ID: 2, Name: "t2", SystemPrompt: "p2", Models: []string{"gpt-3.5"}, IsActive: false},
		},
	}
	body, _ := json.Marshal(map[string]string{"model": "gpt-4"})
	ctx := newTestCtx(fasthttp.MethodPost, "/api/prompt-templates/test", body)
	PromptTemplatesTest(ctx, s)
	assertStatus(t, ctx, fasthttp.StatusOK)

	var resp struct {
		Model     string                   `json:"model"`
		Matched   bool                     `json:"matched"`
		Templates []promptTemplateResponse `json:"templates"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Matched {
		t.Error("expected matched")
	}
	if len(resp.Templates) != 1 {
		t.Errorf("templates = %d, want 1", len(resp.Templates))
	}
}

func TestPromptTemplatesTestNoMatch(t *testing.T) {
	s := &fakePromptTemplateStore{
		templates: []store.PromptTemplate{
			{ID: 1, Name: "t1", SystemPrompt: "p1", Models: []string{"gpt-4"}, IsActive: true},
		},
	}
	body, _ := json.Marshal(map[string]string{"model": "unknown"})
	ctx := newTestCtx(fasthttp.MethodPost, "/api/prompt-templates/test", body)
	PromptTemplatesTest(ctx, s)
	assertStatus(t, ctx, fasthttp.StatusOK)

	var resp struct {
		Matched bool `json:"matched"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Matched {
		t.Error("expected no match")
	}
}

func TestPromptTemplatesTestMissingModel(t *testing.T) {
	s := &fakePromptTemplateStore{}
	body, _ := json.Marshal(map[string]string{})
	ctx := newTestCtx(fasthttp.MethodPost, "/api/prompt-templates/test", body)
	PromptTemplatesTest(ctx, s)
	assertStatus(t, ctx, fasthttp.StatusBadRequest)
}

func TestPromptTemplatesTestNilStore(t *testing.T) {
	ctx := newTestCtx(fasthttp.MethodPost, "/api/prompt-templates/test", nil)
	PromptTemplatesTest(ctx, nil)
	assertStatus(t, ctx, fasthttp.StatusServiceUnavailable)
}

func TestPromptTemplatesTestInvalidMethod(t *testing.T) {
	s := &fakePromptTemplateStore{}
	ctx := newTestCtx(fasthttp.MethodGet, "/api/prompt-templates/test", nil)
	PromptTemplatesTest(ctx, s)
	assertStatus(t, ctx, fasthttp.StatusMethodNotAllowed)
}

func TestPromptTemplatesTestInvalidJSON(t *testing.T) {
	s := &fakePromptTemplateStore{}
	ctx := newTestCtx(fasthttp.MethodPost, "/api/prompt-templates/test", []byte("not-json"))
	PromptTemplatesTest(ctx, s)
	assertStatus(t, ctx, fasthttp.StatusBadRequest)
}

func TestPromptTemplatesCreateInvalidJSON(t *testing.T) {
	s := &fakePromptTemplateStore{}
	ctx := newTestCtx(fasthttp.MethodPost, "/api/prompt-templates", []byte("not-json"))
	PromptTemplates(ctx, s, "")
	assertStatus(t, ctx, fasthttp.StatusBadRequest)
}

func TestPromptTemplatesUpdateInvalidJSON(t *testing.T) {
	s := &fakePromptTemplateStore{}
	ctx := newTestCtx(fasthttp.MethodPut, "/api/prompt-templates/1", []byte("not-json"))
	PromptTemplates(ctx, s, "1")
	assertStatus(t, ctx, fasthttp.StatusBadRequest)
}

type fakePromptTemplateStoreWithErrors struct {
	fakePromptTemplateStore
	listErr    error
	createErr  error
	updateErr  error
	deleteErr  error
}

func (f *fakePromptTemplateStoreWithErrors) ListPromptTemplates() ([]store.PromptTemplate, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.fakePromptTemplateStore.ListPromptTemplates()
}

func (f *fakePromptTemplateStoreWithErrors) CreatePromptTemplate(name, systemPrompt string, models []string, isActive bool) (*store.PromptTemplate, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	return f.fakePromptTemplateStore.CreatePromptTemplate(name, systemPrompt, models, isActive)
}

func (f *fakePromptTemplateStoreWithErrors) UpdatePromptTemplate(id int64, name, systemPrompt string, models []string, isActive bool) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	return f.fakePromptTemplateStore.UpdatePromptTemplate(id, name, systemPrompt, models, isActive)
}

func (f *fakePromptTemplateStoreWithErrors) DeletePromptTemplate(id int64) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	return f.fakePromptTemplateStore.DeletePromptTemplate(id)
}

func TestPromptTemplatesListError(t *testing.T) {
	s := &fakePromptTemplateStoreWithErrors{listErr: errors.New("boom")}
	ctx := newTestCtx(fasthttp.MethodGet, "/api/prompt-templates", nil)
	PromptTemplates(ctx, s, "")
	assertStatus(t, ctx, fasthttp.StatusInternalServerError)
}

func TestPromptTemplatesCreateError(t *testing.T) {
	s := &fakePromptTemplateStoreWithErrors{createErr: errors.New("boom")}
	body, _ := json.Marshal(promptTemplateRequest{Name: "t1", SystemPrompt: "p1"})
	ctx := newTestCtx(fasthttp.MethodPost, "/api/prompt-templates", body)
	PromptTemplates(ctx, s, "")
	assertStatus(t, ctx, fasthttp.StatusInternalServerError)
}

func TestPromptTemplatesUpdateError(t *testing.T) {
	s := &fakePromptTemplateStoreWithErrors{updateErr: errors.New("boom")}
	body, _ := json.Marshal(promptTemplateRequest{Name: "t1", SystemPrompt: "p1"})
	ctx := newTestCtx(fasthttp.MethodPut, "/api/prompt-templates/1", body)
	PromptTemplates(ctx, s, "1")
	assertStatus(t, ctx, fasthttp.StatusInternalServerError)
}

func TestPromptTemplatesDeleteError(t *testing.T) {
	s := &fakePromptTemplateStoreWithErrors{deleteErr: errors.New("boom")}
	ctx := newTestCtx(fasthttp.MethodDelete, "/api/prompt-templates/1", nil)
	PromptTemplates(ctx, s, "1")
	assertStatus(t, ctx, fasthttp.StatusInternalServerError)
}

func TestPromptTemplatesTestListError(t *testing.T) {
	s := &fakePromptTemplateStoreWithErrors{listErr: errors.New("boom")}
	body, _ := json.Marshal(map[string]string{"model": "gpt-4"})
	ctx := newTestCtx(fasthttp.MethodPost, "/api/prompt-templates/test", body)
	PromptTemplatesTest(ctx, s)
	assertStatus(t, ctx, fasthttp.StatusInternalServerError)
}

func TestPromptTemplatesGetInvalidID(t *testing.T) {
	s := &fakePromptTemplateStore{}
	ctx := newTestCtx(fasthttp.MethodGet, "/api/prompt-templates/abc", nil)
	PromptTemplates(ctx, s, "abc")
	assertStatus(t, ctx, fasthttp.StatusBadRequest)
}

func TestPromptTemplatesUpdateMissingID(t *testing.T) {
	s := &fakePromptTemplateStore{}
	body, _ := json.Marshal(promptTemplateRequest{Name: "n", SystemPrompt: "s"})
	ctx := newTestCtx(fasthttp.MethodPut, "/api/prompt-templates", body)
	PromptTemplates(ctx, s, "")
	assertStatus(t, ctx, fasthttp.StatusBadRequest)
}

func TestPromptTemplatesDeleteMissingID(t *testing.T) {
	s := &fakePromptTemplateStore{}
	ctx := newTestCtx(fasthttp.MethodDelete, "/api/prompt-templates", nil)
	PromptTemplates(ctx, s, "")
	assertStatus(t, ctx, fasthttp.StatusBadRequest)
}

func newTestCtx(method, path string, body []byte) *fasthttp.RequestCtx {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(method)
	ctx.Request.SetRequestURI(path)
	if body != nil {
		ctx.Request.SetBody(body)
	}
	return ctx
}

func assertStatus(t *testing.T, ctx *fasthttp.RequestCtx, want int) {
	t.Helper()
	if ctx.Response.StatusCode() != want {
		t.Errorf("status = %d, want %d", ctx.Response.StatusCode(), want)
	}
}
