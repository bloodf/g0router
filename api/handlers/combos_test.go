package handlers

import (
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func TestCombosCreateListUpdateDelete(t *testing.T) {
	s := newHandlerStore(t)

	createBody := `{"name":"research-chain","steps":[{"provider":"anthropic","model":"claude-sonnet-4"},{"provider":"openai","model":"gpt-4o"}],"is_active":true}`
	ctx, body := runHandler(t, fasthttp.MethodPost, createBody, func(ctx *fasthttp.RequestCtx) {
		Combos(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
	var created store.Combo
	decodeJSON(t, body, &created)
	if created.ID == "" || created.Name != "research-chain" || len(created.Steps) != 2 {
		t.Fatalf("created = %+v", created)
	}

	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Combos(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var listed struct {
		Data []store.Combo `json:"data"`
	}
	decodeJSON(t, body, &listed)
	if len(listed.Data) != 1 || listed.Data[0].ID != created.ID {
		t.Fatalf("listed = %+v, want created combo", listed.Data)
	}

	updateBody := `{"name":"renamed","steps":[{"provider":"gemini","model":"gemini-2.5-pro"}],"is_active":false}`
	ctx, body = runHandler(t, fasthttp.MethodPut, updateBody, func(ctx *fasthttp.RequestCtx) {
		Combos(ctx, s, created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var updated store.Combo
	decodeJSON(t, body, &updated)
	if updated.Name != "renamed" || updated.IsActive || updated.Steps[0].Provider != "gemini" {
		t.Fatalf("updated = %+v", updated)
	}

	ctx, body = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		Combos(ctx, s, created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestCombosStrategyInRequestAndResponse(t *testing.T) {
	s := newHandlerStore(t)

	body := `{"name":"rr","steps":[{"provider":"openai","model":"gpt-4o"}],"strategy":"round_robin","is_active":true}`
	ctx, respBody := runHandler(t, fasthttp.MethodPost, body, func(ctx *fasthttp.RequestCtx) {
		Combos(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), respBody)
	}
	var created store.Combo
	decodeJSON(t, respBody, &created)
	if created.Strategy != "round_robin" {
		t.Fatalf("created strategy = %q, want round_robin", created.Strategy)
	}
}

func TestCombosStrategyDefaultsToFallback(t *testing.T) {
	s := newHandlerStore(t)

	body := `{"name":"plain","steps":[{"provider":"openai","model":"gpt-4o"}],"is_active":true}`
	ctx, respBody := runHandler(t, fasthttp.MethodPost, body, func(ctx *fasthttp.RequestCtx) {
		Combos(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), respBody)
	}
	var created store.Combo
	decodeJSON(t, respBody, &created)
	if created.Strategy != "fallback" {
		t.Fatalf("created strategy = %q, want fallback", created.Strategy)
	}
}

func TestCombosInvalidStrategyReturnsBadRequest(t *testing.T) {
	s := newHandlerStore(t)

	body := `{"name":"bad","steps":[{"provider":"openai","model":"gpt-4o"}],"strategy":"nonsense","is_active":true}`
	ctx, respBody := runHandler(t, fasthttp.MethodPost, body, func(ctx *fasthttp.RequestCtx) {
		Combos(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), respBody)
	}
}

func TestCombosMissingReturnsNotFound(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		Combos(ctx, s, "missing")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestCombosInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"name":`, func(ctx *fasthttp.RequestCtx) {
		Combos(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}
