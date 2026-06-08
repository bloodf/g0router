package handlers

import (
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

func TestAPIKeysCreateWithPolicyAndNoSecretLeak(t *testing.T) {
	s := newHandlerStore(t)

	reqBody := `{"name":"scoped","scopes":["gpt-*"],"rate_limit_rpm":60,"rate_limit_tpm":10000,"daily_spend_cap_usd":5.5,"expires_at":4102444800}`
	ctx, body := runHandler(t, fasthttp.MethodPost, reqBody, func(ctx *fasthttp.RequestCtx) {
		APIKeys(ctx, s, "test-secret", "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d; body=%s", ctx.Response.StatusCode(), body)
	}
	if strings.Contains(string(body), "key_hash") || strings.Contains(string(body), "secret") {
		t.Fatalf("response leaks secret material: %s", body)
	}
	var created apiKeyView
	decodeJSON(t, body, &created)
	if len(created.Scopes) != 1 || created.Scopes[0] != "gpt-*" {
		t.Fatalf("scopes = %v", created.Scopes)
	}
	if created.RateLimitRPM == nil || *created.RateLimitRPM != 60 {
		t.Fatalf("rpm = %v", created.RateLimitRPM)
	}
	if created.DailySpendCap == nil || *created.DailySpendCap != 5.5 {
		t.Fatalf("cap = %v", created.DailySpendCap)
	}
	if created.ExpiresAt == nil || *created.ExpiresAt != 4102444800 {
		t.Fatalf("expires_at = %v", created.ExpiresAt)
	}

	// Update policy via PUT.
	putBody := `{"rate_limit_rpm":120,"scopes":["claude-*"]}`
	ctx, body = runHandler(t, fasthttp.MethodPut, putBody, func(ctx *fasthttp.RequestCtx) {
		APIKeys(ctx, s, "test-secret", created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("update status = %d; body=%s", ctx.Response.StatusCode(), body)
	}
	var updated apiKeyView
	decodeJSON(t, body, &updated)
	if updated.RateLimitRPM == nil || *updated.RateLimitRPM != 120 {
		t.Fatalf("updated rpm = %v", updated.RateLimitRPM)
	}
	if len(updated.Scopes) != 1 || updated.Scopes[0] != "claude-*" {
		t.Fatalf("updated scopes = %v", updated.Scopes)
	}
}

func TestAPIKeysPolicyInList(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"name":"k","scopes":["gpt-4*"]}`, func(ctx *fasthttp.RequestCtx) {
		APIKeys(ctx, s, "test-secret", "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d; body=%s", ctx.Response.StatusCode(), body)
	}
	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		APIKeys(ctx, s, "test-secret", "")
	})
	if !strings.Contains(string(body), "gpt-4*") {
		t.Fatalf("list missing scopes: %s", body)
	}
	if strings.Contains(string(body), "key_hash") {
		t.Fatalf("list leaks hash: %s", body)
	}
}

func TestAPIKeysCreateInvalidPolicyRejected(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"name":"bad","rate_limit_rpm":-5}`, func(ctx *fasthttp.RequestCtx) {
		APIKeys(ctx, s, "test-secret", "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestAPIKeysUpdatePolicyInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"name":"k"}`, func(ctx *fasthttp.RequestCtx) {
		APIKeys(ctx, s, "test-secret", "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create failed: %d", ctx.Response.StatusCode())
	}
	ctx, body := runHandler(t, fasthttp.MethodPut, `{bad`, func(ctx *fasthttp.RequestCtx) {
		APIKeys(ctx, s, "test-secret", "some-id")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestAPIKeysUpdateInvalidPolicyRejected(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"name":"k"}`, func(ctx *fasthttp.RequestCtx) {
		APIKeys(ctx, s, "test-secret", "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create failed: %d", ctx.Response.StatusCode())
	}
	var created apiKeyView
	decodeJSON(t, ctx.Response.Body(), &created)
	ctx, body := runHandler(t, fasthttp.MethodPut, `{"rate_limit_rpm":-5}`, func(ctx *fasthttp.RequestCtx) {
		APIKeys(ctx, s, "test-secret", created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestAPIKeysUpdateEmptyIDRejected(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodPut, `{"rate_limit_rpm":10}`, func(ctx *fasthttp.RequestCtx) {
		APIKeys(ctx, s, "test-secret", "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}
