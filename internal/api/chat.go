package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

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
// aborted on an error chunk or write failure. The loop watches ctx.Done()
// so the handler can return promptly on client abort.
func writeSSEStream(ctx context.Context, w streamWriter, ch chan *schemas.StreamChunk) error {
	_, err := translation.ProcessPassthroughStream(ctx, w, ch)
	return err
}

// withRequestCancel returns a cancellable context derived from reqCtx when
// running inside a real fasthttp server. Unit tests often use a bare
// *fasthttp.RequestCtx whose Done() panics, so the helper falls back to
// context.Background() in that case.
func withRequestCancel(reqCtx *fasthttp.RequestCtx) (context.Context, context.CancelFunc) {
	if c, cancel, ok := tryDeriveCancel(reqCtx); ok {
		return c, cancel
	}
	return context.WithCancel(context.Background())
}

// tryDeriveCancel attempts to derive a cancellable context. It recovers from
// panics caused by contexts whose Done() method is not usable (e.g., a bare
// *fasthttp.RequestCtx in unit tests).
func tryDeriveCancel(ctx context.Context) (context.Context, context.CancelFunc, bool) {
	defer func() {
		if r := recover(); r != nil {
			// ctx.Done() panicked; report failure to caller.
		}
	}()
	c, cancel := context.WithCancel(ctx)
	return c, cancel, true
}

