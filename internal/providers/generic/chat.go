package generic

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

// streamError builds the in-band terminal error chunk (AUD-045/046/047).
func streamError(msg string) *schemas.StreamChunk {
	return &schemas.StreamChunk{Error: &schemas.ProviderError{Message: msg, Type: "stream_error"}}
}

// chatURLs returns the ordered candidate endpoint URLs for the provider
// (PAR-ROUTE-035). It mirrors 9router's index-based fallback URL list
// (provider.js:155-209, base.js:20-42): the primary URL is index 0 and any
// configured fallbacks follow in order. The list is derived additively from
// the existing single BaseURL config field — a config carrying multiple
// whitespace/newline-separated URLs yields them in order; a normal single-URL
// config yields a one-element list. Empty segments are dropped. When no URL is
// configured, the list contains a single empty string so chatURL() (= [0]) is
// unchanged.
func (p *Provider) chatURLs() []string {
	fields := strings.Fields(p.config.BaseURL)
	if len(fields) == 0 {
		return []string{p.config.BaseURL}
	}
	return fields
}

// chatURL returns the primary endpoint URL (PAR-ROUTE-035: chatURLs()[0]).
func (p *Provider) chatURL() string {
	return p.chatURLs()[0]
}

// setHeaders configures the request headers for the provider.
func (p *Provider) setHeaders(req *fasthttp.Request, key schemas.Key) {
	for k, v := range p.config.Headers {
		req.Header.Set(k, v)
	}
	if !p.config.NoAuth {
		utils.SetAuthHeader(req, key.Value)
	}
}

// ChatCompletion sends a non-streaming chat completion request.
func (p *Provider) ChatCompletion(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.ChatRequest) (*schemas.ChatResponse, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(p.chatURL())
	req.Header.SetMethod(fasthttp.MethodPost)
	p.setHeaders(req, key)

	if err := utils.SetJSONBody(req, request); err != nil {
		return nil, &schemas.ProviderError{
			Message:    fmt.Sprintf("build request: %v", err),
			Type:       "invalid_request_error",
			StatusCode: 0,
			Meta: schemas.ErrorMeta{
				Provider:       string(p.id),
				ModelRequested: request.Model,
				RequestType:    "chat",
				StatusCode:     0,
			},
		}
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

// ChatCompletionStream sends a streaming chat completion request and returns a channel of SSE chunks.
func (p *Provider) ChatCompletionStream(ctx *schemas.GatewayContext, postHookRunner schemas.PostHookRunner, key schemas.Key, request *schemas.ChatRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	resp := p.client.AcquireResponse()

	req.SetRequestURI(p.chatURL())
	req.Header.SetMethod(fasthttp.MethodPost)
	p.setHeaders(req, key)

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
				Provider:       string(p.id),
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
			Provider:       string(p.id),
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
			Provider:       string(p.id),
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
				// AUD-046: surface read errors in-band before closing.
				ch <- streamError(fmt.Sprintf("read stream: %v", err))
				return
			}
			if line == "[DONE]" {
				return
			}
			var chunk schemas.StreamChunk
			if err := json.Unmarshal([]byte(line), &chunk); err != nil {
				// AUD-045: a malformed chunk means the stream is corrupt;
				// abort with an in-band error instead of silently dropping data.
				ch <- streamError(fmt.Sprintf("decode stream chunk: %v", err))
				return
			}
			ch <- &chunk
			if postHookRunner != nil {
				if err := postHookRunner.Run(ctx, &chunk); err != nil {
					// AUD-047: hook failures abort the stream.
					ch <- streamError(fmt.Sprintf("post hook: %v", err))
					return
				}
			}
		}
	}()

	return ch, nil
}
