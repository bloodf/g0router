package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type routingRuleDTO struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Priority       int    `json:"priority"`
	CondField      string `json:"cond_field"`
	CondOperator   string `json:"cond_operator"`
	CondValue      string `json:"cond_value"`
	TargetProvider string `json:"target_provider"`
	IsActive       bool   `json:"is_active"`
	CreatedAt      string `json:"created_at"`
}

func toRoutingRuleDTO(r *store.RoutingRule) routingRuleDTO {
	return routingRuleDTO{
		ID:             r.ID,
		Name:           r.Name,
		Priority:       r.Priority,
		CondField:      r.CondField,
		CondOperator:   r.CondOperator,
		CondValue:      r.CondValue,
		TargetProvider: r.TargetProvider,
		IsActive:       r.IsActive,
		CreatedAt:      time.Unix(r.CreatedAt, 0).UTC().Format(time.RFC3339),
	}
}

type routingRuleRequest struct {
	Name           string `json:"name"`
	Priority       int    `json:"priority"`
	CondField      string `json:"cond_field"`
	CondOperator   string `json:"cond_operator"`
	CondValue      string `json:"cond_value"`
	TargetProvider string `json:"target_provider"`
	IsActive       *bool  `json:"is_active"`
}

// ListRoutingRules handles GET /api/routing-rules.
func (h *Handlers) ListRoutingRules(ctx *fasthttp.RequestCtx) {
	rules, err := h.store.ListRoutingRules()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list routing rules")
		return
	}
	out := make([]routingRuleDTO, 0, len(rules))
	for _, r := range rules {
		out = append(out, toRoutingRuleDTO(r))
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// CreateRoutingRule handles POST /api/routing-rules.
func (h *Handlers) CreateRoutingRule(ctx *fasthttp.RequestCtx) {
	var req routingRuleRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "name is required")
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	created, err := h.store.CreateRoutingRule(&store.RoutingRule{
		Name:           req.Name,
		Priority:       req.Priority,
		CondField:      req.CondField,
		CondOperator:   req.CondOperator,
		CondValue:      req.CondValue,
		TargetProvider: req.TargetProvider,
		IsActive:       isActive,
	})
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "create routing rule")
		return
	}
	h.recordAudit(ctx, "create_routing_rule", created.ID, fmt.Sprintf("Created routing rule %s", created.Name))
	writeData(ctx, fasthttp.StatusCreated, toRoutingRuleDTO(created))
}

// GetRoutingRule handles GET /api/routing-rules/{id}.
func (h *Handlers) GetRoutingRule(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	rule, err := h.store.GetRoutingRuleByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "routing rule not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load routing rule")
		return
	}
	writeData(ctx, fasthttp.StatusOK, toRoutingRuleDTO(rule))
}

// UpdateRoutingRule handles PUT /api/routing-rules/{id}.
func (h *Handlers) UpdateRoutingRule(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	var req routingRuleRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "name is required")
		return
	}
	existing, err := h.store.GetRoutingRuleByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "routing rule not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load routing rule")
		return
	}
	existing.Name = req.Name
	existing.Priority = req.Priority
	existing.CondField = req.CondField
	existing.CondOperator = req.CondOperator
	existing.CondValue = req.CondValue
	existing.TargetProvider = req.TargetProvider
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	if err := h.store.UpdateRoutingRule(existing); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "update routing rule")
		return
	}
	updated, err := h.store.GetRoutingRuleByID(id)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load routing rule")
		return
	}
	h.recordAudit(ctx, "update_routing_rule", updated.ID, fmt.Sprintf("Updated routing rule %s", updated.Name))
	writeData(ctx, fasthttp.StatusOK, toRoutingRuleDTO(updated))
}

// DeleteRoutingRule handles DELETE /api/routing-rules/{id}.
func (h *Handlers) DeleteRoutingRule(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	if err := h.store.DeleteRoutingRule(id); errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "routing rule not found")
		return
	} else if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "delete routing rule")
		return
	}
	h.recordAudit(ctx, "delete_routing_rule", id, "Deleted routing rule")
	writeData(ctx, fasthttp.StatusOK, map[string]any{"message": "Routing rule deleted successfully"})
}
