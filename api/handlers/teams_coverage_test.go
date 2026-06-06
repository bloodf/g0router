package handlers

import (
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func TestTeamsGetError(t *testing.T) {
	fs := &fakeTeamStore{getErr: errors.New("db error")}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestTeamsUpdateGetAfterUpdateError(t *testing.T) {
	fs := &fakeTeamStore{updateErr: nil, getErr: store.ErrNotFound}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"eng"}`, func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestIsSQLiteConstraintErrorNil(t *testing.T) {
	if isSQLiteConstraintError(nil) {
		t.Fatal("expected false for nil error")
	}
}

func TestTeamsUpdateError(t *testing.T) {
	fs := &fakeTeamStore{updateErr: errors.New("db error")}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"eng"}`, func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestTeamsPutInvalidID(t *testing.T) {
	fs := &fakeTeamStore{}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"eng"}`, func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, fs, "abc")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestTeamsPutInvalidJSON(t *testing.T) {
	fs := &fakeTeamStore{}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":`, func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestTeamsDeleteInvalidID(t *testing.T) {
	fs := &fakeTeamStore{}
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, fs, "abc")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}
