package admin

import (
	"encoding/json"

	"github.com/valyala/fasthttp"
)

// GetDisabledModels handles GET /api/models/disabled[?providerAlias=xxx].
// With providerAlias: returns {data: {ids: [...]}}. Without: returns {data: {disabled: {...}}}.
func (h *Handlers) GetDisabledModels(ctx *fasthttp.RequestCtx) {
	all, err := h.store.ListDisabledModels()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "Failed to fetch disabled models")
		return
	}

	alias := string(ctx.QueryArgs().Peek("providerAlias"))
	if alias != "" {
		ids := all[alias]
		if ids == nil {
			ids = []string{}
		}
		writeData(ctx, fasthttp.StatusOK, map[string]any{"ids": ids})
	} else {
		writeData(ctx, fasthttp.StatusOK, map[string]any{"disabled": all})
	}
}

// PostDisabledModels handles POST /api/models/disabled.
// Body: { "providerAlias": "...", "ids": ["model1", ...] }
func (h *Handlers) PostDisabledModels(ctx *fasthttp.RequestCtx) {
	var req struct {
		ProviderAlias string   `json:"providerAlias"`
		IDs           []string `json:"ids"`
	}
	if err := json.Unmarshal(ctx.Request.Body(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.ProviderAlias == "" || req.IDs == nil {
		writeError(ctx, fasthttp.StatusBadRequest, "providerAlias and ids[] required")
		return
	}

	if err := h.store.DisableModels(req.ProviderAlias, req.IDs); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "Failed to disable models")
		return
	}
	writeData(ctx, fasthttp.StatusOK, map[string]bool{"success": true})
}

// DeleteDisabledModels handles DELETE /api/models/disabled?providerAlias=xxx[&id=yyy].
// If id is omitted, all models for the alias are re-enabled.
func (h *Handlers) DeleteDisabledModels(ctx *fasthttp.RequestCtx) {
	alias := string(ctx.QueryArgs().Peek("providerAlias"))
	if alias == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "providerAlias required")
		return
	}

	id := string(ctx.QueryArgs().Peek("id"))
	var ids []string
	if id != "" {
		ids = []string{id}
	}

	if err := h.store.EnableModels(alias, ids); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "Failed to enable models")
		return
	}
	writeData(ctx, fasthttp.StatusOK, map[string]bool{"success": true})
}
