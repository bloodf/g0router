package gemini

import (
	"encoding/json"
	"fmt"

	"github.com/bloodf/g0router/internal/schemas"
)

// ErrorConverter maps Gemini HTTP error responses to ProviderError.
type ErrorConverter struct{}

// NewErrorConverter creates an error converter.
func NewErrorConverter() *ErrorConverter {
	return &ErrorConverter{}
}

// GeminiError is the error shape returned by Gemini.
type GeminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// GeminiErrorResponse wraps the error.
type GeminiErrorResponse struct {
	Error GeminiError `json:"error"`
}

// Convert parses a Gemini error JSON body and returns a ProviderError.
func (e *ErrorConverter) Convert(statusCode int, body []byte, meta schemas.ErrorMeta) *schemas.ProviderError {
	var envelope GeminiErrorResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		return &schemas.ProviderError{
			Message:    fmt.Sprintf("Gemini error (HTTP %d): %s", statusCode, string(body)),
			Type:       "api_error",
			StatusCode: statusCode,
			Meta:       meta,
		}
	}
	return &schemas.ProviderError{
		Message:    envelope.Error.Message,
		Type:       envelope.Error.Status,
		StatusCode: statusCode,
		Meta:       meta,
	}
}
