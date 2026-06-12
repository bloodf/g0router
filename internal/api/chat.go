package api

import (
	"context"
	"encoding/json"
	"errors"
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
	_, err := translation.ProcessPassthroughStream(ctx, w, ch, nil)
	return err
}

// writeSSEStreamWithSource is the usage-aware variant of writeSSEStream.
// src threads the request body and client format into the stream processor
// so the finish-chunk estimation machinery can fire (PAR-TRANS-046 usage
// clause). The returned summary exposes the accumulated content length and
// the final usage payload for the caller to record.
func writeSSEStreamWithSource(ctx context.Context, w streamWriter, ch chan *schemas.StreamChunk, src *translation.EstimateSource) (translation.StreamSummary, error) {
	return translation.ProcessPassthroughStream(ctx, w, ch, src)
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

// ComboDispatcher resolves combo names and executes their ordered model chain.
// The api layer uses this interface so it does not need to import store types.
type ComboDispatcher interface {
	IsCombo(name string) bool
	ExecuteCombo(name string, fn func(model, connID, credential string) (inference.Verdict, error)) error
}

// ChatHandler handles POST /v1/chat/completions.
type ChatHandler struct {
	router           modelResolver
	refresher        CredentialRefresher
	comboDispatcher  ComboDispatcher
	usageRecorder    UsageRecorder
	pendingTracker   PendingTracker
	detailCapture    DetailCapture
}

// NewChatHandler creates a chat completion handler.
func NewChatHandler(router *inference.Router) *ChatHandler {
	return &ChatHandler{router: router}
}

// SetCredentialRefresher wires an OAuth credential refresher for 401/403 retry.
func (h *ChatHandler) SetCredentialRefresher(cr CredentialRefresher) {
	h.refresher = cr
}

// SetComboDispatcher wires a combo dispatcher for combo model chains.
func (h *ChatHandler) SetComboDispatcher(cd ComboDispatcher) {
	h.comboDispatcher = cd
}

// SetUsageRecorder wires a consumer for request_log entries (PAR-ROUTE-054
// attribution; consumed from w5-b).
func (h *ChatHandler) SetUsageRecorder(r UsageRecorder) { h.usageRecorder = r }

// SetPendingTracker wires a consumer for in-flight request accounting
// (PAR-USAGE-018 wiring half).
func (h *ChatHandler) SetPendingTracker(t PendingTracker) { h.pendingTracker = t }

// SetDetailCapture wires a consumer for full request detail capture
// (PAR-USAGE-026 production call-sites).
func (h *ChatHandler) SetDetailCapture(d DetailCapture) { h.detailCapture = d }

// classifyProviderError maps a provider error to the verdict used by the
// account-fallback and combo engines. It reuses the w4-b classifier.
func classifyProviderError(perr *schemas.ProviderError) inference.Verdict {
	class := inference.Classify(perr.StatusCode, []byte(perr.Message))
	return verdictFromClass(class.Class)
}

func verdictFromClass(class inference.ErrorClass) inference.Verdict {
	switch class {
	case inference.ClassRateLimit:
		return inference.VerdictRateLimit
	case inference.ClassAuthError:
		return inference.VerdictAuth
	case inference.ClassTransient:
		return inference.VerdictTransient
	case inference.ClassPermanent, inference.ClassUnsupportedParam:
		return inference.VerdictPermanent
	default:
		return inference.VerdictUnknown
	}
}

// handleCombo runs the combo chain for the requested model. It falls back through
// models on failure and streams the first model that opens a stream channel.
func (h *ChatHandler) handleCombo(ctx *fasthttp.RequestCtx, req *schemas.ChatRequest, userAgent, accept string) {
	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}

	err := h.comboDispatcher.ExecuteCombo(req.Model, func(model, connID, credential string) (inference.Verdict, error) {
		modelReq := *req
		modelReq.Model = model

		provider, key, err := h.router.ResolveForModel(&modelReq)
		if err != nil {
			return inference.VerdictPermanent, err
		}
		key.ID = connID
		key.Value = credential

		// Stream decision mirrors the single-model path (PAR-ROUTE-043).
		useStream := modelReq.Stream
		if sr, ok := provider.(interface{ RequiresStreaming() bool }); ok && sr.RequiresStreaming() {
			useStream = true
		}
		if strings.Contains(userAgent, "deepseek-tui") && !modelReq.Stream {
			useStream = false
		}
		if strings.Contains(accept, "application/json") && !strings.Contains(accept, "text/event-stream") && !modelReq.Stream {
			useStream = false
		}

		// Thinking config override (PAR-ROUTE-042).
		if tm, ok := provider.(interface{ ThinkingMode() string }); ok {
			applyThinkingOverride(&modelReq, tm.ThinkingMode())
		}

		if useStream {
			ctx.SetContentTypeBytes([]byte("text/event-stream"))
			ctx.Response.Header.Set("Cache-Control", "no-cache")
			ctx.Response.Header.Set("Connection", "keep-alive")

			ch, perr := provider.ChatCompletionStream(gatewayCtx, nil, key, &modelReq)
			if perr != nil {
				return classifyProviderError(perr), perr
			}
			streamCtx, cancel := withRequestCancel(ctx)
			defer cancel()
			if err := writeSSEStream(streamCtx, ctx, ch); err != nil {
				// Channel opened; consume the error and stop fallback (ref parity).
				return inference.VerdictUnknown, nil
			}
			return inference.VerdictUnknown, nil
		}

		resp, perr := provider.ChatCompletion(gatewayCtx, key, &modelReq)
		if perr != nil {
			return classifyProviderError(perr), perr
		}
		b, err := jsonMarshal(resp)
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetContentTypeBytes([]byte("text/plain"))
			ctx.SetBodyString("internal error")
			return inference.VerdictUnknown, nil
		}
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetContentTypeBytes([]byte("application/json"))
		ctx.SetBody(b)
		return inference.VerdictUnknown, nil
	})

	if err != nil {
		var perr *schemas.ProviderError
		if errors.As(err, &perr) {
			status := perr.StatusCode
			if status == 0 {
				status = fasthttp.StatusBadGateway
			}
			writeError(ctx, status, perr.Type, perr.Message, perr.Code)
			return
		}
		writeError(ctx, fasthttp.StatusBadGateway, "server_error", err.Error(), nil)
	}
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

	accept := string(ctx.Request.Header.Peek("Accept"))

	// Combo dispatch: if the requested model is a combo, run its model chain.
	if h.comboDispatcher != nil && h.comboDispatcher.IsCombo(req.Model) {
		h.handleCombo(ctx, &req, userAgent, accept)
		return
	}

	provider, key, err := h.router.ResolveForModel(&req)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", err.Error(), nil)
		return
	}

	// Pending-tracker start (PAR-USAGE-018 wiring half).
	if h.pendingTracker != nil {
		h.pendingTracker.Start(req.Model, key.Provider, key.ID)
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
	if strings.Contains(accept, "application/json") && !strings.Contains(accept, "text/event-stream") && !req.Stream {
		useStream = false
	}

	// Thinking config override (PAR-ROUTE-042).
	if tm, ok := provider.(interface{ ThinkingMode() string }); ok {
		applyThinkingOverride(&req, tm.ThinkingMode())
	}

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}

	// EstimateSource for stream-time usage estimation (PAR-TRANS-046 usage clause).
	src := &translation.EstimateSource{Body: bodyMap, Format: translation.FormatOpenAI}

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
			h.recordError(ctx, req.Model, key.Provider, key.ID, perr)
			writeError(ctx, fasthttp.StatusBadGateway, perr.Type, perr.Message, perr.Code)
			return
		}

		streamCtx, cancel := withRequestCancel(ctx)
		defer cancel()
		summary, sErr := writeSSEStreamWithSource(streamCtx, ctx, ch, src)
		if sErr != nil {
			log.Printf("chat stream error: %v", sErr)
		}
		h.recordStream(ctx, req.Model, key.Provider, key.ID, summary, sErr)
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
		h.recordError(ctx, req.Model, key.Provider, key.ID, perr)
		status := perr.StatusCode
		if status == 0 {
			status = fasthttp.StatusBadGateway
		}
		writeError(ctx, status, perr.Type, perr.Message, perr.Code)
		return
	}

	b, err := jsonMarshal(resp)
	if err != nil {
		h.recordError(ctx, req.Model, key.Provider, key.ID, &schemas.ProviderError{StatusCode: 500, Message: "marshal failure", Type: "internal"})
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentTypeBytes([]byte("text/plain"))
		ctx.SetBodyString("internal error")
		return
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes([]byte("application/json"))
	ctx.SetBody(b)

	h.recordNonStream(ctx, req.Model, key.Provider, key.ID, resp)
}

