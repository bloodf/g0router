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

// modelResolver is the subset of inference.Router used by ChatHandler.
// It exists so tests can inject behavior without relying on the full router.
type modelResolver interface {
	ResolveForModel(*schemas.ChatRequest) (schemas.Provider, schemas.Key, error)
}

// streamWriter is the subset of fasthttp.RequestCtx used by writeSSEStream.
// It exists so tests can inject write failures (AUD-008); fasthttp's
// in-memory response buffer never returns errors.
type streamWriter interface {
	Write(p []byte) (int, error)
	WriteString(s string) (int, error)
}

// writeSSEStream drains ch onto w as framed SSE via the shared passthrough
// processor (PAR-TRANS-049). It returns a non-nil error if the stream
// aborted on an error chunk or write failure.
func writeSSEStream(w streamWriter, ch chan *schemas.StreamChunk) error {
	_, err := translation.ProcessPassthroughStream(w, ch)
	return err
}

// ChatHandler handles POST /v1/chat/completions.
type ChatHandler struct {
	router modelResolver
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

	translation.PreprocessChatRequest(&req)

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

		if err := writeSSEStream(ctx, ch); err != nil {
			log.Printf("chat stream error: %v", err)
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
