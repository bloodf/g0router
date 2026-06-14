package admin

import (
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

// MitmStatus handles GET /api/mitm/status.
func (h *Handlers) MitmStatus(ctx *fasthttp.RequestCtx) {
	writeError(ctx, fasthttp.StatusNotImplemented, "not implemented")
}

// MitmToggle handles POST /api/mitm/toggle.
func (h *Handlers) MitmToggle(ctx *fasthttp.RequestCtx) {
	writeError(ctx, fasthttp.StatusNotImplemented, "not implemented")
}

// MitmCACert handles GET /api/mitm/ca-cert (raw PEM, NOT {data}).
func (h *Handlers) MitmCACert(ctx *fasthttp.RequestCtx) {
	writeError(ctx, fasthttp.StatusNotImplemented, "not implemented")
}

// MitmToolToggle handles POST /api/mitm/tools/{id}.
func (h *Handlers) MitmToolToggle(ctx *fasthttp.RequestCtx) {
	writeError(ctx, fasthttp.StatusNotImplemented, "not implemented")
}
