package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
)

// fakeInputTokensResolver returns a provider whose CountTokens is controllable.
type fakeInputTokensResolver struct {
	captured    *schemas.ChatRequest
	tokens      int
	countErr    *schemas.ProviderError
	resolveErr  error
	lastProv    *fakeInputTokensProvider
}

func (f *fakeInputTokensResolver) ResolveForModel(req *schemas.ChatRequest) (schemas.Provider, schemas.Key, error) {
	f.captured = req
	if f.resolveErr != nil {
		return nil, schemas.Key{}, f.resolveErr
	}
	prov := &fakeInputTokensProvider{tokens: f.tokens, countErr: f.countErr}
	f.lastProv = prov
	return prov, schemas.Key{Provider: "openai"}, nil
}

type fakeInputTokensProvider struct {
	fakeResponsesProvider
	tokens      int
	countErr    *schemas.ProviderError
	countCalled bool
	capturedKey schemas.Key
}

func (p *fakeInputTokensProvider) CountTokens(_ *schemas.GatewayContext, key schemas.Key, _ *schemas.ChatRequest) (*schemas.TokenCountResponse, *schemas.ProviderError) {
	p.countCalled = true
	p.capturedKey = key
	if p.countErr != nil {
		return nil, p.countErr
	}
	return &schemas.TokenCountResponse{Tokens: p.tokens}, nil
}

func newInputTokensCtx(body string) *fasthttp.RequestCtx {
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/responses/input_tokens")
	ctx.Request.SetBody([]byte(body))
	return &ctx
}

// TestInputTokensReturnsBareTokenCount verifies the handler translates the
// responses-shaped body, dispatches CountTokens, and emits the bare
// {"tokens":N} OpenAI shape (no {data,error} admin wrapper) (PAR-BF-OAI-004).
func TestInputTokensReturnsBareTokenCount(t *testing.T) {
	fake := &fakeInputTokensResolver{tokens: 42}
	h := &InputTokensHandler{router: fake, registry: translation.NewRegistry()}

	ctx := newInputTokensCtx(`{"model":"gpt-4","input":[{"role":"user","content":"hi"}]}`)
	h.Handle(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	ct := string(ctx.Response.Header.ContentType())
	if !strings.Contains(ct, "application/json") {
		t.Errorf("content-type = %q, want application/json", ct)
	}
	if strings.Contains(ct, "text/event-stream") {
		t.Errorf("content-type = %q must not be event-stream (non-streaming route)", ct)
	}
	body := ctx.Response.Body()
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal response: %v; body=%s", err, body)
	}
	if _, hasData := got["data"]; hasData {
		t.Errorf("response must NOT carry admin {data} wrapper; body=%s", body)
	}
	if _, hasErr := got["error"]; hasErr {
		t.Errorf("response must NOT carry an error; body=%s", body)
	}
	if tok, ok := got["tokens"].(float64); !ok || int(tok) != 42 {
		t.Errorf("tokens = %v, want 42; body=%s", got["tokens"], body)
	}
	if fake.captured == nil {
		t.Fatal("resolver was never called")
	}
	if !fake.lastProv.countCalled {
		t.Error("provider CountTokens was not called")
	}
}

func TestInputTokensRejectsInvalidJSON(t *testing.T) {
	fake := &fakeInputTokensResolver{tokens: 1}
	h := &InputTokensHandler{router: fake, registry: translation.NewRegistry()}

	ctx := newInputTokensCtx(`{not json`)
	h.Handle(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Errorf("status = %d, want 400", ctx.Response.StatusCode())
	}
	if fake.lastProv != nil && fake.lastProv.countCalled {
		t.Error("CountTokens must not be called on invalid JSON")
	}
}

