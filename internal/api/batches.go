package api

import (
	"encoding/json"
	"fmt"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// BatchesHandler handles the /v1/batches CRUD routes (create/list/retrieve/
// cancel). These are OpenAI-compatible routes returning the bare OpenAI object —
// never the {data,error} admin envelope. State lives upstream at OpenAI (Option A,
// stateless passthrough); requests carry no model, so resolution uses the
// empty-model sentinel which lands the openai provider.
type BatchesHandler struct {
	router         completionsResolver
	usageRecorder  UsageRecorder
	pendingTracker PendingTracker
	detailCapture  DetailCapture
	vkGate         *VKGate
	pinnedResolver VKPinnedKeyResolver
}

// NewBatchesHandler creates a batches handler.
func NewBatchesHandler(router *inference.Router) *BatchesHandler {
	return &BatchesHandler{router: router}
}

// SetUsageRecorder wires a usage recorder.
func (h *BatchesHandler) SetUsageRecorder(r UsageRecorder) { h.usageRecorder = r }

// SetPendingTracker wires a pending tracker.
func (h *BatchesHandler) SetPendingTracker(t PendingTracker) { h.pendingTracker = t }

// SetDetailCapture wires a detail capture.
func (h *BatchesHandler) SetDetailCapture(d DetailCapture) { h.detailCapture = d }

// SetVKGate wires a virtual-key gate.
func (h *BatchesHandler) SetVKGate(g *VKGate) { h.vkGate = g }

// SetVKPinnedResolver wires the virtual-key pinned-key resolver.
func (h *BatchesHandler) SetVKPinnedResolver(r VKPinnedKeyResolver) { h.pinnedResolver = r }

func (h *BatchesHandler) recordGlue() recordGlue {
	return recordGlue{recorder: h.usageRecorder, tracker: h.pendingTracker, detail: h.detailCapture}
}

// resolveAndGate resolves the openai provider (empty-model sentinel) and applies
// the x-g0-vk gate + pinning, mirroring AudioHandler.resolveAndGate.
func (h *BatchesHandler) resolveAndGate(ctx *fasthttp.RequestCtx, raw []byte, headers map[string]string, endpoint string, g *recordGlue) (schemas.Provider, schemas.Key, *schemas.ProviderError) {
	const model = ""
	provider, key, err := h.router.Resolve(model)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", err.Error(), nil)
		return nil, schemas.Key{}, &schemas.ProviderError{StatusCode: 400}
	}

	vkHeader := string(ctx.Request.Header.Peek("x-g0-vk"))
	if vkHeader != "" {
		ok, status, reason, keyIDs := h.vkGate.AllowVK(vkHeader, model, key.Provider)
		if !ok {
			errType := "invalid_request_error"
			if status == 429 {
				errType = "rate_limit_exceeded"
			}
			g.recordError(endpoint, model, key.Provider, key.ID, raw, headers, &schemas.ProviderError{StatusCode: status, Message: reason, Type: errType})
			writeError(ctx, status, errType, reason, nil)
			return nil, schemas.Key{}, &schemas.ProviderError{StatusCode: status}
		}
		if len(keyIDs) > 0 && h.pinnedResolver != nil {
			if connID, credential, ok := h.pinnedResolver.ResolvePinned(key.Provider, model, keyIDs); ok {
				key.ID = connID
				key.Value = credential
			}
		}
		g.apiKey = vkHeader
	}
	return provider, key, nil
}

