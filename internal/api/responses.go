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
	router   modelResolver
	registry *translation.Registry
}

// NewResponsesHandler creates an OpenAI Responses API handler.
func NewResponsesHandler(router *inference.Router) *ResponsesHandler {
	return &ResponsesHandler{
		router:   router,
		registry: translation.NewRegistry(),
	}
}

// Handle processes Responses-format requests, translating to/from OpenAI Chat
// Completions format. The streaming path is the only path — the ref endpoint is
// streaming-only (stream:true forced at openai-responses.js:203,208).
func (h *ResponsesHandler) Handle(ctx *fasthttp.RequestCtx) {
	var body map[string]any
	if err := json.Unmarshal(ctx.PostBody(), &body); err != nil {
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

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}

	// Streaming-only: no non-streaming branch.
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
	if _, err := translation.ProcessTranslateStream(streamCtx, ctx, ch, h.registry, translation.FormatOpenAI, translation.FormatOpenAIResponses, state, nil); err != nil {
		log.Printf("responses stream error: %v", err)
	}
}
