package api

import (
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/ratelimit"
	"github.com/valyala/fasthttp"
)

func intPtr(v int) *int           { return &v }
func floatPtr(v float64) *float64 { return &v }

func policyCtx(model string, identity APIKeyIdentity) *fasthttp.RequestCtx {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetBody([]byte(`{"model":"` + model + `","messages":[]}`))
	ctx.SetUserValue(requestAPIKeyPolicyKey, identity)
	return ctx
}

func newPolicyServer() *Server {
	return &Server{limiter: ratelimit.NewLimiter()}
}

func TestEnforceKeyPolicyNoIdentityPasses(t *testing.T) {
	s := newPolicyServer()
	ctx := &fasthttp.RequestCtx{}
	if !s.enforceKeyPolicy(ctx) {
		t.Fatal("request without a key policy must pass")
	}
}

func TestEnforceKeyPolicyOutOfScopeForbidden(t *testing.T) {
	s := newPolicyServer()
	ctx := policyCtx("claude-3-opus", APIKeyIdentity{ID: "id-1", Scopes: []string{"gpt-*"}})
	if s.enforceKeyPolicy(ctx) {
		t.Fatal("out-of-scope model must be rejected")
	}
	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("status = %d, want 403", ctx.Response.StatusCode())
	}
}

func TestEnforceKeyPolicyInScopeAllowed(t *testing.T) {
	s := newPolicyServer()
	ctx := policyCtx("gpt-4o", APIKeyIdentity{ID: "id-1", Scopes: []string{"gpt-*"}})
	if !s.enforceKeyPolicy(ctx) {
		t.Fatal("in-scope model must pass")
	}
}

func TestEnforceKeyPolicyOverRPMTooManyRequests(t *testing.T) {
	s := newPolicyServer()
	identity := APIKeyIdentity{ID: "id-rpm", RateLimitRPM: intPtr(2)}
	for i := 0; i < 2; i++ {
		if !s.enforceKeyPolicy(policyCtx("gpt-4o", identity)) {
			t.Fatalf("request %d should pass", i)
		}
	}
	ctx := policyCtx("gpt-4o", identity)
	if s.enforceKeyPolicy(ctx) {
		t.Fatal("3rd request should be throttled")
	}
	if ctx.Response.StatusCode() != fasthttp.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", ctx.Response.StatusCode())
	}
}

func TestEnforceKeyPolicyOverSpendBlocked(t *testing.T) {
	s := newPolicyServer()
	identity := APIKeyIdentity{ID: "id-spend", DailySpendCapUSD: floatPtr(1.0)}
	s.limiter.AddSpend("id-spend", 1.5)
	ctx := policyCtx("gpt-4o", identity)
	if s.enforceKeyPolicy(ctx) {
		t.Fatal("over-spend request must be blocked")
	}
	if ctx.Response.StatusCode() != fasthttp.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402", ctx.Response.StatusCode())
	}
}

func TestRecordKeyUsageAddsTokensAndSpend(t *testing.T) {
	s := newPolicyServer()
	s.config.Store = newAPITestStore(t)
	keyID := "id-rec"
	resp := &providers.ChatResponse{
		Model:    "gpt-4o",
		Provider: providers.ProviderOpenAI,
		Usage: &providers.Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}
	req := &providers.ChatRequest{Model: "gpt-4o"}
	s.recordKeyUsage(&keyID, "", req, resp, nil)

	if !s.limiter.AllowTokens(keyID, intPtr(151)) {
		t.Fatal("expected ~150 tokens recorded")
	}
	if s.limiter.AllowTokens(keyID, intPtr(150)) {
		t.Fatal("150 tokens recorded must hit the 150 limit")
	}
	if s.limiter.SpendToday(keyID) <= 0 {
		t.Fatalf("spend not recorded: %v", s.limiter.SpendToday(keyID))
	}
}

func TestRecordKeyUsageNilKeyNoop(t *testing.T) {
	s := newPolicyServer()
	s.recordKeyUsage(nil, "", nil, nil, nil)
}

func TestModelInScopes(t *testing.T) {
	cases := []struct {
		model  string
		scopes []string
		want   bool
	}{
		{"gpt-4o", nil, true},
		{"gpt-4o", []string{}, true},
		{"gpt-4o", []string{"gpt-*"}, true},
		{"claude-3", []string{"gpt-*"}, false},
		{"claude-3", []string{"gpt-*", "claude-*"}, true},
		{"anything", []string{"*"}, true},
	}
	for _, c := range cases {
		if got := modelInScopes(c.model, c.scopes); got != c.want {
			t.Errorf("modelInScopes(%q,%v) = %v, want %v", c.model, c.scopes, got, c.want)
		}
	}
}
