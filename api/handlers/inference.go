package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/proxy"
	"github.com/bloodf/g0router/internal/streaming"
	"github.com/bloodf/g0router/internal/translate"
	"github.com/valyala/fasthttp"
)

type errorResponse struct {
	Error string `json:"error"`
}

type openAIErrorResponse struct {
	Error openAIErrorBody `json:"error"`
}

type openAIErrorBody struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

func Inference(ctx *fasthttp.RequestCtx, engine InferenceEngine) {
	if engine == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "inference engine unavailable")
		return
	}

	var req providers.ChatRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Stream != nil && *req.Stream {
		streamInference(ctx, engine, &req)
		return
	}

	resp, err := engine.Dispatch(requestContext(ctx), &req)
	if err != nil {
		writeDispatchError(ctx, err)
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, resp)
}

func streamInference(ctx *fasthttp.RequestCtx, engine InferenceEngine, req *providers.ChatRequest) {
	streamCtx, cancel := context.WithCancel(context.Background())
	stream, err := engine.DispatchStream(streamCtx, req)
	if err != nil {
		cancel()
		writeDispatchError(ctx, err)
		return
	}

	ctx.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		// Cancelling when the writer loop exits (client disconnect or normal
		// completion) lets producer goroutines blocked on a send abandon and
		// unwind instead of stalling until the upstream stream closes.
		defer cancel()
		for chunk := range stream {
			if chunk.Error != nil {
				writeStreamError(w, chunk.Error)
				return
			}
			data, err := json.Marshal(chunk)
			if err != nil {
				writeStreamMarshalError(w)
				return
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			_ = w.Flush()
		}
		_, _ = w.WriteString("data: [DONE]\n\n")
		_ = w.Flush()
	})
}

// writeStreamMarshalError emits a terminal SSE error event when a chunk cannot
// be serialized, so the client sees a failure signal instead of a stream that
// is silently truncated mid-flight.
func writeStreamMarshalError(w *bufio.Writer) {
	writeStreamError(w, &providers.StreamError{
		Message: "stream encoding error",
		Type:    "server_error",
		Code:    "stream_encoding_error",
	})
}

func writeStreamError(w *bufio.Writer, err *providers.StreamError) {
	message := "upstream provider stream error"
	errorType := "server_error"
	code := "upstream_stream_error"
	if err != nil {
		if err.Type != "" {
			errorType = err.Type
		}
		if err.Code != "" {
			code = err.Code
		}
	}
	data, marshalErr := json.Marshal(openAIErrorResponse{Error: openAIErrorBody{
		Message: message,
		Type:    errorType,
		Code:    code,
	}})
	if marshalErr == nil {
		_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
	}
	_ = w.Flush()
}

func Messages(ctx *fasthttp.RequestCtx, engine InferenceEngine) {
	if engine == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "inference engine unavailable")
		return
	}
	if err := rejectUnsupportedAnthropicMessageShape(ctx.PostBody()); err != nil {
		writeError(ctx, fasthttp.StatusNotImplemented, err.Error())
		return
	}

	req, err := translateAnthropicMessagesRequest(ctx.PostBody())
	if err != nil {
		if errors.Is(err, errAnthropicTranslate) {
			writeError(ctx, fasthttp.StatusNotImplemented, err.Error())
			return
		}
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Stream != nil && *req.Stream {
		streamMessages(ctx, engine, req)
		return
	}

	resp, err := engine.Dispatch(requestContext(ctx), req)
	if err != nil {
		writeDispatchError(ctx, err)
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, anthropicMessageResponse(resp))
}

