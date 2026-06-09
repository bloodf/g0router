package api

import (
	"encoding/json"
	"fmt"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// EmbeddingsHandler handles POST /v1/embeddings.
type EmbeddingsHandler struct {
	router *inference.Router
}

// NewEmbeddingsHandler creates an embeddings handler.
func NewEmbeddingsHandler(router *inference.Router) *EmbeddingsHandler {
	return &EmbeddingsHandler{router: router}
}

// Handle processes embedding requests.
func (h *EmbeddingsHandler) Handle(ctx *fasthttp.RequestCtx) {
	var req schemas.EmbeddingRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "invalid JSON body", nil)
		return
	}

	provider, key, err := h.router.Resolve(req.Model)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", err.Error(), nil)
		return
	}

	// Keys are provided by the management layer (WebUI) via the router.
	// Phase 6+ will wire the key store; empty keys yield provider auth errors.

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}
	resp, perr := provider.Embedding(gatewayCtx, key, &req)
	if perr != nil {
		status := perr.StatusCode
		if status == 0 {
			status = fasthttp.StatusBadGateway
		}
		writeError(ctx, status, perr.Type, perr.Message, perr.Code)
		return
	}

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
