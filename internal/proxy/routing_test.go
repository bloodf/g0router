package proxy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

type fakeRuleStore struct {
	rules []store.RoutingRule
	err   error
}

func (f *fakeRuleStore) ListRoutingRules() ([]store.RoutingRule, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.rules, nil
}

type fakeModelLimitStore struct {
	limit *store.ModelLimit
	err   error
}

func (f *fakeModelLimitStore) GetModelLimitByModel(model string) (*store.ModelLimit, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.limit == nil {
		return nil, store.ErrNotFound
	}
	return f.limit, nil
}

func TestRoutingRuleEvaluatorPriorityOrder(t *testing.T) {
	fs := &fakeRuleStore{
		rules: []store.RoutingRule{
			{ID: 1, Name: "low", Priority: 1, CondField: "model", CondOperator: "equals", CondValue: "gpt-4o", TargetProvider: "openai", TargetModel: stringPtr("gpt-4o-low"), IsActive: true},
			{ID: 2, Name: "high", Priority: 10, CondField: "model", CondOperator: "equals", CondValue: "gpt-4o", TargetProvider: "openai", TargetModel: stringPtr("gpt-4o-high"), IsActive: true},
		},
	}
	eval := NewRoutingRuleEvaluator(fs)
	req := &providers.ChatRequest{Model: "gpt-4o"}
	rewritten, ok := eval.Evaluate(req, nil)
	if !ok {
		t.Fatal("expected match")
	}
	if rewritten != "gpt-4o-high" {
		t.Fatalf("rewritten = %q, want gpt-4o-high", rewritten)
	}
}

func TestRoutingRuleEvaluatorInactiveSkipped(t *testing.T) {
	fs := &fakeRuleStore{
		rules: []store.RoutingRule{
			{ID: 1, Name: "inactive", Priority: 10, CondField: "model", CondOperator: "equals", CondValue: "gpt-4o", TargetProvider: "openai", TargetModel: stringPtr("gpt-4o-alt"), IsActive: false},
		},
	}
	eval := NewRoutingRuleEvaluator(fs)
	req := &providers.ChatRequest{Model: "gpt-4o"}
	_, ok := eval.Evaluate(req, nil)
	if ok {
		t.Fatal("expected no match for inactive rule")
	}
}

func TestRoutingRuleEvaluatorNoMatchFallsThrough(t *testing.T) {
	fs := &fakeRuleStore{
		rules: []store.RoutingRule{
			{ID: 1, Name: "r1", Priority: 1, CondField: "model", CondOperator: "equals", CondValue: "claude-sonnet", TargetProvider: "anthropic", TargetModel: stringPtr("claude-sonnet"), IsActive: true},
		},
	}
	eval := NewRoutingRuleEvaluator(fs)
	req := &providers.ChatRequest{Model: "gpt-4o"}
	_, ok := eval.Evaluate(req, nil)
	if ok {
		t.Fatal("expected no match")
	}
}

func TestRoutingRuleEvaluatorCacheHit(t *testing.T) {
	fs := &fakeRuleStore{
		rules: []store.RoutingRule{
			{ID: 1, Name: "r1", Priority: 1, CondField: "model", CondOperator: "equals", CondValue: "gpt-4o", TargetProvider: "openai", TargetModel: stringPtr("gpt-4o-alt"), IsActive: true},
		},
	}
	clock := &fakeClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	eval := NewRoutingRuleEvaluatorWithClock(fs, clock.Now)

	// First evaluation hits store
	req := &providers.ChatRequest{Model: "gpt-4o"}
	_, ok := eval.Evaluate(req, nil)
	if !ok {
		t.Fatal("expected match")
	}

	// Second evaluation should use cache
	fs.rules = nil // clear rules in store
	_, ok = eval.Evaluate(req, nil)
	if !ok {
		t.Fatal("expected cache hit")
	}
}

func TestRoutingRuleEvaluatorCacheInvalidate(t *testing.T) {
	fs := &fakeRuleStore{
		rules: []store.RoutingRule{
			{ID: 1, Name: "r1", Priority: 1, CondField: "model", CondOperator: "equals", CondValue: "gpt-4o", TargetProvider: "openai", TargetModel: stringPtr("gpt-4o-alt"), IsActive: true},
		},
	}
	clock := &fakeClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	eval := NewRoutingRuleEvaluatorWithClock(fs, clock.Now)

	req := &providers.ChatRequest{Model: "gpt-4o"}
	eval.Evaluate(req, nil)

	// Invalidate cache
	fs.rules = nil
	eval.Invalidate()

	_, ok := eval.Evaluate(req, nil)
	if ok {
		t.Fatal("expected no match after invalidate")
	}
}

