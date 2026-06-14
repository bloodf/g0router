package antigravity

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

// setHeaders configures the antigravity request headers: catalog headers
// (User-Agent), bearer auth, the internal x-request-source header, and the SSE
// Accept for streaming (executors/antigravity.js buildHeaders).
func (p *Provider) setHeaders(req *fasthttp.Request, key schemas.Key, stream bool) {
	for k, v := range p.config.Headers {
		req.Header.Set(k, v)
	}
	if !p.config.NoAuth {
		utils.SetAuthHeader(req, accessToken(key))
	}
	req.Header.Set(internalRequestHeaderName, internalRequestHeaderValue)
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	} else {
		req.Header.Set("Accept", "application/json")
	}
}

// accessToken returns the antigravity access token: the explicit accessToken in
// ProviderSpecificData if present, else the key value.
func accessToken(key schemas.Key) string {
	if key.ProviderSpecificData != nil {
		if t := key.ProviderSpecificData["accessToken"]; t != "" {
			return t
		}
	}
	return key.Value
}

// translatedRequestBody applies the PAR-MCP-060 tool cloaking, then translates
// the OpenAI request to the Antigravity (v1internal Gemini) wire shape via the
// existing converter. The suffixed->original tool name map is returned so the
// response path can restore client tool names.
func (p *Provider) translatedRequestBody(request *schemas.ChatRequest, stream bool) (map[string]any, map[string]string, error) {
	b, err := json.Marshal(request)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal request: %w", err)
	}
	var bodyMap map[string]any
	if err := json.Unmarshal(b, &bodyMap); err != nil {
		return nil, nil, fmt.Errorf("unmarshal request: %w", err)
	}

	// PAR-MCP-060 ride-along: cloak client tools and inject the unavailable
	// decoy tools before translation.
	nameMap := applyToolCloaking(bodyMap)

	reqMap, err := p.registry.TranslateRequest(translation.FormatOpenAI, translation.FormatAntigravity, request.Model, bodyMap, stream, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("translate request: %w", err)
	}
	return reqMap, nameMap, nil
}

// applyToolCloaking replaces bodyMap["tools"] with the cloaked tool list and
// returns the suffixed->original name map. No-op (returns nil) when there are no
// tools.
func applyToolCloaking(bodyMap map[string]any) map[string]string {
	rawTools, ok := bodyMap["tools"].([]any)
	if !ok || len(rawTools) == 0 {
		return nil
	}
	tools := make([]map[string]any, 0, len(rawTools))
	for _, t := range rawTools {
		// OpenAI tools are {type:"function", function:{name,description,...}};
		// flatten to the {name,description} shape cloakTools consumes.
		tm, ok := t.(map[string]any)
		if !ok {
			continue
		}
		if fn, ok := tm["function"].(map[string]any); ok {
			merged := map[string]any{}
			for k, v := range tm {
				merged[k] = v
			}
			if name, ok := fn["name"].(string); ok {
				merged["name"] = name
			}
			if desc, ok := fn["description"].(string); ok {
				merged["description"] = desc
			}
			tools = append(tools, merged)
			continue
		}
		tools = append(tools, tm)
	}
	cloaked, nameMap := cloakTools(tools)
	if cloaked == nil {
		return nameMap
	}
	out := make([]any, len(cloaked))
	for i, c := range cloaked {
		out[i] = c
	}
	bodyMap["tools"] = out
	return nameMap
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

// shouldFallback reports whether an upstream status should trigger a retry on the
// next fallback URL (5xx / 429), matching the ref's fallback semantics.
func shouldFallback(status int) bool {
	return status == fasthttp.StatusTooManyRequests || status >= 500
}

// post sends the translated request, trying each fallback URL in order until one
// succeeds or all fail. It returns the raw response body, or a provider error.
func (p *Provider) post(request *schemas.ChatRequest, key schemas.Key, requestType string, stream bool) ([]byte, *schemas.ProviderError) {
	reqMap, _, err := p.translatedRequestBody(request, stream)
	if err != nil {
		return nil, p.sendError(request, requestType, err.Error(), "invalid_request_error", 0, nil)
	}

	var lastErr *schemas.ProviderError
	for i := 0; i < p.fallbackCount(); i++ {
		body, perr, fallback := p.attempt(p.buildURL(i, stream), reqMap, request, key, requestType, stream)
		if perr == nil {
			return body, nil
		}
		lastErr = perr
		if !fallback || i+1 >= p.fallbackCount() {
			break
		}
	}
	return nil, lastErr
}

// attempt performs a single POST to url. It returns (body, nil, false) on
// success, or (nil, err, fallback) on failure where fallback indicates the next
// URL should be tried.
func (p *Provider) attempt(url string, reqMap map[string]any, request *schemas.ChatRequest, key schemas.Key, requestType string, stream bool) ([]byte, *schemas.ProviderError, bool) {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(url)
	req.Header.SetMethod(fasthttp.MethodPost)
	p.setHeaders(req, key, stream)

	if err := utils.SetJSONBody(req, reqMap); err != nil {
		return nil, p.sendError(request, requestType, fmt.Sprintf("build request: %v", err), "invalid_request_error", 0, nil), false
	}
	if err := p.client.Do(req, resp); err != nil {
		return nil, p.sendError(request, requestType, fmt.Sprintf("request failed: %v", err), "request_error", 0, []byte(err.Error())), true
	}
	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, p.sendError(request, requestType, string(resp.Body()), "provider_error", resp.StatusCode(), resp.Body()), shouldFallback(resp.StatusCode())
	}

	body := append([]byte(nil), resp.Body()...)
	return body, nil, false
}

