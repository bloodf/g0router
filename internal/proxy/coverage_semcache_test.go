package proxy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/semcache"
	"github.com/bloodf/g0router/internal/store"
)

type fakeSemanticCacher struct {
	lookupResp *semcache.CachedResponse
	lookupHit  bool
	lookupErr  error
	storeErr   error
}

func (f *fakeSemanticCacher) Lookup(ctx context.Context, key, model string, promptFn func() string) (*semcache.CachedResponse, bool, error) {
	return f.lookupResp, f.lookupHit, f.lookupErr
}

func (f *fakeSemanticCacher) Store(ctx context.Context, key, model, prompt string, resp *semcache.CachedResponse, ttl time.Duration) error {
	return f.storeErr
}

func TestSemanticCacheKeyStability(t *testing.T) {
	req := &providers.ChatRequest{Model: "gpt-4o", Messages: []providers.Message{{Role: "user", Content: "hello"}}}
	key1 := semanticCacheKey(req)
	key2 := semanticCacheKey(req)
	if key1 != key2 {
		t.Fatalf("cache key not stable: %q vs %q", key1, key2)
	}
}

func TestPromptFromChatRequest(t *testing.T) {
	req := &providers.ChatRequest{Messages: []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}}
	prompt := promptFromChatRequest(req)
	if prompt != "hello\nhi" {
		t.Fatalf("prompt = %q, want hello\\nhi", prompt)
	}
}

func TestPromptFromChatRequestNil(t *testing.T) {
	if promptFromChatRequest(nil) != "" {
		t.Fatal("expected empty prompt for nil request")
	}
}

func TestChatResponseRoundTrip(t *testing.T) {
	reason := "stop"
	original := &providers.ChatResponse{
		ID:      "chat-1",
		Object:  "chat.completion",
		Created: 123,
		Model:   "gpt-4o",
		Choices: []providers.Choice{{
			Index:        0,
			Message:      providers.Message{Role: "assistant", Content: "hello"},
			FinishReason: &reason,
		}},
		Usage: &providers.Usage{PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3},
	}
	cached := chatResponseToCachedResponse(original)
	if cached == nil {
		t.Fatal("expected non-nil cached response")
	}
	back := cachedResponseToChatResponse(cached)
	if back == nil {
		t.Fatal("expected non-nil response")
	}
	if back.ID != original.ID {
		t.Fatalf("ID = %q, want %q", back.ID, original.ID)
	}
	if back.Choices[0].Message.Content != "hello" {
		t.Fatalf("content = %q, want hello", back.Choices[0].Message.Content)
	}
}

func TestChatResponseToCachedResponseNil(t *testing.T) {
	if chatResponseToCachedResponse(nil) != nil {
		t.Fatal("expected nil")
	}
}

func TestCachedResponseToChatResponseNil(t *testing.T) {
	if cachedResponseToChatResponse(nil) != nil {
		t.Fatal("expected nil")
	}
}

func TestRegisterSemanticCache(t *testing.T) {
	engine := NewEngine(nil)
	fake := &fakeSemanticCacher{}
	engine.RegisterSemanticCache(&semcache.Cache{})
	if engine.semanticCache == nil {
		t.Fatal("expected semantic cache to be registered")
	}
	_ = fake
}

func TestDispatchSemanticCacheHit(t *testing.T) {
	s := openProxyTestStore(t)
	key := "sk-openai"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai", Name: "p", AuthType: store.AuthTypeAPIKey, APIKey: &key, IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	openAI := &fakeProvider{
		name:     providers.ProviderOpenAI,
		response: &providers.ChatResponse{ID: "1", Model: "gpt-4o"},
	}
	engine := NewEngine(s)
	engine.Register(openAI)

	cached := &semcache.CachedResponse{ID: "cached-1", Model: "gpt-4o"}
	engine.semanticCache = &fakeSemanticCacher{lookupResp: cached, lookupHit: true}

	resp, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o", Messages: []providers.Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if resp.ID != "cached-1" {
		t.Fatalf("expected cached response, got %q", resp.ID)
	}
	if openAI.called {
		t.Fatal("provider should not be called on cache hit")
	}
}

func TestDispatchSemanticCacheMiss(t *testing.T) {
	s := openProxyTestStore(t)
	key := "sk-openai"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai", Name: "p", AuthType: store.AuthTypeAPIKey, APIKey: &key, IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	openAI := &fakeProvider{
		name:     providers.ProviderOpenAI,
		response: &providers.ChatResponse{ID: "live-1", Model: "gpt-4o"},
	}
	engine := NewEngine(s)
	engine.Register(openAI)

	engine.semanticCache = &fakeSemanticCacher{lookupHit: false}

	resp, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o", Messages: []providers.Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if resp.ID != "live-1" {
		t.Fatalf("expected live response, got %q", resp.ID)
	}
	if !openAI.called {
		t.Fatal("provider should be called on cache miss")
	}
}

func TestDispatchSemanticCacheLookupError(t *testing.T) {
	s := openProxyTestStore(t)
	key := "sk-openai"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai", Name: "p", AuthType: store.AuthTypeAPIKey, APIKey: &key, IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	openAI := &fakeProvider{
		name:     providers.ProviderOpenAI,
		response: &providers.ChatResponse{ID: "live-1", Model: "gpt-4o"},
	}
	engine := NewEngine(s)
	engine.Register(openAI)

	engine.semanticCache = &fakeSemanticCacher{lookupErr: errors.New("lookup fail")}

	resp, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o", Messages: []providers.Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if resp.ID != "live-1" {
		t.Fatalf("expected live response after lookup error, got %q", resp.ID)
	}
}

func TestDispatchSemanticCacheStoreError(t *testing.T) {
	s := openProxyTestStore(t)
	key := "sk-openai"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai", Name: "p", AuthType: store.AuthTypeAPIKey, APIKey: &key, IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	openAI := &fakeProvider{
		name:     providers.ProviderOpenAI,
		response: &providers.ChatResponse{ID: "live-1", Model: "gpt-4o"},
	}
	engine := NewEngine(s)
	engine.Register(openAI)

	engine.semanticCache = &fakeSemanticCacher{storeErr: errors.New("store fail")}

	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o", Messages: []providers.Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatalf("Dispatch should not fail on store error: %v", err)
	}
}

func TestDispatchSemanticCacheSkippedForStreaming(t *testing.T) {
	s := openProxyTestStore(t)
	key := "sk-openai"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai", Name: "p", AuthType: store.AuthTypeAPIKey, APIKey: &key, IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	openAI := &fakeProvider{
		name:     providers.ProviderOpenAI,
		response: &providers.ChatResponse{ID: "live-1", Model: "gpt-4o"},
	}
	engine := NewEngine(s)
	engine.Register(openAI)

	fakeCache := &fakeSemanticCacher{lookupHit: true, lookupResp: &semcache.CachedResponse{ID: "cached"}}
	engine.semanticCache = fakeCache

	stream := true
	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o", Stream: &stream, Messages: []providers.Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if !openAI.called {
		t.Fatal("provider should be called for streaming requests (cache skipped)")
	}
}
