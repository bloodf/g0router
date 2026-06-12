package api

import (
	"sort"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/providers/catalog"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// DisabledChecker reports whether a model is disabled for a given provider alias.
type DisabledChecker interface {
	IsDisabled(providerAlias, modelID string) (bool, error)
}

// ModelsHandler handles GET /v1/models and GET /v1/models/:id.
type ModelsHandler struct {
	router          *inference.Router
	disabledChecker DisabledChecker
}

// NewModelsHandler creates a models handler.
func NewModelsHandler(router *inference.Router) *ModelsHandler {
	return &ModelsHandler{router: router}
}

// SetDisabledChecker wires a disabled-model checker into the handler.
func (h *ModelsHandler) SetDisabledChecker(dc DisabledChecker) {
	h.disabledChecker = dc
}

// List handles GET /v1/models.
func (h *ModelsHandler) List(ctx *fasthttp.RequestCtx) {
	resp := &schemas.ListModelsResponse{
		Object: "list",
	}

	// Aggregate models from all Stage-1 provider catalogs, skipping disabled ones.
	for providerID := range catalog.Providers {
		for _, m := range catalog.ModelsFor(providerID) {
			if h.disabledChecker != nil {
				if disabled, _ := h.disabledChecker.IsDisabled(providerID, m.ID); disabled {
					continue
				}
			}
			resp.Data = append(resp.Data, schemas.ModelEntry{
				ID:      m.ID,
				Object:  "model",
				OwnedBy: providerID,
			})
		}
	}

	// Deterministic order: sort by model ID.
	sort.Slice(resp.Data, func(i, j int) bool {
		return resp.Data[i].ID < resp.Data[j].ID
	})

	b, err := jsonMarshal(resp)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentTypeBytes([]byte("text/plain"))
		ctx.SetBodyString("internal error")
		return
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes([]byte("application/json"))
	ctx.SetBody(b)
}

// Get handles GET /v1/models/:id.
func (h *ModelsHandler) Get(ctx *fasthttp.RequestCtx) {
	id, ok := ctx.UserValue("id").(string)
	if !ok || id == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "missing model id", nil)
		return
	}

	for providerID := range catalog.Providers {
		for _, m := range catalog.ModelsFor(providerID) {
			if m.ID != id {
				continue
			}
			entry := schemas.ModelEntry{
				ID:      m.ID,
				Object:  "model",
				OwnedBy: providerID,
			}
			b, err := jsonMarshal(entry)
			if err != nil {
				ctx.SetStatusCode(fasthttp.StatusInternalServerError)
				ctx.SetContentTypeBytes([]byte("text/plain"))
				ctx.SetBodyString("internal error")
				return
			}
			ctx.SetStatusCode(fasthttp.StatusOK)
			ctx.SetContentTypeBytes([]byte("application/json"))
			ctx.SetBody(b)
			return
		}
	}

	writeError(ctx, fasthttp.StatusNotFound, "not_found_error", "model not found", nil)
}
