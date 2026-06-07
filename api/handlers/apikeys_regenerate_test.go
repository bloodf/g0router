package handlers

import (
	"errors"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeRegenStore struct {
	apiKeyStore
	regenerateKey   *store.APIKey
	regenerateRaw   string
	regenerateErr   error
}

func (f *fakeRegenStore) RegenerateAPIKey(id string, secret string) (*store.APIKey, string, error) {
	if f.regenerateErr != nil {
		return nil, "", f.regenerateErr
	}
	return f.regenerateKey, f.regenerateRaw, nil
}

func TestRegenerateAPIKeySuccess(t *testing.T) {
	fs := &fakeRegenStore{
		regenerateKey: &store.APIKey{ID: "key-1", Name: "test", Prefix: "g0r_newp"},
		regenerateRaw: "g0r_newprefix_rest",
	}

	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		RegenerateAPIKey(ctx, fs, "secret", "key-1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var decoded struct {
		Key struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"key"`
		Raw string `json:"raw"`
	}
	decodeJSON(t, body, &decoded)
	if decoded.Key.ID != "key-1" {
		t.Errorf("id = %q, want key-1", decoded.Key.ID)
	}
	if decoded.Raw != "g0r_newprefix_rest" {
		t.Errorf("raw = %q, want g0r_newprefix_rest", decoded.Raw)
	}
	if !strings.HasPrefix(decoded.Raw, "g0r_") {
		t.Error("raw should have g0r_ prefix")
	}
}

func TestRegenerateAPIKeyMissingID(t *testing.T) {
	fs := &fakeRegenStore{}
	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		RegenerateAPIKey(ctx, fs, "secret", "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestRegenerateAPIKeyStoreError(t *testing.T) {
	fs := &fakeRegenStore{regenerateErr: errors.New("db error")}
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		RegenerateAPIKey(ctx, fs, "secret", "key-1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoInternalDetail(t, body)
}

func TestRegenerateAPIKeyNotFound(t *testing.T) {
	fs := &fakeRegenStore{regenerateErr: errors.New("key not found")}
	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		RegenerateAPIKey(ctx, fs, "secret", "missing")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestRegenerateAPIKeyNilStoreReturns503(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		RegenerateAPIKey(ctx, nil, "secret", "key-1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}
