package handlers

import (
	"bytes"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeAPIKeyStore struct {
	listErr         error
	createKey       *store.APIKey
	createRaw       string
	createErr       error
	updatePolicyErr error
	getErr          error
	deleteErr       error
}

func (f *fakeAPIKeyStore) ListAPIKeys() ([]store.APIKey, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return nil, nil
}

func (f *fakeAPIKeyStore) CreateAPIKey(name, secret string) (*store.APIKey, string, error) {
	if f.createErr != nil {
		return nil, "", f.createErr
	}
	return f.createKey, f.createRaw, nil
}

func (f *fakeAPIKeyStore) UpdateAPIKeyPolicy(id string, policy store.APIKeyPolicy) error {
	return f.updatePolicyErr
}

func (f *fakeAPIKeyStore) GetAPIKey(id string) (*store.APIKey, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.createKey, nil
}

func (f *fakeAPIKeyStore) DeleteAPIKey(id string) error {
	return f.deleteErr
}

func TestAPIKeysCreateUpdatePolicyError(t *testing.T) {
	fs := &fakeAPIKeyStore{
		createKey: &store.APIKey{ID: "key-1", Name: "test"},
		createRaw: "raw-key",
		updatePolicyErr: errors.New("db error"),
	}
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"name":"test","scopes":["gpt-*"]}`, func(ctx *fasthttp.RequestCtx) {
		APIKeys(ctx, fs, "secret", "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoInternalDetail(t, body)
}

func TestAPIKeysCreateReloadError(t *testing.T) {
	fs := &fakeAPIKeyStore{
		createKey: &store.APIKey{ID: "key-1", Name: "test"},
		createRaw: "raw-key",
		getErr:    errors.New("db error"),
	}
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"name":"test","scopes":["gpt-*"]}`, func(ctx *fasthttp.RequestCtx) {
		APIKeys(ctx, fs, "secret", "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoInternalDetail(t, body)
}

func TestAPIKeysPutGetError(t *testing.T) {
	fs := &fakeAPIKeyStore{
		getErr: store.ErrNotFound,
	}
	ctx, body := runHandler(t, fasthttp.MethodPut, `{"rate_limit_rpm":10}`, func(ctx *fasthttp.RequestCtx) {
		APIKeys(ctx, fs, "secret", "key-1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
	if !bytes.Contains(body, []byte("api key not found")) {
		t.Fatalf("body = %s, want api key not found", body)
	}
}
