package handlers

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestFeatureFlagsGet(t *testing.T) {
	s := newHandlerStore(t)
	flags, err := s.ListFeatureFlags()
	if err != nil {
		t.Fatalf("ListFeatureFlags: %v", err)
	}
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		FeatureFlags(ctx, s, string(rune('0'+flags[0].ID)))
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("get status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var got struct {
		Key string `json:"key"`
	}
	decodeJSON(t, body, &got)
	if got.Key != flags[0].Key {
		t.Fatalf("key = %q, want %q", got.Key, flags[0].Key)
	}
}

func TestFeatureFlagsPutInvalidID(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"enabled":true}`, func(ctx *fasthttp.RequestCtx) {
		FeatureFlags(ctx, s, "abc")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}
