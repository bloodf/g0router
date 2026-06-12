package api

import (
	"errors"
	"net/http"
	"testing"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
)

// fakeEmbeddingsResolver resolves embeddings models to a fake provider.
type fakeEmbeddingsResolver struct {
	prov schemas.Provider
}

func (r *fakeEmbeddingsResolver) Resolve(model string) (schemas.Provider, schemas.Key, error) {
	return r.prov, schemas.Key{Provider: "openai"}, nil
}

// fakeEmbeddingsProvider records Embedding calls.
type fakeEmbeddingsProvider struct {
	fakeMessagesProvider
	embeddingCalled bool
}

func (p *fakeEmbeddingsProvider) Embedding(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.EmbeddingRequest) (*schemas.EmbeddingResponse, *schemas.ProviderError) {
	p.embeddingCalled = true
	return &schemas.EmbeddingResponse{Object: "list"}, nil
}

// TestEmbeddingsHandlerMarshalFailureFallsBackTo500 verifies that when
// the response marshal seam fails, the embeddings handler eventually
// writes a 500 status (AUD-010). The provider will fail with a network
// error in this test environment, which routes through writeError —
// writeError then exercises the same failing jsonMarshal seam and falls
// back to a plain-text 500 per the AUD-009–012 acceptance contract.
func TestEmbeddingsHandlerMarshalFailureFallsBackTo500(t *testing.T) {
	prev := jsonMarshal
	t.Cleanup(func() { jsonMarshal = prev })

	jsonMarshal = func(v any) ([]byte, error) {
		return nil, errors.New("simulated marshal failure")
	}

	router := inference.NewRouter(translation.NewRegistry())
	h := NewEmbeddingsHandler(router)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/embeddings")
	ctx.Request.SetBody([]byte(`{"model":"text-embedding-3-small","input":"hello"}`))
	h.Handle(&ctx)

	if got := ctx.Response.StatusCode(); got != fasthttp.StatusInternalServerError {
		t.Errorf("status = %d, want %d", got, fasthttp.StatusInternalServerError)
	}
	if got := string(ctx.Response.Body()); got != "internal error" {
		t.Errorf("body = %q, want %q", got, "internal error")
	}
}

func TestEmbeddingsVKDenied(t *testing.T) {
	resolver := newFakeVKResolver()
	resolver.set("vk-denied", &VKInfo{
		Key: "vk-denied",
		Configs: []VKProviderConfig{
			{Provider: "openai", AllowedModels: []string{"text-embedding-ada-002"}},
		},
		IsActive: true,
	})
	quota := newFakeVKQuotaChecker(struct {
		ok     bool
		status int
		reason string
	}{ok: false, status: 429, reason: "budget exhausted"})

	prov := &fakeEmbeddingsProvider{}
	h := &EmbeddingsHandler{router: &fakeEmbeddingsResolver{prov: prov}}
	h.SetVKGate(NewVKGate(resolver, quota))

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/embeddings")
	ctx.Request.Header.Set("x-g0-vk", "vk-denied")
	ctx.Request.SetBody([]byte(`{"model":"text-embedding-ada-002","input":"hello"}`))
	h.Handle(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", ctx.Response.StatusCode())
	}
	if prov.embeddingCalled {
		t.Fatal("provider Embedding should not be called")
	}
}
