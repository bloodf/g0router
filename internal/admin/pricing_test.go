package admin

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestPricingPatchValidation(t *testing.T) {
	env := newTestEnv(t)
	wireUsageServices(t, env)
	token := loginToken(t, env)
	authHeader := map[string]string{"Authorization": "Bearer " + token}

	// Unknown field.
	status, envl := call(t, env.handlers.RequireSession(env.handlers.PatchPricing), "PATCH", "/api/pricing",
		`{"openai":{"gpt-4o":{"foo":1}}}`, nil, authHeader)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("unknown field status = %d, want 400", status)
	}
	if msg := errMessage(t, envl); msg == "" {
		t.Fatal("expected error message for unknown field")
	}

	// Negative value.
	status, envl = call(t, env.handlers.RequireSession(env.handlers.PatchPricing), "PATCH", "/api/pricing",
		`{"openai":{"gpt-4o":{"input":-1}}}`, nil, authHeader)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("negative value status = %d, want 400", status)
	}
	if msg := errMessage(t, envl); msg == "" {
		t.Fatal("expected error message for negative value")
	}

	// Valid update persists.
	status, envl = call(t, env.handlers.RequireSession(env.handlers.PatchPricing), "PATCH", "/api/pricing",
		`{"openai":{"gpt-4o":{"input":1.5,"output":2.5}}}`, nil, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("valid patch status = %d err=%q", status, errMessage(t, envl))
	}
	pricing := dataField[map[string]map[string]map[string]float64](t, envl)
	openai := pricing["openai"]
	if openai == nil {
		t.Fatalf("pricing missing openai: %v", pricing)
	}
	gpt4o := openai["gpt-4o"]
	if gpt4o["input"] != 1.5 || gpt4o["output"] != 2.5 {
		t.Errorf("gpt-4o pricing = %v, want input=1.5 output=2.5", gpt4o)
	}
}

func TestPricingDelete(t *testing.T) {
	env := newTestEnv(t)
	wireUsageServices(t, env)
	token := loginToken(t, env)
	authHeader := map[string]string{"Authorization": "Bearer " + token}

	// Seed a user override.
	status, _ := call(t, env.handlers.RequireSession(env.handlers.PatchPricing), "PATCH", "/api/pricing",
		`{"openai":{"gpt-4o":{"input":1.5},"gpt-4o-mini":{"input":0.5}},"anthropic":{"claude":{"input":2}}}`, nil, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("seed patch status = %d", status)
	}

	// Reset single model.
	status, envl := call(t, env.handlers.RequireSession(env.handlers.DeletePricing), "DELETE", "/api/pricing?provider=openai&model=gpt-4o", "", nil, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete model status = %d err=%q", status, errMessage(t, envl))
	}
	pricing := dataField[map[string]map[string]map[string]float64](t, envl)
	openai := pricing["openai"]
	if _, ok := openai["gpt-4o"]; ok {
		t.Error("gpt-4o still present after model delete")
	}
	if _, ok := openai["gpt-4o-mini"]; !ok {
		t.Error("gpt-4o-mini should be preserved")
	}

	// Reset provider.
	status, envl = call(t, env.handlers.RequireSession(env.handlers.DeletePricing), "DELETE", "/api/pricing?provider=openai", "", nil, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete provider status = %d err=%q", status, errMessage(t, envl))
	}
	pricing = dataField[map[string]map[string]map[string]float64](t, envl)
	if _, ok := pricing["openai"]; ok {
		t.Error("openai user overrides still present after provider delete")
	}

	// Reset all.
	status, envl = call(t, env.handlers.RequireSession(env.handlers.DeletePricing), "DELETE", "/api/pricing", "", nil, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete all status = %d", status)
	}
	pricing = dataField[map[string]map[string]map[string]float64](t, envl)
	anthropic := pricing["anthropic"]
	if _, ok := anthropic["claude"]; ok {
		t.Error("user-added claude override should be removed after reset all")
	}
}
