package openai

import (
	"fmt"

	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// countTokensResult decodes the upstream POST /v1/responses/input_tokens body.
// OpenAI returns the count under "input_tokens"; the "tokens" alias is accepted
// for shape-tolerance with the bare TokenCountResponse field name.
type countTokensResult struct {
	InputTokens int `json:"input_tokens"`
	Tokens      int `json:"tokens"`
}

// CountTokens proxies a token-count request to POST /v1/responses/input_tokens.
// It mirrors the Embedding transport. The request is the resolved chat-shaped
// body; OpenAI's tokenizer is model-keyed, so the chat shape is accepted by the
// upstream count endpoint. The upstream count is mapped into TokenCountResponse.
func (p *Provider) CountTokens(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.ChatRequest) (*schemas.TokenCountResponse, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(p.baseURL + "/v1/responses/input_tokens")
	req.Header.SetMethod(fasthttp.MethodPost)
	utils.SetAuthHeader(req, key.Value)

	if err := utils.SetJSONBody(req, request); err != nil {
		return nil, &schemas.ProviderError{
			Message:    fmt.Sprintf("build request: %v", err),
			Type:       "invalid_request_error",
			StatusCode: 0,
			Meta: schemas.ErrorMeta{
				Provider:       string(p.provider),
				ModelRequested: request.Model,
				RequestType:    "count_tokens",
				StatusCode:     0,
			},
		}
	}

	if err := p.client.Do(req, resp); err != nil {
		return nil, p.errorConverter.Convert(0, []byte(err.Error()), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: request.Model,
			RequestType:    "count_tokens",
			StatusCode:     0,
			RawBody:        []byte(err.Error()),
		})
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: request.Model,
			RequestType:    "count_tokens",
			StatusCode:     resp.StatusCode(),
			RawBody:        resp.Body(),
		})
	}

	var result countTokensResult
	if err := utils.ReadJSONBody(resp, &result); err != nil {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: request.Model,
			RequestType:    "count_tokens",
			StatusCode:     resp.StatusCode(),
			RawBody:        resp.Body(),
		})
	}

	tokens := result.InputTokens
	if tokens == 0 {
		tokens = result.Tokens
	}
	return &schemas.TokenCountResponse{Tokens: tokens}, nil
}
