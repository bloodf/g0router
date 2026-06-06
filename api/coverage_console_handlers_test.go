package api

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestHandleConsoleLogsStreamMethodNotAllowed(t *testing.T) {
	srv := NewServer(ServerConfig{})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	srv.handleConsoleLogsStream(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", ctx.Response.StatusCode())
	}
}

func TestHandleConsoleLogsStreamNilBroker(t *testing.T) {
	srv := NewServer(ServerConfig{})
	srv.consoleBroker = nil
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	srv.handleConsoleLogsStream(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestHandleConsoleLogsClearMethodNotAllowed(t *testing.T) {
	srv := NewServer(ServerConfig{})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	srv.handleConsoleLogsClear(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", ctx.Response.StatusCode())
	}
}

func TestHandleConsoleLogsClearNilBroker(t *testing.T) {
	srv := NewServer(ServerConfig{})
	srv.consoleBroker = nil
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodDelete)
	srv.handleConsoleLogsClear(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}
