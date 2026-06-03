package handlers

import (
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func TestAliasesCreateListUpdateDelete(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"alias":"fast","provider":"openai","model":"gpt-4o-mini"}`, func(ctx *fasthttp.RequestCtx) {
		Aliases(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
	var created store.ModelAlias
	decodeJSON(t, body, &created)
	if created != (store.ModelAlias{Alias: "fast", Provider: "openai", Model: "gpt-4o-mini"}) {
		t.Fatalf("created = %+v", created)
	}

	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Aliases(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var listed struct {
		Data []store.ModelAlias `json:"data"`
	}
	decodeJSON(t, body, &listed)
	if len(listed.Data) != 1 || listed.Data[0].Alias != "fast" {
		t.Fatalf("listed = %+v", listed.Data)
	}

	ctx, body = runHandler(t, fasthttp.MethodPut, `{"provider":"groq","model":"llama-3.3-70b-versatile"}`, func(ctx *fasthttp.RequestCtx) {
		Aliases(ctx, s, "fast")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var updated store.ModelAlias
	decodeJSON(t, body, &updated)
	if updated != (store.ModelAlias{Alias: "fast", Provider: "groq", Model: "llama-3.3-70b-versatile"}) {
		t.Fatalf("updated = %+v", updated)
	}

	ctx, body = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		Aliases(ctx, s, "fast")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestAliasesMissingReturnsNotFound(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		Aliases(ctx, s, "missing")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
}