func streamMessages(ctx *fasthttp.RequestCtx, engine InferenceEngine, req *providers.ChatRequest) {
	streamCtx, cancel := context.WithCancel(context.Background())
	stream, err := engine.DispatchStream(streamCtx, req)
	if err != nil {
		cancel()
		writeDispatchError(ctx, err)
		return
	}

	ctx.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		// See streamInference: cancel on writer-loop exit to unblock producers.
		defer cancel()
		started := false
		contentStarted := false
		// toolBlockOpen tracks whether a tool_use content block is currently
		// open. blockIndex is the index of the currently open block; it
		// increments as text and tool_use blocks open in sequence.
		toolBlockOpen := false
		blockIndex := 0
		message := anthropicMessageBody{Type: "message", Role: "assistant", Model: req.Model}
		writeErr := false
		emit := func(eventType string, payload any) {
			if writeErr {
				return
			}
			if err := writeAnthropicStreamEvent(w, eventType, payload); err != nil {
				writeStreamMarshalError(w)
				writeErr = true
			}
		}
		ensureStarted := func() {
			if !started {
				emit("message_start", map[string]any{
					"type":    "message_start",
					"message": message,
				})
				started = true
			}
		}
		for chunk := range stream {
			if chunk.Error != nil {
				writeStreamError(w, chunk.Error)
				return
			}
			updateAnthropicStreamMessage(&message, chunk)
			if !started && shouldStartAnthropicMessage(chunk) {
				ensureStarted()
			}
			for _, choice := range chunk.Choices {
				if choice.Delta.Content != nil {
					ensureStarted()
					if !contentStarted {
						emit("content_block_start", map[string]any{
							"type":  "content_block_start",
							"index": blockIndex,
							"content_block": map[string]string{
								"type": "text",
								"text": "",
							},
						})
						contentStarted = true
					}
					if *choice.Delta.Content != "" {
						emit("content_block_delta", map[string]any{
							"type":  "content_block_delta",
							"index": blockIndex,
							"delta": map[string]string{
								"type": "text_delta",
								"text": *choice.Delta.Content,
							},
						})
					}
				}
				for _, call := range choice.Delta.ToolCalls {
					ensureStarted()
					// Heuristic: a fragment starts a NEW tool_use block iff it
					// carries an id or function name; otherwise its Arguments
					// continue the currently open block. Fully-interleaved
					// parallel tool calls without ids are not disambiguated.
					if call.ID != "" || call.Function.Name != "" {
						if contentStarted {
							emit("content_block_stop", map[string]any{
								"type":  "content_block_stop",
								"index": blockIndex,
							})
							contentStarted = false
						}
						if toolBlockOpen {
							emit("content_block_stop", map[string]any{
								"type":  "content_block_stop",
								"index": blockIndex,
							})
						}
						blockIndex++
						toolBlockOpen = true
						emit("content_block_start", map[string]any{
							"type":  "content_block_start",
							"index": blockIndex,
							"content_block": map[string]any{
								"type":  "tool_use",
								"id":    call.ID,
								"name":  call.Function.Name,
								"input": map[string]any{},
							},
						})
					}
					if toolBlockOpen && call.Function.Arguments != "" {
						emit("content_block_delta", map[string]any{
							"type":  "content_block_delta",
							"index": blockIndex,
							"delta": map[string]string{
								"type":         "input_json_delta",
								"partial_json": call.Function.Arguments,
							},
						})
					}
				}
				if choice.FinishReason != nil {
					ensureStarted()
					if contentStarted {
						emit("content_block_stop", map[string]any{
							"type":  "content_block_stop",
							"index": blockIndex,
						})
						contentStarted = false
					}
					if toolBlockOpen {
						emit("content_block_stop", map[string]any{
							"type":  "content_block_stop",
							"index": blockIndex,
						})
						toolBlockOpen = false
					}
					emit("message_delta", map[string]any{
						"type": "message_delta",
						"delta": map[string]any{
							"stop_reason":   anthropicStreamStopReason(choice.FinishReason),
							"stop_sequence": nil,
						},
						"usage": anthropicStreamUsage(chunk.Usage),
					})
					emit("message_stop", map[string]string{
						"type": "message_stop",
					})
				}
			}
			if writeErr {
				return
			}
		}
		_ = w.Flush()
	})
}

func updateAnthropicStreamMessage(message *anthropicMessageBody, chunk providers.StreamChunk) {
	if chunk.ID != "" {
		message.ID = chunk.ID
	}
	if chunk.Model != "" {
		message.Model = chunk.Model
	}
	if chunk.Usage != nil {
		message.Usage = anthropicMessageUsage{
			InputTokens:  chunk.Usage.PromptTokens,
			OutputTokens: chunk.Usage.CompletionTokens,
		}
	}
	for _, choice := range chunk.Choices {
		if choice.Delta.Role != nil && *choice.Delta.Role != "" {
			message.Role = *choice.Delta.Role
		}
	}
}

