package openai

import (
	"fmt"

	"github.com/bloodf/g0router/internal/schemas"
)

func (p *Provider) Responses(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.ResponsesRequest) (*schemas.ResponsesResponse, *schemas.ProviderError) {
	return nil, notImplemented("responses")
}

func (p *Provider) ResponsesStream(ctx *schemas.GatewayContext, postHookRunner schemas.PostHookRunner, key schemas.Key, request *schemas.ResponsesRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	return nil, notImplemented("responses_stream")
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
		Message:    fmt.Sprintf("%s not implemented", method),
		Type:       "not_implemented",
		StatusCode: 501,
	}
}
