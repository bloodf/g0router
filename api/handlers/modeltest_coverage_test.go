package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeModelTestStoreErr struct {
	getActiveConnectionsErr error
	listConnectionsErr      error
	connections             []*store.Connection
}

func (f *fakeModelTestStoreErr) GetActiveConnections(provider string) ([]*store.Connection, error) {
	if f.getActiveConnectionsErr != nil {
		return nil, f.getActiveConnectionsErr
	}
	return f.connections, nil
}

func (f *fakeModelTestStoreErr) ListConnections() ([]*store.Connection, error) {
	if f.listConnectionsErr != nil {
		return nil, f.listConnectionsErr
	}
	return f.connections, nil
}

func (f *fakeModelTestStoreErr) AppendAudit(entry store.AuditEntry) error {
	return nil
}

type fakeModelTestProvider2 struct {
	err error
}

func (f *fakeModelTestProvider2) Name() providers.ModelProvider {
	return "openai"
}

func (f *fakeModelTestProvider2) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &providers.ChatResponse{ID: "test"}, nil
}

func (f *fakeModelTestProvider2) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return nil, nil
}

func (f *fakeModelTestProvider2) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	return nil, nil
}

type fakeModelTestAdapterSource2 struct {
	provider providers.Provider
	ok       bool
}

func (f fakeModelTestAdapterSource2) GetProvider(name providers.ModelProvider) (providers.Provider, bool) {
	return f.provider, f.ok
}