// writeBypassResponse writes a bypass (fake) response to ctx without calling a provider.
// For streaming requests each chunk is framed as an SSE event; for non-streaming the
// first chunk is written as JSON. (PAR-ROUTE-034)
func writeBypassResponse(ctx *fasthttp.RequestCtx, chunks []map[string]any, stream bool) {
	if stream {
		ctx.SetContentTypeBytes([]byte("text/event-stream"))
		ctx.Response.Header.Set("Cache-Control", "no-cache")
		ctx.Response.Header.Set("Connection", "keep-alive")
		for _, chunk := range chunks {
			b, _ := json.Marshal(chunk)
			fmt.Fprintf(ctx, "data: %s\n\n", b)
		}
		fmt.Fprint(ctx, "data: [DONE]\n\n")
		return
	}
	if len(chunks) == 0 {
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetContentTypeBytes([]byte("application/json"))
		ctx.SetBodyString("{}")
		return
	}
	b, err := json.Marshal(chunks[0])
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

// CredentialRefresher refreshes OAuth credentials on 401/403 from a provider.
// It is wired in production via SetCredentialRefresher (PAR-ROUTE-023).
type CredentialRefresher interface {
	RefreshCredentials(connectionID string) (string, error)
}

// ChatHandler handles POST /v1/chat/completions.
type ChatHandler struct {
	router    modelResolver
	refresher CredentialRefresher
}

// NewChatHandler creates a chat completion handler.
func NewChatHandler(router *inference.Router) *ChatHandler {
	return &ChatHandler{router: router}
}

// SetCredentialRefresher wires an OAuth credential refresher for 401/403 retry.
func (h *ChatHandler) SetCredentialRefresher(cr CredentialRefresher) {
	h.refresher = cr
}

// applyThinkingOverride injects provider-level thinking config when not already set.
// Mirrors chatCore.js:48-58 (PAR-ROUTE-042).
func applyThinkingOverride(req *schemas.ChatRequest, mode string) {
	if mode == "" || mode == "auto" {
		return
	}
	switch mode {
	case "on":
		if req.Thinking == nil {
			req.Thinking = &schemas.ThinkingConfig{Type: "enabled", BudgetTokens: 10000}
		}
	case "off":
		if req.Thinking == nil {
			req.Thinking = &schemas.ThinkingConfig{Type: "disabled"}
		}
	default:
		if req.ReasoningEffort == "" {
			req.ReasoningEffort = mode
		}
	}
}

// retryWithRefresh performs up to 3 refresh-then-dispatch cycles on 401/403.
// Each cycle: refresh credentials → re-dispatch. Stops when dispatch succeeds or
// refresh fails. Mirrors chatCore.js:refreshWithRetry (PAR-ROUTE-023).
func (h *ChatHandler) retryWithRefresh(
	dispatch func(key schemas.Key) (*schemas.ChatResponse, *schemas.ProviderError),
	key schemas.Key,
	perr *schemas.ProviderError,
) (*schemas.ChatResponse, *schemas.ProviderError) {
	for attempt := 0; attempt < 3; attempt++ {
		if perr == nil || (perr.StatusCode != 401 && perr.StatusCode != 403) {
			break
		}
		tok, err := h.refresher.RefreshCredentials(key.ID)
		if err != nil || tok == "" {
			break
		}
		key.Value = tok
		resp, retryErr := dispatch(key)
		perr = retryErr
		if perr == nil {
			return resp, nil
		}
	}
	return nil, perr
}

// Handle processes chat completion requests (streaming and non-streaming).
func (h *ChatHandler) Handle(ctx *fasthttp.RequestCtx) {
	raw := ctx.PostBody()

	// Bypass check: short-circuit for Claude CLI patterns without calling provider (PAR-ROUTE-034).
	var bodyMap map[string]any
	if err := json.Unmarshal(raw, &bodyMap); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "invalid JSON body", nil)
		return
	}
	model, _ := bodyMap["model"].(string)
	stream := false
	if s, ok := bodyMap["stream"].(bool); ok {
		stream = s
	}
	userAgent := string(ctx.Request.Header.UserAgent())
	if chunks, bypassed, err := translation.HandleBypassRequest(bodyMap, model, userAgent, false, nil); bypassed {
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, "server_error", "bypass error", nil)
			return
		}
		writeBypassResponse(ctx, chunks, stream)
		return
	}

	var req schemas.ChatRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "invalid JSON body", nil)
		return
	}

	translation.PreprocessChatRequest(&req)

	provider, key, err := h.router.ResolveForModel(&req)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", err.Error(), nil)
		return
	}

	// Stream decision (PAR-ROUTE-043): provider-required streaming, deepseek-tui,
	// and Accept: application/json all override the client's stream field.
	useStream := req.Stream
	if sr, ok := provider.(interface{ RequiresStreaming() bool }); ok && sr.RequiresStreaming() {
		useStream = true
	}
	if strings.Contains(userAgent, "deepseek-tui") && !req.Stream {
		useStream = false
	}
	accept := string(ctx.Request.Header.Peek("Accept"))
	if strings.Contains(accept, "application/json") && !strings.Contains(accept, "text/event-stream") && !req.Stream {
		useStream = false
	}

	// Thinking config override (PAR-ROUTE-042).
	if tm, ok := provider.(interface{ ThinkingMode() string }); ok {
		applyThinkingOverride(&req, tm.ThinkingMode())
	}

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}

	if useStream {
		ctx.SetContentTypeBytes([]byte("text/event-stream"))
		ctx.Response.Header.Set("Cache-Control", "no-cache")
		ctx.Response.Header.Set("Connection", "keep-alive")

		ch, perr := provider.ChatCompletionStream(gatewayCtx, nil, key, &req)
		// Refresh-retry: up to 3 refresh+dispatch cycles on 401/403 (PAR-ROUTE-023).
		if perr != nil && (perr.StatusCode == 401 || perr.StatusCode == 403) && h.refresher != nil {
			for attempt := 0; attempt < 3; attempt++ {
				if perr == nil || (perr.StatusCode != 401 && perr.StatusCode != 403) {
					break
				}
				tok, err := h.refresher.RefreshCredentials(key.ID)
				if err != nil || tok == "" {
					break
				}
				key.Value = tok
				ch, perr = provider.ChatCompletionStream(gatewayCtx, nil, key, &req)
			}
		}
		if perr != nil {
			writeError(ctx, fasthttp.StatusBadGateway, perr.Type, perr.Message, perr.Code)
			return
		}

		streamCtx, cancel := withRequestCancel(ctx)
		defer cancel()
		if err := writeSSEStream(streamCtx, ctx, ch); err != nil {
			log.Printf("chat stream error: %v", err)
		}
		return
	}

	resp, perr := provider.ChatCompletion(gatewayCtx, key, &req)
	// Refresh-retry: up to 3 refresh+dispatch cycles on 401/403 (PAR-ROUTE-023).
	if perr != nil && (perr.StatusCode == 401 || perr.StatusCode == 403) && h.refresher != nil {
		resp, perr = h.retryWithRefresh(func(k schemas.Key) (*schemas.ChatResponse, *schemas.ProviderError) {
			return provider.ChatCompletion(gatewayCtx, k, &req)
		}, key, perr)
	}
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
