package api

import (
	"encoding/json"
	"fmt"

	"github.com/bloodf/g0router/internal/inference"
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
	provider, key, err := h.router.Resolve("")
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", err.Error(), nil)
		return
	}

	if key.Value == "" {
		key.Value = resolveAPIKey(provider)
	}

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}
	resp, perr := provider.ListModels(gatewayCtx, key)
	if perr != nil {
		status := perr.StatusCode
		if status == 0 {
			status = fasthttp.StatusBadGateway
		}
		writeError(ctx, status, perr.Type, perr.Message, perr.Code)
		return
	}

	b, _ := json.Marshal(resp)
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes([]byte("application/json"))
	ctx.SetBody(b)
}

// Get handles GET /v1/models/:id.
func (h *ModelsHandler) Get(ctx *fasthttp.RequestCtx) {
	// Phase 4: delegate to list and filter. Full catalog lookup in Phase 9.
	h.List(ctx)
}
