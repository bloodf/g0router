package api

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/governance"
	"github.com/bloodf/g0router/internal/ratelimit"
	"github.com/valyala/fasthttp"
)

func TestVirtualKeyAuthPassesForValidKey(t *testing.T) {
	s := newAPITestStore(t)
	key, raw, err := s.CreateVirtualKey("test-key", nil, nil, "monthly", nil, nil)
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}

	srv := NewServer(ServerConfig{
		Port:       0,
		Version:    "test",
		Store:      s,
		Governance: governance.New(s, ratelimit.NewLimiter()),
	})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.Header.Set("Authorization", "Bearer "+raw)

	ok, err := srv.validAPIKey(ctx)
	if err != nil {
		t.Fatalf("validAPIKey: %v", err)
	}
	if !ok {
		t.Fatal("expected valid virtual key to pass")
	}
	if vid := userValueStringPtr(ctx, requestVirtualKeyIDKey); vid == nil || *vid != "1" {
		t.Fatalf("virtual key id = %v, want 1", vid)
	}
	if key.Name != "test-key" {
		t.Fatalf("key name = %q, want test-key", key.Name)
	}
}

func TestVirtualKeyAuthRejectsInvalidKey(t *testing.T) {
	s := newAPITestStore(t)
	srv := NewServer(ServerConfig{
		Port:       0,
		Version:    "test",
		Store:      s,
		Governance: governance.New(s, ratelimit.NewLimiter()),
	})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.Header.Set("Authorization", "Bearer gvk-invalid")

	ok, err := srv.validAPIKey(ctx)
	if err != nil {
		t.Fatalf("validAPIKey: %v", err)
	}
	if ok {
		t.Fatal("expected invalid virtual key to be rejected")
	}
}

func TestVirtualKeyAuthRejectsInactiveKey(t *testing.T) {
	s := newAPITestStore(t)
	key, raw, err := s.CreateVirtualKey("inactive-key", nil, nil, "monthly", nil, nil)
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	if err := s.UpdateVirtualKey(key.ID, key.Name, nil, nil, "monthly", nil, nil, false); err != nil {
		t.Fatalf("UpdateVirtualKey: %v", err)
	}

	srv := NewServer(ServerConfig{
		Port:       0,
		Version:    "test",
		Store:      s,
		Governance: governance.New(s, ratelimit.NewLimiter()),
	})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.Header.Set("Authorization", "Bearer "+raw)

	ok, err := srv.validAPIKey(ctx)
	if err != nil {
		t.Fatalf("validAPIKey: %v", err)
	}
	if ok {
		t.Fatal("expected inactive virtual key to be rejected")
	}
	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("status = %d, want 403", ctx.Response.StatusCode())
	}
}

func TestVirtualKeyAuthRejectsBudgetExhausted(t *testing.T) {
	s := newAPITestStore(t)
	budget := 10.0
	futureReset := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	key, raw, err := s.CreateVirtualKey("broke-key", nil, &budget, "monthly", nil, nil)
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	if err := s.UpdateVirtualKey(key.ID, key.Name, nil, &budget, "monthly", nil, nil, true); err != nil {
		t.Fatalf("UpdateVirtualKey: %v", err)
	}
	if err := s.ResetVirtualKeyBudget(key.ID, futureReset); err != nil {
		t.Fatalf("ResetVirtualKeyBudget: %v", err)
	}
	if err := s.AddVirtualKeyBudgetUsed(key.ID, 10.0); err != nil {
		t.Fatalf("AddVirtualKeyBudgetUsed: %v", err)
	}

	srv := NewServer(ServerConfig{
		Port:       0,
		Version:    "test",
		Store:      s,
		Governance: governance.New(s, ratelimit.NewLimiter()),
	})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.Header.Set("Authorization", "Bearer "+raw)

	ok, err := srv.validAPIKey(ctx)
	if err != nil {
		t.Fatalf("validAPIKey: %v", err)
	}
	if ok {
		t.Fatal("expected budget-exhausted virtual key to be rejected")
	}
	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("status = %d, want 403", ctx.Response.StatusCode())
	}
}