func shouldStartAnthropicMessage(chunk providers.StreamChunk) bool {
	if chunk.ID != "" || chunk.Model != "" || chunk.Usage != nil {
		return true
	}
	for _, choice := range chunk.Choices {
		if choice.Delta.Role != nil || choice.Delta.Content != nil || choice.FinishReason != nil {
			return true
		}
	}
	return false
}

func writeAnthropicStreamEvent(w *bufio.Writer, eventType string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(w, "event: %s\n", eventType)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
	_ = w.Flush()
	return nil
}

func anthropicStreamUsage(usage *providers.Usage) map[string]int {
	if usage == nil {
		return map[string]int{"output_tokens": 0}
	}
	return map[string]int{"output_tokens": usage.CompletionTokens}
}

func Responses(ctx *fasthttp.RequestCtx, engine InferenceEngine) {
	if engine == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "inference engine unavailable")
		return
	}

	var req translate.ResponsesRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}

	chatReq, err := translate.ResponsesRequestToOpenAIChat(&req)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}
	if req.Stream != nil && *req.Stream {
		streamResponses(ctx, engine, chatReq)
		return
	}
	resp, err := engine.Dispatch(requestContext(ctx), chatReq)
	if err != nil {
		writeDispatchError(ctx, err)
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, translate.OpenAIChatToResponsesResponse(resp))
}

func streamResponses(ctx *fasthttp.RequestCtx, engine InferenceEngine, req *providers.ChatRequest) {
	streamCtx, cancel := context.WithCancel(context.Background())
	stream, err := engine.DispatchStream(streamCtx, req)
	if err != nil {
		cancel()
		writeDispatchError(ctx, err)
		return
	}

	ctx.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		// See streamInference: cancel on writer-loop exit to unblock producers.
		defer cancel()
		accumulator := streaming.NewResponsesAccumulator()
		toolCalls := newResponsesToolCallAccumulator()
		sequence := 0
		responseID := ""
		model := ""
		createdAt := int64(0)
		for chunk := range stream {
			if chunk.Error != nil {
				writeStreamError(w, chunk.Error)
				return
			}
			if chunk.ID != "" {
				responseID = chunk.ID
			}
			if chunk.Model != "" {
				model = chunk.Model
			}
			if chunk.Created != 0 {
				createdAt = chunk.Created
			}
			for _, choice := range chunk.Choices {
				if choice.Delta.Content != nil && *choice.Delta.Content != "" {
					sequence++
					event := streaming.ResponseEvent{
						Type:           "response.output_text.delta",
						Delta:          *choice.Delta.Content,
						SequenceNumber: sequence,
					}
					accumulator.AddEvent(event)
					if err := writeResponseStreamEvent(w, event.Type, event); err != nil {
						writeStreamMarshalError(w)
						return
					}
				}
				for _, call := range choice.Delta.ToolCalls {
					itemID := toolCalls.add(call)
					if call.Function.Arguments != "" {
						sequence++
						if err := writeResponsesFunctionCallDelta(w, itemID, call.Function.Arguments, sequence); err != nil {
							writeStreamMarshalError(w)
							return
						}
					}
				}
				if choice.FinishReason != nil {
					response := streaming.Response{
						ID:        responseID,
						Object:    "response",
						CreatedAt: createdAt,
						Model:     model,
						Status:    "completed",
						Usage:     streamResponseUsage(chunk.Usage),
					}
					doneEvent := streaming.ResponseEvent{
						Type:           "response.output_text.done",
						Text:           accumulator.Response().OutputText,
						SequenceNumber: sequence + 1,
					}
					accumulator.AddEvent(doneEvent)
					if err := writeResponseStreamEvent(w, doneEvent.Type, doneEvent); err != nil {
						writeStreamMarshalError(w)
						return
					}
					if err := writeResponsesCompleted(w, response, accumulator.Response().OutputText, toolCalls.outputs(), sequence+2); err != nil {
						writeStreamMarshalError(w)
						return
					}
				}
			}
		}
		_, _ = w.WriteString("data: [DONE]\n\n")
		_ = w.Flush()
	})
}

