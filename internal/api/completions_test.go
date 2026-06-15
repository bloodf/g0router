package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
)

// fakeCompletionsResolver resolves completions models to a fake provider.
type fakeCompletionsResolver struct {
	prov schemas.Provider
}

func (r *fakeCompletionsResolver) Resolve(model string) (schemas.Provider, schemas.Key, error) {
	return r.prov, schemas.Key{Provider: "openai"}, nil
}

// fakeCompletionsProvider records TextCompletion / TextCompletionStream calls.
// It embeds fakeMessagesProvider to satisfy the full schemas.Provider interface.
type fakeCompletionsProvider struct {
	fakeMessagesProvider
	completionCalled bool
	streamCalled     bool
	capturedKey      schemas.Key
	resp             *schemas.TextCompletionResponse
	perr             *schemas.ProviderError
	streamCh         chan *schemas.StreamChunk
}

func (p *fakeCompletionsProvider) TextCompletion(_ *schemas.GatewayContext, key schemas.Key, _ *schemas.TextCompletionRequest) (*schemas.TextCompletionResponse, *schemas.ProviderError) {
	p.completionCalled = true
	p.capturedKey = key
	if p.perr != nil {
		return nil, p.perr
	}
	return p.resp, nil
}

func (p *fakeCompletionsProvider) TextCompletionStream(_ *schemas.GatewayContext, _ schemas.PostHookRunner, key schemas.Key, _ *schemas.TextCompletionRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	p.streamCalled = true
	p.capturedKey = key
	if p.perr != nil {
		return nil, p.perr
	}
	return p.streamCh, nil
}

