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
			raw, _ := json.Marshal(body)
			if err := json.Unmarshal(raw, &req); err == nil {
				nativeSkip = true
			}
		}
	}

	if !nativeSkip {
		translated, err := h.registry.TranslateRequest(translation.FormatClaude, translation.FormatOpenAI, model, body, stream, nil)
		if err != nil {
			h.recordError(ctx, model, key.Provider, key.ID, &schemas.ProviderError{StatusCode: 400, Message: err.Error(), Type: "invalid_request_error"})
			writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", err.Error(), nil)
			return
		}

		b, err := json.Marshal(translated)
		if err != nil {
			h.recordError(ctx, model, key.Provider, key.ID, &schemas.ProviderError{StatusCode: 500, Message: "marshal failure", Type: "internal"})
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetContentTypeBytes([]byte("text/plain"))
			ctx.SetBodyString("internal error")
			return
		}

		if err := json.Unmarshal(b, &req); err != nil {
			h.recordError(ctx, model, key.Provider, key.ID, &schemas.ProviderError{StatusCode: 400, Message: err.Error(), Type: "invalid_request_error"})
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
			h.recordError(ctx, model, key.Provider, key.ID, perr)
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
		h.recordStream(ctx, model, key.Provider, key.ID, summary, sErr)
		return
	}

	resp, perr := provider.ChatCompletion(gatewayCtx, key, &req)
	if perr != nil {
		h.recordError(ctx, model, key.Provider, key.ID, perr)
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
		h.recordError(ctx, model, key.Provider, key.ID, &schemas.ProviderError{StatusCode: 500, Message: "marshal failure", Type: "internal"})
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentTypeBytes([]byte("text/plain"))
		ctx.SetBodyString("internal error")
		return
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes([]byte("application/json"))
	ctx.SetBody(out)

	h.recordNonStream(ctx, model, key.Provider, key.ID, resp)
}

// recordError, recordNonStream, recordStream handle the usage glue for
// non-bypass paths (PAR-ROUTE-054, PAR-USAGE-018/026).
func (h *MessagesHandler) recordError(ctx *fasthttp.RequestCtx, model, provider, connID string, perr *schemas.ProviderError) {
	if h.pendingTracker != nil {
		h.pendingTracker.End(model, provider, connID, true)
	}
	statusCode := perr.StatusCode
	if statusCode == 0 {
		statusCode = 502
	}
	statusLabel := fmt.Sprintf("%d", statusCode)
	if h.usageRecorder != nil {
		_ = h.usageRecorder.Record(&UsageEntry{
			Provider:     provider,
			Model:        model,
			ConnectionID: connID,
			Endpoint:     "/v1/messages",
			Status:       "error",
			Tokens:       map[string]int64{},
		})
	}
	if h.detailCapture != nil {
		_ = h.detailCapture.Save(RequestDetailCapture{
			Provider:     provider,
			Model:        model,
			ConnectionID: connID,
			Status:       "error",
			Response:     map[string]any{"error": map[string]any{"message": perr.Message, "status": statusLabel}},
		})
	}
}

func (h *MessagesHandler) recordNonStream(ctx *fasthttp.RequestCtx, model, provider, connID string, resp *schemas.ChatResponse) {
	if h.pendingTracker != nil {
		h.pendingTracker.End(model, provider, connID, false)
	}
	entry := &UsageEntry{
		Provider:     provider,
		Model:        model,
		ConnectionID: connID,
		Endpoint:     "/v1/messages",
		Status:       "ok",
	}
	if resp != nil && resp.Usage != nil {
		entry.PromptTokens = int64(resp.Usage.PromptTokens)
		entry.CompletionTokens = int64(resp.Usage.CompletionTokens)
		entry.Tokens = map[string]int64{
			"prompt_tokens":     entry.PromptTokens,
			"completion_tokens": entry.CompletionTokens,
		}
	}
	if h.usageRecorder != nil {
		_ = h.usageRecorder.Record(entry)
	}
	if h.detailCapture != nil {
		_ = h.detailCapture.Save(RequestDetailCapture{
			Provider:     provider,
			Model:        model,
			ConnectionID: connID,
			Status:       "success",
			Tokens:       entry.Tokens,
			Response:     resp,
		})
	}
}

func (h *MessagesHandler) recordStream(ctx *fasthttp.RequestCtx, model, provider, connID string, summary translation.StreamSummary, sErr error) {
	isError := sErr != nil
	if h.pendingTracker != nil {
		h.pendingTracker.End(model, provider, connID, isError)
	}
	status := "ok"
	if isError {
		status = "error"
	}
	entry := &UsageEntry{
		Provider:     provider,
		Model:        model,
		ConnectionID: connID,
		Endpoint:     "/v1/messages",
		Status:       status,
	}
	if summary.Usage != nil {
		entry.PromptTokens = int64(extractInt(summary.Usage, "prompt_tokens"))
		entry.CompletionTokens = int64(extractInt(summary.Usage, "completion_tokens"))
		entry.Tokens = map[string]int64{}
		if v, ok := summary.Usage["prompt_tokens"]; ok {
			entry.Tokens["prompt_tokens"] = int64(toFloat(v))
		}
		if v, ok := summary.Usage["completion_tokens"]; ok {
			entry.Tokens["completion_tokens"] = int64(toFloat(v))
		}
	}
	if h.usageRecorder != nil {
		_ = h.usageRecorder.Record(entry)
	}
	if h.detailCapture != nil {
		capture := RequestDetailCapture{
			Provider:     provider,
			Model:        model,
			ConnectionID: connID,
			Status:       status,
			Tokens:       entry.Tokens,
		}
		if isError {
			capture.Response = map[string]any{"error": sErr.Error()}
		}
		_ = h.detailCapture.Save(capture)
	}
}
