package cursor

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
)

// streamError builds the in-band terminal error chunk (AUD-045/046/047).
func streamError(msg string) *schemas.StreamChunk {
	return &schemas.StreamChunk{Error: &schemas.ProviderError{Message: msg, Type: "stream_error"}}
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

// requestMessages extracts the OpenAI messages from the request as a slice of
// {role, content} maps for the protobuf encoder.
func requestMessages(request *schemas.ChatRequest) []map[string]any {
	b, err := json.Marshal(request)
	if err != nil {
		return nil
	}
	var body map[string]any
	if err := json.Unmarshal(b, &body); err != nil {
		return nil
	}
	raw, _ := body["messages"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, m := range raw {
		if mm, ok := m.(map[string]any); ok {
			out = append(out, mm)
		}
	}
	return out
}

// requestTools extracts the OpenAI tools from the request as maps.
func requestTools(request *schemas.ChatRequest) []map[string]any {
	b, err := json.Marshal(request)
	if err != nil {
		return nil
	}
	var body map[string]any
	if err := json.Unmarshal(b, &body); err != nil {
		return nil
	}
	raw, _ := body["tools"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, t := range raw {
		if tm, ok := t.(map[string]any); ok {
			out = append(out, tm)
		}
	}
	return out
}

// post sends the connect-framed protobuf request and returns the raw response
// bytes, or a provider error.
func (p *Provider) post(request *schemas.ChatRequest, key schemas.Key, requestType string) ([]byte, *schemas.ProviderError) {
	machineID := ""
	ghost := true
	if key.ProviderSpecificData != nil {
		machineID = key.ProviderSpecificData["machineId"]
		if key.ProviderSpecificData["ghostMode"] == "false" {
			ghost = false
		}
	}

	reasoning := ""
	body := generateCursorBody(requestMessages(request), request.Model, requestTools(request), reasoning, false)

	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(p.chatURL())
	req.Header.SetMethod(fasthttp.MethodPost)
	for k, v := range buildCursorHeaders(key.Value, machineID, ghost, time.Now().UnixMilli()/1_000_000) {
		req.Header.Set(k, v)
	}
	req.SetBodyRaw(body)

	if err := p.client.Do(req, resp); err != nil {
		return nil, p.sendError(request, requestType, fmt.Sprintf("request failed: %v", err), "request_error", 0, []byte(err.Error()))
	}
	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, p.sendError(request, requestType, string(resp.Body()), "provider_error", resp.StatusCode(), resp.Body())
	}

	return append([]byte(nil), resp.Body()...), nil
}

// decodeFrames parses the connect-framed protobuf response into OpenAI stream
// chunk maps, then runs them through cursorToOpenAIResponse (passthrough) via the
// registry. Mirrors cursor.js transformProtobufToSSE chunk emission.
func (p *Provider) decodeFrames(raw []byte, request *schemas.ChatRequest, requestType string) ([]*schemas.StreamChunk, *schemas.ProviderError) {
	responseID := fmt.Sprintf("chatcmpl-cursor-%d", time.Now().UnixMilli())
	created := time.Now().Unix()

	state := translation.NewStreamState()
	state.Model = request.Model

	var out []*schemas.StreamChunk
	var emitted int
	toolIndexByID := map[string]int{}
	var toolCalls int
	var hasToolCalls bool

	emit := func(m map[string]any) *schemas.ProviderError {
		converted, err := p.registry.TranslateResponse(translation.FormatCursor, translation.FormatOpenAI, m, state)
		if err != nil {
			return p.sendError(request, requestType, fmt.Sprintf("translate response: %v", err), "decode_error", 0, raw)
		}
		for _, c := range converted {
			sc, err := mapToStreamChunk(c)
			if err != nil {
				return p.sendError(request, requestType, err.Error(), "decode_error", 0, raw)
			}
			out = append(out, sc)
			emitted++
		}
		return nil
	}

	offset := 0
	for offset < len(raw) {
		flags, payload, consumed, ok := parseConnectFrame(raw[offset:])
		_ = flags
		if !ok {
			break
		}
		offset += consumed

		res := extractTextFromResponse(payload)

		switch {
		case res.toolCall != nil:
			tc := res.toolCall
			hasToolCalls = true
			idx, seen := toolIndexByID[tc.id]
			if !seen {
				idx = toolCalls
				toolIndexByID[tc.id] = idx
				toolCalls++
			}
			delta := map[string]any{
				"tool_calls": []any{map[string]any{
					"index": idx,
					"id":    tc.id,
					"type":  "function",
					"function": map[string]any{
						"name":      tc.name,
						"arguments": tc.arguments,
					},
				}},
			}
			if emitted == 0 {
				delta["role"] = "assistant"
			}
			if perr := emit(chunkMap(responseID, created, request.Model, delta, nil)); perr != nil {
				return nil, perr
			}
		case res.text != "":
			delta := map[string]any{"content": res.text}
			if emitted == 0 {
				delta["role"] = "assistant"
			}
			if perr := emit(chunkMap(responseID, created, request.Model, delta, nil)); perr != nil {
				return nil, perr
			}
		}
	}

	// Final finish chunk.
	finish := "stop"
	if hasToolCalls {
		finish = "tool_calls"
	}
	if perr := emit(chunkMap(responseID, created, request.Model, map[string]any{}, &finish)); perr != nil {
		return nil, perr
	}

	return out, nil
}

// chunkMap builds an OpenAI chat.completion.chunk map.
func chunkMap(id string, created int64, model string, delta map[string]any, finishReason *string) map[string]any {
	choice := map[string]any{
		"index": 0,
		"delta": delta,
	}
	if finishReason != nil {
		choice["finish_reason"] = *finishReason
	} else {
		choice["finish_reason"] = nil
	}
	return map[string]any{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []any{choice},
	}
}

// ChatCompletion sends a non-streaming request and aggregates the decoded chunks
// into a single ChatResponse.
func (p *Provider) ChatCompletion(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.ChatRequest) (*schemas.ChatResponse, *schemas.ProviderError) {
	body, perr := p.post(request, key, "chat")
	if perr != nil {
		return nil, perr
	}
	chunks, perr := p.decodeFrames(body, request, "chat")
	if perr != nil {
		return nil, perr
	}
	return aggregateChunks(chunks, request.Model), nil
}

// ChatCompletionStream sends a streaming request and returns OpenAI chunks
// decoded from the cursor protobuf response.
func (p *Provider) ChatCompletionStream(ctx *schemas.GatewayContext, postHookRunner schemas.PostHookRunner, key schemas.Key, request *schemas.ChatRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	body, perr := p.post(request, key, "chat_stream")
	if perr != nil {
		return nil, perr
	}

	ch := make(chan *schemas.StreamChunk, 16)
	go func() {
		defer close(ch)
		chunks, perr := p.decodeFrames(body, request, "chat_stream")
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
