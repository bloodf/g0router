package api

import (
	"github.com/bloodf/g0router/internal/inference"
	"github.com/valyala/fasthttp"
)

// AudioHandler handles POST /v1/audio/speech and /v1/audio/transcriptions.
type AudioHandler struct {
	router         completionsResolver
	usageRecorder  UsageRecorder
	pendingTracker PendingTracker
	detailCapture  DetailCapture
	vkGate         *VKGate
	pinnedResolver VKPinnedKeyResolver
}

// NewAudioHandler creates an audio handler.
func NewAudioHandler(router *inference.Router) *AudioHandler {
	return &AudioHandler{router: router}
}

// SetUsageRecorder wires a usage recorder.
func (h *AudioHandler) SetUsageRecorder(r UsageRecorder) { h.usageRecorder = r }

// SetPendingTracker wires a pending tracker.
func (h *AudioHandler) SetPendingTracker(t PendingTracker) { h.pendingTracker = t }

// SetDetailCapture wires a detail capture.
func (h *AudioHandler) SetDetailCapture(d DetailCapture) { h.detailCapture = d }

// SetVKGate wires a virtual-key gate.
func (h *AudioHandler) SetVKGate(g *VKGate) { h.vkGate = g }

// SetVKPinnedResolver wires the virtual-key pinned-key resolver.
func (h *AudioHandler) SetVKPinnedResolver(r VKPinnedKeyResolver) { h.pinnedResolver = r }

// Speech is a scaffold; implemented in STEP(b).
func (h *AudioHandler) Speech(ctx *fasthttp.RequestCtx) {}

// Transcription is a scaffold; implemented in STEP(b).
func (h *AudioHandler) Transcription(ctx *fasthttp.RequestCtx) {}