func TestRoutingRuleEvaluatorModelEquals(t *testing.T) {
	fs := &fakeRuleStore{
		rules: []store.RoutingRule{
			{ID: 1, Name: "r1", Priority: 1, CondField: "model", CondOperator: "equals", CondValue: "gpt-4o", TargetProvider: "openai", TargetModel: stringPtr("gpt-4o-alt"), IsActive: true},
		},
	}
	eval := NewRoutingRuleEvaluator(fs)

	cases := []struct {
		model   string
		wantOK  bool
		wantRew string
	}{
		{"gpt-4o", true, "gpt-4o-alt"},
		{"gpt-4o-mini", false, ""},
	}

	for _, tc := range cases {
		req := &providers.ChatRequest{Model: tc.model}
		rewritten, ok := eval.Evaluate(req, nil)
		if ok != tc.wantOK {
			t.Fatalf("model=%q: ok=%v, want %v", tc.model, ok, tc.wantOK)
		}
		if ok && rewritten != tc.wantRew {
			t.Fatalf("model=%q: rewritten=%q, want %q", tc.model, rewritten, tc.wantRew)
		}
	}
}

func TestRoutingRuleEvaluatorModelStartsWith(t *testing.T) {
	fs := &fakeRuleStore{
		rules: []store.RoutingRule{
			{ID: 1, Name: "r1", Priority: 1, CondField: "model", CondOperator: "starts_with", CondValue: "gpt-", TargetProvider: "openai", TargetModel: stringPtr("gpt-fallback"), IsActive: true},
		},
	}
	eval := NewRoutingRuleEvaluator(fs)

	req := &providers.ChatRequest{Model: "gpt-4o"}
	rewritten, ok := eval.Evaluate(req, nil)
	if !ok {
		t.Fatal("expected match")
	}
	if rewritten != "gpt-fallback" {
		t.Fatalf("rewritten = %q, want gpt-fallback", rewritten)
	}
}

func TestRoutingRuleEvaluatorModelContains(t *testing.T) {
	fs := &fakeRuleStore{
		rules: []store.RoutingRule{
			{ID: 1, Name: "r1", Priority: 1, CondField: "model", CondOperator: "contains", CondValue: "4o", TargetProvider: "openai", TargetModel: stringPtr("gpt-4o"), IsActive: true},
		},
	}
	eval := NewRoutingRuleEvaluator(fs)

	req := &providers.ChatRequest{Model: "gpt-4o-mini"}
	rewritten, ok := eval.Evaluate(req, nil)
	if !ok {
		t.Fatal("expected match")
	}
	if rewritten != "gpt-4o" {
		t.Fatalf("rewritten = %q, want gpt-4o", rewritten)
	}
}

func TestRoutingRuleEvaluatorProviderCondition(t *testing.T) {
	fs := &fakeRuleStore{
		rules: []store.RoutingRule{
			{ID: 1, Name: "r1", Priority: 1, CondField: "provider", CondOperator: "equals", CondValue: "openai", TargetProvider: "anthropic", TargetModel: stringPtr("claude-sonnet"), IsActive: true},
		},
	}
	eval := NewRoutingRuleEvaluator(fs)

	// gpt-4o resolves to openai provider
	req := &providers.ChatRequest{Model: "gpt-4o"}
	rewritten, ok := eval.Evaluate(req, nil)
	if !ok {
		t.Fatal("expected match")
	}
	if rewritten != "claude-sonnet" {
		t.Fatalf("rewritten = %q, want claude-sonnet", rewritten)
	}

	// claude-sonnet resolves to anthropic provider
	req = &providers.ChatRequest{Model: "claude-sonnet-4"}
	_, ok = eval.Evaluate(req, nil)
	if ok {
		t.Fatal("expected no match for anthropic provider")
	}
}

func TestModelLimitCheckerMaxTokensReject(t *testing.T) {
	fs := &fakeModelLimitStore{
		limit: &store.ModelLimit{Model: "gpt-4o", MaxTokens: intPtr(4096)},
	}
	checker := NewModelLimitChecker(fs)

	req := &providers.ChatRequest{Model: "gpt-4o", MaxTokens: intPtr(8192)}
	err := checker.Check(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for max_tokens above limit")
	}
	if !errors.Is(err, ErrModelLimitExceeded) {
		t.Fatalf("error = %v, want ErrModelLimitExceeded", err)
	}
}

