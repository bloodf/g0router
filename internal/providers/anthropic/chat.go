package anthropic

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

// ChatCompletion sends a non-streaming chat completion request via Anthropic Messages API.
func (p *Provider) ChatCompletion(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.ChatRequest) (*schemas.ChatResponse, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(p.baseURL + "/v1/messages")
	req.Header.SetMethod(fasthttp.MethodPost)
	setAuthHeader(req, key.Value)

	anthReq := ConvertRequest(request)
	if err := utils.SetJSONBody(req, anthReq); err != nil {
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

	var anthResp MessagesResponse
	if err := utils.ReadJSONBody(resp, &anthResp); err != nil {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: request.Model,
			RequestType:    "chat",
			StatusCode:     resp.StatusCode(),
			RawBody:        resp.Body(),
		})
	}

	return ConvertResponse(&anthResp), nil
}

// ChatCompletionStream sends a streaming chat completion request and returns a channel of SSE chunks.
func (p *Provider) ChatCompletionStream(ctx *schemas.GatewayContext, postHookRunner schemas.PostHookRunner, key schemas.Key, request *schemas.ChatRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	resp := p.client.AcquireResponse()

	req.SetRequestURI(p.baseURL + "/v1/messages")
	req.Header.SetMethod(fasthttp.MethodPost)
	setAuthHeader(req, key.Value)

	anthReq := ConvertRequest(request)
	anthReq.Stream = true
	if err := utils.SetJSONBody(req, anthReq); err != nil {
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
		var messageID, model string

		for {
			line, err := scanner.Scan()
			if err != nil {
				if err == io.EOF {
					return
				}
				// AUD-046: surface read errors in-band before closing.
				ch <- streamError(fmt.Sprintf("read stream: %v", err))
				return
			}
			if line == "[DONE]" {
				return
			}

			var event StreamEvent
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				// AUD-045: a malformed event means the stream is corrupt;
				// abort with an in-band error instead of silently dropping data.
				ch <- streamError(fmt.Sprintf("decode stream event: %v", err))
				return
			}

			switch event.Type {
			case "message_start":
				if event.Message != nil {
					messageID = event.Message.ID
					model = event.Message.Model
				}
			case "content_block_delta", "message_delta":
				chunk := ConvertStreamEventToChunk(&event, messageID, model)
				if chunk != nil {
					ch <- chunk
					if postHookRunner != nil {
						if err := postHookRunner.Run(ctx, chunk); err != nil {
							// AUD-047: hook failures abort the stream.
							ch <- streamError(fmt.Sprintf("post hook: %v", err))
							return
						}
					}
				}
			case "message_stop":
				return
			}
		}
	}()

	return ch, nil
}

func setAuthHeader(req *fasthttp.Request, key string) {
	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", "2023-06-01")
}
