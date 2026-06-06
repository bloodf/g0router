package handlers

import (
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func TestDisabledModelsCreateListDeleteRoundTrip(t *testing.T) {
	s := newHandlerStore(t)

	// Create
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"provider":"openai","model":"gpt-4o"}`, func(ctx *fasthttp.RequestCtx) {
		DisabledModelsCreate(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
	var created disabledModelResponse
	decodeJSON(t, body, &created)
	if created.Provider != "openai" || created.Model != "gpt-4o" {
		t.Fatalf("created = %+v", created)
	}

	// List
	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		DisabledModelsList(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var listed struct {
		Data []disabledModelResponse `json:"data"`
	}
	decodeJSON(t, body, &listed)
	if len(listed.Data) != 1 || listed.Data[0].Model != "gpt-4o" {
		t.Fatalf("listed = %+v", listed.Data)
	}

	// Delete
	ctx, body = runHandler(t, fasthttp.MethodDelete, `{"provider":"openai","model":"gpt-4o"}`, func(ctx *fasthttp.RequestCtx) {
		DisabledModelsDelete(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}

	// Verify deleted
	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		DisabledModelsList(ctx, s)
	})
	var afterDelete struct {
		Data []disabledModelResponse `json:"data"`
	}
	decodeJSON(t, body, &afterDelete)
	if len(afterDelete.Data) != 0 {
		t.Fatalf("expected empty list after delete, got %+v", afterDelete.Data)
	}
}

func TestCustomModelsCreateListDeleteRoundTrip(t *testing.T) {
	s := newHandlerStore(t)

	// Create
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"provider":"openai","model":"gpt-custom","display_name":"My Custom"}`, func(ctx *fasthttp.RequestCtx) {
		CustomModelsCreate(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
	var created customModelResponse
	decodeJSON(t, body, &created)
	if created.Provider != "openai" || created.Model != "gpt-custom" || created.DisplayName != "My Custom" {
		t.Fatalf("created = %+v", created)
	}
	if !created.IsCustom {
		t.Fatal("expected IsCustom=true")
	}

	// List
	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		CustomModelsList(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var listed struct {
		Data []customModelResponse `json:"data"`
	}
	decodeJSON(t, body, &listed)
	if len(listed.Data) != 1 || listed.Data[0].Model != "gpt-custom" {
		t.Fatalf("listed = %+v", listed.Data)
	}

	// Delete
	ctx, body = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		CustomModelsDelete(ctx, s, s, created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}

	// Verify deleted
	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		CustomModelsList(ctx, s)
	})
	var afterDelete struct {
		Data []customModelResponse `json:"data"`
	}
	decodeJSON(t, body, &afterDelete)
	if len(afterDelete.Data) != 0 {
		t.Fatalf("expected empty list after delete, got %+v", afterDelete.Data)
	}
}

func TestDisabledModelsCreateWritesAuditLog(t *testing.T) {
	s := newHandlerStore(t)

	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"provider":"openai","model":"gpt-4o"}`, func(ctx *fasthttp.RequestCtx) {
		DisabledModelsCreate(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201", ctx.Response.StatusCode())
	}

	entries, total, err := s.ListAudit(store.AuditFilter{})
	if err != nil {
		t.Fatalf("ListAudit: %v", err)
	}
	if total == 0 {
		t.Fatal("expected audit log entry")
	}
	found := false
	for _, e := range entries {
		if e.Action == "disabled_model.create" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected disabled_model.create audit entry, got %+v", entries)
	}
}

func TestCustomModelsCreateWritesAuditLog(t *testing.T) {
	s := newHandlerStore(t)

	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"provider":"openai","model":"gpt-custom"}`, func(ctx *fasthttp.RequestCtx) {
		CustomModelsCreate(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201", ctx.Response.StatusCode())
	}

	entries, total, err := s.ListAudit(store.AuditFilter{})
	if err != nil {
		t.Fatalf("ListAudit: %v", err)
	}
	if total == 0 {
		t.Fatal("expected audit log entry")
	}
	found := false
	for _, e := range entries {
		if e.Action == "custom_model.create" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected custom_model.create audit entry, got %+v", entries)
	}
}

func TestDisabledModelsDeleteMissingReturnsNotFound(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodDelete, `{"provider":"missing","model":"model"}`, func(ctx *fasthttp.RequestCtx) {
		DisabledModelsDelete(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestCustomModelsDeleteMissingReturnsNotFound(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		CustomModelsDelete(ctx, s, s, "nonexistent-id")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestDisabledModelsInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"provider":`, func(ctx *fasthttp.RequestCtx) {
		DisabledModelsCreate(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestDisabledModelsMissingFields(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"provider":"openai"}`, func(ctx *fasthttp.RequestCtx) {
		DisabledModelsCreate(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}