func TestModelLimitCheckerMaxTokensWithinLimit(t *testing.T) {
	fs := &fakeModelLimitStore{
		limit: &store.ModelLimit{Model: "gpt-4o", MaxTokens: intPtr(4096)},
	}
	checker := NewModelLimitChecker(fs)

	req := &providers.ChatRequest{Model: "gpt-4o", MaxTokens: intPtr(2048)}
	err := checker.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestModelLimitCheckerMaxTokensNil(t *testing.T) {
	fs := &fakeModelLimitStore{
		limit: &store.ModelLimit{Model: "gpt-4o", MaxTokens: intPtr(4096)},
	}
	checker := NewModelLimitChecker(fs)

	req := &providers.ChatRequest{Model: "gpt-4o"}
	err := checker.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestModelLimitCheckerNoLimit(t *testing.T) {
	fs := &fakeModelLimitStore{}
	checker := NewModelLimitChecker(fs)

	req := &providers.ChatRequest{Model: "gpt-4o", MaxTokens: intPtr(999999)}
	err := checker.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestModelLimitCheckerRPM(t *testing.T) {
	fs := &fakeModelLimitStore{
		limit: &store.ModelLimit{Model: "gpt-4o", MaxRPM: intPtr(2)},
	}
	clock := &fakeClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	checker := NewModelLimitCheckerWithClock(fs, clock.Now)

	ctx := context.Background()
	req := &providers.ChatRequest{Model: "gpt-4o"}

	// First 2 requests should pass
	if err := checker.Check(ctx, req); err != nil {
		t.Fatalf("req 1: %v", err)
	}
	if err := checker.Check(ctx, req); err != nil {
		t.Fatalf("req 2: %v", err)
	}

	// Third request should fail
	err := checker.Check(ctx, req)
	if err == nil {
		t.Fatal("req 3: expected rate limit error")
	}
	if !errors.Is(err, ErrModelRateLimited) {
		t.Fatalf("req 3: error = %v, want ErrModelRateLimited", err)
	}

	// After 1 minute, should pass again
	clock.now = clock.now.Add(time.Minute)
	if err := checker.Check(ctx, req); err != nil {
		t.Fatalf("req after minute: %v", err)
	}
}

func TestModelLimitCheckerAllowedKeyIDs(t *testing.T) {
	fs := &fakeModelLimitStore{
		limit: &store.ModelLimit{Model: "gpt-4o", AllowedKeyIDs: []string{"key-1", "key-2"}},
	}
	checker := NewModelLimitChecker(fs)

	ctx := WithAPIKeyID(context.Background(), "key-1")
	req := &providers.ChatRequest{Model: "gpt-4o"}
	if err := checker.Check(ctx, req); err != nil {
		t.Fatalf("allowed key: %v", err)
	}

	ctx = WithAPIKeyID(context.Background(), "key-3")
	err := checker.Check(ctx, req)
	if err == nil {
		t.Fatal("disallowed key: expected error")
	}
	if !errors.Is(err, ErrModelKeyNotAllowed) {
		t.Fatalf("error = %v, want ErrModelKeyNotAllowed", err)
	}
}

func TestModelLimitCheckerEmptyAllowedKeyIDs(t *testing.T) {
	fs := &fakeModelLimitStore{
		limit: &store.ModelLimit{Model: "gpt-4o", AllowedKeyIDs: []string{}},
	}
	checker := NewModelLimitChecker(fs)

	ctx := WithAPIKeyID(context.Background(), "any-key")
	req := &providers.ChatRequest{Model: "gpt-4o"}
	if err := checker.Check(ctx, req); err != nil {
		t.Fatalf("empty allowlist should allow all: %v", err)
	}
}

func TestModelLimitCheckerNoAPIKeyID(t *testing.T) {
	fs := &fakeModelLimitStore{
		limit: &store.ModelLimit{Model: "gpt-4o", AllowedKeyIDs: []string{"key-1"}},
	}
	checker := NewModelLimitChecker(fs)

	req := &providers.ChatRequest{Model: "gpt-4o"}
	err := checker.Check(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when no api key id in context")
	}
	if !errors.Is(err, ErrModelKeyNotAllowed) {
		t.Fatalf("error = %v, want ErrModelKeyNotAllowed", err)
	}
}

func TestModelLimitCheckerMaxCompletionTokens(t *testing.T) {
	fs := &fakeModelLimitStore{
		limit: &store.ModelLimit{Model: "gpt-4o", MaxTokens: intPtr(4096)},
	}
	checker := NewModelLimitChecker(fs)

	req := &providers.ChatRequest{Model: "gpt-4o", MaxCompletionTokens: intPtr(8192)}
	err := checker.Check(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for max_completion_tokens above limit")
	}
}

func TestRoutingRuleEvaluatorHeaderCondition(t *testing.T) {
	fs := &fakeRuleStore{
		rules: []store.RoutingRule{
			{ID: 1, Name: "r1", Priority: 1, CondField: "header", CondOperator: "equals", CondValue: "x-model:gpt-4o", TargetProvider: "openai", TargetModel: stringPtr("gpt-4o-alt"), IsActive: true},
		},
	}
	eval := NewRoutingRuleEvaluator(fs)

	req := &providers.ChatRequest{Model: "any-model"}
	headers := map[string]string{"x-model": "gpt-4o"}
	rewritten, ok := eval.Evaluate(req, headers)
	if !ok {
		t.Fatal("expected match")
	}
	if rewritten != "gpt-4o-alt" {
		t.Fatalf("rewritten = %q, want gpt-4o-alt", rewritten)
	}

	// No headers should not match
	_, ok = eval.Evaluate(req, nil)
	if ok {
		t.Fatal("expected no match without headers")
	}

	// Wrong header value should not match
	_, ok = eval.Evaluate(req, map[string]string{"x-model": "claude"})
	if ok {
		t.Fatal("expected no match with wrong header value")
	}
}

func TestRoutingRuleEvaluatorHeaderInvalidFormat(t *testing.T) {
	fs := &fakeRuleStore{
		rules: []store.RoutingRule{
			{ID: 1, Name: "r1", Priority: 1, CondField: "header", CondOperator: "equals", CondValue: "no-colon-here", TargetProvider: "openai", TargetModel: stringPtr("gpt-4o"), IsActive: true},
		},
	}
	eval := NewRoutingRuleEvaluator(fs)

	req := &providers.ChatRequest{Model: "any"}
	_, ok := eval.Evaluate(req, map[string]string{"x-model": "gpt-4o"})
	if ok {
		t.Fatal("expected no match for invalid header cond_value format")
	}
}

func TestRoutingRuleEvaluatorUnknownCondField(t *testing.T) {
	fs := &fakeRuleStore{
		rules: []store.RoutingRule{
			{ID: 1, Name: "r1", Priority: 1, CondField: "unknown", CondOperator: "equals", CondValue: "x", TargetProvider: "openai", TargetModel: stringPtr("gpt-4o"), IsActive: true},
		},
	}
	eval := NewRoutingRuleEvaluator(fs)

	req := &providers.ChatRequest{Model: "gpt-4o"}
	_, ok := eval.Evaluate(req, nil)
	if ok {
		t.Fatal("expected no match for unknown cond_field")
	}
}

func TestMatchStringUnknownOperator(t *testing.T) {
	if matchString("unknown", "abc", "a") {
		t.Fatal("expected false for unknown operator")
	}
}

func TestPreviewResolveProviderEmpty(t *testing.T) {
	if previewResolveProvider("") != "" {
		t.Fatal("expected empty provider for empty model")
	}
}

func TestModelRPMTrackerNilMaxRPM(t *testing.T) {
	tracker := newModelRPMTracker(func() time.Time { return time.Now() })
	if !tracker.Allow("gpt-4o", nil) {
		t.Fatal("expected allow when maxRPM is nil")
	}
	if !tracker.Allow("gpt-4o", intPtr(0)) {
		t.Fatal("expected allow when maxRPM is 0")
	}
	if !tracker.Allow("gpt-4o", intPtr(-1)) {
		t.Fatal("expected allow when maxRPM is negative")
	}
}

func TestModelLimitCheckerNilRequest(t *testing.T) {
	checker := NewModelLimitChecker(&fakeModelLimitStore{})
	if err := checker.Check(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestModelLimitCheckerStoreError(t *testing.T) {
	fs := &fakeModelLimitStore{err: errors.New("boom")}
	checker := NewModelLimitChecker(fs)

	req := &providers.ChatRequest{Model: "gpt-4o"}
	err := checker.Check(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSortRulesByPriorityEqual(t *testing.T) {
	rules := []store.RoutingRule{
		{ID: 2, Priority: 5},
		{ID: 1, Priority: 5},
	}
	sortRulesByPriority(rules)
	if rules[0].ID != 1 || rules[1].ID != 2 {
		t.Fatalf("expected stable sort by ID when priority is equal, got %v", rules)
	}
}

func TestRoutingRuleEvaluatorTargetModelEmpty(t *testing.T) {
	fs := &fakeRuleStore{
		rules: []store.RoutingRule{
			{ID: 1, Name: "r1", Priority: 1, CondField: "model", CondOperator: "equals", CondValue: "gpt-4o", TargetProvider: "openai", TargetModel: nil, IsActive: true},
		},
	}
	eval := NewRoutingRuleEvaluator(fs)

	req := &providers.ChatRequest{Model: "gpt-4o"}
	rewritten, ok := eval.Evaluate(req, nil)
	if !ok {
		t.Fatal("expected match")
	}
	if rewritten != "gpt-4o" {
		t.Fatalf("rewritten = %q, want gpt-4o", rewritten)
	}
}

type fakeClock struct {
	now time.Time
}

func (f *fakeClock) Now() time.Time {
	return f.now
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
