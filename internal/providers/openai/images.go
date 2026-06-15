package openai

import (
	"bytes"
	"fmt"
	"mime/multipart"

	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// ImageGeneration sends an image-generation request to
// POST /v1/images/generations as JSON and decodes the bare
// ImageGenerationResponse. It mirrors the Embedding transport.
func (p *Provider) ImageGeneration(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.ImageGenerationRequest) (*schemas.ImageGenerationResponse, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(p.baseURL + "/v1/images/generations")
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
				RequestType:    "image_generation",
				StatusCode:     0,
			},
		}
	}

	return p.doImageJSON(req, resp, request.Model, "image_generation")
}

// ImageGenerationStream sends a streaming image-generation request (JSON body)
// and returns a channel of SSE chunks.
func (p *Provider) ImageGenerationStream(ctx *schemas.GatewayContext, postHookRunner schemas.PostHookRunner, key schemas.Key, request *schemas.ImageGenerationRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	resp := p.client.AcquireResponse()

	req.SetRequestURI(p.baseURL + "/v1/images/generations")
	req.Header.SetMethod(fasthttp.MethodPost)
	utils.SetAuthHeader(req, key.Value)

	streamReq := *request
	if err := utils.SetJSONBody(req, &streamReq); err != nil {
		p.client.ReleaseRequest(req)
		p.client.ReleaseResponse(resp)
		return nil, &schemas.ProviderError{
			Message:    fmt.Sprintf("build request: %v", err),
			Type:       "invalid_request_error",
			StatusCode: 0,
			Meta: schemas.ErrorMeta{
				Provider:       string(p.provider),
				ModelRequested: request.Model,
				RequestType:    "image_generation_stream",
				StatusCode:     0,
			},
		}
	}

	return p.streamSSE(ctx, postHookRunner, req, resp, request.Model, "image_generation_stream")
}

// ImageEdit sends an image-edit request to POST /v1/images/edits as
// multipart/form-data (image + optional mask + prompt; ESC-MULTIPART) and
// decodes the bare ImageGenerationResponse.
func (p *Provider) ImageEdit(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.ImageEditRequest) (*schemas.ImageGenerationResponse, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(p.baseURL + "/v1/images/edits")
	req.Header.SetMethod(fasthttp.MethodPost)
	utils.SetAuthHeader(req, key.Value)

	if perr := p.setImageEditBody(req, request); perr != nil {
		return nil, perr
	}

	return p.doImageJSON(req, resp, request.Model, "image_edit")
}

// ImageVariation sends an image-variation request to
// POST /v1/images/variations as multipart/form-data (image; ESC-MULTIPART) and
// decodes the bare ImageGenerationResponse.
func (p *Provider) ImageVariation(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.ImageVariationRequest) (*schemas.ImageGenerationResponse, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(p.baseURL + "/v1/images/variations")
	req.Header.SetMethod(fasthttp.MethodPost)
	utils.SetAuthHeader(req, key.Value)

	if perr := p.setImageVariationBody(req, request); perr != nil {
		return nil, perr
	}

	return p.doImageJSON(req, resp, request.Model, "image_variation")
}

// doImageJSON issues req, validates status, and decodes an
// ImageGenerationResponse. It is shared by the three image endpoints whose
// success body is JSON. It takes ownership of req/resp via the caller's defers.
func (p *Provider) doImageJSON(req *fasthttp.Request, resp *fasthttp.Response, model, requestType string) (*schemas.ImageGenerationResponse, *schemas.ProviderError) {
	if err := p.client.Do(req, resp); err != nil {
		return nil, p.errorConverter.Convert(0, []byte(err.Error()), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: model,
			RequestType:    requestType,
			StatusCode:     0,
			RawBody:        []byte(err.Error()),
		})
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: model,
			RequestType:    requestType,
			StatusCode:     resp.StatusCode(),
			RawBody:        resp.Body(),
		})
	}

	var result schemas.ImageGenerationResponse
	if err := utils.ReadJSONBody(resp, &result); err != nil {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: model,
			RequestType:    requestType,
			StatusCode:     resp.StatusCode(),
			RawBody:        resp.Body(),
		})
	}
	return &result, nil
}

