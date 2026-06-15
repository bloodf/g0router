package api

import (
	"github.com/bloodf/g0router/internal/inference"
	"github.com/valyala/fasthttp"
)

// ImagesHandler handles POST /v1/images/{generations,edits,variations}.
type ImagesHandler struct {
	router         completionsResolver
	usageRecorder  UsageRecorder
	pendingTracker PendingTracker
	detailCapture  DetailCapture
	vkGate         *VKGate
	pinnedResolver VKPinnedKeyResolver
}

// NewImagesHandler creates an images handler.
func NewImagesHandler(router *inference.Router) *ImagesHandler {
	return &ImagesHandler{router: router}
}

// SetUsageRecorder wires a usage recorder.
func (h *ImagesHandler) SetUsageRecorder(r UsageRecorder) { h.usageRecorder = r }

// SetPendingTracker wires a pending tracker.
func (h *ImagesHandler) SetPendingTracker(t PendingTracker) { h.pendingTracker = t }

// SetDetailCapture wires a detail capture.
func (h *ImagesHandler) SetDetailCapture(d DetailCapture) { h.detailCapture = d }

// SetVKGate wires a virtual-key gate.
func (h *ImagesHandler) SetVKGate(g *VKGate) { h.vkGate = g }

// SetVKPinnedResolver wires the virtual-key pinned-key resolver.
func (h *ImagesHandler) SetVKPinnedResolver(r VKPinnedKeyResolver) { h.pinnedResolver = r }

// Generations is a scaffold; implemented in STEP(b).
func (h *ImagesHandler) Generations(ctx *fasthttp.RequestCtx) {}

// Edits is a scaffold; implemented in STEP(b).
func (h *ImagesHandler) Edits(ctx *fasthttp.RequestCtx) {}

// Variations is a scaffold; implemented in STEP(b).
func (h *ImagesHandler) Variations(ctx *fasthttp.RequestCtx) {}
