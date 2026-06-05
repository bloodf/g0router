package api

import (
	"net"
	"testing"

	"github.com/bloodf/g0router/api/handlers"
	"github.com/bloodf/g0router/internal/ratelimit"
	"github.com/valyala/fasthttp"
)

// TestHandleLoggedInferencePolicyFails exercises the enforceKeyPolicy returning
// false branch inside handleLoggedInference (line 391): when policy fails,
// the inference handler is NOT called.
func TestHandleLoggedInferencePolicyFails(t *testing.T) {
	srv := &Server{limiter: ratelimit.NewLimiter()}

	var req fasthttp.Request
	req.Header.SetMethod(fasthttp.MethodPost)
	req.SetRequestURI("/v1/chat/completions")
	req.SetBody([]byte(`{"model":"claude-3-opus","messages":[]}`))
	ctx := &fasthttp.RequestCtx{}
	ctx.Init(&req, &net.TCPAddr{}, nil)

	// Set an identity with scopes that exclude the requested model.
	ctx.SetUserValue(requestAPIKeyPolicyKey, APIKeyIdentity{
		ID:     "test-key-scope",
		Scopes: []string{"gpt-*"}, // claude- not in scope
	})

	called := false
	srv.handleLoggedInference(ctx, "openai", func(c *fasthttp.RequestCtx, _ handlers.InferenceEngine) {
		called = true
	})

	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("status = %d, want 403 (policy rejected)", ctx.Response.StatusCode())
	}
	if called {
		t.Fatal("handler should not be called when policy fails")
	}
}
