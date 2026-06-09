package admin

import (
	"encoding/json"
	"errors"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type providerDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	BaseURL   string `json:"base_url"`
	Enabled   bool   `json:"enabled"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type providerRequest struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	BaseURL string `json:"base_url"`
	Enabled bool   `json:"enabled"`
}

func toProviderDTO(p *store.ProviderRecord) providerDTO {
	return providerDTO{
		ID:        p.ID,
		Name:      p.Name,
		Type:      p.Type,
		BaseURL:   p.BaseURL,
		Enabled:   p.Enabled,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

// ListProviders handles GET /api/providers.
func (h *Handlers) ListProviders(ctx *fasthttp.RequestCtx) {
	providers, err := h.store.ListProviders()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list providers")
		return
	}
	out := make([]providerDTO, 0, len(providers))
	for _, p := range providers {
		out = append(out, toProviderDTO(p))
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// CreateProvider handles POST /api/providers.
func (h *Handlers) CreateProvider(ctx *fasthttp.RequestCtx) {
	var req providerRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" || req.Type == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "name and type are required")
		return
	}

	rec := &store.ProviderRecord{Name: req.Name, Type: req.Type, BaseURL: req.BaseURL, Enabled: req.Enabled}
	if err := h.store.CreateProvider(rec); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "create provider")
		return
	}
	writeData(ctx, fasthttp.StatusCreated, toProviderDTO(rec))
}

// UpdateProvider handles PUT /api/providers/{id}.
func (h *Handlers) UpdateProvider(ctx *fasthttp.RequestCtx) {
	id := pathID(ctx.UserValue("id"))
	var req providerRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" || req.Type == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "name and type are required")
		return
	}

	rec := &store.ProviderRecord{ID: id, Name: req.Name, Type: req.Type, BaseURL: req.BaseURL, Enabled: req.Enabled}
	err := h.store.UpdateProvider(rec)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "provider not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "update provider")
		return
	}

	updated, err := h.store.GetProvider(id)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load provider")
		return
	}
	writeData(ctx, fasthttp.StatusOK, toProviderDTO(updated))
}

// DeleteProvider handles DELETE /api/providers/{id}.
func (h *Handlers) DeleteProvider(ctx *fasthttp.RequestCtx) {
	id := pathID(ctx.UserValue("id"))
	err := h.store.DeleteProvider(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "provider not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "delete provider")
		return
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{"deleted": true})
}
