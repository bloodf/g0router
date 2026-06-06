package api

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestHandleTrafficStreamMethodNotAllowed(t *testing.T) {
	srv := NewServer(ServerConfig{})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	srv.handleTrafficStream(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", ctx.Response.StatusCode())
	}
}

func TestHandleTrafficStreamNilBroker(t *testing.T) {
	srv := NewServer(ServerConfig{})
	srv.trafficBroker = nil
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	srv.handleTrafficStream(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}
