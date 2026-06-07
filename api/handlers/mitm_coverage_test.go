package handlers

import (
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeMITMFlagStoreErr struct {
	err error
}

func (f *fakeMITMFlagStoreErr) ListFeatureFlags() ([]store.FeatureFlag, error) { return nil, nil }
func (f *fakeMITMFlagStoreErr) GetFeatureFlag(id int64) (*store.FeatureFlag, error) {
	return nil, nil
}
func (f *fakeMITMFlagStoreErr) GetFeatureFlagByKey(key string) (*store.FeatureFlag, error) {
	return nil, f.err
}
func (f *fakeMITMFlagStoreErr) ToggleFeatureFlag(id int64, enabled bool) error { return nil }

type fakeMITMFlagStoreDisabled struct{}

func (f *fakeMITMFlagStoreDisabled) ListFeatureFlags() ([]store.FeatureFlag, error) { return nil, nil }
func (f *fakeMITMFlagStoreDisabled) GetFeatureFlag(id int64) (*store.FeatureFlag, error) {
	return nil, nil
}
func (f *fakeMITMFlagStoreDisabled) GetFeatureFlagByKey(key string) (*store.FeatureFlag, error) {
	return &store.FeatureFlag{Key: "mitm_proxy", Enabled: false}, nil
}
func (f *fakeMITMFlagStoreDisabled) ToggleFeatureFlag(id int64, enabled bool) error { return nil }

type fakeMITMFlagStoreNilFlag struct{}

func (f *fakeMITMFlagStoreNilFlag) ListFeatureFlags() ([]store.FeatureFlag, error) { return nil, nil }
func (f *fakeMITMFlagStoreNilFlag) GetFeatureFlag(id int64) (*store.FeatureFlag, error) {
	return nil, nil
}
func (f *fakeMITMFlagStoreNilFlag) GetFeatureFlagByKey(key string) (*store.FeatureFlag, error) {
	return nil, nil
}
func (f *fakeMITMFlagStoreNilFlag) ToggleFeatureFlag(id int64, enabled bool) error { return nil }

type fakeMITMProxyErr struct {
	fakeMITMProxy
	startErr error
	stopErr  error
}

func (f *fakeMITMProxyErr) Start() error { return f.startErr }
func (f *fakeMITMProxyErr) Stop() error  { return f.stopErr }

func TestMITMStatusNilProxyWithStore(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		MITMStatus(ctx, nil, &fakeMITMFlagStore{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestMITMCACertNilProxyWithStore(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		MITMCACert(ctx, nil, &fakeMITMFlagStore{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestMITMToolsNilProxyWithStore(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"enabled":true}`, func(ctx *fasthttp.RequestCtx) {
		MITMTools(ctx, nil, &fakeMITMFlagStore{}, "cursor")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestMITMRequireFlagStoreError(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		MITMStatus(ctx, &fakeMITMProxy{}, &fakeMITMFlagStoreErr{err: errors.New("db fail")})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestMITMRequireFlagDisabled(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		MITMStatus(ctx, &fakeMITMProxy{}, &fakeMITMFlagStoreDisabled{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestMITMRequireFlagNilFlag(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		MITMStatus(ctx, &fakeMITMProxy{}, &fakeMITMFlagStoreNilFlag{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestMITMStatusEmptyAddr(t *testing.T) {
	p := &fakeMITMProxy{}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		MITMStatus(ctx, p, &fakeMITMFlagStore{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	body := ctx.Response.Body()
	if !contains(body, "not started") {
		t.Fatal("expected 'not started' in response")
	}
}

func TestMITMToggleStartError(t *testing.T) {
	p := &fakeMITMProxyErr{startErr: errors.New("start fail")}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"enabled":true}`, func(ctx *fasthttp.RequestCtx) {
		MITMToggle(ctx, p, &fakeMITMFlagStore{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestMITMToggleStopError(t *testing.T) {
	p := &fakeMITMProxyErr{fakeMITMProxy: fakeMITMProxy{running: true}, stopErr: errors.New("stop fail")}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"enabled":false}`, func(ctx *fasthttp.RequestCtx) {
		MITMToggle(ctx, p, &fakeMITMFlagStore{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestMITMCACertFlagStoreError(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		MITMCACert(ctx, &fakeMITMProxy{}, &fakeMITMFlagStoreErr{err: errors.New("db fail")})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestMITMToolsFlagStoreError(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"enabled":true}`, func(ctx *fasthttp.RequestCtx) {
		MITMTools(ctx, &fakeMITMProxy{}, &fakeMITMFlagStoreErr{err: errors.New("db fail")}, "cursor")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestMITMToolsFlagDisabled(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"enabled":true}`, func(ctx *fasthttp.RequestCtx) {
		MITMTools(ctx, &fakeMITMProxy{}, &fakeMITMFlagStoreDisabled{}, "cursor")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}
