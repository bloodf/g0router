package handlers

import (
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeConnectionStore struct {
	listErr   error
	createErr error
	updateErr error
	getErr    error
	deleteErr error
	conns     []*store.Connection
}

func (f *fakeConnectionStore) ListConnections() ([]*store.Connection, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.conns, nil
}

func (f *fakeConnectionStore) CreateConnection(c *store.Connection) error {
	return f.createErr
}

func (f *fakeConnectionStore) UpdateConnection(c *store.Connection) error {
	return f.updateErr
}

func (f *fakeConnectionStore) GetConnection(id string) (*store.Connection, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return nil, store.ErrNotFound
}

func (f *fakeConnectionStore) DeleteConnection(id string) error {
	return f.deleteErr
}

func TestConnectionsPutGetAfterUpdateError(t *testing.T) {
	fs := &fakeConnectionStore{getErr: errors.New("db error")}
	ctx, body := runHandler(t, fasthttp.MethodPut, `{"provider":"openai","name":"x","auth_type":"api_key","api_key":"sk","is_active":true}`, func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, fs, "id-1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoInternalDetail(t, body)
}

func TestProviderConnectionsMethodNotAllowed(t *testing.T) {
	fs := &fakeConnectionStore{}
	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ProviderConnections(ctx, fs, "openai")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", ctx.Response.StatusCode())
	}
}

func TestProviderConnectionsStoreNil(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderConnections(ctx, nil, "openai")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProviderConnectionsListError(t *testing.T) {
	fs := &fakeConnectionStore{listErr: errors.New("db error")}
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderConnections(ctx, fs, "openai")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoInternalDetail(t, body)
}

func TestIsStoreNilWithStructValue(t *testing.T) {
	if isStoreNil(42) {
		t.Fatal("isStoreNil(42) should be false")
	}
}
