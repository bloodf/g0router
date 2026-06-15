package api

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// CompletionsHandler handles POST /v1/completions (legacy text completions).
type CompletionsHandler struct {
	router         completionsResolver
	usageRecorder  UsageRecorder
	pendingTracker PendingTracker
	detailCapture  DetailCapture
	vkGate         *VKGate
	pinnedResolver VKPinnedKeyResolver
}

// completionsResolver is the subset of *inference.Router used by the completions
// handler. It exists so tests can inject behavior without going through the full
// router.
type completionsResolver interface {
	Resolve(model string) (schemas.Provider, schemas.Key, error)
}

// NewCompletionsHandler creates a completions handler.
func NewCompletionsHandler(router *inference.Router) *CompletionsHandler {
	return &CompletionsHandler{router: router}
}

// SetUsageRecorder wires a consumer for request_log entries (PAR-ROUTE-054).
func (h *CompletionsHandler) SetUsageRecorder(r UsageRecorder) { h.usageRecorder = r }

// SetPendingTracker wires a consumer for in-flight request accounting
// (PAR-USAGE-018 wiring half).
func (h *CompletionsHandler) SetPendingTracker(t PendingTracker) { h.pendingTracker = t }

// SetDetailCapture wires a consumer for full request detail capture
// (PAR-USAGE-026 production call-sites).
func (h *CompletionsHandler) SetDetailCapture(d DetailCapture) { h.detailCapture = d }

// SetVKGate wires a virtual-key gate for x-g0-vk header enforcement (PAR-ROUTE-030).
func (h *CompletionsHandler) SetVKGate(g *VKGate) { h.vkGate = g }

// SetVKPinnedResolver wires the resolver for virtual-key KeyID pinning
// (PAR-ROUTE-030).
func (h *CompletionsHandler) SetVKPinnedResolver(r VKPinnedKeyResolver) { h.pinnedResolver = r }

// Handle processes text completion requests (streaming and non-streaming).
func (h *CompletionsHandler) Handle(ctx *fasthttp.RequestCtx) {
	raw := ctx.PostBody()
	headers := requestHeadersFromCtx(ctx)
	g := h.recordGlue()

	var req schemas.TextCompletionRequest
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
		ok, status, reason, keyIDs := h.vkGate.AllowVK(vkHeader, req.Model, key.Provider)
		if !ok {
			errType := "invalid_request_error"
			if status == 429 {
				errType = "rate_limit_exceeded"
			}
			g.recordError("/v1/completions", req.Model, key.Provider, key.ID, raw, headers, &schemas.ProviderError{StatusCode: status, Message: reason, Type: errType})
			writeError(ctx, status, errType, reason, nil)
			return
		}
		if len(keyIDs) > 0 && h.pinnedResolver != nil {
			if connID, credential, ok := h.pinnedResolver.ResolvePinned(key.Provider, req.Model, keyIDs); ok {
				key.ID = connID
				key.Value = credential
			}
		}
		g.apiKey = vkHeader
	}

	// Pending-tracker start (PAR-USAGE-018 wiring half).
	if h.pendingTracker != nil {
		h.pendingTracker.Start(req.Model, key.Provider, key.ID)
	}

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}

	if req.Stream {
		// Open the provider stream BEFORE setting SSE headers so a stream-open
		// *ProviderError returns an application/json error, not a
		// text/event-stream framing mismatch (PAR-BF-OAI-201).
		ch, perr := provider.TextCompletionStream(gatewayCtx, nil, key, &req)
		if perr != nil {
			g.recordError("/v1/completions", req.Model, key.Provider, key.ID, raw, headers, perr)
			status := perr.StatusCode
			if status == 0 {
				status = fasthttp.StatusBadGateway
			}
			writeError(ctx, status, perr.Type, perr.Message, perr.Code)
			return
		}

		ctx.SetContentTypeBytes([]byte("text/event-stream"))
		ctx.Response.Header.Set("Cache-Control", "no-cache")
		ctx.Response.Header.Set("Connection", "keep-alive")

		streamCtx, cancel := withRequestCancel(ctx)
		defer cancel()
		if sErr := writeSSEStream(streamCtx, ctx, ch); sErr != nil {
			log.Printf("completions stream error: %v", sErr)
		}
		return
	}

	resp, perr := provider.TextCompletion(gatewayCtx, key, &req)
	if perr != nil {
		g.recordError("/v1/completions", req.Model, key.Provider, key.ID, raw, headers, perr)
		status := perr.StatusCode
		if status == 0 {
			status = fasthttp.StatusBadGateway
		}
		writeError(ctx, status, perr.Type, perr.Message, perr.Code)
		return
	}

	b, err := jsonMarshal(resp)
	if err != nil {
		g.recordError("/v1/completions", req.Model, key.Provider, key.ID, raw, headers, &schemas.ProviderError{StatusCode: 500, Message: "marshal failure", Type: "internal"})
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentTypeBytes([]byte("text/plain"))
		ctx.SetBodyString("internal error")
		return
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes([]byte("application/json"))
	ctx.SetBody(b)

	var pt, ct int64
	if resp != nil && resp.Usage != nil {
		pt = int64(resp.Usage.PromptTokens)
		ct = int64(resp.Usage.CompletionTokens)
	}
	g.recordNonStream("/v1/completions", req.Model, key.Provider, key.ID, raw, headers, pt, ct, resp)
}

// recordGlue assembles the shared usage-recording dependencies for this handler.
func (h *CompletionsHandler) recordGlue() recordGlue {
	return recordGlue{recorder: h.usageRecorder, tracker: h.pendingTracker, detail: h.detailCapture}
}
