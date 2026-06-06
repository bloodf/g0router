package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"

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

func writeDispatchError(ctx *fasthttp.RequestCtx, err error) {
	classification := proxy.ClassifyDispatchError(err)
	writeOpenAIError(ctx, classification.StatusCode, classification.Message, classification.Type, classification.Code)
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

// --- Anthropic translation wrappers (preserved for test compatibility) ---

var errAnthropicTranslate = translate.ErrAnthropicTranslate

type anthropicMessageContent = translate.AnthropicMessageContent
type anthropicMessageUsage = translate.AnthropicMessageUsage
type anthropicMessageBody = translate.AnthropicMessageBody
type anthropicRequestEnvelope = translate.AnthropicRequestEnvelope
type anthropicInboundTool = translate.AnthropicInboundTool
type anthropicInboundMessage = translate.AnthropicInboundMessage
type anthropicInboundBlock = translate.AnthropicInboundBlock

func translateAnthropicMessagesRequest(body []byte) (*providers.ChatRequest, error) {
	return translate.AnthropicMessagesRequest(body)
}

func rejectUnsupportedAnthropicMessageShape(body []byte) error {
	return translate.RejectUnsupportedAnthropicMessageShape(body)
}

func rejectUnsupportedAnthropicContent(raw json.RawMessage) error {
	return translate.RejectUnsupportedAnthropicContent(raw)
}

func anthropicMessageResponse(resp *providers.ChatResponse) anthropicMessageBody {
	return translate.AnthropicMessageResponse(resp)
}

func translateAnthropicTools(tools []anthropicInboundTool) ([]providers.Tool, error) {
	return translate.TranslateAnthropicTools(tools)
}

func translateAnthropicToolChoice(raw json.RawMessage) (any, error) {
	return translate.TranslateAnthropicToolChoice(raw)
}

func translateAnthropicInboundMessage(inbound anthropicInboundMessage) ([]providers.Message, error) {
	return translate.TranslateAnthropicInboundMessage(inbound)
}

func anthropicToolInputArguments(input json.RawMessage) string {
	return translate.AnthropicToolInputArguments(input)
}

func anthropicToolResultText(content json.RawMessage) string {
	return translate.AnthropicToolResultText(content)
}

func bytesTrimSpace(raw []byte) []byte {
	return translate.BytesTrimSpace(raw)
}

func anthropicStopReason(reason *string) *string {
	return translate.AnthropicStopReason(reason)
}

func anthropicStreamStopReason(reason *string) string {
	return translate.AnthropicStreamStopReason(reason)
}

func toolCallInput(arguments string) json.RawMessage {
	return translate.ToolCallInput(arguments)
}

func messageContentText(content any) string {
	return translate.MessageContentText(content)
}
