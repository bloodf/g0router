package anthropic

import (
	"encoding/json"
	"fmt"

	"github.com/bloodf/g0router/internal/schemas"
)

// ErrorConverter maps Anthropic HTTP error responses to ProviderError.
type ErrorConverter struct{}

// NewErrorConverter creates an error converter.
func NewErrorConverter() *ErrorConverter {
	return &ErrorConverter{}
}

// AnthropicError is the error shape returned by Anthropic.
type AnthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// AnthropicErrorResponse wraps the error.
type AnthropicErrorResponse struct {
	Type  string         `json:"type"`
	Error AnthropicError `json:"error"`
}

// Convert parses an Anthropic error JSON body and returns a ProviderError.
func (e *ErrorConverter) Convert(statusCode int, body []byte, meta schemas.ErrorMeta) *schemas.ProviderError {
	var envelope AnthropicErrorResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		return &schemas.ProviderError{
			Message:    fmt.Sprintf("Anthropic error (HTTP %d): %s", statusCode, string(body)),
			Type:       "api_error",
			StatusCode: statusCode,
			Meta:       meta,
		}
	}
	return &schemas.ProviderError{
		Message:    envelope.Error.Message,
		Type:       envelope.Error.Type,
		StatusCode: statusCode,
		Meta:       meta,
	}
}
