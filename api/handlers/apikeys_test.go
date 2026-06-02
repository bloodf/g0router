package handlers

import (
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func TestAPIKeysCreateListDelete(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"name":"dashboard"}`, func(ctx *fasthttp.RequestCtx) {
		APIKeys(ctx, s, "test-secret", "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
	var created struct {
		Key store.APIKey `json:"key"`
		Raw string       `json:"raw"`
	}
	decodeJSON(t, body, &created)
	if created.Key.ID == "" || created.Key.Name != "dashboard" {
		t.Fatalf("created = %+v", created.Key)
	}
	if !strings.HasPrefix(created.Raw, "g0r_") {
		t.Fatalf("raw key = %q, want g0r_ prefix", created.Raw)
	}

	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		APIKeys(ctx, s, "test-secret", "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	if strings.Contains(string(body), created.Raw) {
		t.Fatalf("list response exposes raw key: %s", body)
	}
	var listed struct {
		Data []store.APIKey `json:"data"`
	}
	decodeJSON(t, body, &listed)
	if len(listed.Data) != 1 || listed.Data[0].ID != created.Key.ID {
		t.Fatalf("listed = %+v, want created key", listed.Data)
	}

	ctx, body = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		APIKeys(ctx, s, "test-secret", created.Key.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestAPIKeysInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"name":`, func(ctx *fasthttp.RequestCtx) {
		APIKeys(ctx, s, "test-secret", "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}
