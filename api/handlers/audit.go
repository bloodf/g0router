package handlers

import (
	"log"
	"time"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// AuditStore is the read access the audit handler needs.
type AuditStore interface {
	ListAudit(filter store.AuditFilter) ([]store.AuditEntry, int, error)
}

type auditListResponse struct {
	Object string             `json:"object"`
	Data   []auditLogResponse `json:"data"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
	Total  int                `json:"total"`
}

type auditLogResponse struct {
	ID            int64  `json:"id"`
	Timestamp     string `json:"timestamp"`
	ActorAPIKeyID string `json:"actor_api_key_id"`
	Action        string `json:"action"`
	Target        string `json:"target"`
	Details       string `json:"details"`
}

// Audit serves GET /api/audit: a paginated, newest-first list of admin audit
// entries with optional action and actor filters.
func Audit(ctx *fasthttp.RequestCtx, auditStore AuditStore) {
	if auditStore == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "audit store unavailable")
		return
	}
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}

	args := ctx.QueryArgs()
	filter := store.AuditFilter{
		Action: queryString(args, "action"),
		Actor:  queryString(args, "actor"),
	}
	limit, err := parseNonNegativeIntArg(args, "limit")
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}
	offset, err := parseNonNegativeIntArg(args, "offset")
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}
	filter.Limit = limit
	filter.Offset = offset

	entries, total, err := auditStore.ListAudit(filter)
	if err != nil {
		log.Printf("list audit: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to get audit log")
		return
	}

	writeJSON(ctx, fasthttp.StatusOK, auditListResponse{
		Object: "list",
		Data:   auditLogResponses(entries),
		Limit:  filter.Limit,
		Offset: filter.Offset,
		Total:  total,
	})
}

func auditLogResponses(entries []store.AuditEntry) []auditLogResponse {
	responses := make([]auditLogResponse, 0, len(entries))
	for _, entry := range entries {
		responses = append(responses, auditLogResponse{
			ID:            entry.ID,
			Timestamp:     entry.Timestamp.Format(time.RFC3339),
			ActorAPIKeyID: entry.ActorAPIKeyID,
			Action:        entry.Action,
			Target:        entry.Target,
			Details:       entry.Details,
		})
	}
	return responses
}
