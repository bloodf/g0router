package openai

import (
	"fmt"

	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// BatchCreate creates a batch via POST /v1/batches (JSON body) and decodes the
// JSON Batch. The state lives upstream at OpenAI; g0router proxies the request
// and returns the upstream object verbatim (Option A, stateless).
func (p *Provider) BatchCreate(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.BatchCreateRequest) (*schemas.Batch, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(p.baseURL + "/v1/batches")
	req.Header.SetMethod(fasthttp.MethodPost)
	utils.SetAuthHeader(req, key.Value)

	if err := utils.SetJSONBody(req, request); err != nil {
		return nil, &schemas.ProviderError{
			Message:    fmt.Sprintf("build request: %v", err),
			Type:       "invalid_request_error",
			StatusCode: 0,
			Meta: schemas.ErrorMeta{
				Provider:    string(p.provider),
				RequestType: "batch_create",
				StatusCode:  0,
			},
		}
	}

	if err := p.client.Do(req, resp); err != nil {
		return nil, p.errorConverter.Convert(0, []byte(err.Error()), schemas.ErrorMeta{
			Provider:    string(p.provider),
			RequestType: "batch_create",
			StatusCode:  0,
			RawBody:     []byte(err.Error()),
		})
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:    string(p.provider),
			RequestType: "batch_create",
			StatusCode:  resp.StatusCode(),
			RawBody:     resp.Body(),
		})
	}

	var result schemas.Batch
	if err := utils.ReadJSONBody(resp, &result); err != nil {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:    string(p.provider),
			RequestType: "batch_create",
			StatusCode:  resp.StatusCode(),
			RawBody:     resp.Body(),
		})
	}
	return &result, nil
}

// BatchList lists batches via GET /v1/batches and decodes BatchListResponse.
func (p *Provider) BatchList(ctx *schemas.GatewayContext, key schemas.Key) (*schemas.BatchListResponse, *schemas.ProviderError) {
	var result schemas.BatchListResponse
	if perr := p.batchJSON("/v1/batches", fasthttp.MethodGet, "batch_list", key, &result); perr != nil {
		return nil, perr
	}
	return &result, nil
}

// BatchRetrieve fetches a single batch via GET /v1/batches/{id} and decodes the
// JSON Batch. The batch id is interpolated into the upstream URI as-received.
func (p *Provider) BatchRetrieve(ctx *schemas.GatewayContext, key schemas.Key, batchID string) (*schemas.Batch, *schemas.ProviderError) {
	var result schemas.Batch
	if perr := p.batchJSON("/v1/batches/"+batchID, fasthttp.MethodGet, "batch_retrieve", key, &result); perr != nil {
		return nil, perr
	}
	return &result, nil
}

// BatchCancel cancels a batch via POST /v1/batches/{id}/cancel and decodes the
// JSON Batch.
func (p *Provider) BatchCancel(ctx *schemas.GatewayContext, key schemas.Key, batchID string) (*schemas.Batch, *schemas.ProviderError) {
	var result schemas.Batch
	if perr := p.batchJSON("/v1/batches/"+batchID+"/cancel", fasthttp.MethodPost, "batch_cancel", key, &result); perr != nil {
		return nil, perr
	}
	return &result, nil
}

// batchJSON issues a no-body request (GET or POST) to uri, validates the status,
// and decodes the JSON body into out. It centralizes the shared transport for
// the batch endpoints whose body-in is empty and body-out is JSON.
func (p *Provider) batchJSON(uri, method, requestType string, key schemas.Key, out any) *schemas.ProviderError {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(p.baseURL + uri)
	req.Header.SetMethod(method)
	utils.SetAuthHeader(req, key.Value)

	if err := p.client.Do(req, resp); err != nil {
		return p.errorConverter.Convert(0, []byte(err.Error()), schemas.ErrorMeta{
			Provider:    string(p.provider),
			RequestType: requestType,
			StatusCode:  0,
			RawBody:     []byte(err.Error()),
		})
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:    string(p.provider),
			RequestType: requestType,
			StatusCode:  resp.StatusCode(),
			RawBody:     resp.Body(),
		})
	}

	if err := utils.ReadJSONBody(resp, out); err != nil {
		return p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:    string(p.provider),
			RequestType: requestType,
			StatusCode:  resp.StatusCode(),
			RawBody:     resp.Body(),
		})
	}
	return nil
}
