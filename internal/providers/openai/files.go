package openai

import (
	"bytes"
	"fmt"
	"mime/multipart"

	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// FileUpload uploads a file to POST /v1/files as multipart/form-data and decodes
// the JSON FileObject. The state lives upstream at OpenAI; g0router proxies the
// request and returns the upstream object verbatim (Option A, stateless).
func (p *Provider) FileUpload(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.FileUploadRequest) (*schemas.FileObject, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(p.baseURL + "/v1/files")
	req.Header.SetMethod(fasthttp.MethodPost)
	utils.SetAuthHeader(req, key.Value)

	if perr := p.setFileUploadBody(req, request); perr != nil {
		return nil, perr
	}

	if err := p.client.Do(req, resp); err != nil {
		return nil, p.errorConverter.Convert(0, []byte(err.Error()), schemas.ErrorMeta{
			Provider:    string(p.provider),
			RequestType: "file_upload",
			StatusCode:  0,
			RawBody:     []byte(err.Error()),
		})
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:    string(p.provider),
			RequestType: "file_upload",
			StatusCode:  resp.StatusCode(),
			RawBody:     resp.Body(),
		})
	}

	var result schemas.FileObject
	if err := utils.ReadJSONBody(resp, &result); err != nil {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:    string(p.provider),
			RequestType: "file_upload",
			StatusCode:  resp.StatusCode(),
			RawBody:     resp.Body(),
		})
	}
	return &result, nil
}

// setFileUploadBody builds the outbound multipart/form-data body for a file
// upload from the already-parsed schema fields, with an explicit field whitelist
// (file + purpose) (ESC-MULTIPART-UPLOAD).
func (p *Provider) setFileUploadBody(req *fasthttp.Request, request *schemas.FileUploadRequest) *schemas.ProviderError {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	fail := func(err error) *schemas.ProviderError {
		return &schemas.ProviderError{
			Message:    fmt.Sprintf("build multipart request: %v", err),
			Type:       "invalid_request_error",
			StatusCode: 0,
			Meta: schemas.ErrorMeta{
				Provider:    string(p.provider),
				RequestType: "file_upload",
				StatusCode:  0,
			},
		}
	}

	filename := request.Filename
	if filename == "" {
		filename = "file"
	}
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return fail(err)
	}
	if _, err := fw.Write(request.File); err != nil {
		return fail(err)
	}
	if err := mw.WriteField("purpose", request.Purpose); err != nil {
		return fail(err)
	}
	if err := mw.Close(); err != nil {
		return fail(err)
	}

	req.Header.SetContentType(mw.FormDataContentType())
	req.SetBody(buf.Bytes())
	return nil
}

// FileList lists files via GET /v1/files and decodes the JSON FileListResponse.
func (p *Provider) FileList(ctx *schemas.GatewayContext, key schemas.Key) (*schemas.FileListResponse, *schemas.ProviderError) {
	var result schemas.FileListResponse
	if perr := p.fileJSONGet("/v1/files", "file_list", key, &result); perr != nil {
		return nil, perr
	}
	return &result, nil
}

// FileRetrieve fetches a single file via GET /v1/files/{id} and decodes the
// JSON FileObject. The file id is interpolated into the upstream URI as-received.
func (p *Provider) FileRetrieve(ctx *schemas.GatewayContext, key schemas.Key, fileID string) (*schemas.FileObject, *schemas.ProviderError) {
	var result schemas.FileObject
	if perr := p.fileJSONGet("/v1/files/"+fileID, "file_retrieve", key, &result); perr != nil {
		return nil, perr
	}
	return &result, nil
}

// FileDelete deletes a file via DELETE /v1/files/{id} and decodes the JSON
// FileDeleteResponse.
func (p *Provider) FileDelete(ctx *schemas.GatewayContext, key schemas.Key, fileID string) (*schemas.FileDeleteResponse, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(p.baseURL + "/v1/files/" + fileID)
	req.Header.SetMethod(fasthttp.MethodDelete)
	utils.SetAuthHeader(req, key.Value)

	if err := p.client.Do(req, resp); err != nil {
		return nil, p.errorConverter.Convert(0, []byte(err.Error()), schemas.ErrorMeta{
			Provider:    string(p.provider),
			RequestType: "file_delete",
			StatusCode:  0,
			RawBody:     []byte(err.Error()),
		})
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:    string(p.provider),
			RequestType: "file_delete",
			StatusCode:  resp.StatusCode(),
			RawBody:     resp.Body(),
		})
	}

	var result schemas.FileDeleteResponse
	if err := utils.ReadJSONBody(resp, &result); err != nil {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:    string(p.provider),
			RequestType: "file_delete",
			StatusCode:  resp.StatusCode(),
			RawBody:     resp.Body(),
		})
	}
	return &result, nil
}

// FileContent fetches the raw file bytes via GET /v1/files/{id}/content. Unlike
// the other endpoints the success body is binary (e.g. a batch output JSONL), so
// it is copied verbatim rather than decoded (ESC-FILE-CONTENT-BYTES). The bytes
// must outlive the pooled response, so clone rather than alias resp.Body().
func (p *Provider) FileContent(ctx *schemas.GatewayContext, key schemas.Key, fileID string) ([]byte, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(p.baseURL + "/v1/files/" + fileID + "/content")
	req.Header.SetMethod(fasthttp.MethodGet)
	utils.SetAuthHeader(req, key.Value)

	if err := p.client.Do(req, resp); err != nil {
		return nil, p.errorConverter.Convert(0, []byte(err.Error()), schemas.ErrorMeta{
			Provider:    string(p.provider),
			RequestType: "file_content",
			StatusCode:  0,
			RawBody:     []byte(err.Error()),
		})
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:    string(p.provider),
			RequestType: "file_content",
			StatusCode:  resp.StatusCode(),
			RawBody:     resp.Body(),
		})
	}

	return append([]byte(nil), resp.Body()...), nil
}

// fileJSONGet issues a GET to uri, validates the status, and decodes the JSON
// body into out. It centralizes the shared GET-and-decode transport for the
// file endpoints whose success body is JSON.
func (p *Provider) fileJSONGet(uri, requestType string, key schemas.Key, out any) *schemas.ProviderError {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(p.baseURL + uri)
	req.Header.SetMethod(fasthttp.MethodGet)
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
