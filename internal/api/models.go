package api

import (
	"sort"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/providers/catalog"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// DisabledChecker reports whether a model is disabled for a given provider alias.
type DisabledChecker interface {
	IsDisabled(providerAlias, modelID string) (bool, error)
}

// ComboLister returns the list of defined combos (PAR-ROUTE-047: combo names first in /v1/models).
type ComboLister interface {
	ListCombos() ([]*store.Combo, error)
}

// ModelsHandler handles GET /v1/models and GET /v1/models/:id.
type ModelsHandler struct {
	router          *inference.Router
	disabledChecker DisabledChecker
	comboLister     ComboLister
}

// NewModelsHandler creates a models handler.
func NewModelsHandler(router *inference.Router) *ModelsHandler {
	return &ModelsHandler{router: router}
}

// SetDisabledChecker wires a disabled-model checker into the handler.
func (h *ModelsHandler) SetDisabledChecker(dc DisabledChecker) {
	h.disabledChecker = dc
}

// SetComboLister wires a combo lister so combo names appear first in /v1/models (PAR-ROUTE-047).
func (h *ModelsHandler) SetComboLister(cl ComboLister) {
	h.comboLister = cl
}

// List handles GET /v1/models.
func (h *ModelsHandler) List(ctx *fasthttp.RequestCtx) {
	resp := &schemas.ListModelsResponse{
		Object: "list",
	}

	// Combo names appear first (PAR-ROUTE-047).
	if h.comboLister != nil {
		combos, err := h.comboLister.ListCombos()
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, "server_error", "failed to list combos", nil)
			return
		}
		for _, c := range combos {
			resp.Data = append(resp.Data, schemas.ModelEntry{
				ID:      c.Name,
				Object:  "model",
				OwnedBy: "combo",
			})
		}
	}
	providerStart := len(resp.Data)

	// Aggregate models from all Stage-1 provider catalogs, skipping disabled ones.
	for providerID := range catalog.Providers {
		for _, m := range catalog.ModelsFor(providerID) {
			if h.disabledChecker != nil {
				disabled, err := h.disabledChecker.IsDisabled(providerID, m.ID)
				if err != nil {
					writeError(ctx, fasthttp.StatusInternalServerError, "server_error", "failed to check disabled models", nil)
					return
				}
				if disabled {
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

	// Sort only the provider-model section; combo entries keep their list order.
	sort.Slice(resp.Data[providerStart:], func(i, j int) bool {
		return resp.Data[providerStart+i].ID < resp.Data[providerStart+j].ID
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
