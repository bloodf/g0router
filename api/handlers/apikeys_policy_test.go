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
	var created struct {
		Key struct {
			ID               string   `json:"id"`
			Scopes           []string `json:"scopes"`
			RateLimitRPM     *int     `json:"rate_limit_rpm"`
			RateLimitTPM     *int     `json:"rate_limit_tpm"`
			DailySpendCapUSD *float64 `json:"daily_spend_cap_usd"`
			ExpiresAt        *int64   `json:"expires_at"`
		} `json:"key"`
		Raw string `json:"raw"`
	}
	decodeJSON(t, body, &created)
	if len(created.Key.Scopes) != 1 || created.Key.Scopes[0] != "gpt-*" {
		t.Fatalf("scopes = %v", created.Key.Scopes)
	}
	if created.Key.RateLimitRPM == nil || *created.Key.RateLimitRPM != 60 {
		t.Fatalf("rpm = %v", created.Key.RateLimitRPM)
	}
	if created.Key.DailySpendCapUSD == nil || *created.Key.DailySpendCapUSD != 5.5 {
		t.Fatalf("cap = %v", created.Key.DailySpendCapUSD)
	}
	if created.Key.ExpiresAt == nil || *created.Key.ExpiresAt != 4102444800 {
		t.Fatalf("expires_at = %v", created.Key.ExpiresAt)
	}

	// Update policy via PUT.
	putBody := `{"rate_limit_rpm":120,"scopes":["claude-*"]}`
	ctx, body = runHandler(t, fasthttp.MethodPut, putBody, func(ctx *fasthttp.RequestCtx) {
		APIKeys(ctx, s, "test-secret", created.Key.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("update status = %d; body=%s", ctx.Response.StatusCode(), body)
	}
	var updated struct {
		Key struct {
			RateLimitRPM *int     `json:"rate_limit_rpm"`
			Scopes       []string `json:"scopes"`
		} `json:"key"`
	}
	decodeJSON(t, body, &updated)
	if updated.Key.RateLimitRPM == nil || *updated.Key.RateLimitRPM != 120 {
		t.Fatalf("updated rpm = %v", updated.Key.RateLimitRPM)
	}
	if len(updated.Key.Scopes) != 1 || updated.Key.Scopes[0] != "claude-*" {
		t.Fatalf("updated scopes = %v", updated.Key.Scopes)
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
	var created struct {
		Key struct{ ID string `json:"id"` } `json:"key"`
	}
	decodeJSON(t, ctx.Response.Body(), &created)
	ctx, body := runHandler(t, fasthttp.MethodPut, `{"rate_limit_rpm":-5}`, func(ctx *fasthttp.RequestCtx) {
		APIKeys(ctx, s, "test-secret", created.Key.ID)
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
