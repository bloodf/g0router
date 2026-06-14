package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type modelLimitDTO struct {
	ID            int64    `json:"id"`
	Model         string   `json:"model"`
	MaxTokens     int      `json:"max_tokens"`
	MaxRPM        int      `json:"max_rpm"`
	AllowedKeyIDs []string `json:"allowed_key_ids"`
	CreatedAt     string   `json:"created_at"`
}

func toModelLimitDTO(ml *store.ModelLimit) modelLimitDTO {
	keyIDs := ml.AllowedKeyIDs
	if keyIDs == nil {
		keyIDs = []string{}
	}
	return modelLimitDTO{
		ID:            ml.ID,
		Model:         ml.Model,
		MaxTokens:     ml.MaxTokens,
		MaxRPM:        ml.MaxRPM,
		AllowedKeyIDs: keyIDs,
		CreatedAt:     time.Unix(ml.CreatedAt, 0).UTC().Format(time.RFC3339),
	}
}

type modelLimitRequest struct {
	Model         string   `json:"model"`
	MaxTokens     int      `json:"max_tokens"`
	MaxRPM        int      `json:"max_rpm"`
	AllowedKeyIDs []string `json:"allowed_key_ids"`
}

// modelLimitID parses the numeric {id} route parameter (ESC-IDTYPE: the UI
// ModelLimit.id is a number, so the path id is parsed via strconv.ParseInt
// rather than the string-only pathID).
func modelLimitID(v any) (int64, bool) {
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

// ListModelLimits handles GET /api/model-limits.
func (h *Handlers) ListModelLimits(ctx *fasthttp.RequestCtx) {
	limits, err := h.store.ListModelLimits()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list model limits")
		return
	}
	out := make([]modelLimitDTO, 0, len(limits))
	for _, ml := range limits {
		out = append(out, toModelLimitDTO(ml))
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// CreateModelLimit handles POST /api/model-limits.
func (h *Handlers) CreateModelLimit(ctx *fasthttp.RequestCtx) {
	var req modelLimitRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Model == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "model is required")
		return
	}
	created, err := h.store.CreateModelLimit(&store.ModelLimit{
		Model:         req.Model,
		MaxTokens:     req.MaxTokens,
		MaxRPM:        req.MaxRPM,
		AllowedKeyIDs: req.AllowedKeyIDs,
	})
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "create model limit")
		return
	}
	h.recordAudit(ctx, "create_model_limit", strconv.FormatInt(created.ID, 10), fmt.Sprintf("Created model limit %s", created.Model))
	writeData(ctx, fasthttp.StatusCreated, toModelLimitDTO(created))
}

// GetModelLimit handles GET /api/model-limits/{id}.
func (h *Handlers) GetModelLimit(ctx *fasthttp.RequestCtx) {
	id, ok := modelLimitID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	ml, err := h.store.GetModelLimitByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "model limit not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load model limit")
		return
	}
	writeData(ctx, fasthttp.StatusOK, toModelLimitDTO(ml))
}

// UpdateModelLimit handles PUT /api/model-limits/{id}.
func (h *Handlers) UpdateModelLimit(ctx *fasthttp.RequestCtx) {
	id, ok := modelLimitID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	var req modelLimitRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Model == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "model is required")
		return
	}
	existing, err := h.store.GetModelLimitByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "model limit not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load model limit")
		return
	}
	existing.Model = req.Model
	existing.MaxTokens = req.MaxTokens
	existing.MaxRPM = req.MaxRPM
	existing.AllowedKeyIDs = req.AllowedKeyIDs
	if err := h.store.UpdateModelLimit(existing); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "update model limit")
		return
	}
	updated, err := h.store.GetModelLimitByID(id)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load model limit")
		return
	}
	h.recordAudit(ctx, "update_model_limit", strconv.FormatInt(updated.ID, 10), fmt.Sprintf("Updated model limit %s", updated.Model))
	writeData(ctx, fasthttp.StatusOK, toModelLimitDTO(updated))
}

// DeleteModelLimit handles DELETE /api/model-limits/{id}.
func (h *Handlers) DeleteModelLimit(ctx *fasthttp.RequestCtx) {
	id, ok := modelLimitID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	if err := h.store.DeleteModelLimit(id); errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "model limit not found")
		return
	} else if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "delete model limit")
		return
	}
	h.recordAudit(ctx, "delete_model_limit", strconv.FormatInt(id, 10), "Deleted model limit")
	writeData(ctx, fasthttp.StatusOK, map[string]any{"message": "Model limit deleted successfully"})
}
