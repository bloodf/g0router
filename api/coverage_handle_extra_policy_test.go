package api

import (
	"net"
	"testing"

	"github.com/bloodf/g0router/api/handlers"
	"github.com/bloodf/g0router/internal/ratelimit"
	"github.com/valyala/fasthttp"
)

// TestHandleExtraEnforcePolicyFails exercises the enforceKeyPolicy returning false
// inside handleExtra (lines 375-377): when the identity's model scope fails,
// handleExtra returns 403 without calling the handler.
func TestHandleExtraEnforcePolicyFails(t *testing.T) {
	srv := &Server{limiter: ratelimit.NewLimiter()}

	var req fasthttp.Request
	req.Header.SetMethod(fasthttp.MethodPost)
	req.SetRequestURI("/v1/embeddings")
	req.SetBody([]byte(`{"model":"claude-3-opus","input":"hi"}`))
	ctx := &fasthttp.RequestCtx{}
	ctx.Init(&req, &net.TCPAddr{}, nil)

	// Set an identity with scopes that exclude the requested model (claude- not in gpt-*).
	ctx.SetUserValue(requestAPIKeyPolicyKey, APIKeyIdentity{
		ID:     "test-key-id",
		Scopes: []string{"gpt-*"},
	})

	called := false
	srv.handleExtra(ctx, func(ctx *fasthttp.RequestCtx, _ handlers.ExtraEngine) {
		called = true
	})

	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("status = %d, want 403 (policy rejected)", ctx.Response.StatusCode())
	}
	if called {
		t.Fatal("handler should not be called when policy fails")
	}
}

// TestHandleExtraPostPolicyPassesCallsHandler exercises the full POST path
// through handleExtra when enforceKeyPolicy passes (no identity set → passes
// trivially) and the handler is called with a nil ExtraEngine.
func TestHandleExtraPostPolicyPassesCallsHandler(t *testing.T) {
	srv := &Server{limiter: ratelimit.NewLimiter()}

	var req fasthttp.Request
	req.Header.SetMethod(fasthttp.MethodPost)
	req.SetRequestURI("/v1/embeddings")
	req.SetBody([]byte(`{"model":"gpt-4o","input":"hi"}`))
	ctx := &fasthttp.RequestCtx{}
	ctx.Init(&req, &net.TCPAddr{}, nil)
	// No identity set → enforceKeyPolicy returns true.

	called := false
	var gotEngine handlers.ExtraEngine
	srv.handleExtra(ctx, func(c *fasthttp.RequestCtx, e handlers.ExtraEngine) {
		called = true
		gotEngine = e
	})

	if !called {
		t.Fatal("handler should be called when policy passes")
	}
	// Engine is nil (no InferenceEngine configured).
	if gotEngine != nil {
		t.Fatalf("engine = %v, want nil (no InferenceEngine)", gotEngine)
	}
}