// translateSSE parses a Gemini SSE body, translating each chunk to OpenAI chunks
// via geminiToOpenAIResponse.
func (p *Provider) translateSSE(raw []byte, request *schemas.ChatRequest, requestType string) ([]*schemas.StreamChunk, *schemas.ProviderError) {
	scanner := utils.NewSSEScanner(bytes.NewReader(raw))
	state := translation.NewStreamState()
	state.Model = request.Model
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
		var chunk map[string]any
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			return nil, p.sendError(request, requestType, fmt.Sprintf("decode stream chunk: %v", err), "decode_error", 0, raw)
		}
		converted, err := p.registry.TranslateResponse(translation.FormatAntigravity, translation.FormatOpenAI, chunk, state)
		if err != nil {
			return nil, p.sendError(request, requestType, fmt.Sprintf("translate response: %v", err), "decode_error", 0, raw)
		}
		for _, c := range converted {
			sc, err := mapToStreamChunk(c)
			if err != nil {
				return nil, p.sendError(request, requestType, err.Error(), "decode_error", 0, raw)
			}
			out = append(out, sc)
		}
	}
}

// ChatCompletion sends a non-streaming request and aggregates the translated
// chunks into a single ChatResponse.
func (p *Provider) ChatCompletion(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.ChatRequest) (*schemas.ChatResponse, *schemas.ProviderError) {
	body, perr := p.post(request, key, "chat", false)
	if perr != nil {
		return nil, perr
	}
	chunks, perr := p.translateSSE(body, request, "chat")
	if perr != nil {
		return nil, perr
	}
	return aggregateChunks(chunks, request.Model), nil
}

// ChatCompletionStream sends a streaming request and returns OpenAI chunks
// translated from the antigravity (Gemini) SSE response.
func (p *Provider) ChatCompletionStream(ctx *schemas.GatewayContext, postHookRunner schemas.PostHookRunner, key schemas.Key, request *schemas.ChatRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	body, perr := p.post(request, key, "chat_stream", true)
	if perr != nil {
		return nil, perr
	}

	ch := make(chan *schemas.StreamChunk, 16)
	go func() {
		defer close(ch)
		chunks, perr := p.translateSSE(body, request, "chat_stream")
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

// aggregateChunks folds streamed OpenAI delta chunks into a single ChatResponse.
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
