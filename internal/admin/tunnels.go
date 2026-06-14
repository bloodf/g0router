package admin

import (
	"encoding/json"
	"errors"

	"github.com/bloodf/g0router/internal/platform/tunnel"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// tunnelDTO is the canonical snake_case tunnel shape (the 4-field UI Tunnel
// type). It NEVER carries the cleartext token or its ciphertext.
type tunnelDTO struct {
	Type      string `json:"type"`
	IsEnabled bool   `json:"is_enabled"`
	URL       string `json:"url"`
	Status    string `json:"status"`
}

func toTunnelDTO(t store.Tunnel) tunnelDTO {
	return tunnelDTO{
		Type:      t.Type,
		IsEnabled: t.IsEnabled,
		URL:       t.URL,
		Status:    t.Status,
	}
}

type tunnelEnableRequest struct {
	Token string `json:"token"`
	Mode  string `json:"mode"`
}

// ListTunnels handles GET /api/tunnels. It always returns the two known tunnel
// types (cloudflare, tailscale) as a bare array under {data}, mirroring the UI
// mock.
func (h *Handlers) ListTunnels(ctx *fasthttp.RequestCtx) {
	tunnels, err := h.tunnels.List()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list tunnels")
		return
	}
	out := make([]tunnelDTO, 0, len(tunnels))
	for _, t := range tunnels {
		out = append(out, toTunnelDTO(t))
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// TunnelHealth handles GET /api/tunnels/health → {data:{healthy:bool}}.
func (h *Handlers) TunnelHealth(ctx *fasthttp.RequestCtx) {
	healthy, err := h.tunnels.Health()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "tunnel health")
		return
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{"healthy": healthy})
}

// EnableTunnel handles POST /api/tunnels/{type}. The body may carry a token
// (cloudflared named tunnel) and/or mode; both are optional. It returns the
// enabled tunnel DTO. The token is never echoed.
func (h *Handlers) EnableTunnel(ctx *fasthttp.RequestCtx) {
	typ, ok := pathID(ctx.UserValue("type"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	var req tunnelEnableRequest
	if body := ctx.PostBody(); len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
			return
		}
	}

	tn, err := h.tunnels.Enable(typ, req.Token, req.Mode)
	if errors.Is(err, tunnel.ErrUnknownType) {
		writeError(ctx, fasthttp.StatusBadRequest, "unknown tunnel type")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "enable tunnel")
		return
	}
	h.recordAudit(ctx, "tunnel.enable", typ, "Enabled "+typ+" tunnel")
	writeData(ctx, fasthttp.StatusOK, toTunnelDTO(tn))
}

// DisableTunnel handles DELETE /api/tunnels/{type}. It stops the tunnel and
// returns the disabled tunnel DTO. Idempotent.
func (h *Handlers) DisableTunnel(ctx *fasthttp.RequestCtx) {
	typ, ok := pathID(ctx.UserValue("type"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	tn, err := h.tunnels.Disable(typ)
	if errors.Is(err, tunnel.ErrUnknownType) {
		writeError(ctx, fasthttp.StatusBadRequest, "unknown tunnel type")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "disable tunnel")
		return
	}
	h.recordAudit(ctx, "tunnel.disable", typ, "Disabled "+typ+" tunnel")
	writeData(ctx, fasthttp.StatusOK, toTunnelDTO(tn))
}
