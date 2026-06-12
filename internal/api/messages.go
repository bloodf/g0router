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

// MessagesHandler handles POST /v1/messages (Claude-compatible endpoint).
type MessagesHandler struct {
	router         modelResolver
	registry       *translation.Registry
	usageRecorder  UsageRecorder
	pendingTracker PendingTracker
	detailCapture  DetailCapture
	vkGate         *VKGate
}

// NewMessagesHandler creates a Claude-compatible messages handler.
func NewMessagesHandler(router *inference.Router) *MessagesHandler {
	return &MessagesHandler{
		router:   router,
		registry: translation.NewRegistry(),
	}
}

// SetUsageRecorder wires a consumer for request_log entries (PAR-ROUTE-054).
func (h *MessagesHandler) SetUsageRecorder(r UsageRecorder) { h.usageRecorder = r }

// SetPendingTracker wires a consumer for in-flight request accounting
// (PAR-USAGE-018 wiring half).
func (h *MessagesHandler) SetPendingTracker(t PendingTracker) { h.pendingTracker = t }

// SetDetailCapture wires a consumer for full request detail capture
// (PAR-USAGE-026 production call-sites).
func (h *MessagesHandler) SetDetailCapture(d DetailCapture) { h.detailCapture = d }

// SetVKGate wires a virtual-key gate for x-g0-vk header enforcement (PAR-ROUTE-030).
func (h *MessagesHandler) SetVKGate(g *VKGate) { h.vkGate = g }

