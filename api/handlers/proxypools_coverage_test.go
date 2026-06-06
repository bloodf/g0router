package handlers

import (
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeProxyPoolStore struct {
	listErr   error
	getErr    error
	createErr error
	updateErr error
	deleteErr error
	testErr   error
	pools     []store.ProxyPool
	pool      *store.ProxyPool
}

func (f *fakeProxyPoolStore) ListProxyPools() ([]store.ProxyPool, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.pools, nil
}

func (f *fakeProxyPoolStore) GetProxyPool(id string) (*store.ProxyPool, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.pool != nil {
		return f.pool, nil
	}
	return nil, store.ErrNotFound
}

func (f *fakeProxyPoolStore) CreateProxyPool(pool store.ProxyPool) (*store.ProxyPool, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &pool, nil
}

func (f *fakeProxyPoolStore) UpdateProxyPool(id string, pool store.ProxyPool) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	return nil
}

func (f *fakeProxyPoolStore) DeleteProxyPool(id string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	return nil
}

func (f *fakeProxyPoolStore) TestProxyPool(id string) (bool, int, error) {
	if f.testErr != nil {
		return false, 0, f.testErr
	}
	return true, 0, nil
}

func TestProxyPoolGetNotFound(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProxyPoolGet(ctx, s, "nonexistent")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolUpdateNotFound(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	ctx, body := runHandler(t, fasthttp.MethodPut, `{"name":"updated"}`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolUpdate(ctx, s, s, "nonexistent")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolDeleteNotFound(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		ProxyPoolDelete(ctx, s, s, "nonexistent")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolBatchImportEmptyLines(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"lines":[]}`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolBatchImport(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var resp batchImportResponse
	decodeJSON(t, body, &resp)
	if len(resp.Created) != 0 || len(resp.Errors) != 0 {
		t.Fatalf("expected empty created and errors, got %+v", resp)
	}
}

func TestProxyPoolTestStoreError(t *testing.T) {
	fake := &fakeProxyPoolStore{pool: &store.ProxyPool{ID: "1", Name: "test", Protocol: "http", Host: "host", Port: 8080}, testErr: errors.New("test failed")}
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ProxyPoolTest(ctx, fake, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolListStoreError(t *testing.T) {
	fake := &fakeProxyPoolStore{listErr: errors.New("db error")}
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProxyPoolList(ctx, fake)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolCreateMissingName(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"protocol":"http","host":"host.com","port":8080}`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolCreate(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolCreateMissingHost(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"name":"test","protocol":"http","port":8080}`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolCreate(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolUpdateInvalidPort(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	pool := store.ProxyPool{Name: "test", Protocol: "http", Host: "host.com", Port: 8080}
	created, err := s.CreateProxyPool(pool)
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodPut, `{"port":99999}`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolUpdate(ctx, s, s, created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolUpdateInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	pool := store.ProxyPool{Name: "test", Protocol: "http", Host: "host.com", Port: 8080}
	created, err := s.CreateProxyPool(pool)
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodPut, `{"port":`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolUpdate(ctx, s, s, created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolBatchImportInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"lines":`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolBatchImport(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolGetEmptyID(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProxyPoolGet(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolCreateStoreError(t *testing.T) {
	fake := &fakeProxyPoolStore{createErr: errors.New("db error")}
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"name":"test","protocol":"http","host":"host.com","port":8080}`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolCreate(ctx, fake, &fakeAuthAuditWriter{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolUpdateEmptyID(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")
	ctx, body := runHandler(t, fasthttp.MethodPut, `{"name":"updated"}`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolUpdate(ctx, s, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolUpdateStoreError(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")
	pool := store.ProxyPool{Name: "test", Protocol: "http", Host: "host.com", Port: 8080}
	created, err := s.CreateProxyPool(pool)
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}
	fake := &fakeProxyPoolStore{pool: &pool, updateErr: errors.New("db error")}
	ctx, body := runHandler(t, fasthttp.MethodPut, `{"name":"updated"}`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolUpdate(ctx, fake, &fakeAuthAuditWriter{}, created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolUpdateGetAfterUpdateError(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")
	pool := store.ProxyPool{Name: "test", Protocol: "http", Host: "host.com", Port: 8080}
	created, err := s.CreateProxyPool(pool)
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}
	fake := &fakeProxyPoolStore{pool: &pool, getErr: store.ErrNotFound}
	ctx, body := runHandler(t, fasthttp.MethodPut, `{"name":"updated"}`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolUpdate(ctx, fake, &fakeAuthAuditWriter{}, created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolDeleteStoreError(t *testing.T) {
	fake := &fakeProxyPoolStore{deleteErr: errors.New("db error")}
	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		ProxyPoolDelete(ctx, fake, &fakeAuthAuditWriter{}, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolTestEmptyID(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ProxyPoolTest(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolBatchImportAuditError(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"lines":["host.com:8080"]}`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolBatchImport(ctx, s, &fakeAuthAuditWriter{appendErr: errors.New("audit error")})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var resp batchImportResponse
	decodeJSON(t, body, &resp)
	if len(resp.Created) != 1 {
		t.Fatalf("expected 1 created, got %d", len(resp.Created))
	}
}

func TestParseProxyLineInvalidURL(t *testing.T) {
	_, err := parseProxyLine("http://[::1:8080")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestParseProxyLineEmptyPort(t *testing.T) {
	_, err := parseProxyLine("http://host")
	if err == nil {
		t.Fatal("expected error for empty port")
	}
}

func TestParseProxyLineUsernameOnlyURL(t *testing.T) {
	pool, err := parseProxyLine("http://user@host:8080")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pool.Username != "user" {
		t.Fatalf("username = %q, want user", pool.Username)
	}
	if pool.Password != "" {
		t.Fatalf("password = %q, want empty", pool.Password)
	}
}

func TestProxyPoolUpdateHostUsernamePassword(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")
	pool := store.ProxyPool{Name: "test", Protocol: "http", Host: "host.com", Port: 8080}
	created, err := s.CreateProxyPool(pool)
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodPut, `{"host":"newhost.com","username":"user1","password":"pass1"}`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolUpdate(ctx, s, s, created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var resp struct {
		Data proxyPoolResponse `json:"data"`
	}
	decodeJSON(t, body, &resp)
	if resp.Data.Host != "newhost.com" {
		t.Fatalf("host = %q, want newhost.com", resp.Data.Host)
	}
	if resp.Data.Username != "user1" {
		t.Fatalf("username = %q, want user1", resp.Data.Username)
	}

	stored, err := s.GetProxyPool(created.ID)
	if err != nil {
		t.Fatalf("GetProxyPool: %v", err)
	}
	if stored.Password != "pass1" {
		t.Fatalf("password = %q, want pass1", stored.Password)
	}
}

func TestProxyPoolUpdateAuditError(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")
	pool := store.ProxyPool{Name: "test", Protocol: "http", Host: "host.com", Port: 8080}
	created, err := s.CreateProxyPool(pool)
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodPut, `{"name":"updated"}`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolUpdate(ctx, s, &fakeAuthAuditWriter{appendErr: errors.New("audit error")}, created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolDeleteAuditError(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")
	pool := store.ProxyPool{Name: "test", Protocol: "http", Host: "host.com", Port: 8080}
	created, err := s.CreateProxyPool(pool)
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		ProxyPoolDelete(ctx, s, &fakeAuthAuditWriter{appendErr: errors.New("audit error")}, created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolCreateAuditError(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"name":"test","protocol":"http","host":"host.com","port":8080}`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolCreate(ctx, s, &fakeAuthAuditWriter{appendErr: errors.New("audit error")})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolTestGenericStoreError(t *testing.T) {
	fake := &fakeProxyPoolStore{pool: &store.ProxyPool{ID: "1", Name: "test", Protocol: "http", Host: "host", Port: 8080}, getErr: errors.New("db error")}
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ProxyPoolTest(ctx, fake, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolBatchImportCreateError(t *testing.T) {
	fake := &fakeProxyPoolStore{createErr: errors.New("db error")}
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"lines":["host.com:8080"]}`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolBatchImport(ctx, fake, &fakeAuthAuditWriter{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var resp batchImportResponse
	decodeJSON(t, body, &resp)
	if len(resp.Created) != 0 || len(resp.Errors) != 1 {
		t.Fatalf("expected 0 created and 1 error, got %+v", resp)
	}
}

func TestParseProxyLineURLWithPassword(t *testing.T) {
	pool, err := parseProxyLine("http://user:pass@host:8080")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pool.Username != "user" {
		t.Fatalf("username = %q, want user", pool.Username)
	}
	if pool.Password != "pass" {
		t.Fatalf("password = %q, want pass", pool.Password)
	}
}
