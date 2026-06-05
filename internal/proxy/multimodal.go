package proxy

import (
	"context"
	"fmt"

	"github.com/bloodf/g0router/internal/providers"
)

// Embeddings resolves the provider for req.Model and dispatches to it when the
// provider implements providers.EmbeddingsProvider; otherwise it returns
// ErrCapabilityUnsupported.
func (e *Engine) Embeddings(ctx context.Context, req *providers.EmbeddingsRequest) (*providers.EmbeddingsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("embeddings: nil request")
	}
	provider, key, _, _, err := e.providerFor(ctx, req.Model)
	if err != nil {
		return nil, err
	}
	capable, ok := provider.(providers.EmbeddingsProvider)
	if !ok {
		return nil, fmt.Errorf("%w: %s embeddings", ErrCapabilityUnsupported, provider.Name())
	}
	resp, err := capable.Embeddings(ctx, key, req)
	if err != nil {
		return nil, fmt.Errorf("embeddings: %w", err)
	}
	return resp, nil
}

// GenerateImages resolves the provider for req.Model and dispatches to it when
// the provider implements providers.ImagesProvider.
func (e *Engine) GenerateImages(ctx context.Context, req *providers.ImagesRequest) (*providers.ImagesResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("images: nil request")
	}
	provider, key, _, _, err := e.providerFor(ctx, req.Model)
	if err != nil {
		return nil, err
	}
	capable, ok := provider.(providers.ImagesProvider)
	if !ok {
		return nil, fmt.Errorf("%w: %s images", ErrCapabilityUnsupported, provider.Name())
	}
	resp, err := capable.GenerateImages(ctx, key, req)
	if err != nil {
		return nil, fmt.Errorf("images: %w", err)
	}
	return resp, nil
}

// TranscribeAudio resolves the provider for req.Model and dispatches to it when
// the provider implements providers.AudioTranscriptionProvider.
func (e *Engine) TranscribeAudio(ctx context.Context, req *providers.AudioTranscriptionRequest) (*providers.AudioResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("transcription: nil request")
	}
	provider, key, _, _, err := e.providerFor(ctx, req.Model)
	if err != nil {
		return nil, err
	}
	capable, ok := provider.(providers.AudioTranscriptionProvider)
	if !ok {
		return nil, fmt.Errorf("%w: %s transcription", ErrCapabilityUnsupported, provider.Name())
	}
	resp, err := capable.TranscribeAudio(ctx, key, req)
	if err != nil {
		return nil, fmt.Errorf("transcription: %w", err)
	}
	return resp, nil
}

// Speech resolves the provider for req.Model and dispatches to it when the
// provider implements providers.SpeechProvider. It returns the audio bytes and
// upstream content type.
func (e *Engine) Speech(ctx context.Context, req *providers.SpeechRequest) ([]byte, string, error) {
	if req == nil {
		return nil, "", fmt.Errorf("speech: nil request")
	}
	provider, key, _, _, err := e.providerFor(ctx, req.Model)
	if err != nil {
		return nil, "", err
	}
	capable, ok := provider.(providers.SpeechProvider)
	if !ok {
		return nil, "", fmt.Errorf("%w: %s speech", ErrCapabilityUnsupported, provider.Name())
	}
	audio, contentType, err := capable.Speech(ctx, key, req)
	if err != nil {
		return nil, "", fmt.Errorf("speech: %w", err)
	}
	return audio, contentType, nil
}
