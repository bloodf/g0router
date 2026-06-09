package api

import (
	"encoding/json"
	"fmt"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

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

	// Phase 5: inject API key from env based on resolved provider.
	if key.Value == "" {
		key.Value = resolveAPIKey(provider)
	}

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

		for chunk := range ch {
			b, _ := json.Marshal(chunk)
			ctx.WriteString("data: ")
			ctx.Write(b)
			ctx.WriteString("\n\n")
		}
		ctx.WriteString("data: [DONE]\n\n")
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

	b, _ := json.Marshal(resp)
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes([]byte("application/json"))
	ctx.SetBody(b)
}
