package api

import (
	"errors"
	"testing"
	"testing/fstest"

	"github.com/valyala/fasthttp"
)

func uiServer(t *testing.T) *Server {
	t.Helper()
	return &Server{
		uiFS: fstest.MapFS{
			"index.html":     {Data: []byte("<html>index</html>")},
			"assets/app.css": {Data: []byte("body{}")},
		},
	}
}

func TestHandleUIError(t *testing.T) {
	srv := &Server{uiErr: errors.New("ui broken")}
	ctx := &fasthttp.RequestCtx{}
	srv.handleUI(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d", ctx.Response.StatusCode())
	}
}

func TestHandleUIRejectsNonGet(t *testing.T) {
	srv := uiServer(t)
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	srv.handleUI(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestHandleUIServesIndex(t *testing.T) {
	srv := uiServer(t)
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/")
	srv.handleUI(ctx)
	if got := string(ctx.Response.Body()); got != "<html>index</html>" {
		t.Fatalf("body = %q", got)
	}
}

func TestHandleUIServesAsset(t *testing.T) {
	srv := uiServer(t)
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/assets/app.css")
	srv.handleUI(ctx)
	if got := string(ctx.Response.Body()); got != "body{}" {
		t.Fatalf("body = %q", got)
	}
	if ct := string(ctx.Response.Header.ContentType()); ct == "" {
		t.Fatal("expected content type for css")
	}
}

func TestHandleUIMissingAssetIs404(t *testing.T) {
	srv := uiServer(t)
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/assets/missing.js")
	srv.handleUI(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestHandleUIFallsBackToIndex(t *testing.T) {
	srv := uiServer(t)
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/some/spa/route")
	srv.handleUI(ctx)
	if got := string(ctx.Response.Body()); got != "<html>index</html>" {
		t.Fatalf("spa fallback body = %q", got)
	}
}

func TestHandleUIHead(t *testing.T) {
	srv := uiServer(t)
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodHead)
	ctx.Request.SetRequestURI("/index.html")
	srv.handleUI(ctx)
	if len(ctx.Response.Body()) != 0 {
		t.Fatalf("HEAD should have empty body, got %q", ctx.Response.Body())
	}
}
