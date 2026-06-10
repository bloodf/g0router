package api

import (
	"encoding/json"
	"fmt"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// streamWriter is the subset of fasthttp.RequestCtx used by writeSSEStream.
// It exists so tests can inject write failures (AUD-008); fasthttp's
// in-memory response buffer never returns errors.
type streamWriter interface {
	Write(p []byte) (int, error)
	WriteString(s string) (int, error)
}

// writeSSEStream drains ch onto w as SSE frames. It aborts on the first
// marshal failure (AUD-007) or write failure (AUD-008) instead of emitting
// corrupt frames or blocking the producing goroutine on a dead client.
func writeSSEStream(w streamWriter, ch chan *schemas.StreamChunk) {
	for chunk := range ch {
		b, err := jsonMarshal(chunk)
		if err != nil {
			return
		}
		if _, werr := w.WriteString("data: "); werr != nil {
			return
		}
		if _, werr := w.Write(b); werr != nil {
			return
		}
		if _, werr := w.WriteString("\n\n"); werr != nil {
			return
		}
	}
	if _, werr := w.WriteString("data: [DONE]\n\n"); werr != nil {
		return
	}
}

// ChatHandler handles POST /v1/chat/completions.
type ChatHandler struct {
	router *inference.Router
}

// NewChatHandler creates a chat completion handler.
func NewChatHandler(router *inference.Router) *ChatHandler {
	return &ChatHandler{router: router}
}

// Handle processes chat completion requests (streaming and non-streaming).
func (h *ChatHandler) Handle(ctx *fasthttp.RequestCtx) {
	var req schemas.ChatRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "invalid JSON body", nil)
		return
	}

	provider, key, err := h.router.ResolveForModel(&req)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", err.Error(), nil)
		return
	}

	// Keys are provided by the management layer (WebUI) via the router.
	// Phase 6+ will wire the key store; empty keys yield provider auth errors.

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}

	if req.Stream {
		ctx.SetContentTypeBytes([]byte("text/event-stream"))
		ctx.Response.Header.Set("Cache-Control", "no-cache")
		ctx.Response.Header.Set("Connection", "keep-alive")

		ch, perr := provider.ChatCompletionStream(gatewayCtx, nil, key, &req)
		if perr != nil {
			writeError(ctx, fasthttp.StatusBadGateway, perr.Type, perr.Message, perr.Code)
			return
		}

		writeSSEStream(ctx, ch)
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
