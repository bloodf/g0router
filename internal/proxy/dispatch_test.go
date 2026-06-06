package proxy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

func TestRegisterRoutingRuleEvaluator(t *testing.T) {
	e := NewEngine(&fakeEngineStore{})
	if e.ruleEvaluator != nil {
		t.Fatal("expected nil evaluator")
	}
	e.RegisterRoutingRuleEvaluator(&fakeRuleStore{})
	if e.ruleEvaluator == nil {
		t.Fatal("expected evaluator to be set")
	}
}

func TestRegisterModelLimitChecker(t *testing.T) {
	e := NewEngine(&fakeEngineStore{})
	if e.modelLimitChecker != nil {
		t.Fatal("expected nil checker")
	}
	e.RegisterModelLimitChecker(&fakeModelLimitStore{})
	if e.modelLimitChecker == nil {
		t.Fatal("expected checker to be set")
	}
}

func TestInvalidateRoutingRules(t *testing.T) {
	e := NewEngine(&fakeEngineStore{})
	// Should not panic when nil
	e.InvalidateRoutingRules()

	fs := &fakeRuleStore{
		rules: []store.RoutingRule{
			{ID: 1, Name: "r1", Priority: 1, CondField: "model", CondOperator: "equals", CondValue: "gpt-4o", TargetProvider: "openai", TargetModel: stringPtr("gpt-4o"), IsActive: true},
		},
	}
	e.RegisterRoutingRuleEvaluator(fs)

	req := &providers.ChatRequest{Model: "gpt-4o"}
	_, ok := e.ruleEvaluator.Evaluate(req, nil)
	if !ok {
		t.Fatal("expected match")
	}

	fs.rules = nil
	e.InvalidateRoutingRules()

	_, ok = e.ruleEvaluator.Evaluate(req, nil)
	if ok {
		t.Fatal("expected no match after invalidate")
	}
}

func TestApplyRoutingRulesNil(t *testing.T) {
	e := NewEngine(&fakeEngineStore{})
	// Should not panic with nil evaluator or nil req
	e.applyRoutingRules(context.Background(), nil)

	req := &providers.ChatRequest{Model: "gpt-4o"}
	e.applyRoutingRules(context.Background(), req)
	if req.Model != "gpt-4o" {
		t.Fatal("model should not change when evaluator is nil")
	}
}

func TestCheckModelLimitsNil(t *testing.T) {
	e := NewEngine(&fakeEngineStore{})
	// Should not panic with nil checker or nil req
	if err := e.checkModelLimits(context.Background(), nil, "gpt-4o"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := &providers.ChatRequest{Model: "gpt-4o"}
	if err := e.checkModelLimits(context.Background(), req, "gpt-4o"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClassifyModelLimitError(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		wantS  int
		wantM  string
	}{
		{"limit exceeded", ErrModelLimitExceeded, 400, "max_tokens exceeds model limit"},
		{"rate limited", ErrModelRateLimited, 429, "model rate limit exceeded"},
		{"key not allowed", ErrModelKeyNotAllowed, 403, "api key not allowed for this model"},
		{"unknown", errors.New("boom"), 500, "model limit check failed: boom"},
		{"nil", nil, 0, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotS, gotM := classifyModelLimitError(tc.err)
			if gotS != tc.wantS || gotM != tc.wantM {
				t.Fatalf("status=%d msg=%q, want status=%d msg=%q", gotS, gotM, tc.wantS, tc.wantM)
			}
		})
	}
}

func TestIsError(t *testing.T) {
	if !isError(ErrModelLimitExceeded, ErrModelLimitExceeded) {
		t.Fatal("expected true")
	}
	if isError(ErrModelLimitExceeded, ErrModelRateLimited) {
		t.Fatal("expected false")
	}
}

func TestWithRoutingHeaders(t *testing.T) {
	ctx := WithRoutingHeaders(context.Background(), map[string]string{"x-model": "gpt-4o"})
	h := RoutingHeadersFromContext(ctx)
	if h["x-model"] != "gpt-4o" {
		t.Fatalf("header = %q, want gpt-4o", h["x-model"])
	}
}

func TestRoutingHeadersFromContextMissing(t *testing.T) {
	h := RoutingHeadersFromContext(context.Background())
	if h != nil {
		t.Fatal("expected nil")
	}
}

func TestApplyRoutingRulesWithEvaluator(t *testing.T) {
	fs := &fakeRuleStore{
		rules: []store.RoutingRule{
			{ID: 1, Name: "r1", Priority: 1, CondField: "model", CondOperator: "equals", CondValue: "gpt-4o", TargetProvider: "openai", TargetModel: stringPtr("gpt-4o-alt"), IsActive: true},
		},
	}
	e := NewEngine(&fakeEngineStore{})
	e.RegisterRoutingRuleEvaluator(fs)

	req := &providers.ChatRequest{Model: "gpt-4o"}
	e.applyRoutingRules(context.Background(), req)
	if req.Model != "gpt-4o-alt" {
		t.Fatalf("model = %q, want gpt-4o-alt", req.Model)
	}
}

func TestCheckModelLimitsWithChecker(t *testing.T) {
	fs := &fakeModelLimitStore{
		limit: &store.ModelLimit{Model: "gpt-4o", MaxTokens: intPtr(100)},
	}
	e := NewEngine(&fakeEngineStore{})
	e.RegisterModelLimitChecker(fs)

	req := &providers.ChatRequest{Model: "gpt-4o", MaxTokens: intPtr(200)}
	err := e.checkModelLimits(context.Background(), req, "gpt-4o")
	if err == nil {
		t.Fatal("expected error")
	}
}

type fakeEngineStore struct{}

func (f *fakeEngineStore) ResolveModelAlias(string) (store.ModelAlias, error) { return store.ModelAlias{}, nil }
func (f *fakeEngineStore) GetActiveConnections(string) ([]*store.Connection, error) { return nil, nil }
func (f *fakeEngineStore) ListConnections() ([]*store.Connection, error) { return nil, nil }
func (f *fakeEngineStore) MarkConnectionRefreshFailure(string, string) error { return nil }
func (f *fakeEngineStore) UpdateConnectionCredentials(string, *string, *string, *int64) error { return nil }
func (f *fakeEngineStore) ClearConnectionRefreshFailure(string) error { return nil }
func (f *fakeEngineStore) GetActiveCombo(string) (*store.Combo, error) { return nil, nil }
func (f *fakeEngineStore) UpdateConnection(*store.Connection) error { return nil }
func (f *fakeEngineStore) ProviderModelStats(time.Time) (map[string]store.ModelStat, error) { return nil, nil }
func (f *fakeEngineStore) GetConnectionProxyPoolID(string) (*string, error) { return nil, nil }
func (f *fakeEngineStore) GetProxyPool(string) (*store.ProxyPool, error) { return nil, nil }
func (f *fakeEngineStore) IsModelDisabled(provider, model string) (bool, error) { return false, nil }
func (f *fakeEngineStore) ListCustomModels() ([]store.CustomModel, error) { return nil, nil }
func (f *fakeEngineStore) ListRoutingRules() ([]store.RoutingRule, error) { return nil, nil }
func (f *fakeEngineStore) GetModelLimitByModel(model string) (*store.ModelLimit, error) { return nil, nil }
