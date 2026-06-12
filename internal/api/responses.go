package api

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
)

// ResponsesHandler handles POST /v1/responses (OpenAI Responses API endpoint).
// Streaming is forced true — the ref request translator ALWAYS sets stream:true
// (openai-responses.js:203,208), so this endpoint is streaming-only by parity.
type ResponsesHandler struct {
	router         modelResolver
	registry       *translation.Registry
	usageRecorder  UsageRecorder
	pendingTracker PendingTracker
	detailCapture  DetailCapture
}

// NewResponsesHandler creates an OpenAI Responses API handler.
func NewResponsesHandler(router *inference.Router) *ResponsesHandler {
	return &ResponsesHandler{
		router:   router,
		registry: translation.NewRegistry(),
	}
}

// SetUsageRecorder wires a consumer for request_log entries (PAR-ROUTE-054).
func (h *ResponsesHandler) SetUsageRecorder(r UsageRecorder) { h.usageRecorder = r }

// SetPendingTracker wires a consumer for in-flight request accounting
// (PAR-USAGE-018 wiring half).
func (h *ResponsesHandler) SetPendingTracker(t PendingTracker) { h.pendingTracker = t }

// SetDetailCapture wires a consumer for full request detail capture
// (PAR-USAGE-026 production call-sites).
func (h *ResponsesHandler) SetDetailCapture(d DetailCapture) { h.detailCapture = d }

// Handle processes Responses-format requests, translating to/from OpenAI Chat
// Completions format. The streaming path is the only path — the ref endpoint is
// streaming-only (stream:true forced at openai-responses.js:203,208).
func (h *ResponsesHandler) Handle(ctx *fasthttp.RequestCtx) {
	raw := ctx.PostBody()
	headers := requestHeadersFromCtx(ctx)
	g := h.recordGlue()

	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "invalid JSON body", nil)
		return
	}

	model, _ := body["model"].(string)
	stream := true // forced: ref unconditionally sets stream:true

	translated, err := h.registry.TranslateRequest(translation.FormatOpenAIResponses, translation.FormatOpenAI, model, body, stream, nil)
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

	// Pending-tracker start (PAR-USAGE-018 wiring half).
	if h.pendingTracker != nil {
		h.pendingTracker.Start(req.Model, key.Provider, key.ID)
	}

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}

	// Streaming-only: no non-streaming branch.
	ctx.SetContentTypeBytes([]byte("text/event-stream"))
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")

	ch, perr := provider.ChatCompletionStream(gatewayCtx, nil, key, &req)
	if perr != nil {
		g.recordError("/v1/responses", req.Model, key.Provider, key.ID, raw, headers, perr)
		writeError(ctx, fasthttp.StatusBadGateway, perr.Type, perr.Message, perr.Code)
		return
	}

	streamCtx, cancel := withRequestCancel(ctx)
	defer cancel()
	state := translation.NewStreamState()
	src := &translation.EstimateSource{Body: body, Format: translation.FormatOpenAIResponses}
	summary, sErr := translation.ProcessTranslateStream(streamCtx, ctx, ch, h.registry, translation.FormatOpenAI, translation.FormatOpenAIResponses, state, src)
	if sErr != nil {
		log.Printf("responses stream error: %v", sErr)
	}
	g.recordStream("/v1/responses", req.Model, key.Provider, key.ID, raw, headers, summary, sErr)
}

// recordGlue assembles the shared usage-recording dependencies for this handler.
func (h *ResponsesHandler) recordGlue() recordGlue {
	return recordGlue{recorder: h.usageRecorder, tracker: h.pendingTracker, detail: h.detailCapture}
}
