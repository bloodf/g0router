package handlers

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/valyala/fasthttp"
)

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
