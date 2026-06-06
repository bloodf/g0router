package handlers

import (
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeVirtualKeyStore struct {
	listErr      error
	createKey    *store.VirtualKey
	createRaw    string
	createErr    error
	getKey       *store.VirtualKey
	getErr       error
	updateErr    error
	deleteErr    error
}

func (f *fakeVirtualKeyStore) ListVirtualKeys() ([]store.VirtualKey, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return nil, nil
}

func (f *fakeVirtualKeyStore) CreateVirtualKey(name string, teamID *int64, budgetUSD *float64, budgetPeriod string, rateLimitRPM, rateLimitTPM *int) (*store.VirtualKey, string, error) {
	if f.createErr != nil {
		return nil, "", f.createErr
	}
	return f.createKey, f.createRaw, nil
}

func (f *fakeVirtualKeyStore) GetVirtualKey(id int64) (*store.VirtualKey, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.getKey, nil
}

func (f *fakeVirtualKeyStore) UpdateVirtualKey(id int64, name string, teamID *int64, budgetUSD *float64, budgetPeriod string, rateLimitRPM, rateLimitTPM *int, isActive bool) error {
	return f.updateErr
}

func (f *fakeVirtualKeyStore) DeleteVirtualKey(id int64) error {
	return f.deleteErr
}

func TestVirtualKeysListError(t *testing.T) {
	fs := &fakeVirtualKeyStore{listErr: errors.New("db error")}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, fs, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestVirtualKeysGetError(t *testing.T) {
	fs := &fakeVirtualKeyStore{getErr: errors.New("db error")}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestVirtualKeysCreateError(t *testing.T) {
	fs := &fakeVirtualKeyStore{createErr: errors.New("db error")}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"name":"vk"}`, func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, fs, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestVirtualKeysUpdateNotFound(t *testing.T) {
	fs := &fakeVirtualKeyStore{updateErr: store.ErrNotFound}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"vk"}`, func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestVirtualKeysUpdateError(t *testing.T) {
	fs := &fakeVirtualKeyStore{updateErr: errors.New("db error")}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"vk"}`, func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestVirtualKeysUpdateGetAfterUpdateNotFound(t *testing.T) {
	fs := &fakeVirtualKeyStore{getKey: &store.VirtualKey{ID: 1, Name: "vk"}}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"vk"}`, func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
}

func TestVirtualKeysUpdateGetAfterUpdateError(t *testing.T) {
	fs := &fakeVirtualKeyStore{updateErr: nil, getErr: store.ErrNotFound}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"vk"}`, func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestVirtualKeysDeleteError(t *testing.T) {
	fs := &fakeVirtualKeyStore{deleteErr: errors.New("db error")}
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestVirtualKeysPutInvalidID(t *testing.T) {
	fs := &fakeVirtualKeyStore{}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"vk"}`, func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, fs, "abc")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestVirtualKeysPutInvalidJSON(t *testing.T) {
	fs := &fakeVirtualKeyStore{}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":`, func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestVirtualKeysDeleteInvalidID(t *testing.T) {
	fs := &fakeVirtualKeyStore{}
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, fs, "abc")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}
