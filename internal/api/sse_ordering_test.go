package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
)

// streamOpenErrResolver resolves to a provider whose ChatCompletionStream fails
// to open (returns a *ProviderError), exercising the SSE-setup-ordering path.
type streamOpenErrResolver struct {
	perr *schemas.ProviderError
}

func (r *streamOpenErrResolver) ResolveForModel(_ *schemas.ChatRequest) (schemas.Provider, schemas.Key, error) {
	return &streamOpenErrProvider{perr: r.perr}, schemas.Key{Provider: "openai"}, nil
}

type streamOpenErrProvider struct {
	fakeResponsesProvider
	perr *schemas.ProviderError
}

func (p *streamOpenErrProvider) ChatCompletionStream(_ *schemas.GatewayContext, _ schemas.PostHookRunner, _ schemas.Key, _ *schemas.ChatRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	return nil, p.perr
}

// assertJSONErrorNotSSE checks the response carries an application/json error
// body with the real status code, and NOT a text/event-stream content type.
func assertJSONErrorNotSSE(t *testing.T, ctx *fasthttp.RequestCtx, wantStatus int) {
	t.Helper()
	if ctx.Response.StatusCode() != wantStatus {
		t.Errorf("status = %d, want %d", ctx.Response.StatusCode(), wantStatus)
	}
	ct := string(ctx.Response.Header.ContentType())
	if strings.Contains(ct, "text/event-stream") {
		t.Errorf("content-type = %q must NOT be text/event-stream on a stream-open error", ct)
	}
	if !strings.Contains(ct, "application/json") {
		t.Errorf("content-type = %q, want application/json", ct)
	}
	var got map[string]any
	if err := json.Unmarshal(ctx.Response.Body(), &got); err != nil {
		t.Fatalf("error body is not JSON: %v; body=%s", err, ctx.Response.Body())
	}
	if _, ok := got["error"]; !ok {
		t.Errorf("error body missing \"error\" key; body=%s", ctx.Response.Body())
	}
}

// TestChatStreamOpenErrorReturnsJSON verifies PAR-BF-OAI-201: when the provider
// stream fails to open, the chat handler returns an application/json error with
// the provider's real status code, not a text/event-stream framing mismatch.
func TestChatStreamOpenErrorReturnsJSON(t *testing.T) {
	h := &ChatHandler{router: &streamOpenErrResolver{perr: &schemas.ProviderError{StatusCode: 503, Type: "server_error", Message: "upstream down"}}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.SetBody([]byte(`{"model":"gpt-4","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
	h.Handle(&ctx)

	assertJSONErrorNotSSE(t, &ctx, 503)
}

// TestResponsesStreamOpenErrorReturnsJSON verifies the same ordering fix for the
// /v1/responses streaming handler.
func TestResponsesStreamOpenErrorReturnsJSON(t *testing.T) {
	h := &ResponsesHandler{router: &streamOpenErrResolver{perr: &schemas.ProviderError{StatusCode: 503, Type: "server_error", Message: "upstream down"}}, registry: translation.NewRegistry()}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/responses")
	ctx.Request.SetBody([]byte(`{"model":"gpt-4","input":[{"role":"user","content":"hi"}]}`))
	h.Handle(&ctx)

	assertJSONErrorNotSSE(t, &ctx, 503)
}
