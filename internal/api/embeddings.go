package api

import (
	"encoding/json"
	"fmt"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// EmbeddingsHandler handles POST /v1/embeddings.
type EmbeddingsHandler struct {
	router         embeddingsResolver
	usageRecorder  UsageRecorder
	pendingTracker PendingTracker
	detailCapture  DetailCapture
}

// embeddingsResolver is the subset of *inference.Router used by the embeddings
// handler. It exists so tests can inject behavior without going through the
// full router.
type embeddingsResolver interface {
	Resolve(model string) (schemas.Provider, schemas.Key, error)
}

// NewEmbeddingsHandler creates an embeddings handler.
func NewEmbeddingsHandler(router *inference.Router) *EmbeddingsHandler {
	return &EmbeddingsHandler{router: router}
}

// SetUsageRecorder wires a consumer for request_log entries (PAR-ROUTE-054).
func (h *EmbeddingsHandler) SetUsageRecorder(r UsageRecorder) { h.usageRecorder = r }

// SetPendingTracker wires a consumer for in-flight request accounting
// (PAR-USAGE-018 wiring half).
func (h *EmbeddingsHandler) SetPendingTracker(t PendingTracker) { h.pendingTracker = t }

// SetDetailCapture wires a consumer for full request detail capture
// (PAR-USAGE-026 production call-sites).
func (h *EmbeddingsHandler) SetDetailCapture(d DetailCapture) { h.detailCapture = d }

// Handle processes embedding requests.
func (h *EmbeddingsHandler) Handle(ctx *fasthttp.RequestCtx) {
	var req schemas.EmbeddingRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "invalid JSON body", nil)
		return
	}

	provider, key, err := h.router.Resolve(req.Model)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", err.Error(), nil)
		return
	}

	// Pending-tracker start (PAR-USAGE-018 wiring half).
	if h.pendingTracker != nil {
		h.pendingTracker.Start(req.Model, key.Provider, key.ID)
	}

	// Keys are resolved by the router via the wired credential resolver.

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}
	resp, perr := provider.Embedding(gatewayCtx, key, &req)
	if perr != nil {
		h.recordError(req.Model, key.Provider, key.ID, perr)
		status := perr.StatusCode
		if status == 0 {
			status = fasthttp.StatusBadGateway
		}
		writeError(ctx, status, perr.Type, perr.Message, perr.Code)
		return
	}

	b, err := jsonMarshal(resp)
	if err != nil {
		h.recordError(req.Model, key.Provider, key.ID, &schemas.ProviderError{StatusCode: 500, Message: "marshal failure", Type: "internal"})
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentTypeBytes([]byte("text/plain"))
		ctx.SetBodyString("internal error")
		return
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes([]byte("application/json"))
	ctx.SetBody(b)

	h.recordNonStream(req.Model, key.Provider, key.ID, resp)
}

// recordError / recordNonStream handle the usage glue (PAR-ROUTE-054,
// PAR-USAGE-018/026). The Embeddings endpoint is non-stream only.
func (h *EmbeddingsHandler) recordError(model, provider, connID string, perr *schemas.ProviderError) {
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
			Endpoint:     "/v1/embeddings",
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

func (h *EmbeddingsHandler) recordNonStream(model, provider, connID string, resp *schemas.EmbeddingResponse) {
	if h.pendingTracker != nil {
		h.pendingTracker.End(model, provider, connID, false)
	}
	entry := &UsageEntry{
		Provider:     provider,
		Model:        model,
		ConnectionID: connID,
		Endpoint:     "/v1/embeddings",
		Status:       "ok",
	}
	if resp.Usage != nil {
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
