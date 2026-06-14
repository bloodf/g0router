package admin

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bloodf/g0router/internal/providers/catalog"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// ModelProber probes a model's reachability. Production wires a best-effort
// reachability check; tests inject a deterministic fake (no live network).
type ModelProber interface {
	Probe(provider, modelID string) (ok bool, latencyMS int, err error)
}

// customModelDTO is the wire shape the ModelSelectModal consumes. It mirrors the
// UI Model type ({id, provider, name, is_custom, is_disabled, ...}); the id is
// the stored custom-model record id. Config fields are surfaced when present.
type customModelDTO struct {
	ID         string `json:"id"`
	Provider   string `json:"provider"`
	ModelID    string `json:"model_id"`
	Name       string `json:"name"`
	IsCustom   bool   `json:"is_custom"`
	IsDisabled bool   `json:"is_disabled"`
	CreatedAt  int64  `json:"created_at"`

	InputCost     float64 `json:"input_cost"`
	OutputCost    float64 `json:"output_cost"`
	ContextWindow float64 `json:"context_window"`
}

func toCustomModelDTO(cm *store.CustomModel) customModelDTO {
	dto := customModelDTO{
		ID:         cm.ID,
		Provider:   cm.Provider,
		ModelID:    cm.ModelID,
		Name:       cm.Name,
		IsCustom:   true,
		IsDisabled: false,
		CreatedAt:  cm.CreatedAt,
	}
	if v, ok := cm.Config["input_cost"].(float64); ok {
		dto.InputCost = v
	}
	if v, ok := cm.Config["output_cost"].(float64); ok {
		dto.OutputCost = v
	}
	if v, ok := cm.Config["context_window"].(float64); ok {
		dto.ContextWindow = v
	}
	return dto
}

// TestModel handles POST /api/models/test. It resolves the model via the
// catalog then runs the injectable prober, returning {ok, latency_ms}.
func (h *Handlers) TestModel(ctx *fasthttp.RequestCtx) {
	var req struct {
		Provider string `json:"provider"`
		ModelID  string `json:"model_id"`
	}
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.ModelID == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "model_id is required")
		return
	}
	if h.modelProber == nil {
		writeError(ctx, fasthttp.StatusNotImplemented, "model prober not available")
		return
	}

	ok, latencyMS, err := h.modelProber.Probe(req.Provider, req.ModelID)
	if err != nil {
		writeData(ctx, fasthttp.StatusOK, map[string]any{"ok": false, "latency_ms": latencyMS})
		return
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{"ok": ok, "latency_ms": latencyMS})
}

// ModelAvailability handles GET /api/models/availability. It reports which
// catalog models are reachable. Availability is derived deterministically from
// the catalog plus enabled provider connections.
func (h *Handlers) ModelAvailability(ctx *fasthttp.RequestCtx) {
	providers, err := h.store.ListProviders()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load providers")
		return
	}

	enabled := make(map[string]bool, len(providers))
	for _, p := range providers {
		if p.Enabled {
			enabled[p.Type] = true
		}
	}

	type availabilityEntry struct {
		ID        string `json:"id"`
		Available bool   `json:"available"`
	}
	var available []availabilityEntry
	for _, p := range providers {
		for _, m := range catalog.ModelsFor(p.Type) {
			available = append(available, availabilityEntry{
				ID:        m.ID,
				Available: enabled[p.Type],
			})
		}
	}
	// When no providers are configured, fall back to the full catalog marked
	// available so the surface is never empty in a fresh install.
	if len(available) == 0 {
		for provider := range catalog.Models {
			for _, m := range catalog.ModelsFor(provider) {
				available = append(available, availabilityEntry{ID: m.ID, Available: true})
			}
		}
	}

	writeData(ctx, fasthttp.StatusOK, map[string]any{"available": available})
}

// ListCustomModels handles GET /api/models/custom. It returns the custom-model
// list as a bare array under data.
func (h *Handlers) ListCustomModels(ctx *fasthttp.RequestCtx) {
	models, err := h.store.ListCustomModels()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list custom models")
		return
	}
	out := make([]customModelDTO, 0, len(models))
	for _, cm := range models {
		out = append(out, toCustomModelDTO(cm))
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// CreateCustomModel handles POST /api/models/custom.
func (h *Handlers) CreateCustomModel(ctx *fasthttp.RequestCtx) {
	var req struct {
		Provider string         `json:"provider"`
		ModelID  string         `json:"model_id"`
		Name     string         `json:"name"`
		Config   map[string]any `json:"config"`
	}
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.ModelID == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "model_id is required")
		return
	}

	cm, err := h.store.CreateCustomModel(&store.CustomModel{
		Provider: req.Provider,
		ModelID:  req.ModelID,
		Name:     req.Name,
		Config:   req.Config,
	})
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "create custom model")
		return
	}

	h.recordAudit(ctx, "custom_model.create", req.ModelID, fmt.Sprintf("provider=%s", req.Provider))
	writeData(ctx, fasthttp.StatusOK, toCustomModelDTO(cm))
}

// DeleteCustomModel handles DELETE /api/models/custom/{id}.
func (h *Handlers) DeleteCustomModel(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok || id == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "id is required")
		return
	}
	if err := h.store.DeleteCustomModel(id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(ctx, fasthttp.StatusNotFound, "custom model not found")
			return
		}
		writeError(ctx, fasthttp.StatusInternalServerError, "delete custom model")
		return
	}
	h.recordAudit(ctx, "custom_model.delete", id, "")
	writeData(ctx, fasthttp.StatusOK, map[string]any{})
}
