package proxy

import (
	"context"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

func TestChunkErrorVariants(t *testing.T) {
	if chunkError(providers.StreamChunk{}) != nil {
		t.Fatal("no error chunk should yield nil")
	}
	if err := chunkError(providers.StreamChunk{Error: &providers.StreamError{Message: "boom"}}); err == nil || err.Error() != "boom" {
		t.Fatalf("message err = %v", err)
	}
	if err := chunkError(providers.StreamChunk{Error: &providers.StreamError{Code: "code1"}}); err == nil || err.Error() != "code1" {
		t.Fatalf("code err = %v", err)
	}
	if err := chunkError(providers.StreamChunk{Error: &providers.StreamError{}}); err == nil || err.Error() != "stream error" {
		t.Fatalf("default err = %v", err)
	}
}

func TestDispatchStreamFallbackRotatesOnStreamError(t *testing.T) {
	s := openProxyTestStore(t)
	createNamedProxyConnection(t, s, "openai", "primary", "sk-1")
	createNamedProxyConnection(t, s, "openai", "secondary", "sk-2")

	// First connection's stream errors with a fallback-worthy chunk error;
	// engine should rotate to the next connection which streams content.
	openAI := &fakeProvider{
		name: providers.ProviderOpenAI,
		streams: []<-chan providers.StreamChunk{
			errorChunkStream("rate limit exceeded"),
			contentChunkStream("chunk-ok", "hello"),
		},
	}
	engine := NewEngine(s)
	engine.Register(openAI)

	stream, err := engine.DispatchStream(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
	if err != nil {
		t.Fatalf("DispatchStream: %v", err)
	}
	got, ok := <-stream
	if !ok {
		t.Fatal("stream closed before first chunk")
	}
	if got.ID != "chunk-ok" {
		t.Fatalf("chunk ID = %q, want chunk-ok after rotation", got.ID)
	}
}

func TestDispatchStreamNonFallbackStreamErrorReturned(t *testing.T) {
	s := openProxyTestStore(t)
	createNamedProxyConnection(t, s, "openai", "primary", "sk-1")
	openAI := &fakeProvider{
		name:    providers.ProviderOpenAI,
		streams: []<-chan providers.StreamChunk{errorChunkStream("invalid request")},
	}
	engine := NewEngine(s)
	engine.Register(openAI)
	if _, err := engine.DispatchStream(context.Background(), &providers.ChatRequest{Model: "gpt-4o"}); err == nil {
		t.Fatal("non-fallback stream error: want error")
	}
}

func TestDispatchStreamCleanCompletionNoChunks(t *testing.T) {
	s := openProxyTestStore(t)
	createNamedProxyConnection(t, s, "openai", "primary", "sk-1")
	empty := make(chan providers.StreamChunk)
	close(empty)
	openAI := &fakeProvider{
		name:    providers.ProviderOpenAI,
		streams: []<-chan providers.StreamChunk{empty},
	}
	engine := NewEngine(s)
	engine.Register(openAI)
	stream, err := engine.DispatchStream(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
	if err != nil {
		t.Fatalf("DispatchStream: %v", err)
	}
	if _, ok := <-stream; ok {
		t.Fatal("expected closed empty stream")
	}
}

func TestDispatchStreamErrorOnStreamStartRotates(t *testing.T) {
	s := openProxyTestStore(t)
	createNamedProxyConnection(t, s, "openai", "primary", "sk-1")
	createNamedProxyConnection(t, s, "openai", "secondary", "sk-2")
	openAI := &fakeProvider{
		name:       providers.ProviderOpenAI,
		streamErrs: []error{errors.New("server error 500"), nil},
		streams:    []<-chan providers.StreamChunk{nil, contentChunkStream("ok", "hi")},
	}
	engine := NewEngine(s)
	engine.Register(openAI)
	stream, err := engine.DispatchStream(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
	if err != nil {
		t.Fatalf("DispatchStream: %v", err)
	}
	if got := <-stream; got.ID != "ok" {
		t.Fatalf("chunk = %q, want ok after start-error rotation", got.ID)
	}
}

func TestComboDispatchReturnsLastErrorWhenAllStepsFail(t *testing.T) {
	s := openProxyTestStore(t)
	if err := s.CreateCombo(&store.Combo{
		Name: "failer",
		Steps: []store.ComboStep{
			{Provider: "openai", Model: "gpt-4o"},
			{Provider: "anthropic", Model: "claude-sonnet-4"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}
	createNamedProxyConnection(t, s, "openai", "p", "sk-1")
	createNamedProxyConnection(t, s, "anthropic", "p", "sk-2")
	openAI := &fakeProvider{name: providers.ProviderOpenAI, err: errors.New("openai boom")}
	anthropic := &fakeProvider{name: providers.ProviderAnthropic, err: errors.New("anthropic boom")}
	engine := NewEngine(s)
	engine.Register(openAI)
	engine.Register(anthropic)

	if _, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "combo/failer"}); err == nil {
		t.Fatal("combo all-fail: want error")
	}
}

func TestComboResolveUnknownCombo(t *testing.T) {
	s := openProxyTestStore(t)
	engine := NewEngine(s)
	if _, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "combo/missing"}); err == nil {
		t.Fatal("unknown combo: want error")
	}
	if _, err := engine.DispatchStream(context.Background(), &providers.ChatRequest{Model: "combo/missing"}); err == nil {
		t.Fatal("unknown combo stream: want error")
	}
}

func TestComboStreamReturnsLastErrorWhenAllStepsFail(t *testing.T) {
	s := openProxyTestStore(t)
	if err := s.CreateCombo(&store.Combo{
		Name: "failstream",
		Steps: []store.ComboStep{
			{Provider: "openai", Model: "gpt-4o"},
			{Provider: "anthropic", Model: "claude-sonnet-4"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}
	createNamedProxyConnection(t, s, "openai", "p", "sk-1")
	createNamedProxyConnection(t, s, "anthropic", "p", "sk-2")
	openAI := &fakeProvider{name: providers.ProviderOpenAI, streamErrs: []error{errors.New("boom")}}
	anthropic := &fakeProvider{name: providers.ProviderAnthropic, streamErrs: []error{errors.New("boom")}}
	engine := NewEngine(s)
	engine.Register(openAI)
	engine.Register(anthropic)
	if _, err := engine.DispatchStream(context.Background(), &providers.ChatRequest{Model: "combo/failstream"}); err == nil {
		t.Fatal("combo stream all-fail: want error")
	}
}
