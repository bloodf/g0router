package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/proxy"
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
			if chunk.Error != nil {
				writeStreamError(w, chunk.Error)
				return
			}
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

func rejectUnsupportedAnthropicMessageShape(body []byte) error {
	var req struct {
		Tools      []json.RawMessage `json:"tools"`
		ToolChoice json.RawMessage   `json:"tool_choice"`
		Messages   []struct {
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil
	}
	for i, tool := range req.Tools {
		if isAnthropicNativeTool(tool) {
			return fmt.Errorf("messages native tool %d is not supported", i)
		}
	}
	if len(req.ToolChoice) > 0 && string(req.ToolChoice) != "null" && isAnthropicNativeToolChoice(req.ToolChoice) {
		return fmt.Errorf("messages native tool_choice is not supported")
	}
	for i, message := range req.Messages {
		if err := rejectUnsupportedAnthropicContent(message.Content); err != nil {
			return fmt.Errorf("messages content %d: %w", i, err)
		}
	}
	return nil
}

func isAnthropicNativeTool(raw json.RawMessage) bool {
	var tool struct {
		Name        string          `json:"name"`
		InputSchema json.RawMessage `json:"input_schema"`
		Function    json.RawMessage `json:"function"`
	}
	if err := json.Unmarshal(raw, &tool); err != nil {
		return false
	}
	return len(tool.InputSchema) > 0 || (tool.Name != "" && len(tool.Function) == 0)
}

func isAnthropicNativeToolChoice(raw json.RawMessage) bool {
	if len(bytesTrimSpace(raw)) == 0 || bytesTrimSpace(raw)[0] == '"' {
		return false
	}
	var choice struct {
		Type     string          `json:"type"`
		Function json.RawMessage `json:"function"`
	}
	if err := json.Unmarshal(raw, &choice); err != nil {
		return false
	}
	return choice.Type != "" && len(choice.Function) == 0
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
		if block.Type != "" && block.Type != "text" {
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
