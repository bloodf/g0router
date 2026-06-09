package openai

import (
	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// ListModels fetches the available models from OpenAI.
func (p *Provider) ListModels(ctx *schemas.GatewayContext, key schemas.Key) (*schemas.ListModelsResponse, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(p.baseURL + "/v1/models")
	req.Header.SetMethod(fasthttp.MethodGet)
	utils.SetAuthHeader(req, key.Value)

	if err := p.client.Do(req, resp); err != nil {
		return nil, p.errorConverter.Convert(0, []byte(err.Error()), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: "",
			RequestType:    "list_models",
			StatusCode:     0,
			RawBody:        []byte(err.Error()),
		})
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: "",
			RequestType:    "list_models",
			StatusCode:     resp.StatusCode(),
			RawBody:        resp.Body(),
		})
	}

	var result schemas.ListModelsResponse
	if err := utils.ReadJSONBody(resp, &result); err != nil {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: "",
			RequestType:    "list_models",
			StatusCode:     resp.StatusCode(),
			RawBody:        resp.Body(),
		})
	}
	return &result, nil
}
