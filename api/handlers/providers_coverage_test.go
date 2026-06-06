package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// ---- ProviderDetail error branches ----

func TestProviderDetailMethodNotAllowed(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ProviderDetail(ctx, nil, nil, "openai")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("got %d, want 405", ctx.Response.StatusCode())
	}
}

func TestProviderDetailStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderDetail(ctx, nil, nil, "openai")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("got %d, want 503", ctx.Response.StatusCode())
	}
}

func TestProviderDetailListConnectionsError(t *testing.T) {
	fs := &fakeProviderDetailStore{listConnectionsErr: errors.New("boom")}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderDetail(ctx, fs, nil, "openai")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("got %d, want 500", ctx.Response.StatusCode())
	}
}

func TestProviderDetailListModelsError(t *testing.T) {
	fs := &fakeProviderDetailStore{}
	ms := &fakeProviderDetailModelSource{err: errors.New("boom")}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderDetail(ctx, fs, ms, "openai")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("got %d, want 200 (error is logged but 200 returned)", ctx.Response.StatusCode())
	}
}

// ---- ProviderSuggestedModels error branches ----

func TestProviderSuggestedModelsMethodNotAllowed(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ProviderSuggestedModels(ctx, nil, nil, "openai")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("got %d, want 405", ctx.Response.StatusCode())
	}
}

func TestProviderSuggestedModelsStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderSuggestedModels(ctx, nil, nil, "openai")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("got %d, want 503", ctx.Response.StatusCode())
	}
}

func TestProviderSuggestedModelsGetActiveConnectionsError(t *testing.T) {
	fs := &fakeSuggestedModelsStore{getActiveConnectionsErr: errors.New("boom")}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderSuggestedModels(ctx, fs, nil, "openai")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("got %d, want 500", ctx.Response.StatusCode())
	}
}

func TestProviderSuggestedModelsNoActiveConnections(t *testing.T) {
	fs := &fakeSuggestedModelsStore{activeConnections: []*store.Connection{}}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderSuggestedModels(ctx, fs, nil, "openai")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("got %d, want 400", ctx.Response.StatusCode())
	}
}

func TestProviderSuggestedModelsAdapterSourceNil(t *testing.T) {
	fs := &fakeSuggestedModelsStore{activeConnections: []*store.Connection{
		{ID: "c1", Provider: "openai", AuthType: store.AuthTypeAPIKey, IsActive: true},
	}}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderSuggestedModels(ctx, fs, nil, "openai")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("got %d, want 503", ctx.Response.StatusCode())
	}
}

func TestProviderSuggestedModelsAdapterNotFound(t *testing.T) {
	fs := &fakeSuggestedModelsStore{activeConnections: []*store.Connection{
		{ID: "c1", Provider: "openai", AuthType: store.AuthTypeAPIKey, IsActive: true},
	}}
	as := &fakeSuggestedModelsAdapterSource{}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderSuggestedModels(ctx, fs, as, "openai")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("got %d, want 503", ctx.Response.StatusCode())
	}
}

func TestProviderSuggestedModelsListModelsError(t *testing.T) {
	fs := &fakeSuggestedModelsStore{activeConnections: []*store.Connection{
		{ID: "c1", Provider: "openai", AuthType: store.AuthTypeAPIKey, IsActive: true},
	}}
	as := &fakeSuggestedModelsAdapterSource{adapter: &fakeSuggestedModelsAdapter{err: errors.New("boom")}}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderSuggestedModels(ctx, fs, as, "openai")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadGateway {
		t.Fatalf("got %d, want 502", ctx.Response.StatusCode())
	}
}

func TestProviderSuggestedModelsAccessTokenAndAccountID(t *testing.T) {
	tok := "tok"
	acct := "acct"
	fs := &fakeSuggestedModelsStore{activeConnections: []*store.Connection{
		{ID: "c1", Provider: "openai", AuthType: store.AuthTypeOAuth, AccessToken: &tok, AccountID: &acct, IsActive: true},
	}}
	as := &fakeSuggestedModelsAdapterSource{adapter: &fakeSuggestedModelsAdapter{models: []providers.Model{{ID: "m1"}}}}
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderSuggestedModels(ctx, fs, as, "openai")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("got %d, want 200", ctx.Response.StatusCode())
	}
	var resp listResponse[suggestedModelResponse]
	decodeJSON(t, body, &resp)
	if len(resp.Data) != 1 || resp.Data[0].ID != "m1" {
		t.Fatalf("unexpected response: %s", body)
	}
}

// ---- fakes ----

type fakeProviderDetailStore struct {
	listConnectionsErr error
}

func (f *fakeProviderDetailStore) ListConnections() ([]*store.Connection, error) {
	if f.listConnectionsErr != nil {
		return nil, f.listConnectionsErr
	}
	return nil, nil
}

func (f *fakeProviderDetailStore) GetConnectionProxyPoolID(connectionID string) (*string, error) {
	return nil, nil
}

type fakeProviderDetailModelSource struct {
	err error
}

func (f *fakeProviderDetailModelSource) ListModels(ctx context.Context) ([]providers.Model, error) {
	return nil, f.err
}

type fakeSuggestedModelsStore struct {
	activeConnections       []*store.Connection
	getActiveConnectionsErr error
}

func (f *fakeSuggestedModelsStore) GetActiveConnections(provider string) ([]*store.Connection, error) {
	if f.getActiveConnectionsErr != nil {
		return nil, f.getActiveConnectionsErr
	}
	return f.activeConnections, nil
}

type fakeSuggestedModelsAdapterSource struct {
	adapter providers.Provider
}

func (f *fakeSuggestedModelsAdapterSource) GetProvider(name providers.ModelProvider) (providers.Provider, bool) {
	if f.adapter == nil {
		return nil, false
	}
	return f.adapter, true
}

type fakeSuggestedModelsAdapter struct {
	models []providers.Model
	err    error
}

func (f *fakeSuggestedModelsAdapter) Name() providers.ModelProvider { return "openai" }
func (f *fakeSuggestedModelsAdapter) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	return nil, nil
}
func (f *fakeSuggestedModelsAdapter) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return nil, nil
}
func (f *fakeSuggestedModelsAdapter) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	return f.models, f.err
}