// responsesToolCallAccumulator stitches fragmented streaming tool-call deltas
// (id/name on the first fragment, arguments across later fragments) back into
// complete function calls, preserving emission order. Fragments without an id
// or name continue the most recently opened call.
type responsesToolCallAccumulator struct {
	order   []string
	byID    map[string]*responsesToolCall
	lastID  string
	counter int
}

type responsesToolCall struct {
	id        string
	name      string
	arguments string
}

func newResponsesToolCallAccumulator() *responsesToolCallAccumulator {
	return &responsesToolCallAccumulator{byID: make(map[string]*responsesToolCall)}
}

// add records a tool-call delta and returns the stable item id used to
// correlate streaming function-call-arguments events with the final call.
func (a *responsesToolCallAccumulator) add(call providers.ToolCall) string {
	id := call.ID
	if id == "" && call.Function.Name == "" {
		id = a.lastID
	}
	if id == "" {
		a.counter++
		id = fmt.Sprintf("call_%d", a.counter)
	}
	entry, ok := a.byID[id]
	if !ok {
		entry = &responsesToolCall{id: id}
		a.byID[id] = entry
		a.order = append(a.order, id)
	}
	if call.ID != "" {
		entry.id = call.ID
	}
	if call.Function.Name != "" {
		entry.name = call.Function.Name
	}
	entry.arguments += call.Function.Arguments
	a.lastID = id
	return id
}

func (a *responsesToolCallAccumulator) outputs() []responsesCompletedOutput {
	if len(a.order) == 0 {
		return nil
	}
	outputs := make([]responsesCompletedOutput, 0, len(a.order))
	for _, id := range a.order {
		entry := a.byID[id]
		outputs = append(outputs, responsesCompletedOutput{
			Type:      "function_call",
			CallID:    entry.id,
			Name:      entry.name,
			Arguments: entry.arguments,
		})
	}
	return outputs
}

// responsesCompletedOutput mirrors streaming.ResponseOutput while also carrying
// the function-call fields the streaming package's output type omits, so the
// response.completed payload can include emitted function calls.
type responsesCompletedOutput struct {
	Type      string                       `json:"type"`
	Role      string                       `json:"role,omitempty"`
	Content   []streaming.ResponseContent  `json:"content,omitempty"`
	CallID    string                       `json:"call_id,omitempty"`
	Name      string                       `json:"name,omitempty"`
	Arguments string                       `json:"arguments,omitempty"`
}

// responsesCompletedResponse mirrors streaming.Response but replaces Output with
// a type that can represent both message and function_call entries.
type responsesCompletedResponse struct {
	ID         string                     `json:"id"`
	Object     string                     `json:"object"`
	CreatedAt  int64                      `json:"created_at"`
	Model      string                     `json:"model"`
	Status     string                     `json:"status"`
	OutputText string                     `json:"output_text,omitempty"`
	Output     []responsesCompletedOutput `json:"output,omitempty"`
	Usage      *streaming.ResponseUsage   `json:"usage,omitempty"`
}

type responsesCompletedEvent struct {
	Type           string                     `json:"type"`
	SequenceNumber int                        `json:"sequence_number,omitempty"`
	Response       responsesCompletedResponse `json:"response"`
}

type responsesFunctionCallDeltaEvent struct {
	Type           string `json:"type"`
	ItemID         string `json:"item_id"`
	Delta          string `json:"delta"`
	SequenceNumber int    `json:"sequence_number,omitempty"`
}

