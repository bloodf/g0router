package handlers

import (
	"github.com/bloodf/g0router/internal/semcache"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type semanticCacheStore interface {
	SemcacheRepo() *store.SemcacheRepo
}

// SemanticCacheStats returns statistics about the semantic cache.
func SemanticCacheStats(ctx *fasthttp.RequestCtx, store semanticCacheStore) {
	if isStoreNil(store) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}

	repo := store.SemcacheRepo()
	cache := semcache.NewCache(repo, nil, 0.95)
	count, hits, err := cache.Stats()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to get cache stats")
		return
	}

	writeJSON(ctx, fasthttp.StatusOK, map[string]any{
		"total_entries": count,
		"total_hits":    hits,
	})
}

// SemanticCacheClear removes all entries from the semantic cache.
func SemanticCacheClear(ctx *fasthttp.RequestCtx, store semanticCacheStore) {
	if isStoreNil(store) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if string(ctx.Method()) != fasthttp.MethodDelete {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}

	repo := store.SemcacheRepo()
	cache := semcache.NewCache(repo, nil, 0.95)
	if err := cache.Clear(); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to clear cache")
		return
	}

	writeJSON(ctx, fasthttp.StatusOK, map[string]any{"cleared": true})
}
