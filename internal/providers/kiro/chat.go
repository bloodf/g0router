package kiro

import (
	"encoding/json"
	"fmt"

	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
)

// streamError builds the in-band terminal error chunk (AUD-045/046/047).
func streamError(msg string) *schemas.StreamChunk {
	return &schemas.StreamChunk{Error: &schemas.ProviderError{Message: msg, Type: "stream_error"}}
}

// requestURL resolves the endpoint URL. The urlOverride test seam takes
// precedence over the catalog base URL.
func (p *Provider) requestURL() string {
	if p.urlOverride != "" {
		return p.urlOverride
	}
	return p.config.BaseURL
}

// setHeaders configures the catalog headers and bearer auth for the request.
// Kiro authenticates with an OAuth access token carried as a Bearer token
// (ref executors/kiro.js:24-26).
func (p *Provider) setHeaders(req *fasthttp.Request, key schemas.Key) {
	for k, v := range p.config.Headers {
		req.Header.Set(k, v)
	}
	if !p.config.NoAuth {
		utils.SetAuthHeader(req, accessToken(key))
	}
}

// accessToken returns the Kiro access token: the explicit accessToken in
// ProviderSpecificData if present, else the key value.
func accessToken(key schemas.Key) string {
	if key.ProviderSpecificData != nil {
		if t := key.ProviderSpecificData["accessToken"]; t != "" {
			return t
		}
	}
	return key.Value
}

// credentialsMap builds the credentials map buildKiroPayload reads (it consumes
// providerSpecificData.profileArn). No secret is logged.
func credentialsMap(key schemas.Key) map[string]any {
	psd := map[string]any{}
	for k, v := range key.ProviderSpecificData {
		psd[k] = v
	}
	return map[string]any{"providerSpecificData": psd}
}

// translatedRequestBody marshals the OpenAI request, translates it to the Kiro
// custom-JSON shape via the existing converter, and returns the JSON-ready map.
func (p *Provider) translatedRequestBody(request *schemas.ChatRequest, key schemas.Key, stream bool) (map[string]any, error) {
	b, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	var bodyMap map[string]any
	if err := json.Unmarshal(b, &bodyMap); err != nil {
		return nil, fmt.Errorf("unmarshal request: %w", err)
	}
	reqMap, err := p.registry.TranslateRequest(translation.FormatOpenAI, translation.FormatKiro, request.Model, bodyMap, stream, credentialsMap(key))
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

// post sends the translated request and returns the raw eventstream response
// body, or a provider error.
func (p *Provider) post(request *schemas.ChatRequest, key schemas.Key, requestType string, stream bool) ([]byte, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(p.requestURL())
	req.Header.SetMethod(fasthttp.MethodPost)
	p.setHeaders(req, key)

	reqMap, err := p.translatedRequestBody(request, key, stream)
	if err != nil {
		return nil, p.sendError(request, requestType, err.Error(), "invalid_request_error", 0, nil)
	}
	if err := utils.SetJSONBody(req, reqMap); err != nil {
		return nil, p.sendError(request, requestType, fmt.Sprintf("build request: %v", err), "invalid_request_error", 0, nil)
	}

	if err := p.client.Do(req, resp); err != nil {
		return nil, p.sendError(request, requestType, fmt.Sprintf("request failed: %v", err), "request_error", 0, []byte(err.Error()))
	}
	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, p.sendError(request, requestType, string(resp.Body()), "provider_error", resp.StatusCode(), resp.Body())
	}

	// Copy the body out of the pooled response before it is released.
	body := append([]byte(nil), resp.Body()...)
	return body, nil
}

// decodeToChunks decodes the eventstream body into frames and translates each
// via kiroToOpenAIResponse to OpenAI stream chunks.
func (p *Provider) decodeToChunks(body []byte, request *schemas.ChatRequest, requestType string) ([]*schemas.StreamChunk, *schemas.ProviderError) {
	events, err := DecodeEventStream(body)
	if err != nil {
		return nil, p.sendError(request, requestType, fmt.Sprintf("decode eventstream: %v", err), "decode_error", 0, body)
	}

	state := translation.NewStreamState()
	state.Model = request.Model
	var out []*schemas.StreamChunk
	for _, ev := range events {
		converted, err := p.registry.TranslateResponse(translation.FormatKiro, translation.FormatOpenAI, ev, state)
		if err != nil {
			return nil, p.sendError(request, requestType, fmt.Sprintf("translate response: %v", err), "decode_error", 0, body)
		}
		for _, c := range converted {
			chunk, err := mapToStreamChunk(c)
			if err != nil {
				return nil, p.sendError(request, requestType, err.Error(), "decode_error", 0, body)
			}
			out = append(out, chunk)
		}
	}
	return out, nil
}

// ChatCompletion sends a non-streaming request, decodes the eventstream, and
// aggregates the resulting OpenAI chunks into a single ChatResponse.
func (p *Provider) ChatCompletion(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.ChatRequest) (*schemas.ChatResponse, *schemas.ProviderError) {
	body, perr := p.post(request, key, "chat", false)
	if perr != nil {
		return nil, perr
	}
	chunks, perr := p.decodeToChunks(body, request, "chat")
	if perr != nil {
		return nil, perr
	}
	return aggregateChunks(chunks, request.Model), nil
}

// ChatCompletionStream sends a streaming request and returns OpenAI chunks
// translated from the Kiro eventstream.
func (p *Provider) ChatCompletionStream(ctx *schemas.GatewayContext, postHookRunner schemas.PostHookRunner, key schemas.Key, request *schemas.ChatRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	body, perr := p.post(request, key, "chat_stream", true)
	if perr != nil {
		return nil, perr
	}

	ch := make(chan *schemas.StreamChunk, 16)
	go func() {
		defer close(ch)

		chunks, perr := p.decodeToChunks(body, request, "chat_stream")
		if perr != nil {
			ch <- streamError(perr.Message)
			return
		}
		for _, chunk := range chunks {
			ch <- chunk
			if postHookRunner != nil {
				if err := postHookRunner.Run(ctx, chunk); err != nil {
					ch <- streamError(fmt.Sprintf("post hook: %v", err))
					return
				}
			}
		}
	}()

	return ch, nil
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
