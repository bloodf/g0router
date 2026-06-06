package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type createRoutingRuleRequest struct {
	Name           string  `json:"name"`
	Priority       int     `json:"priority"`
	CondField      string  `json:"cond_field"`
	CondOperator   string  `json:"cond_operator"`
	CondValue      string  `json:"cond_value"`
	TargetProvider string  `json:"target_provider"`
	TargetModel    *string `json:"target_model"`
}

type updateRoutingRuleRequest struct {
	Name           string  `json:"name"`
	Priority       int     `json:"priority"`
	CondField      string  `json:"cond_field"`
	CondOperator   string  `json:"cond_operator"`
	CondValue      string  `json:"cond_value"`
	TargetProvider string  `json:"target_provider"`
	TargetModel    *string `json:"target_model"`
	IsActive       bool    `json:"is_active"`
}

type routingRuleView struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	Priority       int     `json:"priority"`
	CondField      string  `json:"cond_field"`
	CondOperator   string  `json:"cond_operator"`
	CondValue      string  `json:"cond_value"`
	TargetProvider string  `json:"target_provider"`
	TargetModel    *string `json:"target_model"`
	IsActive       bool    `json:"is_active"`
	CreatedAt      string  `json:"created_at"`
}

func newRoutingRuleView(rule store.RoutingRule) routingRuleView {
	return routingRuleView{
		ID:             rule.ID,
		Name:           rule.Name,
		Priority:       rule.Priority,
		CondField:      rule.CondField,
		CondOperator:   rule.CondOperator,
		CondValue:      rule.CondValue,
		TargetProvider: rule.TargetProvider,
		TargetModel:    rule.TargetModel,
		IsActive:       rule.IsActive,
		CreatedAt:      rule.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type routingRuleStore interface {
	ListRoutingRules() ([]store.RoutingRule, error)
	CreateRoutingRule(name string, priority int, condField, condOperator, condValue, targetProvider string, targetModel *string) (*store.RoutingRule, error)
	GetRoutingRule(id int64) (*store.RoutingRule, error)
	UpdateRoutingRule(id int64, name string, priority int, condField, condOperator, condValue, targetProvider string, targetModel *string, isActive bool) error
	DeleteRoutingRule(id int64) error
}

func RoutingRules(ctx *fasthttp.RequestCtx, s routingRuleStore, id string) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		if id == "" {
			rules, err := s.ListRoutingRules()
			if err != nil {
				log.Printf("list routing rules: %v", err)
				writeError(ctx, fasthttp.StatusInternalServerError, "failed to list routing rules")
				return
			}
			views := make([]routingRuleView, 0, len(rules))
			for _, rule := range rules {
				views = append(views, newRoutingRuleView(rule))
			}
			writeJSON(ctx, fasthttp.StatusOK, listResponse[routingRuleView]{Data: views})
			return
		}
		ruleID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid routing rule id")
			return
		}
		rule, err := s.GetRoutingRule(ruleID)
		if err != nil {
			writeError(ctx, fasthttp.StatusNotFound, "routing rule not found")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, newRoutingRuleView(*rule))
	case fasthttp.MethodPost:
		var req createRoutingRuleRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		if strings.TrimSpace(req.Name) == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "name is required")
			return
		}
		if strings.TrimSpace(req.CondField) == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "cond_field is required")
			return
		}
		if strings.TrimSpace(req.CondOperator) == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "cond_operator is required")
			return
		}
		if strings.TrimSpace(req.TargetProvider) == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "target_provider is required")
			return
		}
		rule, err := s.CreateRoutingRule(req.Name, req.Priority, req.CondField, req.CondOperator, req.CondValue, req.TargetProvider, req.TargetModel)
		if err != nil {
			log.Printf("create routing rule: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to create routing rule")
			return
		}
		writeJSON(ctx, fasthttp.StatusCreated, newRoutingRuleView(*rule))
	case fasthttp.MethodPut:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "routing rule id required")
			return
		}
		ruleID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid routing rule id")
			return
		}
		var req updateRoutingRuleRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		if err := s.UpdateRoutingRule(ruleID, req.Name, req.Priority, req.CondField, req.CondOperator, req.CondValue, req.TargetProvider, req.TargetModel, req.IsActive); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(ctx, fasthttp.StatusNotFound, "routing rule not found")
				return
			}
			log.Printf("update routing rule: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to update routing rule")
			return
		}
		updated, err := s.GetRoutingRule(ruleID)
		if err != nil {
			writeError(ctx, fasthttp.StatusNotFound, "routing rule not found")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, newRoutingRuleView(*updated))
	case fasthttp.MethodDelete:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "routing rule id required")
			return
		}
		ruleID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid routing rule id")
			return
		}
		if err := s.DeleteRoutingRule(ruleID); err != nil {
			log.Printf("delete routing rule: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to delete routing rule")
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}
