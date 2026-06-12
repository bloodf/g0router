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
	router         embeddingsResolver
	usageRecorder  UsageRecorder
	pendingTracker PendingTracker
	detailCapture  DetailCapture
	vkGate         *VKGate
}

// embeddingsResolver is the subset of *inference.Router used by the embeddings
// handler. It exists so tests can inject behavior without going through the
// full router.
type embeddingsResolver interface {
	Resolve(model string) (schemas.Provider, schemas.Key, error)
}

// NewEmbeddingsHandler creates an embeddings handler.
func NewEmbeddingsHandler(router *inference.Router) *EmbeddingsHandler {
	return &EmbeddingsHandler{router: router}
}

// SetUsageRecorder wires a consumer for request_log entries (PAR-ROUTE-054).
func (h *EmbeddingsHandler) SetUsageRecorder(r UsageRecorder) { h.usageRecorder = r }

// SetPendingTracker wires a consumer for in-flight request accounting
// (PAR-USAGE-018 wiring half).
func (h *EmbeddingsHandler) SetPendingTracker(t PendingTracker) { h.pendingTracker = t }

// SetDetailCapture wires a consumer for full request detail capture
// (PAR-USAGE-026 production call-sites).
func (h *EmbeddingsHandler) SetDetailCapture(d DetailCapture) { h.detailCapture = d }

// SetVKGate wires a virtual-key gate for x-g0-vk header enforcement (PAR-ROUTE-030).
func (h *EmbeddingsHandler) SetVKGate(g *VKGate) { h.vkGate = g }

// Handle processes embedding requests.
func (h *EmbeddingsHandler) Handle(ctx *fasthttp.RequestCtx) {
	raw := ctx.PostBody()
	headers := requestHeadersFromCtx(ctx)
	g := h.recordGlue()

	var req schemas.EmbeddingRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "invalid JSON body", nil)
		return
	}

	provider, key, err := h.router.Resolve(req.Model)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", err.Error(), nil)
		return
	}

	// x-g0-vk virtual-key gate (PAR-ROUTE-030): after model resolution, before dispatch.
	vkHeader := string(ctx.Request.Header.Peek("x-g0-vk"))
	if vkHeader != "" {
		if ok, status, reason, _ := h.vkGate.AllowVK(vkHeader, req.Model, key.Provider); !ok {
			errType := "invalid_request_error"
			if status == 429 {
				errType = "rate_limit_exceeded"
			}
			g.recordError("/v1/embeddings", req.Model, key.Provider, key.ID, raw, headers, &schemas.ProviderError{StatusCode: status, Message: reason, Type: errType})
			writeError(ctx, status, errType, reason, nil)
			return
		}
		g.apiKey = vkHeader
	}

	// Pending-tracker start (PAR-USAGE-018 wiring half).
	if h.pendingTracker != nil {
		h.pendingTracker.Start(req.Model, key.Provider, key.ID)
	}

	// Keys are resolved by the router via the wired credential resolver.

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}
	resp, perr := provider.Embedding(gatewayCtx, key, &req)
	if perr != nil {
		g.recordError("/v1/embeddings", req.Model, key.Provider, key.ID, raw, headers, perr)
		status := perr.StatusCode
		if status == 0 {
			status = fasthttp.StatusBadGateway
		}
		writeError(ctx, status, perr.Type, perr.Message, perr.Code)
		return
	}

	b, err := jsonMarshal(resp)
	if err != nil {
		g.recordError("/v1/embeddings", req.Model, key.Provider, key.ID, raw, headers, &schemas.ProviderError{StatusCode: 500, Message: "marshal failure", Type: "internal"})
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentTypeBytes([]byte("text/plain"))
		ctx.SetBodyString("internal error")
		return
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes([]byte("application/json"))
	ctx.SetBody(b)

	var pt, ct int64
	if resp.Usage != nil {
		pt = int64(resp.Usage.PromptTokens)
		ct = int64(resp.Usage.CompletionTokens)
	}
	g.recordNonStream("/v1/embeddings", req.Model, key.Provider, key.ID, raw, headers, pt, ct, resp)
}

// recordGlue assembles the shared usage-recording dependencies for this handler.
func (h *EmbeddingsHandler) recordGlue() recordGlue {
	return recordGlue{recorder: h.usageRecorder, tracker: h.pendingTracker, detail: h.detailCapture}
}