func TestInputTokensResolveErrorIs400(t *testing.T) {
	fake := &fakeInputTokensResolver{resolveErr: errors.New("no provider for model")}
	h := &InputTokensHandler{router: fake, registry: translation.NewRegistry()}

	ctx := newInputTokensCtx(`{"model":"nope","input":[{"role":"user","content":"hi"}]}`)
	h.Handle(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Errorf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

// TestInputTokensProviderErrorPassthrough verifies a provider 501 (the default
// stub state for non-openai providers) is surfaced with its status code.
func TestInputTokensProviderErrorPassthrough(t *testing.T) {
	fake := &fakeInputTokensResolver{countErr: &schemas.ProviderError{StatusCode: 501, Type: "not_implemented", Message: "count_tokens not implemented"}}
	h := &InputTokensHandler{router: fake, registry: translation.NewRegistry()}

	ctx := newInputTokensCtx(`{"model":"gpt-4","input":[{"role":"user","content":"hi"}]}`)
	h.Handle(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusNotImplemented {
		t.Errorf("status = %d, want 501", ctx.Response.StatusCode())
	}
}

func TestInputTokensVKDenied(t *testing.T) {
	resolver := newFakeVKResolver()
	resolver.set("vk-denied", &VKInfo{
		Key: "vk-denied",
		Configs: []VKProviderConfig{
			{Provider: "openai", AllowedModels: []string{"gpt-3.5-turbo"}},
		},
		IsActive: true,
	})
	quota := newFakeVKQuotaChecker(struct {
		ok     bool
		status int
		reason string
	}{ok: false, status: 403, reason: "model not allowed for virtual key"})

	fake := &fakeInputTokensResolver{tokens: 5}
	h := &InputTokensHandler{router: fake, registry: translation.NewRegistry()}
	h.SetVKGate(NewVKGate(resolver, quota))

	ctx := newInputTokensCtx(`{"model":"gpt-4","input":[{"role":"user","content":"hi"}]}`)
	ctx.Request.Header.Set("x-g0-vk", "vk-denied")
	h.Handle(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("status = %d, want 403", ctx.Response.StatusCode())
	}
	if fake.lastProv != nil && fake.lastProv.countCalled {
		t.Error("CountTokens must not be called when VK denied")
	}
}

func TestInputTokensVKPinnedOverridesKey(t *testing.T) {
	resolver := newFakeVKResolver()
	resolver.set("vk-pinned", &VKInfo{
		Key: "vk-pinned",
		Configs: []VKProviderConfig{
			{Provider: "openai", AllowedModels: []string{"gpt-4"}, KeyIDs: []string{"conn-2"}},
		},
		IsActive: true,
	})

	fake := &fakeInputTokensResolver{tokens: 3}
	h := &InputTokensHandler{router: fake, registry: translation.NewRegistry()}
	h.SetVKGate(NewVKGate(resolver, newFakeVKQuotaChecker()))
	h.SetVKPinnedResolver(&fakePinnedKeyResolver{connID: "conn-2", credential: "cred-2", ok: true})

	ctx := newInputTokensCtx(`{"model":"gpt-4","input":[{"role":"user","content":"hi"}]}`)
	ctx.Request.Header.Set("x-g0-vk", "vk-pinned")
	h.Handle(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if fake.lastProv == nil {
		t.Fatal("provider not resolved")
	}
	if fake.lastProv.capturedKey.ID != "conn-2" {
		t.Errorf("key.ID = %q, want conn-2", fake.lastProv.capturedKey.ID)
	}
	if fake.lastProv.capturedKey.Value != "cred-2" {
		t.Errorf("key.Value = %q, want cred-2", fake.lastProv.capturedKey.Value)
	}
}

// TestInputTokensMarshalFailure verifies a marshal failure falls back to a
// plain-text 500 (mirrors responses.go).
func TestInputTokensMarshalFailure(t *testing.T) {
	fake := &fakeInputTokensResolver{tokens: 9}
	h := &InputTokensHandler{router: fake, registry: translation.NewRegistry()}

	orig := jsonMarshal
	jsonMarshal = func(any) ([]byte, error) { return nil, errors.New("boom") }
	defer func() { jsonMarshal = orig }()

	ctx := newInputTokensCtx(`{"model":"gpt-4","input":[{"role":"user","content":"hi"}]}`)
	h.Handle(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Errorf("status = %d, want 500", ctx.Response.StatusCode())
	}
}
