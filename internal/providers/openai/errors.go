package openai

import (
	"encoding/json"
	"fmt"

	"github.com/bloodf/g0router/internal/schemas"
)

// ErrorConverter maps OpenAI HTTP error responses to ProviderError.
type ErrorConverter struct{}

// NewErrorConverter creates an error converter.
func NewErrorConverter() *ErrorConverter {
	return &ErrorConverter{}
}

// Convert parses an OpenAI error JSON body and returns a ProviderError.
func (e *ErrorConverter) Convert(statusCode int, body []byte, meta schemas.ErrorMeta) *schemas.ProviderError {
	var envelope schemas.ErrorResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		return &schemas.ProviderError{
			Message:    fmt.Sprintf("OpenAI error (HTTP %d): %s", statusCode, string(body)),
			Type:       "api_error",
			StatusCode: statusCode,
			Meta:       meta,
		}
	}
	return &schemas.ProviderError{
		Message:    envelope.Error.Message,
		Type:       envelope.Error.Type,
		Param:      envelope.Error.Param,
		Code:       envelope.Error.Code,
		StatusCode: statusCode,
		Meta:       meta,
	}
}
