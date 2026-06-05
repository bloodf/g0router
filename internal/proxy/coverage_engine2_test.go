package proxy

import (
	"context"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/usage"
)

func TestAnnotateDispatchResponseNil(t *testing.T) {
	// Should be a no-op and not panic.
	annotateDispatchResponse(nil, providers.Key{})
}

func TestDispatchNoConnectionReturnsError(t *testing.T) {
	s := openProxyTestStore(t)
	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "x"}}
	engine := NewEngine(s)
	engine.Register(openAI)
	// No connection stored for openai -> providerForRoute returns no connections.
	if _, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"}); err == nil {
		t.Fatal("dispatch without connection: want error")
	}
	if openAI.called {
		t.Fatal("provider should not be called without a connection")
	}
}

func TestDispatchStreamNoConnectionReturnsError(t *testing.T) {
	s := openProxyTestStore(t)
	openAI := &fakeProvider{name: providers.ProviderOpenAI}
	engine := NewEngine(s)
	engine.Register(openAI)
	if _, err := engine.DispatchStream(context.Background(), &providers.ChatRequest{Model: "gpt-4o"}); err == nil {
		t.Fatal("dispatch stream without connection: want error")
	}
}

func TestDispatchStreamQuotaExhaustionBlocks(t *testing.T) {
	s := openProxyTestStore(t)
	createNamedProxyConnection(t, s, "openai", "p", "sk-1")
	openAI := &fakeProvider{name: providers.ProviderOpenAI, stream: contentChunkStream("c", "hi")}
	engine := NewEngine(s)
	engine.Register(openAI)
	engine.RegisterQuotaFetcher(providers.ProviderOpenAI, &fakeQuotaFetcher{
		quota: usage.Quota{Limit: 100, Used: 100, Remaining: 0},
	})
	if _, err := engine.DispatchStream(context.Background(), &providers.ChatRequest{Model: "gpt-4o"}); err == nil {
		t.Fatal("stream quota exhausted: want error")
	}
	if openAI.streamed {
		t.Fatal("provider stream should not be called when quota exhausted")
	}
}

func TestDispatchUnknownModelReturnsError(t *testing.T) {
	s := openProxyTestStore(t)
	engine := NewEngine(s)
	if _, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "no-such-model-xyz"}); err == nil {
		t.Fatal("unknown model: want error")
	}
	if _, err := engine.DispatchStream(context.Background(), &providers.ChatRequest{Model: "no-such-model-xyz"}); err == nil {
		t.Fatal("unknown model stream: want error")
	}
}
