package proxy

import (
	"context"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

// errCapableProvider is a multimodal-capable provider whose methods always
// return the injected error. Used to cover the dispatch-error branches in
// Embeddings, GenerateImages, TranscribeAudio, and Speech.
type errCapableProvider struct {
	fakeProvider
	dispatchErr error
}

func (f *errCapableProvider) Embeddings(_ context.Context, _ providers.Key, _ *providers.EmbeddingsRequest) (*providers.EmbeddingsResponse, error) {
	return nil, f.dispatchErr
}
func (f *errCapableProvider) GenerateImages(_ context.Context, _ providers.Key, _ *providers.ImagesRequest) (*providers.ImagesResponse, error) {
	return nil, f.dispatchErr
}
func (f *errCapableProvider) TranscribeAudio(_ context.Context, _ providers.Key, _ *providers.AudioTranscriptionRequest) (*providers.AudioResponse, error) {
	return nil, f.dispatchErr
}
func (f *errCapableProvider) Speech(_ context.Context, _ providers.Key, _ *providers.SpeechRequest) ([]byte, string, error) {
	return nil, "", f.dispatchErr
}

func newErrMultimodalEngine(t *testing.T, dispatchErr error) *Engine {
	t.Helper()
	p := &errCapableProvider{
		fakeProvider: fakeProvider{name: providers.ProviderOpenAI},
		dispatchErr:  dispatchErr,
	}
	return newMultimodalEngine(t, p)
}

func TestEmbeddingsDispatchError(t *testing.T) {
	engine := newErrMultimodalEngine(t, errors.New("upstream embeddings failure"))
	_, err := engine.Embeddings(context.Background(), &providers.EmbeddingsRequest{Model: "gpt-4o"})
	if err == nil {
		t.Fatal("Embeddings should propagate provider error")
	}
}

func TestGenerateImagesDispatchError(t *testing.T) {
	engine := newErrMultimodalEngine(t, errors.New("image gen failed"))
	_, err := engine.GenerateImages(context.Background(), &providers.ImagesRequest{Model: "gpt-4o", Prompt: "x"})
	if err == nil {
		t.Fatal("GenerateImages should propagate error")
	}
}

func TestTranscribeAudioDispatchError(t *testing.T) {
	engine := newErrMultimodalEngine(t, errors.New("transcription failed"))
	_, err := engine.TranscribeAudio(context.Background(), &providers.AudioTranscriptionRequest{Model: "gpt-4o"})
	if err == nil {
		t.Fatal("TranscribeAudio should propagate error")
	}
}

func TestSpeechDispatchError(t *testing.T) {
	engine := newErrMultimodalEngine(t, errors.New("speech synthesis failed"))
	_, _, err := engine.Speech(context.Background(), &providers.SpeechRequest{Model: "gpt-4o", Input: "hi", Voice: "alloy"})
	if err == nil {
		t.Fatal("Speech should propagate error")
	}
}

// TestTranscribeAudioUnsupportedProvider exercises the !ok capability branch for
// TranscribeAudio (no test exists for it in multimodal_test.go).
func TestTranscribeAudioUnsupportedProvider(t *testing.T) {
	plain := &fakeProvider{name: providers.ProviderOpenAI}
	engine := newMultimodalEngine(t, plain)
	_, err := engine.TranscribeAudio(context.Background(), &providers.AudioTranscriptionRequest{Model: "gpt-4o"})
	if !errors.Is(err, ErrCapabilityUnsupported) {
		t.Fatalf("error = %v, want ErrCapabilityUnsupported", err)
	}
}

// Nil-request tests for GenerateImages, TranscribeAudio, Speech (Embeddings nil
// is already covered in multimodal_test.go).

func TestGenerateImagesNilRequest(t *testing.T) {
	engine := newErrMultimodalEngine(t, nil)
	if _, err := engine.GenerateImages(context.Background(), nil); err == nil {
		t.Fatal("GenerateImages(nil) should error")
	}
}

func TestTranscribeAudioNilRequest(t *testing.T) {
	engine := newErrMultimodalEngine(t, nil)
	if _, err := engine.TranscribeAudio(context.Background(), nil); err == nil {
		t.Fatal("TranscribeAudio(nil) should error")
	}
}

func TestSpeechNilRequest(t *testing.T) {
	engine := newErrMultimodalEngine(t, nil)
	if _, _, err := engine.Speech(context.Background(), nil); err == nil {
		t.Fatal("Speech(nil) should error")
	}
}

// providerFor-error tests (no registered provider for the model).

func TestGenerateImagesProviderNotFound(t *testing.T) {
	s := openProxyTestStore(t)
	engine := NewEngine(s) // no providers registered
	if _, err := engine.GenerateImages(context.Background(), &providers.ImagesRequest{Model: "no-model"}); err == nil {
		t.Fatal("GenerateImages with no provider should error")
	}
}

func TestTranscribeAudioProviderNotFound(t *testing.T) {
	s := openProxyTestStore(t)
	engine := NewEngine(s)
	if _, err := engine.TranscribeAudio(context.Background(), &providers.AudioTranscriptionRequest{Model: "no-model"}); err == nil {
		t.Fatal("TranscribeAudio with no provider should error")
	}
}

func TestSpeechProviderNotFound(t *testing.T) {
	s := openProxyTestStore(t)
	engine := NewEngine(s)
	if _, _, err := engine.Speech(context.Background(), &providers.SpeechRequest{Model: "no-model"}); err == nil {
		t.Fatal("Speech with no provider should error")
	}
}
