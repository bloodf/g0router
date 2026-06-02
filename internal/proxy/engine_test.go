package proxy

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

type fakeProvider struct {
	name        providers.ModelProvider
	response    *providers.ChatResponse
	stream      <-chan providers.StreamChunk
	err         error
	called      bool
	streamed    bool
	received    *providers.ChatRequest
	receivedKey providers.Key
}

func (f *fakeProvider) Name() providers.ModelProvider {
	return f.name
}

func (f *fakeProvider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	f.called = true
	f.receivedKey = key
	f.received = req
	return f.response, f.err
}

func (f *fakeProvider) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	f.streamed = true
	f.receivedKey = key
	f.received = req
	return f.stream, f.err
}

func (f *fakeProvider) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	return nil, nil
}

func TestDispatchRoutesToCorrectProvider(t *testing.T) {
	s := openProxyTestStore(t)
	openAIKey := "sk-openai"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "primary",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &openAIKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection openai: %v", err)
	}
	anthropicKey := "sk-anthropic"
	if err := s.CreateConnection(&store.Connection{
		Provider: "anthropic",
		Name:     "backup",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &anthropicKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection anthropic: %v", err)
	}

	openAI := &fakeProvider{
		name: providers.ProviderOpenAI,
		response: &providers.ChatResponse{
			ID:    "chatcmpl-1",
			Model: "gpt-4o",
		},
	}
	anthropic := &fakeProvider{name: providers.ProviderAnthropic}
	engine := NewEngine(s)
	engine.Register(openAI)
	engine.Register(anthropic)

	req := &providers.ChatRequest{Model: "gpt-4o"}
	resp, err := engine.Dispatch(context.Background(), req)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if resp.ID != "chatcmpl-1" {
		t.Fatalf("response ID = %q, want chatcmpl-1", resp.ID)
	}
	if !openAI.called {
		t.Fatal("openai provider was not called")
	}
	if anthropic.called {
		t.Fatal("anthropic provider should not be called")
	}
	if openAI.received != req {
		t.Fatal("provider should receive original request")
	}
	if openAI.receivedKey.Provider != providers.ProviderOpenAI {
		t.Fatalf("key provider = %q, want openai", openAI.receivedKey.Provider)
	}
	if openAI.receivedKey.Value != openAIKey {
		t.Fatalf("key value = %q, want %q", openAI.receivedKey.Value, openAIKey)
	}
	if openAI.receivedKey.ConnID == "" {
		t.Fatal("connection ID should be set")
	}
	if openAI.receivedKey.AuthType != string(store.AuthTypeAPIKey) {
		t.Fatalf("auth type = %q, want api_key", openAI.receivedKey.AuthType)
	}
}

func TestDispatchUnknownModel(t *testing.T) {
	engine := NewEngine(openProxyTestStore(t))
	engine.Register(&fakeProvider{name: providers.ProviderOpenAI})

	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "unknown-model"})
	if !errors.Is(err, ErrProviderNotFound) {
		t.Fatalf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestDispatchNoConnections(t *testing.T) {
	engine := NewEngine(openProxyTestStore(t))
	engine.Register(&fakeProvider{name: providers.ProviderOpenAI})

	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
	if !errors.Is(err, ErrNoConnections) {
		t.Fatalf("expected ErrNoConnections, got %v", err)
	}
}

func TestDispatchStreamReturnsChannel(t *testing.T) {
	s := openProxyTestStore(t)
	token := "token-anthropic"
	if err := s.CreateConnection(&store.Connection{
		Provider:    "anthropic",
		Name:        "oauth",
		AuthType:    store.AuthTypeOAuth,
		AccessToken: &token,
		IsActive:    true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	content := "hello"
	chunks := make(chan providers.StreamChunk, 1)
	chunks <- providers.StreamChunk{
		ID:    "chunk-1",
		Model: "claude-3-5-sonnet",
		Choices: []providers.StreamChoice{
			{Delta: providers.StreamDelta{Content: &content}},
		},
	}
	close(chunks)

	anthropic := &fakeProvider{name: providers.ProviderAnthropic, stream: chunks}
	engine := NewEngine(s)
	engine.Register(anthropic)

	stream, err := engine.DispatchStream(context.Background(), &providers.ChatRequest{Model: "claude-3-5-sonnet"})
	if err != nil {
		t.Fatalf("DispatchStream: %v", err)
	}
	got, ok := <-stream
	if !ok {
		t.Fatal("stream closed before first chunk")
	}
	if got.ID != "chunk-1" {
		t.Fatalf("chunk ID = %q, want chunk-1", got.ID)
	}
	if !anthropic.streamed {
		t.Fatal("anthropic stream provider was not called")
	}
	if anthropic.receivedKey.Value != token {
		t.Fatalf("key value = %q, want %q", anthropic.receivedKey.Value, token)
	}
	if anthropic.receivedKey.AuthType != string(store.AuthTypeOAuth) {
		t.Fatalf("auth type = %q, want oauth", anthropic.receivedKey.AuthType)
	}
}

func openProxyTestStore(t *testing.T) *store.Store {
	t.Helper()

	s, err := store.NewStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	return s
}
