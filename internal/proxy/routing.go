package proxy

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bloodf/g0router/internal/modelcatalog"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

var (
	ErrModelLimitExceeded  = errors.New("model limit exceeded")
	ErrModelRateLimited    = errors.New("model rate limited")
	ErrModelKeyNotAllowed  = errors.New("model key not allowed")
)

const defaultRuleCacheTTL = 5 * time.Minute

// --- Context keys for passing request metadata through context.Context ---

type apiKeyIDKey struct{}
type routingHeadersKey struct{}

// WithAPIKeyID attaches the authenticated API key ID to the context.
func WithAPIKeyID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, apiKeyIDKey{}, id)
}

// APIKeyIDFromContext retrieves the API key ID from the context.
func APIKeyIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(apiKeyIDKey{}).(string); ok {
		return id
	}
	return ""
}

// WithRoutingHeaders attaches HTTP headers to the context for routing rule evaluation.
func WithRoutingHeaders(ctx context.Context, headers map[string]string) context.Context {
	return context.WithValue(ctx, routingHeadersKey{}, headers)
}

// RoutingHeadersFromContext retrieves HTTP headers from the context.
func RoutingHeadersFromContext(ctx context.Context) map[string]string {
	if h, ok := ctx.Value(routingHeadersKey{}).(map[string]string); ok {
		return h
	}
	return nil
}

// --- Routing rule cache ---

type ruleCache struct {
	ttl      time.Duration
	mu       sync.Mutex
	rules    []store.RoutingRule
	loadedAt time.Time
	version  int64
}

func newRuleCache(ttl time.Duration) *ruleCache {
	return &ruleCache{ttl: ttl}
}

func (c *ruleCache) get(now time.Time) ([]store.RoutingRule, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rules == nil || now.Sub(c.loadedAt) >= c.ttl {
		return nil, false
	}
	// Return a copy to prevent external mutation
	out := make([]store.RoutingRule, len(c.rules))
	copy(out, c.rules)
	return out, true
}

func (c *ruleCache) set(rules []store.RoutingRule, now time.Time) {
	c.mu.Lock()
	c.rules = make([]store.RoutingRule, len(rules))
	copy(c.rules, rules)
	c.loadedAt = now
	c.mu.Unlock()
}

func (c *ruleCache) invalidate() {
	c.mu.Lock()
	c.rules = nil
	c.version++
	c.mu.Unlock()
}

// --- Routing rule evaluator ---

type ruleStore interface {
	ListRoutingRules() ([]store.RoutingRule, error)
}

// RoutingRuleEvaluator evaluates routing rules in priority order.
type RoutingRuleEvaluator struct {
	store ruleStore
	cache *ruleCache
	now   func() time.Time
}

// NewRoutingRuleEvaluator creates a RoutingRuleEvaluator with default TTL.
func NewRoutingRuleEvaluator(store ruleStore) *RoutingRuleEvaluator {
	return NewRoutingRuleEvaluatorWithClock(store, time.Now)
}

// NewRoutingRuleEvaluatorWithClock creates a RoutingRuleEvaluator with a custom clock.
func NewRoutingRuleEvaluatorWithClock(store ruleStore, now func() time.Time) *RoutingRuleEvaluator {
	return &RoutingRuleEvaluator{
		store: store,
		cache: newRuleCache(defaultRuleCacheTTL),
		now:   now,
	}
}

// Evaluate checks routing rules against the request. If a rule matches, it returns
// the rewritten model name and true. Otherwise it returns "", false.
func (e *RoutingRuleEvaluator) Evaluate(req *providers.ChatRequest, headers map[string]string) (string, bool) {
	rules, err := e.loadRules()
	if err != nil || len(rules) == 0 {
		return "", false
	}

	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}
		if e.ruleMatches(rule, req, headers) {
			if rule.TargetModel != nil && *rule.TargetModel != "" {
				return *rule.TargetModel, true
			}
			// If target_model is empty, we can't meaningfully rewrite.
			// Return the original model so the caller can handle provider forcing if needed.
			return req.Model, true
		}
	}
	return "", false
}

// Invalidate clears the rule cache so the next evaluation reloads from store.
func (e *RoutingRuleEvaluator) Invalidate() {
	e.cache.invalidate()
}

func (e *RoutingRuleEvaluator) loadRules() ([]store.RoutingRule, error) {
	now := e.now()
	if cached, ok := e.cache.get(now); ok {
		return cached, nil
	}

	rules, err := e.store.ListRoutingRules()
	if err != nil {
		return nil, err
	}
	sortRulesByPriority(rules)
	e.cache.set(rules, now)
	return rules, nil
}

func (e *RoutingRuleEvaluator) ruleMatches(rule store.RoutingRule, req *providers.ChatRequest, headers map[string]string) bool {
	switch rule.CondField {
	case "model":
		return matchString(rule.CondOperator, req.Model, rule.CondValue)
	case "provider":
		provider := previewResolveProvider(req.Model)
		return matchString(rule.CondOperator, provider, rule.CondValue)
	case "header":
		if headers == nil {
			return false
		}
		// cond_value for header is expected to be "header_name:expected_value"
		// This is a pragmatic interpretation; future schema changes may add a dedicated column.
		headerName, expectedValue, ok := strings.Cut(rule.CondValue, ":")
		if !ok {
			return false
		}
		actualValue := headers[strings.ToLower(strings.TrimSpace(headerName))]
		return matchString(rule.CondOperator, actualValue, strings.TrimSpace(expectedValue))
	default:
		return false
	}
}

