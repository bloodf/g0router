package handlers

import (
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeMITMProxy struct {
	running     bool
	addr        string
	pem         []byte
	toolEnabled map[string]bool
}

func (f *fakeMITMProxy) IsRunning() bool        { return f.running }
func (f *fakeMITMProxy) Addr() string            { return f.addr }
func (f *fakeMITMProxy) Start() error            { f.running = true; return nil }
func (f *fakeMITMProxy) Stop() error             { f.running = false; return nil }
func (f *fakeMITMProxy) CACertPEM() []byte       { return f.pem }
func (f *fakeMITMProxy) ToolEnabled(n string) bool {
	if f.toolEnabled == nil {
		return false
	}
	return f.toolEnabled[n]
}
func (f *fakeMITMProxy) SetToolEnabled(n string, e bool) {
	if f.toolEnabled == nil {
		f.toolEnabled = make(map[string]bool)
	}
	f.toolEnabled[n] = e
}

type fakeMITMFlagStore struct {
	enabled bool
	err     error
}

func (f *fakeMITMFlagStore) ListFeatureFlags() ([]store.FeatureFlag, error) { return nil, nil }
func (f *fakeMITMFlagStore) GetFeatureFlag(id int64) (*store.FeatureFlag, error) { return nil, nil }
func (f *fakeMITMFlagStore) GetFeatureFlagByKey(key string) (*store.FeatureFlag, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &store.FeatureFlag{Key: key, Enabled: true}, nil
}
func (f *fakeMITMFlagStore) ToggleFeatureFlag(id int64, enabled bool) error { return nil }

func TestMITMStatusNilProxy(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	MITMStatus(ctx, nil, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestMITMStatus(t *testing.T) {
	p := &fakeMITMProxy{running: true, addr: "127.0.0.1:8081"}
	ctx := &fasthttp.RequestCtx{}
	MITMStatus(ctx, p, &fakeMITMFlagStore{})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	body := ctx.Response.Body()
	if !contains(body, "running") {
		t.Fatal("response missing running field")
	}
}

func TestMITMToggleNilProxy(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	MITMToggle(ctx, nil, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestMITMToggleStart(t *testing.T) {
	p := &fakeMITMProxy{}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"enabled":true}`, func(ctx *fasthttp.RequestCtx) {
		MITMToggle(ctx, p, &fakeMITMFlagStore{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if !p.running {
		t.Fatal("expected proxy to be running")
	}
}

func TestMITMToggleStop(t *testing.T) {
	p := &fakeMITMProxy{running: true}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"enabled":false}`, func(ctx *fasthttp.RequestCtx) {
		MITMToggle(ctx, p, &fakeMITMFlagStore{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if p.running {
		t.Fatal("expected proxy to be stopped")
	}
}

func TestMITMToggleInvalidJSON(t *testing.T) {
	p := &fakeMITMProxy{}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `bad`, func(ctx *fasthttp.RequestCtx) {
		MITMToggle(ctx, p, &fakeMITMFlagStore{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestMITMCACertNilProxy(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	MITMCACert(ctx, nil, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestMITMCACert(t *testing.T) {
	p := &fakeMITMProxy{pem: []byte("-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----\n")}
	ctx := &fasthttp.RequestCtx{}
	MITMCACert(ctx, p, &fakeMITMFlagStore{})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if string(ctx.Response.Body()) != string(p.pem) {
		t.Fatalf("body mismatch")
	}
}

func TestMITMToolsNilProxy(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	MITMTools(ctx, nil, nil, "cursor")
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestMITMToolsMissingID(t *testing.T) {
	p := &fakeMITMProxy{}
	ctx := &fasthttp.RequestCtx{}
	MITMTools(ctx, p, &fakeMITMFlagStore{}, "")
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestMITMToolsToggle(t *testing.T) {
	p := &fakeMITMProxy{}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"enabled":true}`, func(ctx *fasthttp.RequestCtx) {
		MITMTools(ctx, p, &fakeMITMFlagStore{}, "cursor")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if !p.ToolEnabled("cursor") {
		t.Fatal("expected cursor to be enabled")
	}
}

func TestMITMToolsInvalidJSON(t *testing.T) {
	p := &fakeMITMProxy{}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `bad`, func(ctx *fasthttp.RequestCtx) {
		MITMTools(ctx, p, &fakeMITMFlagStore{}, "cursor")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}
