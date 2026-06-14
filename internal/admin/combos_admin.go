package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type comboStepDTO struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type comboAdminDTO struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Strategy string         `json:"strategy"`
	Steps    []comboStepDTO `json:"steps"`
	IsActive bool           `json:"is_active"`
}

func toComboAdminDTO(c *store.ComboAdmin) comboAdminDTO {
	steps := make([]comboStepDTO, 0, len(c.Steps))
	for _, s := range c.Steps {
		steps = append(steps, comboStepDTO{Provider: s.Provider, Model: s.Model})
	}
	return comboAdminDTO{
		ID:       c.ID,
		Name:     c.Name,
		Strategy: c.Strategy,
		Steps:    steps,
		IsActive: c.IsActive,
	}
}

type comboAdminRequest struct {
	Name     string         `json:"name"`
	Strategy string         `json:"strategy"`
	Steps    []comboStepDTO `json:"steps"`
	IsActive *bool          `json:"is_active"`
}

func toStoreSteps(in []comboStepDTO) []store.ComboStep {
	out := make([]store.ComboStep, 0, len(in))
	for _, s := range in {
		out = append(out, store.ComboStep{Provider: s.Provider, Model: s.Model})
	}
	return out
}

// mirrorEngineCombo best-effort upserts the admin combo into the engine combos
// table (name → models[]) so /v1/models still lists the combo by name. The
// engine combo name must match comboNameRe; non-matching names are skipped. A
// failure is logged and never fails the parent request (ESC-COMBOS).
func (h *Handlers) mirrorEngineCombo(name string, steps []store.ComboStep) {
	if name == "" || !comboNameRe.MatchString(name) {
		return
	}
	models := make([]string, 0, len(steps))
	for _, s := range steps {
		if s.Model != "" {
			models = append(models, s.Model)
		}
	}
	if err := h.store.UpdateCombo(name, models); err != nil {
		if createErr := h.store.CreateCombo(&store.Combo{Name: name, Models: models}); createErr != nil {
			log.Printf("combos-admin: mirror engine combo %q failed: update=%v create=%v", name, err, createErr)
		}
	}
}

// ListCombosAdmin handles GET /api/combos.
func (h *Handlers) ListCombosAdmin(ctx *fasthttp.RequestCtx) {
	combos, err := h.store.ListComboAdmins()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list combos")
		return
	}
	out := make([]comboAdminDTO, 0, len(combos))
	for _, c := range combos {
		out = append(out, toComboAdminDTO(c))
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// CreateComboAdmin handles POST /api/combos.
func (h *Handlers) CreateComboAdmin(ctx *fasthttp.RequestCtx) {
	var req comboAdminRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "name is required")
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	steps := toStoreSteps(req.Steps)
	created, err := h.store.CreateComboAdmin(&store.ComboAdmin{
		Name:     req.Name,
		Strategy: req.Strategy,
		Steps:    steps,
		IsActive: isActive,
	})
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "create combo")
		return
	}
	h.mirrorEngineCombo(created.Name, created.Steps)
	h.recordAudit(ctx, "create_combo", created.ID, fmt.Sprintf("Created combo %s", created.Name))
	writeData(ctx, fasthttp.StatusCreated, toComboAdminDTO(created))
}

// GetComboAdmin handles GET /api/combos/{id}.
func (h *Handlers) GetComboAdmin(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	c, err := h.store.GetComboAdminByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "combo not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load combo")
		return
	}
	writeData(ctx, fasthttp.StatusOK, toComboAdminDTO(c))
}

// UpdateComboAdmin handles PUT /api/combos/{id}. The frozen /combos page sends a
// body of {steps:[{provider,model}]} (and optionally name/strategy/is_active);
// fields absent from the body preserve their existing values.
func (h *Handlers) UpdateComboAdmin(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	var req comboAdminRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	existing, err := h.store.GetComboAdminByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "combo not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load combo")
		return
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Strategy != "" {
		existing.Strategy = req.Strategy
	}
	if req.Steps != nil {
		existing.Steps = toStoreSteps(req.Steps)
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	if err := h.store.UpdateComboAdmin(existing); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "update combo")
		return
	}
	updated, err := h.store.GetComboAdminByID(id)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load combo")
		return
	}
	h.mirrorEngineCombo(updated.Name, updated.Steps)
	h.recordAudit(ctx, "update_combo", updated.ID, fmt.Sprintf("Updated combo %s", updated.Name))
	writeData(ctx, fasthttp.StatusOK, toComboAdminDTO(updated))
}

// DeleteComboAdmin handles DELETE /api/combos/{id}.
func (h *Handlers) DeleteComboAdmin(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	existing, err := h.store.GetComboAdminByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "combo not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load combo")
		return
	}
	if err := h.store.DeleteComboAdmin(id); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "delete combo")
		return
	}
	// Best-effort: drop the mirrored engine combo so /v1/models stops listing it.
	if existing.Name != "" {
		if err := h.store.DeleteCombo(existing.Name); err != nil && !errors.Is(err, store.ErrNotFound) {
			log.Printf("combos-admin: mirror delete engine combo %q failed: %v", existing.Name, err)
		}
	}
	h.recordAudit(ctx, "delete_combo", id, fmt.Sprintf("Deleted combo %s", existing.Name))
	writeData(ctx, fasthttp.StatusOK, map[string]any{"message": "Combo deleted successfully"})
}
