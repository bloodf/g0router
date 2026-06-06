package handlers

import (
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeSettingsStore struct {
	settings store.Settings
}

func (f *fakeSettingsStore) GetSettings() (store.Settings, error) {
	return f.settings, nil
}

func (f *fakeSettingsStore) UpdateSettings(s store.Settings) error {
	f.settings = s
	return nil
}

func TestSettingsWithFakeStore(t *testing.T) {
	fs := &fakeSettingsStore{settings: store.Settings{
		RequireAPIKey: true,
		RTKEnabled:    true,
	}}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Settings(ctx, fs)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("get status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	updateBody := `{"require_api_key":false,"rtk_enabled":false}`
	ctx, body = runHandler(t, fasthttp.MethodPut, updateBody, func(ctx *fasthttp.RequestCtx) {
		Settings(ctx, fs)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("put status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	if fs.settings.RequireAPIKey || fs.settings.RTKEnabled {
		t.Fatalf("fake store not updated")
	}
}