func writeResponsesFunctionCallDelta(w *bufio.Writer, itemID, delta string, sequence int) error {
	event := responsesFunctionCallDeltaEvent{
		Type:           "response.function_call_arguments.delta",
		ItemID:         itemID,
		Delta:          delta,
		SequenceNumber: sequence,
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(w, "event: %s\n", event.Type)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
	_ = w.Flush()
	return nil
}

func writeResponsesCompleted(w *bufio.Writer, resp streaming.Response, outputText string, toolOutputs []responsesCompletedOutput, sequence int) error {
	completed := responsesCompletedResponse{
		ID:         resp.ID,
		Object:     "response",
		CreatedAt:  resp.CreatedAt,
		Model:      resp.Model,
		Status:     resp.Status,
		OutputText: outputText,
		Usage:      resp.Usage,
	}
	if outputText != "" {
		completed.Output = append(completed.Output, responsesCompletedOutput{
			Type: "message",
			Role: "assistant",
			Content: []streaming.ResponseContent{{
				Type: "output_text",
				Text: outputText,
			}},
		})
	}
	completed.Output = append(completed.Output, toolOutputs...)
	event := responsesCompletedEvent{
		Type:           "response.completed",
		SequenceNumber: sequence,
		Response:       completed,
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(w, "event: %s\n", event.Type)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
	_ = w.Flush()
	return nil
}

func writeResponseStreamEvent(w *bufio.Writer, eventType string, event streaming.ResponseEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(w, "event: %s\n", eventType)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
	_ = w.Flush()
	return nil
}

func streamResponseUsage(usage *providers.Usage) *streaming.ResponseUsage {
	if usage == nil {
		return nil
	}
	return &streaming.ResponseUsage{
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
		TotalTokens:  usage.TotalTokens,
	}
}

func writeDispatchError(ctx *fasthttp.RequestCtx, err error) {
	classification := proxy.ClassifyDispatchError(err)
	writeOpenAIError(ctx, classification.StatusCode, classification.Message, classification.Type, classification.Code)
}

type anthropicMessageContent struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type anthropicMessageUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicMessageBody struct {
	ID         string                    `json:"id"`
	Type       string                    `json:"type"`
	Role       string                    `json:"role"`
	Model      string                    `json:"model"`
	Content    []anthropicMessageContent `json:"content"`
	StopReason *string                   `json:"stop_reason,omitempty"`
	Usage      anthropicMessageUsage     `json:"usage"`
}

func anthropicMessageResponse(resp *providers.ChatResponse) anthropicMessageBody {
	body := anthropicMessageBody{Type: "message"}
	if resp == nil {
		return body
	}
	body.ID = resp.ID
	body.Model = resp.Model
	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		body.Role = choice.Message.Role
		body.StopReason = anthropicStopReason(choice.FinishReason)
		if text := messageContentText(choice.Message.Content); text != "" {
			body.Content = append(body.Content, anthropicMessageContent{Type: "text", Text: text})
		}
		for _, toolCall := range choice.Message.ToolCalls {
			body.Content = append(body.Content, anthropicMessageContent{
				Type:  "tool_use",
				ID:    toolCall.ID,
				Name:  toolCall.Function.Name,
				Input: toolCallInput(toolCall.Function.Arguments),
			})
		}
	}
	if body.Role == "" {
		body.Role = "assistant"
	}
	if resp.Usage != nil {
		body.Usage = anthropicMessageUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		}
	}
	return body
}

// anthropicRequestEnvelope mirrors the inbound Anthropic /v1/messages body
// closely enough to translate its content blocks into the internal/OpenAI
// ChatRequest shape while preserving tool-call identifiers.
type anthropicRequestEnvelope struct {
	Model      string                    `json:"model"`
	Messages   []anthropicInboundMessage `json:"messages"`
	Stream     *bool                     `json:"stream,omitempty"`
	Tools      []anthropicInboundTool    `json:"tools"`
	ToolChoice json.RawMessage           `json:"tool_choice"`
}

type anthropicInboundTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
	Type        string          `json:"type"`
}

type anthropicInboundMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type anthropicInboundBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
}

