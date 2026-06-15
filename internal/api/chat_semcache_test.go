package api

import (
	"context"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// cacheProbe is a hermetic fake SemanticCache that records calls so tests can
// prove short-circuit / no-op / write-through behavior without any store,
// network, clock, or embedder (bf-core-2 D9).
type cacheProbe struct {
	enabled      bool
	hit          []byte // non-nil ⇒ Lookup reports a hit returning these bytes
	enabledCalls int
	lookupCalls  int
	storeCalls   int
	storedModel  string
	storedResp   []byte
}

func (c *cacheProbe) Enabled() bool {
	c.enabledCalls++
	return c.enabled
}

func (c *cacheProbe) Lookup(_ context.Context, _, _ string) ([]byte, bool, error) {
	c.lookupCalls++
	if c.hit != nil {
		return c.hit, true, nil
	}
	return nil, false, nil
}

func (c *cacheProbe) Store(_ context.Context, model, _ string, response []byte) error {
	c.storeCalls++
	c.storedModel = model
	c.storedResp = response
	return nil
}

// countingProvider records how many times ChatCompletion / ChatCompletionStream
// were invoked, so a cache short-circuit (count 0) is provable.
type countingProvider struct {
	fakeMessagesProvider
	resp         *schemas.ChatResponse
	chatCalls    int
	streamCalls  int
}

func (p *countingProvider) ChatCompletion(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.ChatRequest) (*schemas.ChatResponse, *schemas.ProviderError) {
	p.chatCalls++
	return p.resp, nil
}

func (p *countingProvider) ChatCompletionStream(_ *schemas.GatewayContext, _ schemas.PostHookRunner, _ schemas.Key, _ *schemas.ChatRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	p.streamCalls++
	return p.fakeMessagesProvider.streamCh, nil
}

const semcacheBody = `{"model":"gpt-4","messages":[{"role":"user","content":"hi"}]}`

func semcacheHandler(prov schemas.Provider, cache SemanticCache) *ChatHandler {
	h := &ChatHandler{router: &testProviderResolver{prov: prov}}
	if cache != nil {
		h.SetSemanticCache(cache)
	}
	return h
}

func TestChatSemanticCache_NilCacheNoOp(t *testing.T) {
	prov := &countingProvider{resp: &schemas.ChatResponse{ID: "r1", Object: "chat.completion"}}
	h := semcacheHandler(prov, nil)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.Set("Accept", "application/json")
	ctx.Request.SetBody([]byte(semcacheBody))
	h.Handle(&ctx)

	if prov.chatCalls != 1 {
		t.Fatalf("nil cache: provider chat calls = %d, want 1", prov.chatCalls)
	}
}

func TestChatSemanticCache_FlagOffNoOp(t *testing.T) {
	prov := &countingProvider{resp: &schemas.ChatResponse{ID: "r1", Object: "chat.completion"}}
	cache := &cacheProbe{enabled: false, hit: []byte(`{"id":"cached"}`)}
	h := semcacheHandler(prov, cache)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.Set("Accept", "application/json")
	ctx.Request.SetBody([]byte(semcacheBody))
	h.Handle(&ctx)

	if prov.chatCalls != 1 {
		t.Fatalf("flag off: provider chat calls = %d, want 1", prov.chatCalls)
	}
	if cache.lookupCalls != 0 {
		t.Fatalf("flag off: Lookup calls = %d, want 0 (no read when disabled)", cache.lookupCalls)
	}
	if cache.storeCalls != 0 {
		t.Fatalf("flag off: Store calls = %d, want 0 (no write when disabled)", cache.storeCalls)
	}
}

func TestChatSemanticCache_HitShortCircuits(t *testing.T) {
	prov := &countingProvider{resp: &schemas.ChatResponse{ID: "live", Object: "chat.completion"}}
	cached := []byte(`{"id":"cached","object":"chat.completion"}`)
	cache := &cacheProbe{enabled: true, hit: cached}
	h := semcacheHandler(prov, cache)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.Set("Accept", "application/json")
	ctx.Request.SetBody([]byte(semcacheBody))
	h.Handle(&ctx)

	// SHORT-CIRCUIT PROOF: provider NOT called on a cache hit.
	if prov.chatCalls != 0 {
		t.Fatalf("cache hit: provider chat calls = %d, want 0 (short-circuit)", prov.chatCalls)
	}
	if got := string(ctx.Response.Body()); got != string(cached) {
		t.Fatalf("cache hit body = %q, want cached bytes %q", got, cached)
	}
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("cache hit status = %d, want 200", ctx.Response.StatusCode())
	}
	ct := string(ctx.Response.Header.ContentType())
	if !strings.Contains(ct, "application/json") {
		t.Fatalf("cache hit content-type = %q, want application/json", ct)
	}
	// No write-through on a hit.
	if cache.storeCalls != 0 {
		t.Fatalf("cache hit: Store calls = %d, want 0", cache.storeCalls)
	}
}

func TestChatSemanticCache_MissWritesThrough(t *testing.T) {
	prov := &countingProvider{resp: &schemas.ChatResponse{ID: "live", Object: "chat.completion"}}
	cache := &cacheProbe{enabled: true} // hit nil ⇒ miss
	h := semcacheHandler(prov, cache)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.Set("Accept", "application/json")
	ctx.Request.SetBody([]byte(semcacheBody))
	h.Handle(&ctx)

	if prov.chatCalls != 1 {
		t.Fatalf("cache miss: provider chat calls = %d, want 1", prov.chatCalls)
	}
	if cache.lookupCalls != 1 {
		t.Fatalf("cache miss: Lookup calls = %d, want 1", cache.lookupCalls)
	}
	if cache.storeCalls != 1 {
		t.Fatalf("cache miss: Store calls = %d, want 1 (write-through)", cache.storeCalls)
	}
	if cache.storedModel != "gpt-4" {
		t.Fatalf("cache miss: stored model = %q, want gpt-4", cache.storedModel)
	}
	if len(cache.storedResp) == 0 {
		t.Fatal("cache miss: stored response is empty, want the marshaled provider response")
	}
}

func TestChatSemanticCache_StreamNeverConsulted(t *testing.T) {
	ch := make(chan *schemas.StreamChunk)
	close(ch)
	prov := &countingProvider{fakeMessagesProvider: fakeMessagesProvider{streamCh: ch}}
	cache := &cacheProbe{enabled: true, hit: []byte(`{"id":"cached"}`)}
	h := semcacheHandler(prov, cache)

	var ctx fasthttp.RequestCtx
	// stream:true ⇒ streaming branch; the cache must never be consulted (D6).
	ctx.Request.SetBody([]byte(`{"model":"gpt-4","messages":[{"role":"user","content":"hi"}],"stream":true}`))
	h.Handle(&ctx)

	if prov.streamCalls != 1 {
		t.Fatalf("stream: provider stream calls = %d, want 1", prov.streamCalls)
	}
	if cache.lookupCalls != 0 {
		t.Fatalf("stream: Lookup calls = %d, want 0 (cache never consulted on stream)", cache.lookupCalls)
	}
	if cache.storeCalls != 0 {
		t.Fatalf("stream: Store calls = %d, want 0 (cache never consulted on stream)", cache.storeCalls)
	}
}
