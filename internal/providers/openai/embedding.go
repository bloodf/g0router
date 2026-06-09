package openai

import (
	"fmt"

	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// Embedding sends an embedding request.
func (p *Provider) Embedding(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.EmbeddingRequest) (*schemas.EmbeddingResponse, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(p.baseURL + "/v1/embeddings")
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
				RequestType:    "embedding",
				StatusCode:     0,
			},
		}
	}

	if err := p.client.Do(req, resp); err != nil {
		return nil, p.errorConverter.Convert(0, []byte(err.Error()), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: request.Model,
			RequestType:    "embedding",
			StatusCode:     0,
			RawBody:        []byte(err.Error()),
		})
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: request.Model,
			RequestType:    "embedding",
			StatusCode:     resp.StatusCode(),
			RawBody:        resp.Body(),
		})
	}

	var result schemas.EmbeddingResponse
	if err := utils.ReadJSONBody(resp, &result); err != nil {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: request.Model,
			RequestType:    "embedding",
			StatusCode:     resp.StatusCode(),
			RawBody:        resp.Body(),
		})
	}
	return &result, nil
}
