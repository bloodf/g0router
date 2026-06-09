package gemini

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// ChatCompletion sends a non-streaming chat completion request via Gemini.
func (p *Provider) ChatCompletion(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.ChatRequest) (*schemas.ChatResponse, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	gemReq := ConvertChatRequest(request)
	uri := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, gemReq.Model, key.Value)
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

	var gemResp GenerateContentResponse
	if err := utils.ReadJSONBody(resp, &gemResp); err != nil {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: request.Model,
			RequestType:    "chat",
			StatusCode:     resp.StatusCode(),
			RawBody:        resp.Body(),
		})
	}

	return ConvertChatResponse(&gemResp, request.Model), nil
}

// ChatCompletionStream sends a streaming chat completion request and returns a channel of SSE chunks.
func (p *Provider) ChatCompletionStream(ctx *schemas.GatewayContext, postHookRunner schemas.PostHookRunner, key schemas.Key, request *schemas.ChatRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	resp := p.client.AcquireResponse()

	gemReq := ConvertChatRequest(request)
	uri := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s", p.baseURL, gemReq.Model, key.Value)
	req.SetRequestURI(uri)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/json")

	if err := utils.SetJSONBody(req, gemReq); err != nil {
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

			var gemResp GenerateContentResponse
			if err := json.Unmarshal([]byte(line), &gemResp); err != nil {
				continue
			}

			chunk := ConvertStreamChunk(&gemResp, request.Model)
			if chunk != nil && len(chunk.Choices) > 0 {
				ch <- chunk
				if postHookRunner != nil {
					_ = postHookRunner.Run(ctx, chunk)
				}
			}
		}
	}()

	return ch, nil
}

// setAuthHeader sets the Gemini API key header.
func setAuthHeader(req *fasthttp.Request, key string) {
	req.Header.Set("x-goog-api-key", key)
}

// buildModelURI constructs a Gemini API URI for a given model and method.
func buildModelURI(baseURL, model, method, key string) string {
	return fmt.Sprintf("%s/models/%s:%s?key=%s", baseURL, model, method, key)
}

// sanitizeModelName removes provider prefixes like "gemini/" from model names.
func sanitizeModelName(model string) string {
	return strings.TrimPrefix(model, "gemini/")
}