func TestVirtualKeyAuthRejectsTeamBudgetExhausted(t *testing.T) {
	s := newAPITestStore(t)
	teamBudget := 50.0
	futureReset := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	team, err := s.CreateTeam("eng", &teamBudget, "monthly", nil)
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if err := s.ResetTeamBudget(team.ID, futureReset); err != nil {
		t.Fatalf("ResetTeamBudget: %v", err)
	}
	if err := s.AddTeamBudgetUsed(team.ID, 50.0); err != nil {
		t.Fatalf("AddTeamBudgetUsed: %v", err)
	}

	_, raw, err := s.CreateVirtualKey("team-key", &team.ID, nil, "monthly", nil, nil)
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}

	srv := NewServer(ServerConfig{
		Port:       0,
		Version:    "test",
		Store:      s,
		Governance: governance.New(s, ratelimit.NewLimiter()),
	})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.Header.Set("Authorization", "Bearer "+raw)

	ok, err := srv.validAPIKey(ctx)
	if err != nil {
		t.Fatalf("validAPIKey: %v", err)
	}
	if ok {
		t.Fatal("expected team-budget-exhausted virtual key to be rejected")
	}
	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("status = %d, want 403", ctx.Response.StatusCode())
	}
}

func TestVirtualKeyAuthRejectsTeamRateLimit(t *testing.T) {
	s := newAPITestStore(t)
	teamRPM := 5
	team, err := s.CreateTeam("eng", nil, "monthly", &teamRPM)
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	_, raw, err := s.CreateVirtualKey("team-key", &team.ID, nil, "monthly", nil, nil)
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}

	lim := ratelimit.NewLimiter()
	// Exhaust team RPM
	for i := 0; i < 5; i++ {
		_ = lim.AllowRequest("team-1", &teamRPM)
	}

	srv := NewServer(ServerConfig{
		Port:       0,
		Version:    "test",
		Store:      s,
		Governance: governance.New(s, lim),
	})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.Header.Set("Authorization", "Bearer "+raw)

	ok, err := srv.validAPIKey(ctx)
	if err != nil {
		t.Fatalf("validAPIKey: %v", err)
	}
	if ok {
		t.Fatal("expected team-rate-limited virtual key to be rejected")
	}
	if ctx.Response.StatusCode() != fasthttp.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", ctx.Response.StatusCode())
	}
}

func TestVirtualKeyWorksOnV1Endpoint(t *testing.T) {
	s := newAPITestStore(t)
	key, raw, err := s.CreateVirtualKey("v1-key", nil, nil, "monthly", nil, nil)
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}

	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		Governance:      governance.New(s, ratelimit.NewLimiter()),
		InferenceEngine: routeInferenceEngine{response: routeChatResponseWithUsage()},
	})

	req, err := http.NewRequest(http.MethodPost, baseURL+"/v1/chat/completions", strings.NewReader(`{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+raw)
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("POST /v1/chat/completions: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	// Verify virtual key usage was recorded
	got, err := s.GetVirtualKey(key.ID)
	if err != nil {
		t.Fatalf("GetVirtualKey: %v", err)
	}
	if got.BudgetUsedUSD == 0 {
		t.Fatal("expected virtual key budget to be used after inference")
	}
}

func TestRegularAPIKeyUnaffectedByVirtualKeyCheck(t *testing.T) {
	store := newAPITestStore(t)
	_, baseURL := startTestServer(t, ServerConfig{
		Port:          0,
		Version:       "test",
		Store:         store,
		RequireAPIKey: true,
		APIKeySecret:  "test-secret",
		APIKeyValidator: fakeAPIKeyValidator{
			validKeys: map[string]bool{"g0r_valid": true},
		},
	})

	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/settings", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer g0r_valid")
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("GET /api/settings: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestApplyMiddlewareVirtualKeyForbiddenDoesNotFallThroughToSession(t *testing.T) {
	s := newAPITestStore(t)
	key, raw, err := s.CreateVirtualKey("inactive", nil, nil, "monthly", nil, nil)
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	if err := s.UpdateVirtualKey(key.ID, key.Name, nil, nil, "monthly", nil, nil, false); err != nil {
		t.Fatalf("UpdateVirtualKey: %v", err)
	}

	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		RequireAPIKey:   true,
		Store:           s,
		Governance:      governance.New(s, ratelimit.NewLimiter()),
		InferenceEngine: routeInferenceEngine{response: routeChatResponseWithUsage()},
	})

	req, err := http.NewRequest(http.MethodPost, baseURL+"/v1/chat/completions", strings.NewReader(`{"model":"gpt-4o","messages":[]}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+raw)
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("POST /v1/chat/completions: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (virtual key inactive, not 401)", resp.StatusCode)
	}
}
