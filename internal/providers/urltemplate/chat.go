package urltemplate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// streamError builds the in-band terminal error chunk (AUD-045/046/047).
func streamError(msg string) *schemas.StreamChunk {
	return &schemas.StreamChunk{Error: &schemas.ProviderError{Message: msg, Type: "stream_error"}}
}

// requestURL resolves the endpoint URL for the request. The urlOverride test
// seam takes precedence over the computed URL.
func (p *Provider) requestURL(key schemas.Key, model string) string {
	if p.urlOverride != "" {
		return p.urlOverride
	}
	return p.buildURL(key, model)
}

// setHeaders configures the catalog headers and auth for the request. azure
// uses the api-key header; the others use the bearer Authorization header.
func (p *Provider) setHeaders(req *fasthttp.Request, key schemas.Key) {
	for k, v := range p.config.Headers {
		req.Header.Set(k, v)
	}
	if p.config.NoAuth {
		return
	}
	if name := p.authHeaderName(); name != "" {
		req.Header.Set(name, key.Value)
		return
	}
	utils.SetAuthHeader(req, key.Value)
}

func (p *Provider) buildError(request *schemas.ChatRequest, requestType, msg, errType string, status int, raw []byte) *schemas.ProviderError {
	return &schemas.ProviderError{
		Message:    msg,
		Type:       errType,
		StatusCode: status,
		Meta: schemas.ErrorMeta{
			Provider:       string(p.id),
			ModelRequested: request.Model,
			RequestType:    requestType,
			StatusCode:     status,
			RawBody:        raw,
		},
	}
}

// ChatCompletion sends a non-streaming OpenAI chat request to the runtime-built
// endpoint.
func (p *Provider) ChatCompletion(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.ChatRequest) (*schemas.ChatResponse, *schemas.ProviderError) {
	url := p.requestURL(key, request.Model)
	if url == "" {
		return nil, p.buildError(request, "chat", fmt.Sprintf("%s requires provider-specific URL data (e.g. accountId/azureEndpoint)", p.id), "invalid_request_error", 0, nil)
	}

	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(url)
	req.Header.SetMethod(fasthttp.MethodPost)
	p.setHeaders(req, key)

	if err := utils.SetJSONBody(req, request); err != nil {
		return nil, p.buildError(request, "chat", fmt.Sprintf("build request: %v", err), "invalid_request_error", 0, nil)
	}

	if err := p.client.Do(req, resp); err != nil {
		return nil, p.errorConverter.Convert(0, []byte(err.Error()), schemas.ErrorMeta{
			Provider:       string(p.id),
			ModelRequested: request.Model,
			RequestType:    "chat",
			StatusCode:     0,
			RawBody:        []byte(err.Error()),
		})
	}
	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:       string(p.id),
			ModelRequested: request.Model,
			RequestType:    "chat",
			StatusCode:     resp.StatusCode(),
			RawBody:        resp.Body(),
		})
	}

	var result schemas.ChatResponse
	if err := utils.ReadJSONBody(resp, &result); err != nil {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:       string(p.id),
			ModelRequested: request.Model,
			RequestType:    "chat",
			StatusCode:     resp.StatusCode(),
			RawBody:        resp.Body(),
		})
	}
	return &result, nil
}

// ChatCompletionStream sends a streaming OpenAI chat request to the
// runtime-built endpoint and returns a channel of SSE chunks.
func (p *Provider) ChatCompletionStream(ctx *schemas.GatewayContext, postHookRunner schemas.PostHookRunner, key schemas.Key, request *schemas.ChatRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	url := p.requestURL(key, request.Model)
	if url == "" {
		return nil, p.buildError(request, "chat_stream", fmt.Sprintf("%s requires provider-specific URL data (e.g. accountId/azureEndpoint)", p.id), "invalid_request_error", 0, nil)
	}

	req := p.client.AcquireRequest()
	resp := p.client.AcquireResponse()

	req.SetRequestURI(url)
	req.Header.SetMethod(fasthttp.MethodPost)
	p.setHeaders(req, key)

	streamReq := *request
	streamReq.Stream = true
	if err := utils.SetJSONBody(req, &streamReq); err != nil {
		p.client.ReleaseRequest(req)
		p.client.ReleaseResponse(resp)
		return nil, p.buildError(request, "chat_stream", fmt.Sprintf("build request: %v", err), "invalid_request_error", 0, nil)
	}

	if err := p.client.Do(req, resp); err != nil {
		p.client.ReleaseRequest(req)
		p.client.ReleaseResponse(resp)
		return nil, p.errorConverter.Convert(0, []byte(err.Error()), schemas.ErrorMeta{
			Provider:       string(p.id),
			ModelRequested: request.Model,
			RequestType:    "chat_stream",
			StatusCode:     0,
			RawBody:        []byte(err.Error()),
		})
	}
	if resp.StatusCode() != fasthttp.StatusOK {
		perr := p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:       string(p.id),
			ModelRequested: request.Model,
			RequestType:    "chat_stream",
			StatusCode:     resp.StatusCode(),
			RawBody:        resp.Body(),
		})
		p.client.ReleaseRequest(req)
		p.client.ReleaseResponse(resp)
		return nil, perr
	}

	p.client.ReleaseRequest(req)

	ch := make(chan *schemas.StreamChunk, 16)
	go func() {
		defer close(ch)
		defer p.client.ReleaseResponse(resp)

		body := bytes.NewReader(resp.Body())
		scanner := utils.NewSSEScanner(body)
		for {
			line, err := scanner.Scan()
			if err != nil {
				if err == io.EOF {
					return
				}
				ch <- streamError(fmt.Sprintf("read stream: %v", err))
				return
			}
			if line == "[DONE]" {
				return
			}
			var chunk schemas.StreamChunk
			if err := json.Unmarshal([]byte(line), &chunk); err != nil {
				ch <- streamError(fmt.Sprintf("decode stream chunk: %v", err))
				return
			}
			ch <- &chunk
			if postHookRunner != nil {
				if err := postHookRunner.Run(ctx, &chunk); err != nil {
					ch <- streamError(fmt.Sprintf("post hook: %v", err))
					return
				}
			}
		}
	}()

	return ch, nil
}
