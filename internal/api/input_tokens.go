package api

import (
	"encoding/json"
	"fmt"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
)

// InputTokensHandler handles POST /v1/responses/input_tokens (OpenAI Responses
// token-count endpoint). It translates the Responses-shaped body into a
// ChatRequest (reusing the SHIPPED responses translation path with stream=false)
// and dispatches provider.CountTokens, returning the bare TokenCountResponse
// JSON ({"tokens":N}). This route is non-streaming.
type InputTokensHandler struct {
	router         modelResolver
	registry       *translation.Registry
	usageRecorder  UsageRecorder
	pendingTracker PendingTracker
	detailCapture  DetailCapture
	vkGate         *VKGate
	pinnedResolver VKPinnedKeyResolver
}

// NewInputTokensHandler creates a /v1/responses/input_tokens handler.
func NewInputTokensHandler(router *inference.Router) *InputTokensHandler {
	return &InputTokensHandler{router: router, registry: translation.NewRegistry()}
}

// SetUsageRecorder wires a consumer for request_log entries (PAR-ROUTE-054).
func (h *InputTokensHandler) SetUsageRecorder(r UsageRecorder) { h.usageRecorder = r }

// SetPendingTracker wires a consumer for in-flight request accounting.
func (h *InputTokensHandler) SetPendingTracker(t PendingTracker) { h.pendingTracker = t }

// SetDetailCapture wires a consumer for full request detail capture.
func (h *InputTokensHandler) SetDetailCapture(d DetailCapture) { h.detailCapture = d }

// SetVKGate wires a virtual-key gate for x-g0-vk header enforcement (PAR-ROUTE-030).
func (h *InputTokensHandler) SetVKGate(g *VKGate) { h.vkGate = g }

// SetVKPinnedResolver wires the resolver for virtual-key KeyID pinning (PAR-ROUTE-030).
func (h *InputTokensHandler) SetVKPinnedResolver(r VKPinnedKeyResolver) { h.pinnedResolver = r }

// Handle processes a token-count request: translate the Responses-shaped body to
// a ChatRequest (non-streaming), resolve+gate, dispatch CountTokens, and emit the
// bare TokenCountResponse JSON.
func (h *InputTokensHandler) Handle(ctx *fasthttp.RequestCtx) {
	const endpoint = "/v1/responses/input_tokens"
	raw := ctx.PostBody()
	headers := requestHeadersFromCtx(ctx)
	g := h.recordGlue()

	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "invalid JSON body", nil)
		return
	}

	model, _ := body["model"].(string)

	// Reuse the SHIPPED responses→chat translation (stream=false; count is non-streaming).
	translated, err := h.registry.TranslateRequest(translation.FormatOpenAIResponses, translation.FormatOpenAI, model, body, false, nil)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", err.Error(), nil)
		return
	}

	b, err := json.Marshal(translated)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentTypeBytes([]byte("text/plain"))
		ctx.SetBodyString("internal error")
		return
	}

	var req schemas.ChatRequest
	if err := json.Unmarshal(b, &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", err.Error(), nil)
		return
	}

	translation.PreprocessChatRequest(&req)

	provider, key, err := h.router.ResolveForModel(&req)
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
			g.recordError(endpoint, req.Model, key.Provider, key.ID, raw, headers, &schemas.ProviderError{StatusCode: status, Message: reason, Type: errType})
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

	if h.pendingTracker != nil {
		h.pendingTracker.Start(req.Model, key.Provider, key.ID)
	}

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}

	resp, perr := provider.CountTokens(gatewayCtx, key, &req)
	if perr != nil {
		g.recordError(endpoint, req.Model, key.Provider, key.ID, raw, headers, perr)
		writeProviderError(ctx, perr)
		return
	}

	out, mErr := jsonMarshal(resp)
	if mErr != nil {
		g.recordError(endpoint, req.Model, key.Provider, key.ID, raw, headers, &schemas.ProviderError{StatusCode: 500, Message: "marshal failure", Type: "internal"})
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentTypeBytes([]byte("text/plain"))
		ctx.SetBodyString("internal error")
		return
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes([]byte("application/json"))
	ctx.SetBody(out)

	// Count is not a billed inference; record 0/0 tokens (mirror audio's 0/0).
	g.recordNonStream(endpoint, req.Model, key.Provider, key.ID, raw, headers, 0, 0, resp)
}

// recordGlue assembles the shared usage-recording dependencies for this handler.
func (h *InputTokensHandler) recordGlue() recordGlue {
	return recordGlue{recorder: h.usageRecorder, tracker: h.pendingTracker, detail: h.detailCapture}
}
