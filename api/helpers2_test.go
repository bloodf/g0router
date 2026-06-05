package api

import (
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/usage"
	"github.com/valyala/fasthttp"
)

func TestProviderFromModel(t *testing.T) {
	cases := map[string]string{
		"claude-sonnet-4":    "anthropic", // resolved via the model catalog
		"gpt-4o":             "openai",
		"gpt-unknown-prefix": "openai",    // gpt- prefix fallback (not in catalog)
		"claude-3-5-sonnet":  "anthropic", // claude- prefix fallback
		"totally-unknown-xx": "unknown",
	}
	for model, want := range cases {
		if got := providerFromModel(model); got != want {
			t.Fatalf("providerFromModel(%q) = %q, want %q", model, got, want)
		}
	}
}

func TestAuthTypeForRequest(t *testing.T) {
	if got := authTypeForRequest(true); got != "api_key" {
		t.Fatalf("authTypeForRequest(true) = %q", got)
	}
	if got := authTypeForRequest(false); got != "noauth" {
		t.Fatalf("authTypeForRequest(false) = %q", got)
	}
}

func TestCostForUsageUnknownProvider(t *testing.T) {
	if got := costForUsage(nil, "", "m", nil); got != nil {
		t.Fatalf("empty provider cost = %v, want nil", got)
	}
	if got := costForUsage(nil, "unknown", "m", nil); got != nil {
		t.Fatalf("unknown provider cost = %v, want nil", got)
	}
}

func TestCostForUsageComputes(t *testing.T) {
	u := &usage.Usage{InputTokens: 1000, OutputTokens: 500, TotalTokens: 1500}
	got := costForUsage(nil, "anthropic", "claude-sonnet-4", u)
	if got == nil {
		t.Fatal("expected a computed cost for a known anthropic model")
	}
	if *got < 0 {
		t.Fatalf("cost = %v, want non-negative", *got)
	}
}

func TestCostForUsageUnpricedModelReturnsNil(t *testing.T) {
	// A non-empty provider with a model the catalog cannot price -> the
	// CalculateCost error branch returns nil.
	u := &usage.Usage{InputTokens: 10, OutputTokens: 10, TotalTokens: 20}
	if got := costForUsage(nil, "anthropic", "no-such-priced-model-xyz", u); got != nil {
		t.Fatalf("unpriced model cost = %v, want nil", got)
	}
}

func TestSanitizedLogErrorDispatchError(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	got := sanitizedLogError(ctx, errors.New("boom"), 200)
	if got == nil || *got == "" {
		t.Fatalf("dispatch error = %v, want classified string", got)
	}
}

func TestSanitizedLogErrorSuccessStatus(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	if got := sanitizedLogError(ctx, nil, 200); got != nil {
		t.Fatalf("success status = %v, want nil", got)
	}
}

func TestSanitizedLogErrorOpenAIBody(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	ctx.Response.SetBody([]byte(`{"error":{"message":"bad input","code":"invalid_request"}}`))
	got := sanitizedLogError(ctx, nil, 400)
	if got == nil || *got != "invalid_request: bad input" {
		t.Fatalf("openai body error = %v", got)
	}
}

func TestSanitizedLogErrorOpenAIBodyNoCode(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	ctx.Response.SetBody([]byte(`{"error":{"message":"oops"}}`))
	got := sanitizedLogError(ctx, nil, 400)
	if got == nil || *got != "request_error: oops" {
		t.Fatalf("openai body no-code error = %v", got)
	}
}

func TestSanitizedLogErrorSimpleBody(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	ctx.Response.SetBody([]byte(`{"error":"plain failure"}`))
	got := sanitizedLogError(ctx, nil, 500)
	if got == nil || *got != "request_error: plain failure" {
		t.Fatalf("simple body error = %v", got)
	}
}

func TestSanitizedLogErrorFallback(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	ctx.Response.SetBody([]byte("not json"))
	got := sanitizedLogError(ctx, nil, 503)
	if got == nil || *got != "request_error: status 503" {
		t.Fatalf("fallback error = %v", got)
	}
}
