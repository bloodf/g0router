package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/streaming"
	"github.com/bloodf/g0router/internal/translate"
	"github.com/valyala/fasthttp"
)

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
	streamCtx, cancel := streamContext(ctx)
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
	streamCtx, cancel := streamContext(ctx)
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
