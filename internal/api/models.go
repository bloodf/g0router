package api

import (
	"sort"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/providers/catalog"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// ModelsHandler handles GET /v1/models and GET /v1/models/:id.
type ModelsHandler struct {
	router *inference.Router
}

// NewModelsHandler creates a models handler.
func NewModelsHandler(router *inference.Router) *ModelsHandler {
	return &ModelsHandler{router: router}
}

// List handles GET /v1/models.
func (h *ModelsHandler) List(ctx *fasthttp.RequestCtx) {
	resp := &schemas.ListModelsResponse{
		Object: "list",
	}

	// Aggregate models from all Stage-1 provider catalogs.
	for providerID := range catalog.Providers {
		for _, m := range catalog.ModelsFor(providerID) {
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
	// Phase 4: delegate to list and filter. Full catalog lookup in Phase 9.
	h.List(ctx)
}