// TestCompletionsHandlerNonStreamSuccess verifies a non-streaming success
// returns the bare OpenAI TextCompletionResponse (NOT the admin {data,error}
// envelope).
func TestCompletionsHandlerNonStreamSuccess(t *testing.T) {
	prov := &fakeCompletionsProvider{
		resp: &schemas.TextCompletionResponse{
			ID:     "cmpl-1",
			Object: "text_completion",
			Model:  "gpt-3.5-turbo-instruct",
			Choices: []schemas.TextCompletionChoice{
				{Text: "hello world", Index: 0, FinishReason: "stop"},
			},
		},
	}
	h := &CompletionsHandler{router: &fakeCompletionsResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/completions")
	ctx.Request.SetBody([]byte(`{"model":"gpt-3.5-turbo-instruct","prompt":"hi"}`))
	h.Handle(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if !prov.completionCalled {
		t.Fatal("provider TextCompletion not called")
	}

	body := ctx.Response.Body()
	var resp schemas.TextCompletionResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Choices) != 1 || resp.Choices[0].Text != "hello world" {
		t.Errorf("choices = %+v, want one with text 'hello world'", resp.Choices)
	}

	// OpenAI shape proof: no admin {data,error} wrapper keys at top level.
	var top map[string]json.RawMessage
	if err := json.Unmarshal(body, &top); err != nil {
		t.Fatalf("unmarshal top-level: %v", err)
	}
	if _, ok := top["data"]; ok {
		t.Error("response has top-level 'data' wrapper (admin envelope leaked)")
	}
	if _, ok := top["error"]; ok {
		t.Error("response has top-level 'error' key on success")
	}
}

// TestCompletionsHandlerInvalidJSON verifies a malformed body returns a 400
// OpenAI error shape.
func TestCompletionsHandlerInvalidJSON(t *testing.T) {
	prov := &fakeCompletionsProvider{}
	h := &CompletionsHandler{router: &fakeCompletionsResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/completions")
	ctx.Request.SetBody([]byte(`{not json`))
	h.Handle(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
	if prov.completionCalled {
		t.Fatal("provider should not be called on invalid JSON")
	}
	var resp struct {
		Error struct {
			Type string `json:"type"`
		} `json:"error"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.Error.Type != "invalid_request_error" {
		t.Errorf("error.type = %q, want invalid_request_error", resp.Error.Type)
	}
}

// TestCompletionsHandlerProviderError verifies a provider *ProviderError is
// surfaced with its status code (e.g. a 501 from an unimplemented provider).
func TestCompletionsHandlerProviderError(t *testing.T) {
	prov := &fakeCompletionsProvider{
		perr: &schemas.ProviderError{StatusCode: 501, Type: "not_implemented", Message: "text_completion not implemented"},
	}
	h := &CompletionsHandler{router: &fakeCompletionsResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/completions")
	ctx.Request.SetBody([]byte(`{"model":"some-model","prompt":"hi"}`))
	h.Handle(&ctx)

	if ctx.Response.StatusCode() != 501 {
		t.Fatalf("status = %d, want 501", ctx.Response.StatusCode())
	}
}

// TestCompletionsHandlerStream verifies stream:true sets the SSE content type
// and frames a [DONE] terminator.
func TestCompletionsHandlerStream(t *testing.T) {
	ch := make(chan *schemas.StreamChunk, 2)
	ch <- &schemas.StreamChunk{ID: "cmpl-1", Object: "text_completion"}
	close(ch)

	prov := &fakeCompletionsProvider{streamCh: ch}
	h := &CompletionsHandler{router: &fakeCompletionsResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/completions")
	ctx.Request.SetBody([]byte(`{"model":"gpt-3.5-turbo-instruct","prompt":"hi","stream":true}`))
	h.Handle(&ctx)

	if !prov.streamCalled {
		t.Fatal("provider TextCompletionStream not called")
	}
	ct := string(ctx.Response.Header.ContentType())
	if ct != "text/event-stream" {
		t.Errorf("content-type = %q, want text/event-stream", ct)
	}
	body := string(ctx.Response.Body())
	if !contains(body, "[DONE]") {
		t.Errorf("stream body missing [DONE] terminator: %q", body)
	}
}

// TestCompletionsHandlerMarshalFailureFallsBackTo500 verifies that when the
// response marshal seam fails, the handler falls back to a plain-text 500
// (AUD-010), matching the embeddings handler contract.
func TestCompletionsHandlerMarshalFailureFallsBackTo500(t *testing.T) {
	prev := jsonMarshal
	t.Cleanup(func() { jsonMarshal = prev })
	jsonMarshal = func(v any) ([]byte, error) {
		return nil, errors.New("simulated marshal failure")
	}

	router := inference.NewRouter(translation.NewRegistry())
	h := NewCompletionsHandler(router)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/completions")
	ctx.Request.SetBody([]byte(`{"model":"gpt-3.5-turbo-instruct","prompt":"hi"}`))
	h.Handle(&ctx)

	if got := ctx.Response.StatusCode(); got != fasthttp.StatusInternalServerError {
		t.Errorf("status = %d, want %d", got, fasthttp.StatusInternalServerError)
	}
	if got := string(ctx.Response.Body()); got != "internal error" {
		t.Errorf("body = %q, want %q", got, "internal error")
	}
}

// TestCompletionsVKDenied verifies the x-g0-vk gate denies before dispatch.
func TestCompletionsVKDenied(t *testing.T) {
	resolver := newFakeVKResolver()
	resolver.set("vk-denied", &VKInfo{
		Key: "vk-denied",
		Configs: []VKProviderConfig{
			{Provider: "openai", AllowedModels: []string{"gpt-3.5-turbo-instruct"}},
		},
		IsActive: true,
	})
	quota := newFakeVKQuotaChecker(struct {
		ok     bool
		status int
		reason string
	}{ok: false, status: 429, reason: "budget exhausted"})

	prov := &fakeCompletionsProvider{}
	h := &CompletionsHandler{router: &fakeCompletionsResolver{prov: prov}}
	h.SetVKGate(NewVKGate(resolver, quota))

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/completions")
	ctx.Request.Header.Set("x-g0-vk", "vk-denied")
	ctx.Request.SetBody([]byte(`{"model":"gpt-3.5-turbo-instruct","prompt":"hi"}`))
	h.Handle(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", ctx.Response.StatusCode())
	}
	if prov.completionCalled {
		t.Fatal("provider TextCompletion should not be called")
	}
}

// TestCompletionsHandle_VKPinnedKeyOverridesDispatch verifies PAR-ROUTE-030
// pinning for the /v1/completions handler.
func TestCompletionsHandle_VKPinnedKeyOverridesDispatch(t *testing.T) {
	resolver := newFakeVKResolver()
	resolver.set("vk-pinned", &VKInfo{
		Key: "vk-pinned",
		Configs: []VKProviderConfig{
			{Provider: "openai", AllowedModels: []string{"gpt-3.5-turbo-instruct"}, KeyIDs: []string{"conn-2"}},
		},
		IsActive: true,
	})

	prov := &fakeCompletionsProvider{
		resp: &schemas.TextCompletionResponse{ID: "cmpl-1", Object: "text_completion"},
	}
	h := &CompletionsHandler{router: &fakeCompletionsResolver{prov: prov}}
	h.SetVKGate(NewVKGate(resolver, newFakeVKQuotaChecker()))
	h.SetVKPinnedResolver(&fakePinnedKeyResolver{connID: "conn-2", credential: "cred-2", ok: true})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/completions")
	ctx.Request.Header.Set("x-g0-vk", "vk-pinned")
	ctx.Request.SetBody([]byte(`{"model":"gpt-3.5-turbo-instruct","prompt":"hi"}`))
	h.Handle(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if !prov.completionCalled {
		t.Fatal("provider TextCompletion not called")
	}
	if prov.capturedKey.ID != "conn-2" {
		t.Errorf("key.ID = %q, want conn-2", prov.capturedKey.ID)
	}
	if prov.capturedKey.Value != "cred-2" {
		t.Errorf("key.Value = %q, want cred-2", prov.capturedKey.Value)
	}
}
