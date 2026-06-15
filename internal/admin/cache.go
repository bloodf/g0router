package admin

import (
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// semanticCacheStatsDTO summarizes the cache for the admin GET (counts only).
type semanticCacheStatsDTO struct {
	Entries   int64 `json:"entries"`
	TotalHits int64 `json:"total_hits"`
}

// semanticCacheEntryDTO is the metadata view of a cache row. It deliberately
// OMITS the full response_json (phase-19:50) so the GET never leaks cached
// payloads — only key, model, hits, and expiry are exposed.
type semanticCacheEntryDTO struct {
	Key     string `json:"key"`
	Model   string `json:"model"`
	Hits    int64  `json:"hits"`
	Expires string `json:"expires"`
}

type semanticCacheDTO struct {
	Stats   semanticCacheStatsDTO   `json:"stats"`
	Entries []semanticCacheEntryDTO `json:"entries"`
}

func toSemanticCacheEntryDTO(e *store.SemanticCacheEntry) semanticCacheEntryDTO {
	return semanticCacheEntryDTO{
		Key:     e.CacheKey,
		Model:   e.Model,
		Hits:    e.HitCount,
		Expires: e.ExpiresAt,
	}
}

// GetSemanticCache handles GET /api/cache/semantic. It returns cache statistics
// plus per-entry metadata (key, model, hits, expires) — never the full cached
// response payloads (phase-19:50).
func (h *Handlers) GetSemanticCache(ctx *fasthttp.RequestCtx) {
	stats, err := h.store.SemanticCacheStats()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "semantic cache stats")
		return
	}
	entries, err := h.store.ListSemanticCacheEntries()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list semantic cache entries")
		return
	}
	out := semanticCacheDTO{
		Stats:   semanticCacheStatsDTO{Entries: stats.Entries, TotalHits: stats.TotalHits},
		Entries: make([]semanticCacheEntryDTO, 0, len(entries)),
	}
	for _, e := range entries {
		out.Entries = append(out.Entries, toSemanticCacheEntryDTO(e))
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// ClearSemanticCache handles DELETE /api/cache/semantic. It empties the cache
// table and records an audit entry (phase-19:51 "clear (audited)").
func (h *Handlers) ClearSemanticCache(ctx *fasthttp.RequestCtx) {
	if err := h.store.ClearSemanticCache(); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "clear semantic cache")
		return
	}
	h.recordAudit(ctx, "semantic_cache.clear", "semantic_cache", "cleared semantic cache")
	writeData(ctx, fasthttp.StatusOK, map[string]any{"cleared": true})
}