func previewResolveProvider(model string) string {
	if model == "" {
		return ""
	}
	catalog := modelcatalog.NewCatalog()
	if provider, ok := catalog.ProviderForModel(model); ok {
		return provider.String()
	}
	if provider, ok := resolveProvider(model); ok {
		return provider.String()
	}
	return ""
}

func matchString(operator, value, target string) bool {
	switch operator {
	case "equals":
		return value == target
	case "contains":
		return strings.Contains(value, target)
	case "starts_with":
		return strings.HasPrefix(value, target)
	default:
		return false
	}
}

// --- Model limit checker ---

type modelLimitStore interface {
	GetModelLimitByModel(model string) (*store.ModelLimit, error)
}

type modelRPMTracker struct {
	mu      sync.Mutex
	buckets map[string]*rpmBucket
	now     func() time.Time
}

type rpmBucket struct {
	windowStart time.Time
	count       int
}

func newModelRPMTracker(now func() time.Time) *modelRPMTracker {
	return &modelRPMTracker{
		buckets: make(map[string]*rpmBucket),
		now:     now,
	}
}

func (t *modelRPMTracker) Allow(model string, maxRPM *int) bool {
	if maxRPM == nil || *maxRPM <= 0 {
		return true
	}
	now := t.now()
	t.mu.Lock()
	defer t.mu.Unlock()

	bucket := t.buckets[model]
	if bucket == nil || now.Sub(bucket.windowStart) >= time.Minute {
		bucket = &rpmBucket{windowStart: now, count: 0}
		t.buckets[model] = bucket
	}
	if bucket.count >= *maxRPM {
		return false
	}
	bucket.count++
	return true
}

// ModelLimitChecker checks model limits at dispatch time.
type ModelLimitChecker struct {
	store modelLimitStore
	rpm   *modelRPMTracker
	now   func() time.Time
}

// NewModelLimitChecker creates a ModelLimitChecker with the real wall clock.
func NewModelLimitChecker(store modelLimitStore) *ModelLimitChecker {
	return NewModelLimitCheckerWithClock(store, time.Now)
}

// NewModelLimitCheckerWithClock creates a ModelLimitChecker with a custom clock.
func NewModelLimitCheckerWithClock(store modelLimitStore, now func() time.Time) *ModelLimitChecker {
	return &ModelLimitChecker{
		store: store,
		rpm:   newModelRPMTracker(now),
		now:   now,
	}
}

// Check validates model limits for the request. It returns an error if any limit is exceeded.
// Limits checked: max_tokens, max_rpm, allowed_key_ids.
func (c *ModelLimitChecker) Check(ctx context.Context, req *providers.ChatRequest) error {
	if req == nil {
		return nil
	}

	limit, err := c.store.GetModelLimitByModel(req.Model)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("check model limit: %w", err)
	}
	if limit == nil {
		return nil
	}

	// max_tokens limit: reject if above limit
	if limit.MaxTokens != nil && *limit.MaxTokens > 0 {
		if req.MaxTokens != nil && *req.MaxTokens > *limit.MaxTokens {
			return fmt.Errorf("max_tokens %d exceeds model limit %d: %w", *req.MaxTokens, *limit.MaxTokens, ErrModelLimitExceeded)
		}
		if req.MaxCompletionTokens != nil && *req.MaxCompletionTokens > *limit.MaxTokens {
			return fmt.Errorf("max_completion_tokens %d exceeds model limit %d: %w", *req.MaxCompletionTokens, *limit.MaxTokens, ErrModelLimitExceeded)
		}
	}

	// RPM limit
	if limit.MaxRPM != nil && *limit.MaxRPM > 0 {
		if !c.rpm.Allow(req.Model, limit.MaxRPM) {
			return fmt.Errorf("model %s rate limited at %d rpm: %w", req.Model, *limit.MaxRPM, ErrModelRateLimited)
		}
	}

	// Key allowlist
	if len(limit.AllowedKeyIDs) > 0 {
		keyID := APIKeyIDFromContext(ctx)
		if keyID == "" || !sliceContains(limit.AllowedKeyIDs, keyID) {
			return fmt.Errorf("key %q not allowed for model %s: %w", keyID, req.Model, ErrModelKeyNotAllowed)
		}
	}

	return nil
}

func sliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// sortRulesByPriority sorts rules by priority descending, then by ID ascending for stability.
func sortRulesByPriority(rules []store.RoutingRule) {
	sort.SliceStable(rules, func(i, j int) bool {
		if rules[i].Priority != rules[j].Priority {
			return rules[i].Priority > rules[j].Priority
		}
		return rules[i].ID < rules[j].ID
	})
}
