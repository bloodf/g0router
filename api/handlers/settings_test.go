package handlers

import (
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func TestSettingsGetAndUpdate(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Settings(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("get status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var defaults store.Settings
	decodeJSON(t, body, &defaults)
	if !defaults.RequireAPIKey || !defaults.RTKEnabled || defaults.CavemanEnabled {
		t.Fatalf("defaults = %+v", defaults)
	}

	updateBody := `{"RequireAPIKey":false,"RTKEnabled":false,"CavemanEnabled":true,"CavemanLevel":"lite","EnableRequestLogs":true,"ProxyURL":"http://proxy.local:8080","DataDir":"/tmp/g0router"}`
	ctx, body = runHandler(t, fasthttp.MethodPut, updateBody, func(ctx *fasthttp.RequestCtx) {
		Settings(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("put status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var updated store.Settings
	decodeJSON(t, body, &updated)
	if updated.RequireAPIKey || updated.RTKEnabled || !updated.CavemanEnabled || updated.CavemanLevel != "lite" {
		t.Fatalf("updated = %+v", updated)
	}
}

func TestSettingsInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPut, `{"RequireAPIKey":`, func(ctx *fasthttp.RequestCtx) {
		Settings(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}
