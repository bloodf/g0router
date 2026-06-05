package api

import (
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/valyala/fasthttp"
)

// TestEnforceKeyPolicyTPMDenied covers the token-rate-limit 429 branch.
func TestEnforceKeyPolicyTPMDenied(t *testing.T) {
	s := newPolicyServer()
	identity := APIKeyIdentity{ID: "id-tpm", RateLimitTPM: intPtr(10)}
	// Exhaust the TPM budget.
	s.limiter.AddTokens("id-tpm", 20)
	ctx := policyCtx("gpt-4o", identity)
	if s.enforceKeyPolicy(ctx) {
		t.Fatal("over-TPM request must be rejected")
	}
	if ctx.Response.StatusCode() != fasthttp.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", ctx.Response.StatusCode())
	}
}

// TestEnforceKeyPolicyNoLimiterPasses covers the s.limiter == nil fast-path.
func TestEnforceKeyPolicyNoLimiterPasses(t *testing.T) {
	s := &Server{} // limiter is nil
	ctx := policyCtx("gpt-4o", APIKeyIdentity{ID: "id-nolim"})
	if !s.enforceKeyPolicy(ctx) {
		t.Fatal("nil limiter must always pass")
	}
}

// TestModelFromBodyInvalidJSON covers the json.Unmarshal error branch.
func TestModelFromBodyInvalidJSON(t *testing.T) {
	if got := modelFromBody([]byte("not-json")); got != "" {
		t.Fatalf("invalid JSON should return empty string, got %q", got)
	}
}

// TestModelFromBodyMissingField covers valid JSON with no "model" key.
func TestModelFromBodyMissingField(t *testing.T) {
	if got := modelFromBody([]byte(`{"messages":[]}`)); got != "" {
		t.Fatalf("missing model field should return empty string, got %q", got)
	}
}

// TestWritePolicyErrorSetsBodyAndContentType covers the happy path of writePolicyError.
func TestWritePolicyErrorSetsBodyAndContentType(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	writePolicyError(ctx, fasthttp.StatusForbidden, "nope")
	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("status = %d, want 403", ctx.Response.StatusCode())
	}
	ct := string(ctx.Response.Header.ContentType())
	if ct != "application/json" {
		t.Fatalf("content-type = %q, want application/json", ct)
	}
}

// TestRecordKeyUsageStreamUsagePath covers the streamUsage != nil branch.
func TestRecordKeyUsageStreamUsagePath(t *testing.T) {
	s := newPolicyServer()
	s.config.Store = newAPITestStore(t)
	keyID := "id-stream"
	streamUsage := &providers.Usage{
		PromptTokens:     50,
		CompletionTokens: 50,
		TotalTokens:      100,
	}
	req := &providers.ChatRequest{Model: "gpt-4o"}
	s.recordKeyUsage(&keyID, "gpt-4o", req, nil, streamUsage)

	if s.limiter.SpendToday(keyID) <= 0 {
		t.Fatal("spend must be recorded from stream usage")
	}
	// 100 tokens recorded; tpm=101 must allow, tpm=100 must deny.
	if !s.limiter.AllowTokens(keyID, intPtr(101)) {
		t.Fatal("101 tpm must allow after 100 tokens recorded")
	}
}

// TestRecordKeyUsageEmptyKeyNoop covers the *keyID == "" branch.
func TestRecordKeyUsageEmptyKeyNoop(t *testing.T) {
	s := newPolicyServer()
	empty := ""
	s.recordKeyUsage(&empty, "", nil, nil, nil)
	// No panic and no state written.
}

// TestRecordKeyUsageNilResponseAndNilUsageNoop covers extracted == nil path.
func TestRecordKeyUsageNilResponseAndNilUsageNoop(t *testing.T) {
	s := newPolicyServer()
	keyID := "id-noop"
	s.recordKeyUsage(&keyID, "", nil, nil, nil)
	if s.limiter.SpendToday(keyID) != 0 {
		t.Fatal("no usage provided must not record spend")
	}
}
