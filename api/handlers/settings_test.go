package handlers

import (
	"strings"
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

func TestSettingsCacheFieldsRoundTrip(t *testing.T) {
	s := newHandlerStore(t)

	updateBody := `{"cache_enabled":true,"cache_ttl_seconds":600}`
	ctx, body := runHandler(t, fasthttp.MethodPut, updateBody, func(ctx *fasthttp.RequestCtx) {
		Settings(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("put status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var updated store.Settings
	decodeJSON(t, body, &updated)
	if !updated.CacheEnabled || updated.CacheTTLSeconds != 600 {
		t.Fatalf("updated = %+v, want cache enabled ttl 600", updated)
	}
}

func TestSettingsRejectsNegativeCacheTTL(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPut, `{"cache_ttl_seconds":-1}`, func(ctx *fasthttp.RequestCtx) {
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

func TestSettingsRequireLoginRejectsNoUsers(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPut, `{"require_login":true}`, func(ctx *fasthttp.RequestCtx) {
		Settings(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", ctx.Response.StatusCode(), body)
	}
	if !strings.Contains(string(body), "require_login cannot be enabled without at least one dashboard user") {
		t.Fatalf("body = %s, want conflict message", body)
	}
}

func TestSettingsRequireLoginAcceptsWithUsers(t *testing.T) {
	s := newHandlerStore(t)

	if _, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin"); err != nil {
		t.Fatalf("CreateDashboardUser: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodPut, `{"require_login":true}`, func(ctx *fasthttp.RequestCtx) {
		Settings(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var updated store.Settings
	decodeJSON(t, body, &updated)
	if !updated.RequireLogin {
		t.Fatalf("RequireLogin = false, want true")
	}
}

func TestSettingsRequireLoginFalseAlwaysWorks(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPut, `{"require_login":false}`, func(ctx *fasthttp.RequestCtx) {
		Settings(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var updated store.Settings
	decodeJSON(t, body, &updated)
	if updated.RequireLogin {
		t.Fatalf("RequireLogin = true, want false")
	}
}

func TestSettingsGetIncludesRequireLoginAndTrustProxyHeaders(t *testing.T) {
	s := newHandlerStore(t)

	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.RequireLogin = true
	settings.TrustProxyHeaders = true
	if _, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin"); err != nil {
		t.Fatalf("CreateDashboardUser: %v", err)
	}
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Settings(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("get status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var got store.Settings
	decodeJSON(t, body, &got)
	if !got.RequireLogin {
		t.Error("RequireLogin missing or false in GET response")
	}
	if !got.TrustProxyHeaders {
		t.Error("TrustProxyHeaders missing or false in GET response")
	}
}

func TestSettingsUpdateAPIKeySecret(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPut, `{"require_api_key":false,"api_key_secret":"new-secret-value"}`, func(ctx *fasthttp.RequestCtx) {
		Settings(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("put status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	secret, err := s.GetAPIKeySecret()
	if err != nil {
		t.Fatalf("GetAPIKeySecret: %v", err)
	}
	if secret != "new-secret-value" {
		t.Fatalf("secret = %q, want new-secret-value", secret)
	}
}
