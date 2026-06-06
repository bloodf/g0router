package proxy

import (
	"context"
	"fmt"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

// RoutingRuleStore is the narrow store interface for routing rule evaluation.
type RoutingRuleStore interface {
	ListRoutingRules() ([]store.RoutingRule, error)
}

// ModelLimitStore is the narrow store interface for model limit checking.
type ModelLimitStore interface {
	GetModelLimitByModel(model string) (*store.ModelLimit, error)
}

// RegisterRoutingRuleEvaluator attaches a routing rule evaluator to the engine.
func (e *Engine) RegisterRoutingRuleEvaluator(store RoutingRuleStore) {
	e.ruleEvaluator = NewRoutingRuleEvaluator(store)
}

// RegisterModelLimitChecker attaches a model limit checker to the engine.
func (e *Engine) RegisterModelLimitChecker(store ModelLimitStore) {
	e.modelLimitChecker = NewModelLimitChecker(store)
}

// InvalidateRoutingRules clears the routing rule cache so the next dispatch
// reloads rules from the store.
func (e *Engine) InvalidateRoutingRules() {
	if e.ruleEvaluator != nil {
		e.ruleEvaluator.Invalidate()
	}
}

// applyRoutingRules evaluates routing rules against the request and rewrites
// the model if a rule matches. Rules are evaluated in priority order before
// alias/combo resolution.
func (e *Engine) applyRoutingRules(ctx context.Context, req *providers.ChatRequest) {
	if e.ruleEvaluator == nil || req == nil {
		return
	}
	headers := RoutingHeadersFromContext(ctx)
	if rewritten, ok := e.ruleEvaluator.Evaluate(req, headers); ok {
		req.Model = rewritten
	}
}

// checkModelLimits validates model limits for the resolved route. It checks
// max_tokens, max_rpm, and allowed_key_ids against the upstream model.
func (e *Engine) checkModelLimits(ctx context.Context, req *providers.ChatRequest, model string) error {
	if e.modelLimitChecker == nil || req == nil {
		return nil
	}
	checkReq := *req
	checkReq.Model = model
	return e.modelLimitChecker.Check(ctx, &checkReq)
}

// dispatchErrorClassification maps model limit errors to HTTP status codes
// and user-facing messages.
func classifyModelLimitError(err error) (status int, message string) {
	if err == nil {
		return 0, ""
	}
	switch {
	case isError(err, ErrModelLimitExceeded):
		return 400, "max_tokens exceeds model limit"
	case isError(err, ErrModelRateLimited):
		return 429, "model rate limit exceeded"
	case isError(err, ErrModelKeyNotAllowed):
		return 403, "api key not allowed for this model"
	default:
		return 500, fmt.Sprintf("model limit check failed: %v", err)
	}
}

func isError(err, target error) bool {
	return err == target
}
