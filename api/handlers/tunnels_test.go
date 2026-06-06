package handlers

import (
	"bytes"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeTunnelManager struct {
	startCloudflareFunc func(port string) (string, error)
	stopCloudflareFunc  func() error
	startTailscaleFunc  func(port string) (string, error)
	stopTailscaleFunc   func() error
}

func (f *fakeTunnelManager) StartCloudflare(port string) (string, error) {
	if f.startCloudflareFunc != nil {
		return f.startCloudflareFunc(port)
	}
	return "https://example.trycloudflare.com", nil
}

func (f *fakeTunnelManager) StopCloudflare() error {
	if f.stopCloudflareFunc != nil {
		return f.stopCloudflareFunc()
	}
	return nil
}

func (f *fakeTunnelManager) StartTailscale(port string) (string, error) {
	if f.startTailscaleFunc != nil {
		return f.startTailscaleFunc(port)
	}
	return "https://tailscale.example.com", nil
}

func (f *fakeTunnelManager) StopTailscale() error {
	if f.stopTailscaleFunc != nil {
		return f.stopTailscaleFunc()
	}
	return nil
}

func TestTunnelList(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-tunnels")

	_ = s.UpsertTunnelConfig(store.TunnelConfig{Type: "cloudflare", IsEnabled: true, URL: "https://cf.example.com", Status: "active", Config: "secret1"})
	_ = s.UpsertTunnelConfig(store.TunnelConfig{Type: "tailscale", IsEnabled: false, URL: "", Status: "inactive", Config: "secret2"})

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		TunnelList(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	if bytes.Contains(body, []byte(`"config"`)) {
		t.Fatalf("response contains config field: %s", body)
	}

	var resp struct {
		Data []tunnelResponse `json:"data"`
	}
	decodeJSON(t, body, &resp)
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 tunnels, got %d", len(resp.Data))
	}
	if resp.Data[0].Type != "cloudflare" || !resp.Data[0].IsEnabled {
		t.Fatalf("unexpected first tunnel: %+v", resp.Data[0])
	}
	if resp.Data[1].Type != "tailscale" || resp.Data[1].IsEnabled {
		t.Fatalf("unexpected second tunnel: %+v", resp.Data[1])
	}
}

func TestTunnelCloudflareCreate(t *testing.T) {
	s := newHandlerStore(t)
	mgr := &fakeTunnelManager{}

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"port":"8080"}`, func(ctx *fasthttp.RequestCtx) {
		TunnelCloudflareCreate(ctx, s, mgr, s, "3000")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}

	var resp struct {
		URL    string `json:"url"`
		Status string `json:"status"`
	}
	decodeJSON(t, body, &resp)
	if resp.URL == "" || resp.Status != "active" {
		t.Fatalf("unexpected response: %+v", resp)
	}

	entry := lastAuditEntry(t, s, "tunnel.cloudflare.create")
	if entry == nil || entry.Target != "8080" {
		t.Fatalf("audit entry = %+v", entry)
	}
}

func TestTunnelCloudflareCreateDefaultPort(t *testing.T) {
	s := newHandlerStore(t)
	mgr := &fakeTunnelManager{
		startCloudflareFunc: func(port string) (string, error) {
			if port != "3000" {
				t.Fatalf("expected default port 3000, got %s", port)
			}
			return "https://cf.example.com", nil
		},
	}

	ctx, body := runHandler(t, fasthttp.MethodPost, `{}`, func(ctx *fasthttp.RequestCtx) {
		TunnelCloudflareCreate(ctx, s, mgr, s, "3000")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}

	var resp struct {
		URL    string `json:"url"`
		Status string `json:"status"`
	}
	decodeJSON(t, body, &resp)
	if resp.URL != "https://cf.example.com" {
		t.Fatalf("unexpected url: %s", resp.URL)
	}
}

func TestTunnelCloudflareDelete(t *testing.T) {
	s := newHandlerStore(t)
	mgr := &fakeTunnelManager{}

	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		TunnelCloudflareDelete(ctx, s, mgr, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}

	entry := lastAuditEntry(t, s, "tunnel.cloudflare.delete")
	if entry == nil {
		t.Fatalf("expected audit entry")
	}
}

func TestTunnelTailscaleCreate(t *testing.T) {
	s := newHandlerStore(t)
	mgr := &fakeTunnelManager{}

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"port":"8080"}`, func(ctx *fasthttp.RequestCtx) {
		TunnelTailscaleCreate(ctx, s, mgr, s, "3000")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}

	var resp struct {
		URL    string `json:"url"`
		Status string `json:"status"`
	}
	decodeJSON(t, body, &resp)
	if resp.URL == "" || resp.Status != "active" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestTunnelTailscaleCreateNotFound(t *testing.T) {
	s := newHandlerStore(t)
	mgr := &fakeTunnelManager{
		startTailscaleFunc: func(port string) (string, error) {
			return "", errors.New("tailscale not found on PATH")
		},
	}

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"port":"8080"}`, func(ctx *fasthttp.RequestCtx) {
		TunnelTailscaleCreate(ctx, s, mgr, s, "3000")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusConflict {
		t.Fatalf("create status = %d, want 409; body=%s", ctx.Response.StatusCode(), body)
	}
	if !bytes.Contains(body, []byte("tailscale is not installed")) {
		t.Fatalf("expected install instructions, got: %s", body)
	}
}

func TestTunnelTailscaleDelete(t *testing.T) {
	s := newHandlerStore(t)
	mgr := &fakeTunnelManager{}

	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		TunnelTailscaleDelete(ctx, s, mgr, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestTunnelHealth(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-tunnels")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	_ = s.UpsertTunnelConfig(store.TunnelConfig{Type: "cloudflare", IsEnabled: true, URL: ts.URL, Status: "active"})
	_ = s.UpsertTunnelConfig(store.TunnelConfig{Type: "tailscale", IsEnabled: false, URL: "http://localhost:99999", Status: "inactive"})

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		TunnelHealth(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("health status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var resp struct {
		Data []struct {
			Type      string `json:"type"`
			URL       string `json:"url"`
			Reachable bool   `json:"reachable"`
			LatencyMS int    `json:"latency_ms"`
		} `json:"data"`
	}
	decodeJSON(t, body, &resp)
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 health result, got %d", len(resp.Data))
	}
	if resp.Data[0].Type != "cloudflare" || !resp.Data[0].Reachable {
		t.Fatalf("unexpected health result: %+v", resp.Data[0])
	}
	if resp.Data[0].LatencyMS < 0 {
		t.Fatalf("latency should be non-negative")
	}
}

func TestProxyTestValid(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().(*net.TCPAddr)
	proxyURL := "http://127.0.0.1:" + strconv.Itoa(addr.Port)

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		conn.Close()
	}()

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"url":"`+proxyURL+`"}`, func(ctx *fasthttp.RequestCtx) {
		ProxyTest(ctx)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("proxy test status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var resp proxyTestResponse
	decodeJSON(t, body, &resp)
	if !resp.OK {
		t.Fatalf("expected ok=true, got ok=%v, error=%s", resp.OK, resp.Error)
	}
	if resp.LatencyMS < 0 {
		t.Fatalf("latency should be non-negative")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for connection accept")
	}
}

func TestProxyTestInvalid(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"url":"http://127.0.0.1:1"}`, func(ctx *fasthttp.RequestCtx) {
		ProxyTest(ctx)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("proxy test status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var resp proxyTestResponse
	decodeJSON(t, body, &resp)
	if resp.OK {
		t.Fatalf("expected ok=false")
	}
	if resp.Error == "" {
		t.Fatalf("expected error message")
	}
}

func TestProxyTestInvalidJSON(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"url":`, func(ctx *fasthttp.RequestCtx) {
		ProxyTest(ctx)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
	_ = body
}

func TestProxyTestMissingURL(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodPost, `{}`, func(ctx *fasthttp.RequestCtx) {
		ProxyTest(ctx)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
	_ = body
}

func TestTunnelListStoreUnavailable(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		TunnelList(ctx, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
	_ = body
}

func TestTunnelCloudflareCreateStoreUnavailable(t *testing.T) {
	mgr := &fakeTunnelManager{}
	ctx, body := runHandler(t, fasthttp.MethodPost, `{}`, func(ctx *fasthttp.RequestCtx) {
		TunnelCloudflareCreate(ctx, nil, mgr, nil, "3000")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
	_ = body
}

func TestTunnelTailscaleCreateStoreUnavailable(t *testing.T) {
	mgr := &fakeTunnelManager{}
	ctx, body := runHandler(t, fasthttp.MethodPost, `{}`, func(ctx *fasthttp.RequestCtx) {
		TunnelTailscaleCreate(ctx, nil, mgr, nil, "3000")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
	_ = body
}