// recordError terminates the pending request accounting and persists the
// failure as a single request_log row + a request_details row. Called on
// non-stream error paths (PAR-ROUTE-054, PAR-USAGE-018/026).
func (h *ChatHandler) recordError(ctx *fasthttp.RequestCtx, model, provider, connID string, perr *schemas.ProviderError) {
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
			Endpoint:     "/v1/chat/completions",
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

// recordNonStream terminates the pending request accounting and persists the
// success as a single request_log row + a request_details row. Called on
// non-stream success paths (PAR-ROUTE-054, PAR-USAGE-018/026).
func (h *ChatHandler) recordNonStream(ctx *fasthttp.RequestCtx, model, provider, connID string, resp *schemas.ChatResponse) {
	if h.pendingTracker != nil {
		h.pendingTracker.End(model, provider, connID, false)
	}
	entry := &UsageEntry{
		Provider:     provider,
		Model:        model,
		ConnectionID: connID,
		Endpoint:     "/v1/chat/completions",
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

// recordStream terminates the pending request accounting and persists the
// stream's accumulated/estimated usage + a request_details row. Called on
// stream completion (success or error). src is unused here because the
// processor already applied the estimation machinery.
func (h *ChatHandler) recordStream(ctx *fasthttp.RequestCtx, model, provider, connID string, summary translation.StreamSummary, sErr error) {
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
		Endpoint:     "/v1/chat/completions",
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

// extractInt / toFloat / toFloatMapInt64: small helpers for pulling numeric
// fields out of an untyped usage map. Kept local to avoid exposing them
// across the package.
func extractInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		return int(toFloat(v))
	}
	return 0
}

func toFloat(v any) float64 {
	switch x := v.(type) {
	case int:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	case uint:
		return float64(x)
	case uint32:
		return float64(x)
	case uint64:
		return float64(x)
	case float32:
		return float64(x)
	case float64:
		return x
	}
	return 0
}
