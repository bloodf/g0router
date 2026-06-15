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

// SemanticCache is the api-local seam for the exact-key response cache
// (bf-core-2). It is wired via SetSemanticCache so the api package does not
// import internal/semcache or internal/store directly (mirrors modelResolver /
// ComboDispatcher). Enabled() reports the [semantic_cache] feature flag;
// Lookup/Store implement read-through/write-through over a deterministic
// (model, prompt) key. The semantic-similarity half is deferred (D2): there is
// no embedder here.
type SemanticCache interface {
	Enabled() bool
	Lookup(ctx context.Context, model, prompt string) ([]byte, bool, error)
	Store(ctx context.Context, model, prompt string, response []byte) error
}

// ChatHandler handles POST /v1/chat/completions.
type ChatHandler struct {
	router           modelResolver
	refresher        CredentialRefresher
	comboDispatcher  ComboDispatcher
	usageRecorder    UsageRecorder
	pendingTracker   PendingTracker
	detailCapture    DetailCapture
	vkGate           *VKGate
	pinnedResolver   VKPinnedKeyResolver
	semanticCache    SemanticCache
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

// SetVKGate wires a virtual-key gate for x-g0-vk header enforcement (PAR-ROUTE-030).
func (h *ChatHandler) SetVKGate(g *VKGate) { h.vkGate = g }

// PinnedResolver setter wires the resolver for virtual-key KeyID pinning
// (PAR-ROUTE-030).
func (h *ChatHandler) SetVKPinnedResolver(r VKPinnedKeyResolver) { h.pinnedResolver = r }

// SetSemanticCache wires the exact-key response cache (bf-core-2). When unset
// (nil) or with the [semantic_cache] flag off, the chat path is byte-identical
// to pre-bf-core-2 (clean no-op). Read-through/write-through runs ONLY in the
// non-streaming branch (D6).
func (h *ChatHandler) SetSemanticCache(c SemanticCache) { h.semanticCache = c }

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

	// x-g0-vk virtual-key gate (PAR-ROUTE-030): after model resolution, before
	// dispatch. AllowVK is called unconditionally so the injected mandatory
	// predicate (bf-gov-4, D2/Option-A) can reject an absent VK when the
	// vk_mandatory flag is ON. AllowVK("") short-circuits at its own key==""
	// branch — when mandatory OFF it returns (true,0,"",nil) and the blocks
	// below are no-ops; when mandatory ON it returns (false,401,...) and we
	// reject. Byte- and perf-identical to pre-bf-gov-4 when the flag is OFF.
	vkHeader := string(ctx.Request.Header.Peek("x-g0-vk"))
	ok, status, reason, keyIDs := h.vkGate.AllowVK(vkHeader, req.Model, key.Provider)
	if !ok {
		errType := "invalid_request_error"
		if status == 429 {
			errType = "rate_limit_exceeded"
		}
		// Surface the typed governance Decision as error.code (bf-gov-3, D8):
		// a token/request/budget/rate denial carries its snake_case code.
		writeError(ctx, status, errType, reason, DecisionCodeForReason(reason))
		return
	}
	if len(keyIDs) > 0 && h.pinnedResolver != nil {
		if connID, credential, ok := h.pinnedResolver.ResolvePinned(key.Provider, req.Model, keyIDs); ok {
			key.ID = connID
			key.Value = credential
		}
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

	g := h.recordGlue()
	g.apiKey = vkHeader
	headers := requestHeadersFromCtx(ctx)

	if useStream {
		// Open the provider stream BEFORE setting SSE headers so a stream-open
		// *ProviderError returns an application/json error (with the provider's
		// real status), not a text/event-stream framing mismatch (PAR-BF-OAI-201).
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
			g.recordError("/v1/chat/completions", req.Model, key.Provider, key.ID, raw, headers, perr)
			status := perr.StatusCode
			if status == 0 {
				status = fasthttp.StatusBadGateway
			}
			writeError(ctx, status, perr.Type, perr.Message, perr.Code)
			return
		}

		ctx.SetContentTypeBytes([]byte("text/event-stream"))
		ctx.Response.Header.Set("Cache-Control", "no-cache")
		ctx.Response.Header.Set("Connection", "keep-alive")

		streamCtx, cancel := withRequestCancel(ctx)
		defer cancel()
		summary, sErr := writeSSEStreamWithSource(streamCtx, ctx, ch, src)
		if sErr != nil {
			log.Printf("chat stream error: %v", sErr)
		}
		g.recordStream("/v1/chat/completions", req.Model, key.Provider, key.ID, raw, headers, summary, sErr)
		return
	}

	// Exact-key semantic-cache read-through (bf-core-2, D4/D6): non-stream branch
	// only, positioned where a guardrail check would sit (after the VK gate,
	// before dispatch). On a hit the cached bytes are returned and the provider
	// is short-circuited; flag-off or nil-cache is a clean no-op.
	cachePrompt := ""
	cacheActive := h.semanticCache != nil && h.semanticCache.Enabled()
	if cacheActive {
		cachePrompt = semanticCachePrompt(&req)
		if cached, hit, err := h.semanticCache.Lookup(context.Background(), req.Model, cachePrompt); err == nil && hit {
			ctx.SetStatusCode(fasthttp.StatusOK)
			ctx.SetContentTypeBytes([]byte("application/json"))
			ctx.SetBody(cached)
			return
		}
	}

	resp, perr := provider.ChatCompletion(gatewayCtx, key, &req)
	// Refresh-retry: up to 3 refresh+dispatch cycles on 401/403 (PAR-ROUTE-023).
	if perr != nil && (perr.StatusCode == 401 || perr.StatusCode == 403) && h.refresher != nil {
		resp, perr = h.retryWithRefresh(func(k schemas.Key) (*schemas.ChatResponse, *schemas.ProviderError) {
			return provider.ChatCompletion(gatewayCtx, k, &req)
		}, key, perr)
	}
	if perr != nil {
		g.recordError("/v1/chat/completions", req.Model, key.Provider, key.ID, raw, headers, perr)
		status := perr.StatusCode
		if status == 0 {
			status = fasthttp.StatusBadGateway
		}
		writeError(ctx, status, perr.Type, perr.Message, perr.Code)
		return
	}

	b, err := jsonMarshal(resp)
	if err != nil {
		g.recordError("/v1/chat/completions", req.Model, key.Provider, key.ID, raw, headers, &schemas.ProviderError{StatusCode: 500, Message: "marshal failure", Type: "internal"})
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentTypeBytes([]byte("text/plain"))
		ctx.SetBodyString("internal error")
		return
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes([]byte("application/json"))
	ctx.SetBody(b)

	// Exact-key write-through on a miss (bf-core-2): persist the marshaled
	// response under the (model, prompt) key so an identical follow-up
	// short-circuits the provider. Best-effort — a cache write failure must not
	// fail the request that already succeeded.
	if cacheActive {
		_ = h.semanticCache.Store(context.Background(), req.Model, cachePrompt, b)
	}

	var pt, ct int64
	if resp != nil && resp.Usage != nil {
		pt = int64(resp.Usage.PromptTokens)
		ct = int64(resp.Usage.CompletionTokens)
	}
	g.recordNonStream("/v1/chat/completions", req.Model, key.Provider, key.ID, raw, headers, pt, ct, resp)
}

// semanticCachePrompt builds a deterministic prompt string for the exact-key
// cache from the request messages (bf-core-2 D1). Go marshals struct fields in
// declaration order and slices in order, so identical message lists yield an
// identical string; the semcache key layer then normalizes + hashes it. A
// marshal failure (unexpected for a request that already unmarshaled) yields ""
// — a stable but low-entropy key, which the caller still gates on the flag.
func semanticCachePrompt(req *schemas.ChatRequest) string {
	b, err := json.Marshal(req.Messages)
	if err != nil {
		return ""
	}
	return string(b)
}

// recordGlue assembles the shared usage-recording dependencies for this handler.
func (h *ChatHandler) recordGlue() recordGlue {
	return recordGlue{recorder: h.usageRecorder, tracker: h.pendingTracker, detail: h.detailCapture}
}


