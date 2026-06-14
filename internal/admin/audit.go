package admin

import (
	"log"
	"strconv"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

const (
	auditDefaultLimit = 100
	auditMaxLimit     = 1000
)

type auditDTO struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Actor     string `json:"actor"`
	Action    string `json:"action"`
	Target    string `json:"target"`
	Details   string `json:"details,omitempty"`
}

func toAuditDTO(e store.AuditEntry) auditDTO {
	return auditDTO{
		ID:        e.ID,
		Timestamp: e.Timestamp,
		Actor:     e.Actor,
		Action:    e.Action,
		Target:    e.Target,
		Details:   e.Details,
	}
}

// GetAudit handles GET /api/audit?limit=N.
func (h *Handlers) GetAudit(ctx *fasthttp.RequestCtx) {
	limit := auditDefaultLimit
	if raw := string(ctx.QueryArgs().Peek("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > auditMaxLimit {
		limit = auditMaxLimit
	}

	items, total, err := h.audit.List(limit)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list audit log")
		return
	}
	out := make([]auditDTO, 0, len(items))
	for _, e := range items {
		out = append(out, toAuditDTO(e))
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{"items": out, "total": total})
}

// recordAudit writes a best-effort audit entry for an administrative action.
// The actor is resolved from the authenticated session user; details must be a
// human-readable summary and must never contain secrets. A write failure is
// logged and never propagated to the parent request.
func (h *Handlers) recordAudit(ctx *fasthttp.RequestCtx, action, target, details string) {
	actor := "system"
	if u, ok := ctx.UserValue(userKey).(*store.User); ok && u != nil {
		actor = u.Username
	}
	if err := h.audit.WriteAudit(actor, action, target, details); err != nil {
		log.Printf("audit: write %s/%s failed: %v", action, target, err)
	}
}
