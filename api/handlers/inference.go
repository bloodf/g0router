package handlers

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/proxy"
	"github.com/bloodf/g0router/internal/translate"
	"github.com/valyala/fasthttp"
)

type errorResponse struct {
	Error string `json:"error"`
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
	stream, err := engine.DispatchStream(requestContext(ctx), req)
	if err != nil {
		writeDispatchError(ctx, err)
		return
	}

	ctx.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		for chunk := range stream {
			data, err := json.Marshal(chunk)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			_ = w.Flush()
		}
		_, _ = w.WriteString("data: [DONE]\n\n")
		_ = w.Flush()
	})
}

func Messages(ctx *fasthttp.RequestCtx, engine InferenceEngine) {
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
		writeError(ctx, fasthttp.StatusNotImplemented, "messages streaming unavailable")
		return
	}

	resp, err := engine.Dispatch(requestContext(ctx), &req)
	if err != nil {
		writeDispatchError(ctx, err)
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, anthropicMessageResponse(resp))
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
	if req.Stream != nil && *req.Stream {
		writeError(ctx, fasthttp.StatusNotImplemented, "responses streaming unavailable")
		return
	}

	chatReq, err := translate.ResponsesRequestToOpenAIChat(&req)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}
	resp, err := engine.Dispatch(requestContext(ctx), chatReq)
	if err != nil {
		writeDispatchError(ctx, err)
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, translate.OpenAIChatToResponsesResponse(resp))
}

func writeDispatchError(ctx *fasthttp.RequestCtx, err error) {
	switch {
	case errors.Is(err, proxy.ErrProviderNotFound):
		writeError(ctx, fasthttp.StatusNotFound, err.Error())
	case errors.Is(err, proxy.ErrNoConnections):
		writeError(ctx, fasthttp.StatusServiceUnavailable, err.Error())
	default:
		writeError(ctx, fasthttp.StatusInternalServerError, err.Error())
	}
}

type anthropicMessageContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
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
		body.StopReason = choice.FinishReason
		if text := messageContentText(choice.Message.Content); text != "" {
			body.Content = []anthropicMessageContent{{Type: "text", Text: text}}
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
