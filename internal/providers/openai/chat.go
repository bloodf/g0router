package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// ChatCompletion sends a non-streaming chat completion request.
func (p *Provider) ChatCompletion(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.ChatRequest) (*schemas.ChatResponse, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(p.baseURL + "/v1/chat/completions")
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
				RequestType:    "chat",
				StatusCode:     0,
			},
		}
	}

	if err := p.client.Do(req, resp); err != nil {
		return nil, p.errorConverter.Convert(0, []byte(err.Error()), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: request.Model,
			RequestType:    "chat",
			StatusCode:     0,
			RawBody:        []byte(err.Error()),
		})
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: request.Model,
			RequestType:    "chat",
			StatusCode:     resp.StatusCode(),
			RawBody:        resp.Body(),
		})
	}

	var result schemas.ChatResponse
	if err := utils.ReadJSONBody(resp, &result); err != nil {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: request.Model,
			RequestType:    "chat",
			StatusCode:     resp.StatusCode(),
			RawBody:        resp.Body(),
		})
	}
	return &result, nil
}

// ChatCompletionStream sends a streaming chat completion request and returns a channel of SSE chunks.
func (p *Provider) ChatCompletionStream(ctx *schemas.GatewayContext, postHookRunner schemas.PostHookRunner, key schemas.Key, request *schemas.ChatRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	resp := p.client.AcquireResponse()

	req.SetRequestURI(p.baseURL + "/v1/chat/completions")
	req.Header.SetMethod(fasthttp.MethodPost)
	utils.SetAuthHeader(req, key.Value)

	streamReq := *request
	streamReq.Stream = true
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
				RequestType:    "chat_stream",
				StatusCode:     0,
			},
		}
	}

	if err := p.client.Do(req, resp); err != nil {
		p.client.ReleaseRequest(req)
		p.client.ReleaseResponse(resp)
		return nil, p.errorConverter.Convert(0, []byte(err.Error()), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: request.Model,
			RequestType:    "chat_stream",
			StatusCode:     0,
			RawBody:        []byte(err.Error()),
		})
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		p.client.ReleaseRequest(req)
		p.client.ReleaseResponse(resp)
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: request.Model,
			RequestType:    "chat_stream",
			StatusCode:     resp.StatusCode(),
			RawBody:        resp.Body(),
		})
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
				break
			}
			if line == "[DONE]" {
				return
			}
			var chunk schemas.StreamChunk
			if err := json.Unmarshal([]byte(line), &chunk); err != nil {
				// AUD-045: a malformed chunk means the stream is corrupt;
				// abort instead of silently dropping data.
				return
			}
			ch <- &chunk
			if postHookRunner != nil {
				_ = postHookRunner.Run(ctx, &chunk)
			}
		}
	}()

	return ch, nil
}