// translateAnthropicMessagesRequest converts an inbound Anthropic messages body
// into the internal ChatRequest. Anthropic tool_use blocks map to assistant
// tool_calls (preserving ids), and tool_result blocks map to tool-role messages
// whose tool_call_id is the originating tool_use_id, so identifiers survive the
// translation. Bodies whose content is a plain string fall through to the
// standard decoder unchanged.
func translateAnthropicMessagesRequest(body []byte) (*providers.ChatRequest, error) {
	var req providers.ChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	var envelope anthropicRequestEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}

	messages := make([]providers.Message, 0, len(envelope.Messages))
	for _, inbound := range envelope.Messages {
		translated, err := translateAnthropicInboundMessage(inbound)
		if err != nil {
			return nil, err
		}
		messages = append(messages, translated...)
	}
	req.Messages = messages

	tools, err := translateAnthropicTools(envelope.Tools)
	if err != nil {
		return nil, err
	}
	req.Tools = tools

	choice, err := translateAnthropicToolChoice(envelope.ToolChoice)
	if err != nil {
		return nil, err
	}
	req.ToolChoice = choice

	return &req, nil
}

// errAnthropicTranslate marks an Anthropic feature that has no representable
// OpenAI equivalent, so the handler can answer 501 rather than 400.
var errAnthropicTranslate = errors.New("messages translation unsupported")

// translateAnthropicTools converts inbound Anthropic tool definitions
// ({name, description, input_schema}) into internal/OpenAI function tools
// ({type:"function", function:{name, description, parameters: <input_schema>}}).
// Server-side tools (identified by a non-empty type, e.g. web_search) have no
// OpenAI function equivalent and are rejected specifically.
func translateAnthropicTools(tools []anthropicInboundTool) ([]providers.Tool, error) {
	if len(tools) == 0 {
		return nil, nil
	}
	out := make([]providers.Tool, 0, len(tools))
	for i, tool := range tools {
		if tool.Type != "" && tool.Type != "custom" && tool.Type != "function" {
			return nil, fmt.Errorf("%w: tool %d type %q has no OpenAI equivalent", errAnthropicTranslate, i, tool.Type)
		}
		out = append(out, providers.Tool{
			Type: "function",
			Function: providers.ToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}
	return out, nil
}

// translateAnthropicToolChoice maps an Anthropic tool_choice to its OpenAI
// equivalent: {type:auto}->"auto", {type:any}->"required",
// {type:tool,name:X}->{type:"function",function:{name:X}}. Absent/null leaves
// it unset. A bare string passes through. Unknown variants are rejected.
func translateAnthropicToolChoice(raw json.RawMessage) (any, error) {
	trimmed := bytesTrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil, nil
	}
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return nil, err
		}
		return s, nil
	}
	var choice struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(trimmed, &choice); err != nil {
		return nil, err
	}
	switch choice.Type {
	case "auto":
		return "auto", nil
	case "any":
		return "required", nil
	case "tool":
		return map[string]any{
			"type":     "function",
			"function": map[string]any{"name": choice.Name},
		}, nil
	default:
		return nil, fmt.Errorf("%w: tool_choice type %q has no OpenAI equivalent", errAnthropicTranslate, choice.Type)
	}
}

func translateAnthropicInboundMessage(inbound anthropicInboundMessage) ([]providers.Message, error) {
	trimmed := bytesTrimSpace(inbound.Content)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		var content any
		if len(trimmed) > 0 && string(trimmed) != "null" {
			if err := json.Unmarshal(trimmed, &content); err != nil {
				return nil, err
			}
		}
		return []providers.Message{{Role: inbound.Role, Content: content}}, nil
	}

	var blocks []anthropicInboundBlock
	if err := json.Unmarshal(trimmed, &blocks); err != nil {
		return nil, err
	}

	var textParts []string
	var toolCalls []providers.ToolCall
	var results []providers.Message
	for _, block := range blocks {
		switch block.Type {
		case "", "text":
			if block.Text != "" {
				textParts = append(textParts, block.Text)
			}
		case "tool_use":
			toolCalls = append(toolCalls, providers.ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: providers.ToolCallFunc{
					Name:      block.Name,
					Arguments: anthropicToolInputArguments(block.Input),
				},
			})
		case "tool_result":
			id := block.ToolUseID
			results = append(results, providers.Message{
				Role:       "tool",
				Content:    anthropicToolResultText(block.Content),
				ToolCallID: &id,
			})
		}
	}

	var out []providers.Message
	if len(textParts) > 0 || len(toolCalls) > 0 || len(results) == 0 {
		message := providers.Message{Role: inbound.Role, Content: strings.Join(textParts, "")}
		if len(toolCalls) > 0 {
			message.ToolCalls = toolCalls
		}
		out = append(out, message)
	}
	out = append(out, results...)
	return out, nil
}

