package handlers

import (
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func TestPricingOverridesCreateListUpdateDelete(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"provider":"openai","model":"gpt-4o-mini","input_cost_per_token":0.000001,"output_cost_per_token":0.000002}`, func(ctx *fasthttp.RequestCtx) {
		Pricing(ctx, s, "", "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
	var created store.PricingOverride
	decodeJSON(t, body, &created)
	if created.Provider != "openai" || created.Model != "gpt-4o-mini" || created.InputCostPerToken != 0.000001 || created.OutputCostPerToken != 0.000002 {
		t.Fatalf("created = %+v", created)
	}

	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Pricing(ctx, s, "", "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var listed struct {
		Data []store.PricingOverride `json:"data"`
	}
	decodeJSON(t, body, &listed)
	if len(listed.Data) != 1 || listed.Data[0].Provider != "openai" {
		t.Fatalf("listed = %+v", listed.Data)
	}

	ctx, body = runHandler(t, fasthttp.MethodPut, `{"input_cost_per_token":0.000003,"output_cost_per_token":0.000004}`, func(ctx *fasthttp.RequestCtx) {
		Pricing(ctx, s, "openai", "gpt-4o-mini")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var updated store.PricingOverride
	decodeJSON(t, body, &updated)
	if updated.Provider != "openai" || updated.Model != "gpt-4o-mini" || updated.InputCostPerToken != 0.000003 || updated.OutputCostPerToken != 0.000004 {
		t.Fatalf("updated = %+v", updated)
	}

	ctx, body = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		Pricing(ctx, s, "openai", "gpt-4o-mini")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestPricingOverridesMissingReturnsNotFound(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		Pricing(ctx, s, "openai", "missing")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
}
