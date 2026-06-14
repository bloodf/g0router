package admin

import (
	"errors"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// mitmToolDTO is the canonical snake_case MITM tool shape (the 5-field UI
// MitmTool type). It NEVER carries any CA key material.
type mitmToolDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	DNSOverride string `json:"dns_override"`
	Status      string `json:"status"`
}

func toMitmToolDTO(t store.MitmTool) mitmToolDTO {
	return mitmToolDTO{
		ID:          t.ID,
		Name:        t.Name,
		Enabled:     t.Enabled,
		DNSOverride: t.DNSOverride,
		Status:      t.Status,
	}
}

// MitmStatus handles GET /api/mitm/status → {data:{enabled, tools:[mitmToolDTO]}}.
// It always surfaces the two seeded tools.
func (h *Handlers) MitmStatus(ctx *fasthttp.RequestCtx) {
	enabled, tools, err := h.mitm.Status()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "mitm status")
		return
	}
	out := make([]mitmToolDTO, 0, len(tools))
	for _, t := range tools {
		out = append(out, toMitmToolDTO(t))
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{"enabled": enabled, "tools": out})
}

// MitmToggle handles POST /api/mitm/toggle → {data:{enabled}}. It flips the
// global flag, records an audit entry, and best-effort starts/stops the listener.
func (h *Handlers) MitmToggle(ctx *fasthttp.RequestCtx) {
	enabled, err := h.mitm.Toggle()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "mitm toggle")
		return
	}
	action := "disabled"
	if enabled {
		action = "enabled"
	}
	h.recordAudit(ctx, "mitm.toggle", "mitm", "MITM "+action)
	writeData(ctx, fasthttp.StatusOK, map[string]any{"enabled": enabled})
}

// MitmCACert handles GET /api/mitm/ca-cert. It is the raw-PEM EXCEPTION: it writes
// the PUBLIC root CA certificate as application/x-pem-file, NOT a {data} envelope,
// mirroring the dashboard's plain-fetch download. The CA private key is NEVER
// served.
func (h *Handlers) MitmCACert(ctx *fasthttp.RequestCtx) {
	pemBytes, err := h.mitm.CACertPEM()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load CA certificate")
		return
	}
	ctx.SetContentType("application/x-pem-file")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(pemBytes)
}

// MitmToolToggle handles POST /api/mitm/tools/{id} → {data:mitmToolDTO}. It flips
// the tool's enabled flag (deriving status), records an audit entry, and returns
// the updated tool. 404 on an unknown id.
func (h *Handlers) MitmToolToggle(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	tool, err := h.mitm.ToggleTool(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "tool not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "toggle mitm tool")
		return
	}
	h.recordAudit(ctx, "mitm.tool.toggle", id, "Toggled MITM tool "+id)
	writeData(ctx, fasthttp.StatusOK, toMitmToolDTO(tool))
}