func anthropicToolInputArguments(input json.RawMessage) string {
	trimmed := bytesTrimSpace(input)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return "{}"
	}
	return string(trimmed)
}

func anthropicToolResultText(content json.RawMessage) string {
	trimmed := bytesTrimSpace(content)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return ""
	}
	if trimmed[0] == '"' {
		var text string
		if err := json.Unmarshal(trimmed, &text); err == nil {
			return text
		}
		return string(trimmed)
	}
	if trimmed[0] == '[' {
		var blocks []anthropicInboundBlock
		if err := json.Unmarshal(trimmed, &blocks); err == nil {
			var parts []string
			for _, block := range blocks {
				if block.Text != "" {
					parts = append(parts, block.Text)
				}
			}
			return strings.Join(parts, "")
		}
	}
	return string(trimmed)
}

// rejectUnsupportedAnthropicMessageShape rejects only content block types we
// cannot represent. Native tool definitions and tool_choice are now translated
// (see translateAnthropicTools / translateAnthropicToolChoice), so they are no
// longer rejected here; genuinely-unsupported tool variants are reported during
// translation instead.
func rejectUnsupportedAnthropicMessageShape(body []byte) error {
	var req struct {
		Messages []struct {
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil
	}
	for i, message := range req.Messages {
		if err := rejectUnsupportedAnthropicContent(message.Content); err != nil {
			return fmt.Errorf("messages content %d: %w", i, err)
		}
	}
	return nil
}

func rejectUnsupportedAnthropicContent(raw json.RawMessage) error {
	trimmed := bytesTrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] == '"' || string(trimmed) == "null" {
		return nil
	}
	if trimmed[0] != '[' {
		return fmt.Errorf("unsupported content shape")
	}
	var blocks []struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(trimmed, &blocks); err != nil {
		return nil
	}
	for i, block := range blocks {
		switch block.Type {
		case "", "text", "tool_use", "tool_result":
		default:
			return fmt.Errorf("unsupported content block %d type %q", i, block.Type)
		}
	}
	return nil
}

func bytesTrimSpace(raw []byte) []byte {
	return []byte(strings.TrimSpace(string(raw)))
}

func anthropicStopReason(reason *string) *string {
	if reason == nil {
		return nil
	}
	if *reason == "tool_calls" {
		toolUse := "tool_use"
		return &toolUse
	}
	return reason
}

func anthropicStreamStopReason(reason *string) string {
	if reason == nil {
		return ""
	}
	switch *reason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	default:
		return *reason
	}
}

func toolCallInput(arguments string) json.RawMessage {
	trimmed := strings.TrimSpace(arguments)
	if trimmed == "" {
		return json.RawMessage(`{}`)
	}
	var object map[string]any
	if err := json.Unmarshal([]byte(trimmed), &object); err == nil {
		return json.RawMessage(trimmed)
	}
	wrapped, err := json.Marshal(map[string]string{"arguments": arguments})
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return wrapped
}

func messageContentText(content any) string {
	switch value := content.(type) {
	case nil:
		return ""
	case string:
		return value
	default:
		return fmt.Sprint(value)
	}
}

func writeJSON(ctx *fasthttp.RequestCtx, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "marshal response")
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(status)
	ctx.SetBody(body)
}

func writeError(ctx *fasthttp.RequestCtx, status int, message string) {
	body, err := json.Marshal(errorResponse{Error: message})
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(status)
	ctx.SetBody(body)
}

func writeOpenAIError(ctx *fasthttp.RequestCtx, status int, message string, typ string, code string) {
	body, err := json.Marshal(openAIErrorResponse{
		Error: openAIErrorBody{
			Message: message,
			Type:    typ,
			Code:    code,
		},
	})
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(status)
	ctx.SetBody(body)
}
