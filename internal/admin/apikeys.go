package admin

import (
	"encoding/json"
	"errors"

	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type apiKeyDTO struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	Name      string `json:"name"`
	MachineID string `json:"machine_id"`
	IsActive  bool   `json:"is_active"`
	CreatedAt int64  `json:"created_at"`
}

func toAPIKeyDTO(k *store.APIKey) apiKeyDTO {
	return apiKeyDTO{
		ID:        k.ID,
		Key:       k.Key,
		Name:      k.Name,
		MachineID: k.MachineID,
		IsActive:  k.IsActive,
		CreatedAt: k.CreatedAt,
	}
}

// ListAPIKeys handles GET /api/keys.
func (h *Handlers) ListAPIKeys(ctx *fasthttp.RequestCtx) {
	keys, err := h.store.ListAPIKeys()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list api keys")
		return
	}
	out := make([]apiKeyDTO, 0, len(keys))
	for _, k := range keys {
		out = append(out, toAPIKeyDTO(k))
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{"keys": out})
}

// CreateAPIKey handles POST /api/keys.
func (h *Handlers) CreateAPIKey(ctx *fasthttp.RequestCtx) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "name is required")
		return
	}

	machineID, err := auth.MachineID(h.store.DataDir(), "")
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "derive machine id")
		return
	}
	key, _, err := auth.GenerateAPIKey(machineID)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "generate api key")
		return
	}

	rec, err := h.store.CreateAPIKey(req.Name, key, machineID)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "create api key")
		return
	}

	writeData(ctx, fasthttp.StatusCreated, map[string]any{
		"key":        rec.Key,
		"name":       rec.Name,
		"id":         rec.ID,
		"machine_id": rec.MachineID,
	})
}

// GetAPIKey handles GET /api/keys/{id}.
func (h *Handlers) GetAPIKey(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	rec, err := h.store.GetAPIKeyByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "key not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load api key")
		return
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{"key": toAPIKeyDTO(rec)})
}

// UpdateAPIKey handles PUT /api/keys/{id}.
func (h *Handlers) UpdateAPIKey(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	var req struct {
		IsActive *bool `json:"is_active"`
	}
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}

	if _, err := h.store.GetAPIKeyByID(id); errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "key not found")
		return
	} else if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load api key")
		return
	}

	if req.IsActive != nil {
		if err := h.store.SetAPIKeyActive(id, *req.IsActive); err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, "update api key")
			return
		}
	}

	rec, err := h.store.GetAPIKeyByID(id)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load api key")
		return
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{"key": toAPIKeyDTO(rec)})
}

// DeleteAPIKey handles DELETE /api/keys/{id}.
func (h *Handlers) DeleteAPIKey(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	if err := h.store.DeleteAPIKey(id); errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "key not found")
		return
	} else if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "delete api key")
		return
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{"message": "Key deleted successfully"})
}
