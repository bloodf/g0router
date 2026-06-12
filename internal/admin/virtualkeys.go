package admin

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type virtualKeyDTO struct {
	ID              string                  `json:"id"`
	Key             string                  `json:"key"`
	Name            string                  `json:"name"`
	ProviderConfigs []schemas.ProviderConfig `json:"provider_configs"`
	Budget          *schemas.Budget         `json:"budget,omitempty"`
	RateLimitRPM    *int                    `json:"rate_limit_rpm,omitempty"`
	IsActive        bool                    `json:"is_active"`
	CreatedAt       int64                   `json:"created_at"`
	UpdatedAt       int64                   `json:"updated_at"`
}

func toVirtualKeyDTO(vk *store.VirtualKey) virtualKeyDTO {
	dto := virtualKeyDTO{
		ID:              vk.ID,
		Key:             vk.Key,
		Name:            vk.Name,
		ProviderConfigs: vk.ProviderConfigs,
		Budget:          vk.Budget,
		RateLimitRPM:    vk.RateLimitRPM,
		IsActive:        vk.IsActive,
		CreatedAt:       vk.CreatedAt,
		UpdatedAt:       vk.UpdatedAt,
	}
	if dto.ProviderConfigs == nil {
		dto.ProviderConfigs = []schemas.ProviderConfig{}
	}
	return dto
}

type virtualKeyRequest struct {
	Name            string                  `json:"name"`
	ProviderConfigs []schemas.ProviderConfig `json:"provider_configs"`
	Budget          *schemas.Budget         `json:"budget,omitempty"`
	RateLimitRPM    *int                    `json:"rate_limit_rpm,omitempty"`
	IsActive        *bool                   `json:"is_active,omitempty"`
}

func validateVirtualKeyRequest(req *virtualKeyRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(req.ProviderConfigs) == 0 {
		return fmt.Errorf("provider_configs is required")
	}
	for i, pc := range req.ProviderConfigs {
		if pc.Provider == "" {
			return fmt.Errorf("provider_configs[%d].provider is required", i)
		}
		if len(pc.AllowedModels) == 0 {
			return fmt.Errorf("provider_configs[%d].allowed_models is required", i)
		}
		if len(pc.KeyIDs) == 0 {
			return fmt.Errorf("provider_configs[%d].key_ids is required", i)
		}
	}
	if req.Budget != nil {
		if req.Budget.Limit < 0 {
			return fmt.Errorf("budget.limit must be non-negative")
		}
		if req.Budget.Used < 0 {
			return fmt.Errorf("budget.used must be non-negative")
		}
		if req.Budget.Period != "" && req.Budget.Period != "daily" && req.Budget.Period != "weekly" && req.Budget.Period != "monthly" {
			return fmt.Errorf("budget.period must be daily, weekly, or monthly")
		}
	}
	if req.RateLimitRPM != nil && *req.RateLimitRPM < 0 {
		return fmt.Errorf("rate_limit_rpm must be non-negative")
	}
	return nil
}

// ListVirtualKeys handles GET /api/virtual-keys.
func (h *Handlers) ListVirtualKeys(ctx *fasthttp.RequestCtx) {
	vks, err := h.store.ListVirtualKeys()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list virtual keys")
		return
	}
	out := make([]virtualKeyDTO, 0, len(vks))
	for _, vk := range vks {
		out = append(out, toVirtualKeyDTO(vk))
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{"virtual_keys": out})
}

// CreateVirtualKey handles POST /api/virtual-keys.
func (h *Handlers) CreateVirtualKey(ctx *fasthttp.RequestCtx) {
	var req virtualKeyRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := validateVirtualKeyRequest(&req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}

	vk := &store.VirtualKey{
		VirtualKey: schemas.VirtualKey{
			Name:            req.Name,
			ProviderConfigs: req.ProviderConfigs,
			Budget:          req.Budget,
			RateLimitRPM:    req.RateLimitRPM,
		},
	}
	created, err := h.store.CreateVirtualKey(vk)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "create virtual key")
		return
	}
	writeData(ctx, fasthttp.StatusCreated, map[string]any{"virtual_key": toVirtualKeyDTO(created)})
}

// GetVirtualKey handles GET /api/virtual-keys/{id}.
func (h *Handlers) GetVirtualKey(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	vk, err := h.store.GetVirtualKeyByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "virtual key not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load virtual key")
		return
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{"virtual_key": toVirtualKeyDTO(vk)})
}

// UpdateVirtualKey handles PUT /api/virtual-keys/{id}.
func (h *Handlers) UpdateVirtualKey(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	var req virtualKeyRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := validateVirtualKeyRequest(&req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}

	existing, err := h.store.GetVirtualKeyByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "virtual key not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load virtual key")
		return
	}

	isActive := existing.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	vk := &store.VirtualKey{
		VirtualKey: schemas.VirtualKey{
			ID:              id,
			Name:            req.Name,
			ProviderConfigs: req.ProviderConfigs,
			Budget:          req.Budget,
			RateLimitRPM:    req.RateLimitRPM,
		},
		IsActive: isActive,
	}
	if err := h.store.UpdateVirtualKey(vk); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "update virtual key")
		return
	}

	updated, err := h.store.GetVirtualKeyByID(id)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load virtual key")
		return
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{"virtual_key": toVirtualKeyDTO(updated)})
}

// DeleteVirtualKey handles DELETE /api/virtual-keys/{id}.
func (h *Handlers) DeleteVirtualKey(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	if err := h.store.DeleteVirtualKey(id); errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "virtual key not found")
		return
	} else if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "delete virtual key")
		return
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{"message": "Virtual key deleted successfully"})
}
