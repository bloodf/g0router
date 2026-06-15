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
