package antigravity

import (
	"fmt"

	"github.com/bloodf/g0router/internal/schemas"
)

// Compile-time assertion that Provider satisfies schemas.Provider.
var _ schemas.Provider = (*Provider)(nil)

func (p *Provider) ListModels(ctx *schemas.GatewayContext, key schemas.Key) (*schemas.ListModelsResponse, *schemas.ProviderError) {
	return nil, notImplemented("list_models")
}

func (p *Provider) TextCompletion(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.TextCompletionRequest) (*schemas.TextCompletionResponse, *schemas.ProviderError) {
	return nil, notImplemented("text_completion")
}

func (p *Provider) TextCompletionStream(ctx *schemas.GatewayContext, postHookRunner schemas.PostHookRunner, key schemas.Key, request *schemas.TextCompletionRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	return nil, notImplemented("text_completion_stream")
}

func (p *Provider) Responses(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.ResponsesRequest) (*schemas.ResponsesResponse, *schemas.ProviderError) {
	return nil, notImplemented("responses")
}

func (p *Provider) ResponsesStream(ctx *schemas.GatewayContext, postHookRunner schemas.PostHookRunner, key schemas.Key, request *schemas.ResponsesRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	return nil, notImplemented("responses_stream")
}

func (p *Provider) Embedding(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.EmbeddingRequest) (*schemas.EmbeddingResponse, *schemas.ProviderError) {
	return nil, notImplemented("embedding")
}

func (p *Provider) ImageGeneration(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.ImageGenerationRequest) (*schemas.ImageGenerationResponse, *schemas.ProviderError) {
	return nil, notImplemented("image_generation")
}

func (p *Provider) ImageGenerationStream(ctx *schemas.GatewayContext, postHookRunner schemas.PostHookRunner, key schemas.Key, request *schemas.ImageGenerationRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	return nil, notImplemented("image_generation_stream")
}

func (p *Provider) ImageEdit(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.ImageEditRequest) (*schemas.ImageGenerationResponse, *schemas.ProviderError) {
	return nil, notImplemented("image_edit")
}

func (p *Provider) ImageVariation(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.ImageVariationRequest) (*schemas.ImageGenerationResponse, *schemas.ProviderError) {
	return nil, notImplemented("image_variation")
}

func (p *Provider) Speech(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.SpeechRequest) (*schemas.SpeechResponse, *schemas.ProviderError) {
	return nil, notImplemented("speech")
}

func (p *Provider) SpeechStream(ctx *schemas.GatewayContext, postHookRunner schemas.PostHookRunner, key schemas.Key, request *schemas.SpeechRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	return nil, notImplemented("speech_stream")
}

func (p *Provider) Transcription(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.TranscriptionRequest) (*schemas.TranscriptionResponse, *schemas.ProviderError) {
	return nil, notImplemented("transcription")
}

func (p *Provider) TranscriptionStream(ctx *schemas.GatewayContext, postHookRunner schemas.PostHookRunner, key schemas.Key, request *schemas.TranscriptionRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	return nil, notImplemented("transcription_stream")
}

func (p *Provider) FileUpload(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.FileUploadRequest) (*schemas.FileObject, *schemas.ProviderError) {
	return nil, notImplemented("file_upload")
}

func (p *Provider) FileList(ctx *schemas.GatewayContext, key schemas.Key) (*schemas.FileListResponse, *schemas.ProviderError) {
	return nil, notImplemented("file_list")
}

func (p *Provider) FileRetrieve(ctx *schemas.GatewayContext, key schemas.Key, fileID string) (*schemas.FileObject, *schemas.ProviderError) {
	return nil, notImplemented("file_retrieve")
}

func (p *Provider) FileDelete(ctx *schemas.GatewayContext, key schemas.Key, fileID string) (*schemas.FileDeleteResponse, *schemas.ProviderError) {
	return nil, notImplemented("file_delete")
}

func (p *Provider) FileContent(ctx *schemas.GatewayContext, key schemas.Key, fileID string) ([]byte, *schemas.ProviderError) {
	return nil, notImplemented("file_content")
}

func (p *Provider) BatchCreate(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.BatchCreateRequest) (*schemas.Batch, *schemas.ProviderError) {
	return nil, notImplemented("batch_create")
}

func (p *Provider) BatchList(ctx *schemas.GatewayContext, key schemas.Key) (*schemas.BatchListResponse, *schemas.ProviderError) {
	return nil, notImplemented("batch_list")
}

func (p *Provider) BatchRetrieve(ctx *schemas.GatewayContext, key schemas.Key, batchID string) (*schemas.Batch, *schemas.ProviderError) {
	return nil, notImplemented("batch_retrieve")
}

func (p *Provider) BatchCancel(ctx *schemas.GatewayContext, key schemas.Key, batchID string) (*schemas.Batch, *schemas.ProviderError) {
	return nil, notImplemented("batch_cancel")
}

func (p *Provider) CountTokens(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.ChatRequest) (*schemas.TokenCountResponse, *schemas.ProviderError) {
	return nil, notImplemented("count_tokens")
}

func notImplemented(method string) *schemas.ProviderError {
	return &schemas.ProviderError{
		Message:    fmt.Sprintf("%s not implemented for antigravity provider", method),
		Type:       "not_implemented",
		StatusCode: 501,
	}
}