// setImageEditBody builds the outbound multipart/form-data body for an image
// edit from the already-parsed schema fields, with an explicit field whitelist.
func (p *Provider) setImageEditBody(req *fasthttp.Request, request *schemas.ImageEditRequest) *schemas.ProviderError {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	fail := func(err error) *schemas.ProviderError {
		return multipartBuildError(string(p.provider), request.Model, "image_edit", err)
	}

	if err := writeFilePart(mw, "image", "image", request.Image); err != nil {
		return fail(err)
	}
	if len(request.Mask) > 0 {
		if err := writeFilePart(mw, "mask", "mask", request.Mask); err != nil {
			return fail(err)
		}
	}
	if err := mw.WriteField("prompt", request.Prompt); err != nil {
		return fail(err)
	}
	if request.Model != "" {
		if err := mw.WriteField("model", request.Model); err != nil {
			return fail(err)
		}
	}
	if request.N != nil {
		if err := mw.WriteField("n", fmt.Sprintf("%d", *request.N)); err != nil {
			return fail(err)
		}
	}
	if request.Size != nil {
		if err := mw.WriteField("size", *request.Size); err != nil {
			return fail(err)
		}
	}
	if request.ResponseFormat != nil {
		if err := mw.WriteField("response_format", *request.ResponseFormat); err != nil {
			return fail(err)
		}
	}
	if request.User != "" {
		if err := mw.WriteField("user", request.User); err != nil {
			return fail(err)
		}
	}
	if err := mw.Close(); err != nil {
		return fail(err)
	}

	req.Header.SetContentType(mw.FormDataContentType())
	req.SetBody(buf.Bytes())
	return nil
}

// setImageVariationBody builds the outbound multipart/form-data body for an
// image variation from the already-parsed schema fields.
func (p *Provider) setImageVariationBody(req *fasthttp.Request, request *schemas.ImageVariationRequest) *schemas.ProviderError {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	fail := func(err error) *schemas.ProviderError {
		return multipartBuildError(string(p.provider), request.Model, "image_variation", err)
	}

	if err := writeFilePart(mw, "image", "image", request.Image); err != nil {
		return fail(err)
	}
	if request.Model != "" {
		if err := mw.WriteField("model", request.Model); err != nil {
			return fail(err)
		}
	}
	if request.N != nil {
		if err := mw.WriteField("n", fmt.Sprintf("%d", *request.N)); err != nil {
			return fail(err)
		}
	}
	if request.Size != nil {
		if err := mw.WriteField("size", *request.Size); err != nil {
			return fail(err)
		}
	}
	if request.ResponseFormat != nil {
		if err := mw.WriteField("response_format", *request.ResponseFormat); err != nil {
			return fail(err)
		}
	}
	if request.User != "" {
		if err := mw.WriteField("user", request.User); err != nil {
			return fail(err)
		}
	}
	if err := mw.Close(); err != nil {
		return fail(err)
	}

	req.Header.SetContentType(mw.FormDataContentType())
	req.SetBody(buf.Bytes())
	return nil
}

// writeFilePart writes a single file part to the multipart writer.
func writeFilePart(mw *multipart.Writer, field, filename string, data []byte) error {
	fw, err := mw.CreateFormFile(field, filename)
	if err != nil {
		return err
	}
	if _, err := fw.Write(data); err != nil {
		return err
	}
	return nil
}

// multipartBuildError builds a *ProviderError for an outbound multipart-body
// construction failure.
func multipartBuildError(provider, model, requestType string, err error) *schemas.ProviderError {
	return &schemas.ProviderError{
		Message:    fmt.Sprintf("build multipart request: %v", err),
		Type:       "invalid_request_error",
		StatusCode: 0,
		Meta: schemas.ErrorMeta{
			Provider:       provider,
			ModelRequested: model,
			RequestType:    requestType,
			StatusCode:     0,
		},
	}
}
