package api

import (
	"encoding/json"
	"fmt"

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

	translated, err := h.registry.TranslateRequest(translation.FormatClaude, translation.FormatOpenAI, model, body, stream)
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

	if stream {
		ctx.SetContentTypeBytes([]byte("text/event-stream"))
		ctx.Response.Header.Set("Cache-Control", "no-cache")
		ctx.Response.Header.Set("Connection", "keep-alive")

		ch, perr := provider.ChatCompletionStream(gatewayCtx, nil, key, &req)
		if perr != nil {
			writeError(ctx, fasthttp.StatusBadGateway, perr.Type, perr.Message, perr.Code)
			return
		}

		writeClaudeSSEStream(ctx, ch, h.registry)
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

// claudeStreamWriter implements streamWriter and emits Claude-format SSE frames.
type claudeStreamWriter struct {
	ctx      *fasthttp.RequestCtx
	registry *translation.Registry
	state    *translation.StreamState
}

func (w *claudeStreamWriter) Write(p []byte) (int, error) {
	return w.ctx.Write(p)
}

func (w *claudeStreamWriter) WriteString(s string) (int, error) {
	return w.ctx.WriteString(s)
}

// writeClaudeSSEStream drains ch, translating each OpenAI chunk to Claude
// format and framing it with FormatSSE.
func writeClaudeSSEStream(ctx *fasthttp.RequestCtx, ch chan *schemas.StreamChunk, registry *translation.Registry) {
	state := translation.NewStreamState()
	w := &claudeStreamWriter{ctx: ctx, registry: registry, state: state}
	for chunk := range ch {
		if chunk.Error != nil {
			return
		}
		b, err := jsonMarshal(chunk)
		if err != nil {
			return
		}
		var openaiChunk map[string]any
		if err := json.Unmarshal(b, &openaiChunk); err != nil {
			return
		}
		events, err := registry.TranslateResponse(translation.FormatOpenAI, translation.FormatClaude, openaiChunk, state)
		if err != nil {
			return
		}
		for _, ev := range events {
			if _, werr := w.Write(translation.FormatSSE(translation.FormatClaude, ev)); werr != nil {
				return
			}
		}
	}
	if _, werr := w.WriteString("data: [DONE]\n\n"); werr != nil {
		return
	}
}
