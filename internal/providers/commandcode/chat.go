package commandcode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
)

// streamError builds the in-band terminal error chunk (AUD-045/046/047).
func streamError(msg string) *schemas.StreamChunk {
	return &schemas.StreamChunk{Error: &schemas.ProviderError{Message: msg, Type: "stream_error"}}
}

// setHeaders configures the catalog headers and auth for the request.
func (p *Provider) setHeaders(req *fasthttp.Request, key schemas.Key) {
	for k, v := range p.config.Headers {
		req.Header.Set(k, v)
	}
	if !p.config.NoAuth {
		utils.SetAuthHeader(req, key.Value)
	}
}

// translatedRequestBody marshals the OpenAI request, translates it to the
// CommandCode custom-JSON shape via the existing converter, and returns the
// JSON-ready map.
func (p *Provider) translatedRequestBody(request *schemas.ChatRequest, stream bool) (map[string]any, error) {
	b, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	var bodyMap map[string]any
	if err := json.Unmarshal(b, &bodyMap); err != nil {
		return nil, fmt.Errorf("unmarshal request: %w", err)
	}
	reqMap, err := p.registry.TranslateRequest(translation.FormatOpenAI, translation.FormatCommandCode, request.Model, bodyMap, stream, nil)
	if err != nil {
		return nil, fmt.Errorf("translate request: %w", err)
	}
	return reqMap, nil
}

// sendError builds a ProviderError for the chat request type.
func (p *Provider) sendError(request *schemas.ChatRequest, requestType, msg, errType string, status int, raw []byte) *schemas.ProviderError {
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

// ChatCompletion sends a non-streaming request. CommandCode is SSE-only, so the
// adapter reads the event stream, translates each event commandcode->openai, and
// aggregates the resulting chunks into a single OpenAI ChatResponse.
func (p *Provider) ChatCompletion(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.ChatRequest) (*schemas.ChatResponse, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(p.config.BaseURL)
	req.Header.SetMethod(fasthttp.MethodPost)
	p.setHeaders(req, key)

	reqMap, err := p.translatedRequestBody(request, false)
	if err != nil {
		return nil, p.sendError(request, "chat", err.Error(), "invalid_request_error", 0, nil)
	}
	if err := utils.SetJSONBody(req, reqMap); err != nil {
		return nil, p.sendError(request, "chat", fmt.Sprintf("build request: %v", err), "invalid_request_error", 0, nil)
	}

	if err := p.client.Do(req, resp); err != nil {
		return nil, p.sendError(request, "chat", fmt.Sprintf("request failed: %v", err), "request_error", 0, []byte(err.Error()))
	}
	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, p.sendError(request, "chat", string(resp.Body()), "provider_error", resp.StatusCode(), resp.Body())
	}

	chunks, perr := p.translateStreamBody(resp.Body(), request, "chat")
	if perr != nil {
		return nil, perr
	}
	return aggregateChunks(chunks, request.Model), nil
}

// ChatCompletionStream sends a streaming request and returns OpenAI chunks
// translated from the CommandCode event stream.
func (p *Provider) ChatCompletionStream(ctx *schemas.GatewayContext, postHookRunner schemas.PostHookRunner, key schemas.Key, request *schemas.ChatRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	resp := p.client.AcquireResponse()

	req.SetRequestURI(p.config.BaseURL)
	req.Header.SetMethod(fasthttp.MethodPost)
	p.setHeaders(req, key)

	reqMap, err := p.translatedRequestBody(request, true)
	if err != nil {
		p.client.ReleaseRequest(req)
		p.client.ReleaseResponse(resp)
		return nil, p.sendError(request, "chat_stream", err.Error(), "invalid_request_error", 0, nil)
	}
	if err := utils.SetJSONBody(req, reqMap); err != nil {
		p.client.ReleaseRequest(req)
		p.client.ReleaseResponse(resp)
		return nil, p.sendError(request, "chat_stream", fmt.Sprintf("build request: %v", err), "invalid_request_error", 0, nil)
	}

	if err := p.client.Do(req, resp); err != nil {
		p.client.ReleaseRequest(req)
		p.client.ReleaseResponse(resp)
		return nil, p.sendError(request, "chat_stream", fmt.Sprintf("request failed: %v", err), "request_error", 0, []byte(err.Error()))
	}
	if resp.StatusCode() != fasthttp.StatusOK {
		perr := p.sendError(request, "chat_stream", string(resp.Body()), "provider_error", resp.StatusCode(), resp.Body())
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
		state := translation.NewStreamState()

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

			var event map[string]any
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				ch <- streamError(fmt.Sprintf("decode stream event: %v", err))
				return
			}

			converted, err := p.registry.TranslateResponse(translation.FormatCommandCode, translation.FormatOpenAI, event, state)
			if err != nil {
				ch <- streamError(fmt.Sprintf("translate response: %v", err))
				return
			}
			for _, c := range converted {
				chunk, err := mapToStreamChunk(c)
				if err != nil {
					ch <- streamError(err.Error())
					return
				}
				ch <- chunk
				if postHookRunner != nil {
					if err := postHookRunner.Run(ctx, chunk); err != nil {
						ch <- streamError(fmt.Sprintf("post hook: %v", err))
						return
					}
				}
			}
		}
	}()

	return ch, nil
}