func TestModelTestStoreNil(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ModelTest(ctx, nil, nil, "openai", "gpt-4o")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestModelTestBatchStoreNil(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ModelTestBatch(ctx, nil, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestModelTestAdapterSourceNil(t *testing.T) {
	s := &fakeModelTestStoreErr{connections: []*store.Connection{
		{Provider: "openai", AuthType: store.AuthTypeAPIKey, APIKey: strPtr("key"), IsActive: true},
	}}
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ModelTest(ctx, s, nil, "openai", "gpt-4o")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestModelTestBatchAdapterSourceNil(t *testing.T) {
	s := &fakeModelTestStoreErr{connections: []*store.Connection{
		{Provider: "openai", AuthType: store.AuthTypeAPIKey, APIKey: strPtr("key"), IsActive: true},
	}}
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ModelTestBatch(ctx, s, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestModelTestAdapterNotFound(t *testing.T) {
	s := &fakeModelTestStoreErr{connections: []*store.Connection{
		{Provider: "openai", AuthType: store.AuthTypeAPIKey, APIKey: strPtr("key"), IsActive: true},
	}}
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ModelTest(ctx, s, fakeModelTestAdapterSource2{ok: false}, "openai", "gpt-4o")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestModelTestGetActiveConnectionsError(t *testing.T) {
	s := &fakeModelTestStoreErr{getActiveConnectionsErr: errors.New("db error")}
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ModelTest(ctx, s, fakeModelTestAdapterSource2{provider: &fakeModelTestProvider2{}, ok: true}, "openai", "gpt-4o")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestModelTestBatchListConnectionsError(t *testing.T) {
	s := &fakeModelTestStoreErr{listConnectionsErr: errors.New("db error")}
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ModelTestBatch(ctx, s, fakeModelTestAdapterSource2{provider: &fakeModelTestProvider2{}, ok: true})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestModelTestBatchAdapterNotFoundInTestConnection(t *testing.T) {
	// Test the testConnection helper path when adapter is not found
	s := &fakeModelTestStoreErr{connections: []*store.Connection{
		{Provider: "openai", AuthType: store.AuthTypeAPIKey, APIKey: strPtr("key"), IsActive: true},
	}}
	adapter := fakeModelTestAdapterSource2{ok: false}
	result := testConnection(context.Background(), adapter, s.connections[0], nil)
	if result.OK {
		t.Fatal("expected OK=false when adapter not found")
	}
	if result.Error == nil || *result.Error != "provider adapter unavailable" {
		t.Fatalf("expected provider adapter unavailable error, got %v", result.Error)
	}
}

func TestModelTestBatchUpstreamFailureInTestConnection(t *testing.T) {
	s := &fakeModelTestStoreErr{connections: []*store.Connection{
		{Provider: "openai", AuthType: store.AuthTypeAPIKey, APIKey: strPtr("key"), IsActive: true},
	}}
	adapter := fakeModelTestAdapterSource2{provider: &fakeModelTestProvider2{err: errors.New("upstream fail")}, ok: true}
	result := testConnection(context.Background(), adapter, s.connections[0], nil)
	if result.OK {
		t.Fatal("expected OK=false when upstream fails")
	}
	if result.Error == nil || *result.Error != "upstream fail" {
		t.Fatalf("expected upstream fail error, got %v", result.Error)
	}
}

func TestPickTestModelFromCatalog(t *testing.T) {
	conn := &store.Connection{Provider: "openai"}
	model := pickTestModel(providers.ProviderOpenAI, conn)
	if model == "" {
		t.Fatal("expected non-empty model from catalog")
	}
}

func TestPickTestModelFallback(t *testing.T) {
	conn := &store.Connection{Provider: "unknown"}
	model := pickTestModel(providers.ModelProvider("unknown"), conn)
	if model != "unknown" {
		t.Fatalf("model = %q, want unknown", model)
	}
}

func TestModelTestBatchEmptyProviderListAllConnections(t *testing.T) {
	s := &fakeModelTestStoreErr{connections: []*store.Connection{
		{Provider: "openai", AuthType: store.AuthTypeAPIKey, APIKey: strPtr("key"), IsActive: true},
		{Provider: "anthropic", AuthType: store.AuthTypeAPIKey, APIKey: strPtr("key"), IsActive: true},
	}}
	adapter := fakeModelTestAdapterSource2{provider: &fakeModelTestProvider2{}, ok: true}
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"providers":[]}`, func(ctx *fasthttp.RequestCtx) {
		ModelTestBatch(ctx, s, adapter)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	_ = body
}

func TestPickTestModelWithModelLocks(t *testing.T) {
	conn := &store.Connection{Provider: "openai", ModelLocks: map[string]int64{"gpt-4o-locked": 1}}
	model := pickTestModel(providers.ProviderOpenAI, conn)
	if model != "gpt-4o-locked" {
		t.Fatalf("model = %q, want gpt-4o-locked", model)
	}
}

func TestTestConnectionWithAccessToken(t *testing.T) {
	token := "token123"
	conn := &store.Connection{
		Provider:    "openai",
		AuthType:    store.AuthTypeOAuth,
		AccessToken: &token,
		IsActive:    true,
	}
	adapter := fakeModelTestAdapterSource2{provider: &fakeModelTestProvider2{}, ok: true}
	result := testConnection(context.Background(), adapter, conn, nil)
	if !result.OK {
		t.Fatalf("expected OK, got error: %v", result.Error)
	}
}

func TestTestConnectionWithAccountID(t *testing.T) {
	key := "sk-test"
	account := "acc123"
	conn := &store.Connection{
		Provider:  "openai",
		AuthType:  store.AuthTypeAPIKey,
		APIKey:    &key,
		AccountID: &account,
		IsActive:  true,
	}
	adapter := fakeModelTestAdapterSource2{provider: &fakeModelTestProvider2{}, ok: true}
	result := testConnection(context.Background(), adapter, conn, nil)
	if !result.OK {
		t.Fatalf("expected OK, got error: %v", result.Error)
	}
}

func TestModelTestAuditAppendError(t *testing.T) {
	s := &fakeModelTestStoreErr{connections: []*store.Connection{
		{Provider: "openai", AuthType: store.AuthTypeAPIKey, APIKey: strPtr("key"), IsActive: true},
	}}
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ModelTest(ctx, s, fakeModelTestAdapterSource2{provider: &fakeModelTestProvider2{}, ok: true}, "openai", "gpt-4o")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var decoded struct {
		Data struct {
			OK        bool    `json:"ok"`
			LatencyMS int64   `json:"latency_ms"`
			Error     *string `json:"error"`
		} `json:"data"`
	}
	decodeJSON(t, body, &decoded)
	if !decoded.Data.OK {
		t.Fatalf("ok = false, want true; error=%v", decoded.Data.Error)
	}
}

func TestModelTestWithAccessToken(t *testing.T) {
	token := "access-token-123"
	s := &fakeModelTestStoreErr{connections: []*store.Connection{
		{Provider: "openai", AuthType: store.AuthTypeOAuth, AccessToken: &token, IsActive: true},
	}}
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ModelTest(ctx, s, fakeModelTestAdapterSource2{provider: &fakeModelTestProvider2{}, ok: true}, "openai", "gpt-4o")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestModelTestWithAccountID(t *testing.T) {
	key := "sk-test"
	account := "acc123"
	s := &fakeModelTestStoreErr{connections: []*store.Connection{
		{Provider: "openai", AuthType: store.AuthTypeAPIKey, APIKey: &key, AccountID: &account, IsActive: true},
	}}
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ModelTest(ctx, s, fakeModelTestAdapterSource2{provider: &fakeModelTestProvider2{}, ok: true}, "openai", "gpt-4o")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
}
