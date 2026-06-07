package handlers

import (
	"errors"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

func TestLocaleGetDefault(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Locale(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded struct {
		Data struct {
			Locale string `json:"locale"`
		} `json:"data"`
	}
	decodeJSON(t, body, &decoded)
	if decoded.Data.Locale != "en" {
		t.Fatalf("locale = %q, want en", decoded.Data.Locale)
	}
}

func TestLocalePostUpdates(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"locale":"pt-BR"}`, func(ctx *fasthttp.RequestCtx) {
		Locale(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded struct {
		Data struct {
			Locale string `json:"locale"`
		} `json:"data"`
	}
	decodeJSON(t, body, &decoded)
	if decoded.Data.Locale != "pt-BR" {
		t.Fatalf("locale = %q, want pt-BR", decoded.Data.Locale)
	}

	// Verify persistence
	ctx2, body2 := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Locale(ctx, s)
	})
	if ctx2.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("get status = %d, want 200; body=%s", ctx2.Response.StatusCode(), body2)
	}
	decodeJSON(t, body2, &decoded)
	if decoded.Data.Locale != "pt-BR" {
		t.Fatalf("persisted locale = %q, want pt-BR", decoded.Data.Locale)
	}
}

func TestLocalePostInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPost, `{`, func(ctx *fasthttp.RequestCtx) {
		Locale(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestLocalePostEmpty(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"locale":""}`, func(ctx *fasthttp.RequestCtx) {
		Locale(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
	if !strings.Contains(string(body), "locale") {
		t.Fatalf("body = %s, want locale validation error", body)
	}
}

func TestLocaleStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Locale(ctx, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestLocaleMethodNotAllowed(t *testing.T) {
	s := newHandlerStore(t)

	ctx, _ := runHandler(t, fasthttp.MethodPut, ``, func(ctx *fasthttp.RequestCtx) {
		Locale(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", ctx.Response.StatusCode())
	}
}

func TestLocaleGetSettingsError(t *testing.T) {
	fs := &fakeSettingsStore{getSettingsErr: errors.New("boom")}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Locale(ctx, fs)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}