// Handle processes Claude-format requests, translating to/from OpenAI format.
func (h *MessagesHandler) Handle(ctx *fasthttp.RequestCtx) {
	raw := ctx.PostBody()
	headers := requestHeadersFromCtx(ctx)
	g := h.recordGlue()

	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "invalid JSON body", nil)
		return
	}

	model, _ := body["model"].(string)
	stream := false
	if s, ok := body["stream"].(bool); ok {
		stream = s
	}

	// Bypass check: short-circuit for Claude CLI patterns without calling provider (PAR-ROUTE-034).
	userAgent := string(ctx.Request.Header.UserAgent())
	if chunks, bypassed, err := translation.HandleBypassRequest(body, model, userAgent, false, h.registry); bypassed {
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, "server_error", "bypass error", nil)
			return
		}
		writeBypassResponse(ctx, chunks, stream)
		return
	}

	// Resolve provider first so native-format detection can skip translation (PAR-ROUTE-041).
	minReq := &schemas.ChatRequest{Model: model}
	provider, key, err := h.router.ResolveForModel(minReq)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", err.Error(), nil)
		return
	}

	// x-g0-vk virtual-key gate (PAR-ROUTE-030): after model resolution, before dispatch.
	if vkHeader := string(ctx.Request.Header.Peek("x-g0-vk")); vkHeader != "" {
		if ok, status, reason := h.vkGate.AllowVK(vkHeader, model); !ok {
			errType := "invalid_request_error"
			if status == 429 {
				errType = "rate_limit_exceeded"
			}
			g.recordError("/v1/messages", model, key.Provider, key.ID, raw, headers, &schemas.ProviderError{StatusCode: status, Message: reason, Type: errType})
			writeError(ctx, status, errType, reason, nil)
			return
		}
	}

	// Pending-tracker start (PAR-USAGE-018 wiring half).
	if h.pendingTracker != nil {
		h.pendingTracker.Start(model, key.Provider, key.ID)
	}

	var req schemas.ChatRequest

	// Native passthrough (PAR-ROUTE-041): when provider's native format matches the
	// detected source format, skip translation entirely.
	nativeSkip := false
	if nfp, ok := provider.(interface{ NativeFormat() string }); ok {
		if nfp.NativeFormat() == DetectFormat(body) {
			rawTranslated, _ := json.Marshal(body)
			if err := json.Unmarshal(rawTranslated, &req); err == nil {
				nativeSkip = true
			}
		}
	}

	if !nativeSkip {
		translated, err := h.registry.TranslateRequest(translation.FormatClaude, translation.FormatOpenAI, model, body, stream, nil)
		if err != nil {
			g.recordError("/v1/messages", model, key.Provider, key.ID, raw, headers, &schemas.ProviderError{StatusCode: 400, Message: err.Error(), Type: "invalid_request_error"})
			writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", err.Error(), nil)
			return
		}

		b, err := json.Marshal(translated)
		if err != nil {
			g.recordError("/v1/messages", model, key.Provider, key.ID, raw, headers, &schemas.ProviderError{StatusCode: 500, Message: "marshal failure", Type: "internal"})
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetContentTypeBytes([]byte("text/plain"))
			ctx.SetBodyString("internal error")
			return
		}

		if err := json.Unmarshal(b, &req); err != nil {
			g.recordError("/v1/messages", model, key.Provider, key.ID, raw, headers, &schemas.ProviderError{StatusCode: 400, Message: err.Error(), Type: "invalid_request_error"})
			writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", err.Error(), nil)
			return
		}
	}

	translation.PreprocessChatRequest(&req)

	// Thinking config override (PAR-ROUTE-042).
	if tm, ok := provider.(interface{ ThinkingMode() string }); ok {
		applyThinkingOverride(&req, tm.ThinkingMode())
	}

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}

	if stream {
		ctx.SetContentTypeBytes([]byte("text/event-stream"))
		ctx.Response.Header.Set("Cache-Control", "no-cache")
		ctx.Response.Header.Set("Connection", "keep-alive")

		ch, perr := provider.ChatCompletionStream(gatewayCtx, nil, key, &req)
		if perr != nil {
			g.recordError("/v1/messages", model, key.Provider, key.ID, raw, headers, perr)
			writeError(ctx, fasthttp.StatusBadGateway, perr.Type, perr.Message, perr.Code)
			return
		}

		streamCtx, cancel := withRequestCancel(ctx)
		defer cancel()
		state := translation.NewStreamState()
		src := &translation.EstimateSource{Body: body, Format: translation.FormatClaude}
		summary, sErr := translation.ProcessTranslateStream(streamCtx, ctx, ch, h.registry, translation.FormatOpenAI, translation.FormatClaude, state, src)
		if sErr != nil {
			log.Printf("messages stream error: %v", sErr)
		}
		g.recordStream("/v1/messages", model, key.Provider, key.ID, raw, headers, summary, sErr)
		return
	}

	resp, perr := provider.ChatCompletion(gatewayCtx, key, &req)
	if perr != nil {
		g.recordError("/v1/messages", model, key.Provider, key.ID, raw, headers, perr)
		status := perr.StatusCode
		if status == 0 {
			status = fasthttp.StatusBadGateway
		}
		writeError(ctx, status, perr.Type, perr.Message, perr.Code)
		return
	}

	// Non-streaming /v1/messages returns the OpenAI-shaped response unchanged.
	// This matches 9router: translateNonStreamingResponse only converts provider->OpenAI
	// and never synthesizes Claude JSON.
	out, err := jsonMarshal(resp)
	if err != nil {
		g.recordError("/v1/messages", model, key.Provider, key.ID, raw, headers, &schemas.ProviderError{StatusCode: 500, Message: "marshal failure", Type: "internal"})
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentTypeBytes([]byte("text/plain"))
		ctx.SetBodyString("internal error")
		return
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes([]byte("application/json"))
	ctx.SetBody(out)

	var pt, ct int64
	if resp != nil && resp.Usage != nil {
		pt = int64(resp.Usage.PromptTokens)
		ct = int64(resp.Usage.CompletionTokens)
	}
	g.recordNonStream("/v1/messages", model, key.Provider, key.ID, raw, headers, pt, ct, resp)
}

// recordGlue assembles the shared usage-recording dependencies for this handler.
func (h *MessagesHandler) recordGlue() recordGlue {
	return recordGlue{recorder: h.usageRecorder, tracker: h.pendingTracker, detail: h.detailCapture}
}
