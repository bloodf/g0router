package proxy

import (
	"context"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

// capableFakeProvider embeds fakeProvider and adds the optional multimodal
// capability interfaces so engine routing can be exercised.
type capableFakeProvider struct {
	fakeProvider
	embeddings *providers.EmbeddingsResponse
	images     *providers.ImagesResponse
	audio      *providers.AudioResponse
	speech     []byte
	speechCT   string
	gotKey     providers.Key
}

func (f *capableFakeProvider) Embeddings(ctx context.Context, key providers.Key, req *providers.EmbeddingsRequest) (*providers.EmbeddingsResponse, error) {
	f.gotKey = key
	return f.embeddings, nil
}

func (f *capableFakeProvider) GenerateImages(ctx context.Context, key providers.Key, req *providers.ImagesRequest) (*providers.ImagesResponse, error) {
	f.gotKey = key
	return f.images, nil
}

func (f *capableFakeProvider) TranscribeAudio(ctx context.Context, key providers.Key, req *providers.AudioTranscriptionRequest) (*providers.AudioResponse, error) {
	f.gotKey = key
	return f.audio, nil
}

func (f *capableFakeProvider) Speech(ctx context.Context, key providers.Key, req *providers.SpeechRequest) ([]byte, string, error) {
	f.gotKey = key
	return f.speech, f.speechCT, nil
}

func newMultimodalEngine(t *testing.T, provider providers.Provider) *Engine {
	t.Helper()
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "openai", "sk-openai")
	engine := NewEngine(s)
	engine.Register(provider)
	return engine
}

func TestEngineEmbeddingsRoutesToCapableProvider(t *testing.T) {
	provider := &capableFakeProvider{
		fakeProvider: fakeProvider{name: providers.ProviderOpenAI},
		embeddings:   &providers.EmbeddingsResponse{Object: "list", Model: "gpt-4o"},
	}
	engine := newMultimodalEngine(t, provider)

	resp, err := engine.Embeddings(context.Background(), &providers.EmbeddingsRequest{Model: "gpt-4o", Input: "hi"})
	if err != nil {
		t.Fatalf("Embeddings: %v", err)
	}
	if resp.Object != "list" {
		t.Fatalf("resp = %+v", resp)
	}
	if provider.gotKey.Value != "sk-openai" {
		t.Fatalf("key = %+v", provider.gotKey)
	}
}

func TestEngineImagesRoutesToCapableProvider(t *testing.T) {
	provider := &capableFakeProvider{
		fakeProvider: fakeProvider{name: providers.ProviderOpenAI},
		images:       &providers.ImagesResponse{Created: 1, Data: []providers.ImageData{{URL: "u"}}},
	}
	engine := newMultimodalEngine(t, provider)

	resp, err := engine.GenerateImages(context.Background(), &providers.ImagesRequest{Model: "gpt-4o", Prompt: "cat"})
	if err != nil {
		t.Fatalf("GenerateImages: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].URL != "u" {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestEngineTranscribeRoutesToCapableProvider(t *testing.T) {
	provider := &capableFakeProvider{
		fakeProvider: fakeProvider{name: providers.ProviderOpenAI},
		audio:        &providers.AudioResponse{Text: "hi"},
	}
	engine := newMultimodalEngine(t, provider)

	resp, err := engine.TranscribeAudio(context.Background(), &providers.AudioTranscriptionRequest{Model: "gpt-4o", Filename: "a.mp3", File: []byte("x")})
	if err != nil {
		t.Fatalf("TranscribeAudio: %v", err)
	}
	if resp.Text != "hi" {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestEngineSpeechRoutesToCapableProvider(t *testing.T) {
	provider := &capableFakeProvider{
		fakeProvider: fakeProvider{name: providers.ProviderOpenAI},
		speech:       []byte("MP3"),
		speechCT:     "audio/mpeg",
	}
	engine := newMultimodalEngine(t, provider)

	audio, ct, err := engine.Speech(context.Background(), &providers.SpeechRequest{Model: "gpt-4o", Input: "hi", Voice: "alloy"})
	if err != nil {
		t.Fatalf("Speech: %v", err)
	}
	if string(audio) != "MP3" || ct != "audio/mpeg" {
		t.Fatalf("audio=%q ct=%q", audio, ct)
	}
}

func TestEngineEmbeddingsUnsupportedProvider(t *testing.T) {
	// plain fakeProvider does not implement EmbeddingsProvider.
	provider := &fakeProvider{name: providers.ProviderOpenAI}
	engine := newMultimodalEngine(t, provider)

	_, err := engine.Embeddings(context.Background(), &providers.EmbeddingsRequest{Model: "gpt-4o", Input: "hi"})
	if !errors.Is(err, ErrCapabilityUnsupported) {
		t.Fatalf("expected ErrCapabilityUnsupported, got %v", err)
	}
}

func TestEngineImagesUnsupportedProvider(t *testing.T) {
	provider := &fakeProvider{name: providers.ProviderOpenAI}
	engine := newMultimodalEngine(t, provider)
	if _, err := engine.GenerateImages(context.Background(), &providers.ImagesRequest{Model: "gpt-4o", Prompt: "x"}); !errors.Is(err, ErrCapabilityUnsupported) {
		t.Fatalf("expected ErrCapabilityUnsupported, got %v", err)
	}
}

func TestEngineSpeechUnsupportedProvider(t *testing.T) {
	provider := &fakeProvider{name: providers.ProviderOpenAI}
	engine := newMultimodalEngine(t, provider)
	if _, _, err := engine.Speech(context.Background(), &providers.SpeechRequest{Model: "gpt-4o", Input: "x", Voice: "alloy"}); !errors.Is(err, ErrCapabilityUnsupported) {
		t.Fatalf("expected ErrCapabilityUnsupported, got %v", err)
	}
}

func TestEngineEmbeddingsNilRequest(t *testing.T) {
	provider := &fakeProvider{name: providers.ProviderOpenAI}
	engine := newMultimodalEngine(t, provider)
	if _, err := engine.Embeddings(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestEngineEmbeddingsProviderNotFound(t *testing.T) {
	provider := &capableFakeProvider{fakeProvider: fakeProvider{name: providers.ProviderOpenAI}}
	engine := newMultimodalEngine(t, provider)
	if _, err := engine.Embeddings(context.Background(), &providers.EmbeddingsRequest{Model: "no-such-model-xyz"}); !errors.Is(err, ErrProviderNotFound) {
		t.Fatalf("expected ErrProviderNotFound, got %v", err)
	}
}
