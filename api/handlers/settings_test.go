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

	updateBody := `{"require_api_key":false,"rtk_enabled":false,"caveman_enabled":true,"caveman_level":"lite","enable_request_logs":true,"proxy_url":"http://proxy.local:8080","data_dir":"/tmp/g0router","log_retention_days":15}`
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
	if updated.LogRetentionDays != 15 {
		t.Fatalf("log retention = %d, want 15", updated.LogRetentionDays)
	}
}

func TestSettingsRejectsNegativeRetention(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPut, `{"log_retention_days":-5}`, func(ctx *fasthttp.RequestCtx) {
		Settings(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestSettingsRejectsRetentionOverCap(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPut, `{"log_retention_days":99999}`, func(ctx *fasthttp.RequestCtx) {
		Settings(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
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
