package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type flagDTO struct {
	ID          int64  `json:"id"`
	Key         string `json:"key"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

func toFlagDTO(f *store.FeatureFlag) flagDTO {
	return flagDTO{
		ID:          f.ID,
		Key:         f.Key,
		Enabled:     f.Enabled,
		Description: f.Description,
		CreatedAt:   f.CreatedAt,
	}
}

type toggleFlagRequest struct {
	Enabled bool `json:"enabled"`
}

// flagID parses the numeric {id} route parameter (feature flags use integer ids).
func flagID(v any) (int64, bool) {
	s, ok := v.(string)
	if !ok {
		return 0, false
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

// ListFeatureFlags handles GET /api/feature-flags. The response data is a bare array.
func (h *Handlers) ListFeatureFlags(ctx *fasthttp.RequestCtx) {
	flags, err := h.store.ListFeatureFlags()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list feature flags")
		return
	}
	out := make([]flagDTO, 0, len(flags))
	for _, f := range flags {
		out = append(out, toFlagDTO(f))
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// GetFeatureFlag handles GET /api/feature-flags/{id}.
func (h *Handlers) GetFeatureFlag(ctx *fasthttp.RequestCtx) {
	id, ok := flagID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	f, err := h.store.GetFeatureFlagByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "feature flag not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load feature flag")
		return
	}
	writeData(ctx, fasthttp.StatusOK, toFlagDTO(f))
}

// ToggleFeatureFlag handles PUT /api/feature-flags/{id}.
func (h *Handlers) ToggleFeatureFlag(ctx *fasthttp.RequestCtx) {
	id, ok := flagID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	var req toggleFlagRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}

	updated, err := h.store.SetFeatureFlagEnabled(id, req.Enabled)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "feature flag not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "update feature flag")
		return
	}
	h.recordAudit(ctx, "feature_flag.toggle", updated.Key,
		fmt.Sprintf("set %s enabled=%v", updated.Key, updated.Enabled))
	writeData(ctx, fasthttp.StatusOK, toFlagDTO(updated))
}
