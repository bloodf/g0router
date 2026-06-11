package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/bloodf/g0router/internal/providers/catalog"
	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
)

// streamError builds the in-band terminal error chunk (AUD-045/046/047).
func streamError(msg string) *schemas.StreamChunk {
	return &schemas.StreamChunk{Error: &schemas.ProviderError{Message: msg, Type: "stream_error"}}
}

// chatURL returns the target URL for chat requests.
// For ollama-local it uses the default host or the given override;
// for cloud ollama it uses config.BaseURL.
func (p *Provider) chatURL(override string) string {
	if p.id == schemas.ModelProvider("ollama-local") {
		return catalog.ResolveOllamaHost(override) + "/api/chat"
	}
	return p.config.BaseURL
}

// ChatCompletion sends a non-streaming chat completion request.
func (p *Provider) ChatCompletion(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.ChatRequest) (*schemas.ChatResponse, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(p.chatURL(key.ProviderSpecificData["baseUrl"]))
	req.Header.SetMethod(fasthttp.MethodPost)
	// No auth header for ollama (NoAuth == true).

	bodyMap, err := requestToMap(request)
	if err != nil {
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

	reqMap, err := p.registry.TranslateRequest(translation.FormatOpenAI, translation.FormatOllama, request.Model, bodyMap, false, nil)
	if err != nil {
		return nil, &schemas.ProviderError{
			Message:    fmt.Sprintf("translate request: %v", err),
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

	if err := utils.SetJSONBody(req, reqMap); err != nil {
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
		return nil, &schemas.ProviderError{
			Message:    fmt.Sprintf("request failed: %v", err),
			Type:       "request_error",
			StatusCode: 0,
			Meta: schemas.ErrorMeta{
				Provider:       string(p.id),
				ModelRequested: request.Model,
				RequestType:    "chat",
				StatusCode:     0,
				RawBody:        []byte(err.Error()),
			},
		}
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, &schemas.ProviderError{
			Message:    string(resp.Body()),
			Type:       "provider_error",
			StatusCode: resp.StatusCode(),
			Meta: schemas.ErrorMeta{
				Provider:       string(p.id),
				ModelRequested: request.Model,
				RequestType:    "chat",
				StatusCode:     resp.StatusCode(),
				RawBody:        resp.Body(),
			},
		}
	}

	var ollamaBody map[string]any
	if err := utils.ReadJSONBody(resp, &ollamaBody); err != nil {
		return nil, &schemas.ProviderError{
			Message:    fmt.Sprintf("decode response: %v", err),
			Type:       "decode_error",
			StatusCode: resp.StatusCode(),
			Meta: schemas.ErrorMeta{
				Provider:       string(p.id),
				ModelRequested: request.Model,
				RequestType:    "chat",
				StatusCode:     resp.StatusCode(),
				RawBody:        resp.Body(),
			},
		}
	}

	openaiMap := translation.OllamaBodyToOpenAI(ollamaBody)
	b, err := json.Marshal(openaiMap)
	if err != nil {
		return nil, &schemas.ProviderError{
			Message:    fmt.Sprintf("marshal response: %v", err),
			Type:       "decode_error",
			StatusCode: resp.StatusCode(),
			Meta: schemas.ErrorMeta{
				Provider:       string(p.id),
				ModelRequested: request.Model,
				RequestType:    "chat",
				StatusCode:     resp.StatusCode(),
				RawBody:        resp.Body(),
			},
		}
	}

	var result schemas.ChatResponse
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, &schemas.ProviderError{
			Message:    fmt.Sprintf("unmarshal response: %v", err),
			Type:       "decode_error",
			StatusCode: resp.StatusCode(),
			Meta: schemas.ErrorMeta{
				Provider:       string(p.id),
				ModelRequested: request.Model,
				RequestType:    "chat",
				StatusCode:     resp.StatusCode(),
				RawBody:        resp.Body(),
			},
		}
	}
	return &result, nil
}

// ChatCompletionStream sends a streaming chat completion request and returns a channel of chunks.
func (p *Provider) ChatCompletionStream(ctx *schemas.GatewayContext, postHookRunner schemas.PostHookRunner, key schemas.Key, request *schemas.ChatRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	resp := p.client.AcquireResponse()

	req.SetRequestURI(p.chatURL(key.ProviderSpecificData["baseUrl"]))
	req.Header.SetMethod(fasthttp.MethodPost)
	// No auth header for ollama (NoAuth == true).

	bodyMap, err := requestToMap(request)
	if err != nil {
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

	reqMap, err := p.registry.TranslateRequest(translation.FormatOpenAI, translation.FormatOllama, request.Model, bodyMap, true, nil)
	if err != nil {
		p.client.ReleaseRequest(req)
		p.client.ReleaseResponse(resp)
		return nil, &schemas.ProviderError{
			Message:    fmt.Sprintf("translate request: %v", err),
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

	if err := utils.SetJSONBody(req, reqMap); err != nil {
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
		return nil, &schemas.ProviderError{
			Message:    fmt.Sprintf("request failed: %v", err),
			Type:       "request_error",
			StatusCode: 0,
			Meta: schemas.ErrorMeta{
				Provider:       string(p.id),
				ModelRequested: request.Model,
				RequestType:    "chat_stream",
				StatusCode:     0,
				RawBody:        []byte(err.Error()),
			},
		}
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		p.client.ReleaseRequest(req)
		p.client.ReleaseResponse(resp)
		return nil, &schemas.ProviderError{
			Message:    string(resp.Body()),
			Type:       "provider_error",
			StatusCode: resp.StatusCode(),
			Meta: schemas.ErrorMeta{
				Provider:       string(p.id),
				ModelRequested: request.Model,
				RequestType:    "chat_stream",
				StatusCode:     resp.StatusCode(),
				RawBody:        resp.Body(),
			},
		}
	}

	p.client.ReleaseRequest(req)

	ch := make(chan *schemas.StreamChunk, 16)
	go func() {
		defer close(ch)
		defer p.client.ReleaseResponse(resp)

		body := bytes.NewReader(resp.Body())
		scanner := utils.NewNDJSONScanner(body)
		state := translation.NewStreamState()

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

			var lineMap map[string]any
			if err := json.Unmarshal([]byte(line), &lineMap); err != nil {
				// Malformed NDJSON is already skipped by the scanner (sse.go:71-78);
				// this path covers post-hook/read failures only.
				ch <- streamError(fmt.Sprintf("decode stream chunk: %v", err))
				return
			}

			chunks, err := p.registry.TranslateResponse(translation.FormatOllama, translation.FormatOpenAI, lineMap, state)
			if err != nil {
				ch <- streamError(fmt.Sprintf("translate response: %v", err))
				return
			}

			for _, c := range chunks {
				b, err := json.Marshal(c)
				if err != nil {
					ch <- streamError(fmt.Sprintf("marshal chunk: %v", err))
					return
				}
				var chunk schemas.StreamChunk
				if err := json.Unmarshal(b, &chunk); err != nil {
					ch <- streamError(fmt.Sprintf("unmarshal chunk: %v", err))
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
		}
	}()

	return ch, nil
}

// requestToMap converts a ChatRequest struct into a generic map.
func requestToMap(req *schemas.ChatRequest) (map[string]any, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("unmarshal request: %w", err)
	}
	return m, nil
}
