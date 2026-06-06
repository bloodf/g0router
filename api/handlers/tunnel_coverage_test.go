package handlers

import (
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func TestTunnelListStoreError(t *testing.T) {
	ts := &fakeTunnelStore{
		listTunnelConfigsFunc: func() ([]store.TunnelConfig, error) {
			return nil, errors.New("list failed")
		},
	}
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		TunnelList(ctx, ts)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestTunnelCloudflareCreateInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)
	mgr := &fakeTunnelManager{}
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"port":`, func(ctx *fasthttp.RequestCtx) {
		TunnelCloudflareCreate(ctx, s, mgr, s, "3000")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestTunnelCloudflareCreateStartError(t *testing.T) {
	s := newHandlerStore(t)
	mgr := &fakeTunnelManager{
		startCloudflareFunc: func(port string) (string, error) {
			return "", errors.New("start failed")
		},
	}
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"port":"8080"}`, func(ctx *fasthttp.RequestCtx) {
		TunnelCloudflareCreate(ctx, s, mgr, s, "3000")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestTunnelCloudflareCreateAuditError(t *testing.T) {
	s := newHandlerStore(t)
	mgr := &fakeTunnelManager{}
	aw := &fakeAuditWriter{err: errors.New("audit failed")}
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"port":"8080"}`, func(ctx *fasthttp.RequestCtx) {
		TunnelCloudflareCreate(ctx, s, mgr, aw, "3000")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestTunnelCloudflareDeleteNilStore(t *testing.T) {
	mgr := &fakeTunnelManager{}
	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		TunnelCloudflareDelete(ctx, nil, mgr, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestTunnelCloudflareDeleteStopError(t *testing.T) {
	s := newHandlerStore(t)
	mgr := &fakeTunnelManager{
		stopCloudflareFunc: func() error {
			return errors.New("stop failed")
		},
	}
	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		TunnelCloudflareDelete(ctx, s, mgr, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestTunnelCloudflareDeleteAuditError(t *testing.T) {
	s := newHandlerStore(t)
	mgr := &fakeTunnelManager{}
	aw := &fakeAuditWriter{err: errors.New("audit failed")}
	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		TunnelCloudflareDelete(ctx, s, mgr, aw)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestTunnelTailscaleCreateInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)
	mgr := &fakeTunnelManager{}
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"port":`, func(ctx *fasthttp.RequestCtx) {
		TunnelTailscaleCreate(ctx, s, mgr, s, "3000")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestTunnelTailscaleCreateStartError(t *testing.T) {
	s := newHandlerStore(t)
	mgr := &fakeTunnelManager{
		startTailscaleFunc: func(port string) (string, error) {
			return "", errors.New("start failed")
		},
	}
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"port":"8080"}`, func(ctx *fasthttp.RequestCtx) {
		TunnelTailscaleCreate(ctx, s, mgr, s, "3000")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestTunnelTailscaleCreateDefaultPortError(t *testing.T) {
	s := newHandlerStore(t)
	mgr := &fakeTunnelManager{
		startTailscaleFunc: func(port string) (string, error) {
			if port != "3000" {
				t.Fatalf("expected default port 3000, got %s", port)
			}
			return "", errors.New("start failed")
		},
	}
	ctx, body := runHandler(t, fasthttp.MethodPost, `{}`, func(ctx *fasthttp.RequestCtx) {
		TunnelTailscaleCreate(ctx, s, mgr, s, "3000")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestTunnelTailscaleCreateAuditError(t *testing.T) {
	s := newHandlerStore(t)
	mgr := &fakeTunnelManager{}
	aw := &fakeAuditWriter{err: errors.New("audit failed")}
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"port":"8080"}`, func(ctx *fasthttp.RequestCtx) {
		TunnelTailscaleCreate(ctx, s, mgr, aw, "3000")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestTunnelTailscaleDeleteNilStore(t *testing.T) {
	mgr := &fakeTunnelManager{}
	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		TunnelTailscaleDelete(ctx, nil, mgr, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestTunnelTailscaleDeleteStopError(t *testing.T) {
	s := newHandlerStore(t)
	mgr := &fakeTunnelManager{
		stopTailscaleFunc: func() error {
			return errors.New("stop failed")
		},
	}
	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		TunnelTailscaleDelete(ctx, s, mgr, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestTunnelTailscaleDeleteAuditError(t *testing.T) {
	s := newHandlerStore(t)
	mgr := &fakeTunnelManager{}
	aw := &fakeAuditWriter{err: errors.New("audit failed")}
	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		TunnelTailscaleDelete(ctx, s, mgr, aw)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestTunnelHealthNilStore(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		TunnelHealth(ctx, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestTunnelHealthListError(t *testing.T) {
	ts := &fakeTunnelStore{
		listTunnelConfigsFunc: func() ([]store.TunnelConfig, error) {
			return nil, errors.New("list failed")
		},
	}
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		TunnelHealth(ctx, ts)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyTestInvalidURL(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"url":"://invalid"}`, func(ctx *fasthttp.RequestCtx) {
		ProxyTest(ctx)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	if !contains(body, `"ok":false`) {
		t.Fatalf("expected ok=false, got: %s", body)
	}
}

func TestProxyTestHTTPSPort(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"url":"https://127.0.0.1"}`, func(ctx *fasthttp.RequestCtx) {
		ProxyTest(ctx)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	if !contains(body, `"ok":false`) {
		t.Fatalf("expected ok=false, got: %s", body)
	}
}

func TestProxyTestSocks5Port(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"url":"socks5://127.0.0.1"}`, func(ctx *fasthttp.RequestCtx) {
		ProxyTest(ctx)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	if !contains(body, `"ok":false`) {
		t.Fatalf("expected ok=false, got: %s", body)
	}
}

func TestProxyTestHTTPPort(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"url":"http://127.0.0.1"}`, func(ctx *fasthttp.RequestCtx) {
		ProxyTest(ctx)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	if !contains(body, `"ok":false`) {
		t.Fatalf("expected ok=false, got: %s", body)
	}
}

func TestProxyTestDefaultPort(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"url":"ftp://127.0.0.1"}`, func(ctx *fasthttp.RequestCtx) {
		ProxyTest(ctx)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	if !contains(body, `"ok":false`) {
		t.Fatalf("expected ok=false, got: %s", body)
	}
}

type fakeTunnelStore struct {
	listTunnelConfigsFunc func() ([]store.TunnelConfig, error)
}

func (f *fakeTunnelStore) ListTunnelConfigs() ([]store.TunnelConfig, error) {
	if f.listTunnelConfigsFunc != nil {
		return f.listTunnelConfigsFunc()
	}
	return nil, nil
}

func (f *fakeTunnelStore) UpsertTunnelConfig(cfg store.TunnelConfig) error {
	return nil
}

func (f *fakeTunnelStore) UpdateTunnelStatus(tunnelType, status, lastError string) error {
	return nil
}

type fakeAuditWriter struct {
	err error
}

func (f *fakeAuditWriter) AppendAudit(entry store.AuditEntry) error {
	return f.err
}

func contains(b []byte, s string) bool {
	return len(b) >= len(s) && string(b) == s || len(b) > len(s) && (string(b[:len(s)]) == s || string(b[len(b)-len(s):]) == s || containsSubstring(b, s))
}

func containsSubstring(b []byte, s string) bool {
	for i := 0; i <= len(b)-len(s); i++ {
		if string(b[i:i+len(s)]) == s {
			return true
		}
	}
	return false
}
