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
	router   modelResolver
	registry *translation.Registry
}

// NewMessagesHandler creates a Claude-compatible messages handler.
func NewMessagesHandler(router *inference.Router) *MessagesHandler {
	return &MessagesHandler{
		router:   router,
		registry: translation.NewRegistry(),
	}
}

// Handle processes Claude-format requests, translating to/from OpenAI format.
func (h *MessagesHandler) Handle(ctx *fasthttp.RequestCtx) {
	var body map[string]any
	if err := json.Unmarshal(ctx.PostBody(), &body); err != nil {
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

	var req schemas.ChatRequest

	// Native passthrough (PAR-ROUTE-041): when provider's native format matches the
	// detected source format, skip translation entirely.
	nativeSkip := false
	if nfp, ok := provider.(interface{ NativeFormat() string }); ok {
		if nfp.NativeFormat() == DetectFormat(body) {
			raw, _ := json.Marshal(body)
			if err := json.Unmarshal(raw, &req); err == nil {
				nativeSkip = true
			}
		}
	}

	if !nativeSkip {
		translated, err := h.registry.TranslateRequest(translation.FormatClaude, translation.FormatOpenAI, model, body, stream, nil)
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

		if err := json.Unmarshal(b, &req); err != nil {
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
			writeError(ctx, fasthttp.StatusBadGateway, perr.Type, perr.Message, perr.Code)
			return
		}

		streamCtx, cancel := withRequestCancel(ctx)
		defer cancel()
		state := translation.NewStreamState()
		if _, err := translation.ProcessTranslateStream(streamCtx, ctx, ch, h.registry, translation.FormatOpenAI, translation.FormatClaude, state, nil); err != nil {
			log.Printf("messages stream error: %v", err)
		}
		return
	}

	resp, perr := provider.ChatCompletion(gatewayCtx, key, &req)
	if perr != nil {
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
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentTypeBytes([]byte("text/plain"))
		ctx.SetBodyString("internal error")
		return
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes([]byte("application/json"))
	ctx.SetBody(out)
}
