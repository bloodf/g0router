package api

import (
	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
)

// InputTokensHandler handles POST /v1/responses/input_tokens.
//
// TDD scaffold — behavior implemented in STEP(b).
type InputTokensHandler struct {
	router         modelResolver
	registry       *translation.Registry
	usageRecorder  UsageRecorder
	pendingTracker PendingTracker
	detailCapture  DetailCapture
	vkGate         *VKGate
	pinnedResolver VKPinnedKeyResolver
}

// NewInputTokensHandler creates a /v1/responses/input_tokens handler.
func NewInputTokensHandler(router *inference.Router) *InputTokensHandler {
	return &InputTokensHandler{router: router, registry: translation.NewRegistry()}
}

func (h *InputTokensHandler) SetUsageRecorder(r UsageRecorder)        { h.usageRecorder = r }
func (h *InputTokensHandler) SetPendingTracker(t PendingTracker)      { h.pendingTracker = t }
func (h *InputTokensHandler) SetDetailCapture(d DetailCapture)        { h.detailCapture = d }
func (h *InputTokensHandler) SetVKGate(g *VKGate)                     { h.vkGate = g }
func (h *InputTokensHandler) SetVKPinnedResolver(r VKPinnedKeyResolver) { h.pinnedResolver = r }

// Handle is a TDD scaffold; STEP(b) implements the real behavior.
func (h *InputTokensHandler) Handle(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusNotImplemented)
}