// writeJSON marshals resp as the bare OpenAI object and writes it 200, falling
// back to a plain-text 500 on marshal failure (mirrors AudioHandler).
func (h *BatchesHandler) writeJSON(ctx *fasthttp.RequestCtx, g *recordGlue, endpoint, model string, key schemas.Key, raw []byte, headers map[string]string, resp any) {
	b, mErr := jsonMarshal(resp)
	if mErr != nil {
		g.recordError(endpoint, model, key.Provider, key.ID, raw, headers, &schemas.ProviderError{StatusCode: 500, Message: "marshal failure", Type: "internal"})
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentTypeBytes([]byte("text/plain"))
		ctx.SetBodyString("internal error")
		return
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes([]byte("application/json"))
	ctx.SetBody(b)
	g.recordNonStream(endpoint, model, key.Provider, key.ID, raw, headers, 0, 0, resp)
}

// Create handles POST /v1/batches. It parses the JSON BatchCreateRequest and
// returns the bare Batch JSON.
func (h *BatchesHandler) Create(ctx *fasthttp.RequestCtx) {
	const endpoint = "/v1/batches"
	raw := ctx.PostBody()
	headers := requestHeadersFromCtx(ctx)
	g := h.recordGlue()

	var req schemas.BatchCreateRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "invalid JSON body", nil)
		return
	}

	provider, key, perr := h.resolveAndGate(ctx, raw, headers, endpoint, &g)
	if perr != nil {
		return
	}
	if h.pendingTracker != nil {
		h.pendingTracker.Start("", key.Provider, key.ID)
	}

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}
	resp, sperr := provider.BatchCreate(gatewayCtx, key, &req)
	if sperr != nil {
		g.recordError(endpoint, "", key.Provider, key.ID, raw, headers, sperr)
		writeProviderError(ctx, sperr)
		return
	}
	h.writeJSON(ctx, &g, endpoint, "", key, raw, headers, resp)
}

// List handles GET /v1/batches and returns the bare BatchListResponse JSON.
func (h *BatchesHandler) List(ctx *fasthttp.RequestCtx) {
	const endpoint = "/v1/batches"
	raw := ctx.PostBody()
	headers := requestHeadersFromCtx(ctx)
	g := h.recordGlue()

	provider, key, perr := h.resolveAndGate(ctx, raw, headers, endpoint, &g)
	if perr != nil {
		return
	}
	if h.pendingTracker != nil {
		h.pendingTracker.Start("", key.Provider, key.ID)
	}

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}
	resp, sperr := provider.BatchList(gatewayCtx, key)
	if sperr != nil {
		g.recordError(endpoint, "", key.Provider, key.ID, raw, headers, sperr)
		writeProviderError(ctx, sperr)
		return
	}
	h.writeJSON(ctx, &g, endpoint, "", key, raw, headers, resp)
}

// Retrieve handles GET /v1/batches/{batch_id} and returns the bare Batch JSON.
func (h *BatchesHandler) Retrieve(ctx *fasthttp.RequestCtx) {
	const endpoint = "/v1/batches/{batch_id}"
	raw := ctx.PostBody()
	headers := requestHeadersFromCtx(ctx)
	g := h.recordGlue()

	batchID, _ := ctx.UserValue("batch_id").(string)
	if batchID == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "missing batch id", nil)
		return
	}

	provider, key, perr := h.resolveAndGate(ctx, raw, headers, endpoint, &g)
	if perr != nil {
		return
	}
	if h.pendingTracker != nil {
		h.pendingTracker.Start("", key.Provider, key.ID)
	}

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}
	resp, sperr := provider.BatchRetrieve(gatewayCtx, key, batchID)
	if sperr != nil {
		g.recordError(endpoint, "", key.Provider, key.ID, raw, headers, sperr)
		writeProviderError(ctx, sperr)
		return
	}
	h.writeJSON(ctx, &g, endpoint, "", key, raw, headers, resp)
}

// Cancel handles POST /v1/batches/{batch_id}/cancel and returns the bare Batch.
func (h *BatchesHandler) Cancel(ctx *fasthttp.RequestCtx) {
	const endpoint = "/v1/batches/{batch_id}/cancel"
	raw := ctx.PostBody()
	headers := requestHeadersFromCtx(ctx)
	g := h.recordGlue()

	batchID, _ := ctx.UserValue("batch_id").(string)
	if batchID == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "missing batch id", nil)
		return
	}

	provider, key, perr := h.resolveAndGate(ctx, raw, headers, endpoint, &g)
	if perr != nil {
		return
	}
	if h.pendingTracker != nil {
		h.pendingTracker.Start("", key.Provider, key.ID)
	}

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}
	resp, sperr := provider.BatchCancel(gatewayCtx, key, batchID)
	if sperr != nil {
		g.recordError(endpoint, "", key.Provider, key.ID, raw, headers, sperr)
		writeProviderError(ctx, sperr)
		return
	}
	h.writeJSON(ctx, &g, endpoint, "", key, raw, headers, resp)
}