// translateStreamBody parses an SSE body, translating each CommandCode event to
// OpenAI chunks. Used by the non-streaming path to collect the full response.
func (p *Provider) translateStreamBody(raw []byte, request *schemas.ChatRequest, requestType string) ([]*schemas.StreamChunk, *schemas.ProviderError) {
	scanner := utils.NewSSEScanner(bytes.NewReader(raw))
	state := translation.NewStreamState()
	var out []*schemas.StreamChunk
	for {
		line, err := scanner.Scan()
		if err != nil {
			if err == io.EOF {
				return out, nil
			}
			return nil, p.sendError(request, requestType, fmt.Sprintf("read stream: %v", err), "decode_error", 0, raw)
		}
		if line == "[DONE]" {
			return out, nil
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, p.sendError(request, requestType, fmt.Sprintf("decode stream event: %v", err), "decode_error", 0, raw)
		}
		converted, err := p.registry.TranslateResponse(translation.FormatCommandCode, translation.FormatOpenAI, event, state)
		if err != nil {
			return nil, p.sendError(request, requestType, fmt.Sprintf("translate response: %v", err), "decode_error", 0, raw)
		}
		for _, c := range converted {
			chunk, err := mapToStreamChunk(c)
			if err != nil {
				return nil, p.sendError(request, requestType, err.Error(), "decode_error", 0, raw)
			}
			out = append(out, chunk)
		}
	}
}

// mapToStreamChunk converts a translated OpenAI chunk map into a StreamChunk.
func mapToStreamChunk(m map[string]any) (*schemas.StreamChunk, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal chunk: %w", err)
	}
	var chunk schemas.StreamChunk
	if err := json.Unmarshal(b, &chunk); err != nil {
		return nil, fmt.Errorf("unmarshal chunk: %w", err)
	}
	return &chunk, nil
}

// aggregateChunks folds streamed OpenAI delta chunks into a single
// non-streaming ChatResponse.
func aggregateChunks(chunks []*schemas.StreamChunk, model string) *schemas.ChatResponse {
	resp := &schemas.ChatResponse{
		Object: "chat.completion",
		Model:  model,
		Choices: []schemas.Choice{{
			Index:   0,
			Message: &schemas.Message{Role: "assistant"},
		}},
	}
	var content string
	var reasoning string
	var toolCalls []schemas.ToolCall
	for _, c := range chunks {
		if c == nil {
			continue
		}
		if resp.ID == "" && c.ID != "" {
			resp.ID = c.ID
		}
		if c.Usage != nil {
			resp.Usage = c.Usage
		}
		for _, ch := range c.Choices {
			content += ch.Delta.Content
			if ch.Delta.ReasoningContent != nil {
				reasoning += *ch.Delta.ReasoningContent
			}
			toolCalls = append(toolCalls, ch.Delta.ToolCalls...)
			if ch.FinishReason != nil && *ch.FinishReason != "" {
				resp.Choices[0].FinishReason = *ch.FinishReason
			}
		}
	}
	resp.Choices[0].Message.Content = content
	if reasoning != "" {
		resp.Choices[0].Message.ReasoningContent = &reasoning
	}
	if len(toolCalls) > 0 {
		resp.Choices[0].Message.ToolCalls = toolCalls
	}
	return resp
}
