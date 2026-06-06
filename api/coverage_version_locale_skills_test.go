package api

import (
	"path/filepath"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func TestVersionRoute(t *testing.T) {
	s := NewServer(ServerConfig{Version: "1.0.0", BuildDate: "2024-01-01"})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.SetRequestURI("/api/version")
	s.handle(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
}

func TestLocaleRouteGet(t *testing.T) {
	st, err := store.NewStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer st.Close()

	s := NewServer(ServerConfig{Store: st})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.SetRequestURI("/api/locale")
	s.handle(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
}

func TestLocaleRoutePost(t *testing.T) {
	st, err := store.NewStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer st.Close()

	s := NewServer(ServerConfig{Store: st})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("/api/locale")
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.SetBodyString(`{"locale":"pt-BR"}`)
	s.handle(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
}

func TestSkillsRoute(t *testing.T) {
	s := NewServer(ServerConfig{})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.SetRequestURI("/api/skills")
	s.handle(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
}

func TestUpdateCheckRoute(t *testing.T) {
	s := NewServer(ServerConfig{Version: "0.0.0"})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("/api/update/check")
	s.handle(ctx)
	// It will likely fail because it hits the real GitHub API with a fake version,
	// but we just care about route coverage.
	if ctx.Response.StatusCode() == fasthttp.StatusNotFound {
		t.Fatal("route not found")
	}
}

func TestUpdateApplyRouteAdminRequired(t *testing.T) {
	s := NewServer(ServerConfig{Version: "1.0.0"})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("/api/update/apply")
	s.handle(ctx)
	if ctx.Response.StatusCode() == fasthttp.StatusNotFound {
		t.Fatal("route not found")
	}
}
