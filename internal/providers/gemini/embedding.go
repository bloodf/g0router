package gemini

import (
	"fmt"

	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// Embedding sends an embedding request via Gemini.
func (p *Provider) Embedding(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.EmbeddingRequest) (*schemas.EmbeddingResponse, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	model := sanitizeModelName(request.Model)
	if model == "" {
		model = "text-embedding-004"
	}

	gemReq := ConvertEmbeddingRequest(request)
	uri := fmt.Sprintf("%s/models/%s:embedContent?key=%s", p.baseURL, model, key.Value)
	req.SetRequestURI(uri)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/json")

	if err := utils.SetJSONBody(req, gemReq); err != nil {
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

	var gemResp EmbedContentResponse
	if err := utils.ReadJSONBody(resp, &gemResp); err != nil {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: request.Model,
			RequestType:    "embedding",
			StatusCode:     resp.StatusCode(),
			RawBody:        resp.Body(),
		})
	}

	return ConvertEmbeddingResponse(&gemResp, request.Model), nil
}


