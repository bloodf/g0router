package handlers

import (
	"context"
	"testing"

	"github.com/bloodf/g0router/internal/semcache"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeSemanticCacheStore struct {
	repo *store.SemcacheRepo
}

func (f *fakeSemanticCacheStore) SemcacheRepo() *store.SemcacheRepo {
	return f.repo
}

func TestSemanticCacheStats(t *testing.T) {
	s := newHandlerStore(t)
	repo := s.SemcacheRepo()
	fs := &fakeSemanticCacheStore{repo: repo}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		SemanticCacheStats(ctx, fs)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("stats status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var result struct {
		TotalEntries int   `json:"total_entries"`
		TotalHits    int64 `json:"total_hits"`
	}
	decodeJSON(t, body, &result)
	if result.TotalEntries != 0 {
		t.Fatalf("total_entries = %d, want 0", result.TotalEntries)
	}
}

func TestSemanticCacheStatsMethodNotAllowed(t *testing.T) {
	s := newHandlerStore(t)
	repo := s.SemcacheRepo()
	fs := &fakeSemanticCacheStore{repo: repo}

	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		SemanticCacheStats(ctx, fs)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", ctx.Response.StatusCode())
	}
}

func TestSemanticCacheStatsStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		SemanticCacheStats(ctx, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestSemanticCacheClear(t *testing.T) {
	s := newHandlerStore(t)
	repo := s.SemcacheRepo()
	fs := &fakeSemanticCacheStore{repo: repo}

	// Seed an entry
	cache := semcache.NewCache(repo, nil, 0.95)
	_ = cache.Store(context.Background(), "key", "model", "prompt", &semcache.CachedResponse{ID: "1"}, 0)

	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		SemanticCacheClear(ctx, fs)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("clear status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var result struct {
		Cleared bool `json:"cleared"`
	}
	decodeJSON(t, body, &result)
	if !result.Cleared {
		t.Fatal("expected cleared=true")
	}

	// Verify empty
	count, _, _ := cache.Stats()
	if count != 0 {
		t.Fatalf("count = %d, want 0 after clear", count)
	}
}

func TestSemanticCacheClearMethodNotAllowed(t *testing.T) {
	s := newHandlerStore(t)
	repo := s.SemcacheRepo()
	fs := &fakeSemanticCacheStore{repo: repo}

	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		SemanticCacheClear(ctx, fs)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", ctx.Response.StatusCode())
	}
}

func TestSemanticCacheClearStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		SemanticCacheClear(ctx, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestSemanticCacheStatsError(t *testing.T) {
	s := newHandlerStore(t)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	repo := s.SemcacheRepo()
	fs := &fakeSemanticCacheStore{repo: repo}

	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		SemanticCacheStats(ctx, fs)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestSemanticCacheClearError(t *testing.T) {
	s := newHandlerStore(t)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	repo := s.SemcacheRepo()
	fs := &fakeSemanticCacheStore{repo: repo}

	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		SemanticCacheClear(ctx, fs)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}
