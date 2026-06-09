package schemas

// APIError matches the OpenAI error object shape.
type APIError struct {
	Message string  `json:"message"`
	Type    string  `json:"type"`
	Param   *string `json:"param,omitempty"`
	Code    *string `json:"code,omitempty"`
}

// ErrorResponse is the standard OpenAI error envelope.
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// ErrorMeta carries metadata about a provider error.
type ErrorMeta struct {
	Provider       string
	ModelRequested string
	RequestType    string
	StatusCode     int
	RawBody        []byte
}

// ProviderError wraps an APIError with provider context.
type ProviderError struct {
	Message    string
	Type       string
	Param      *string
	Code       *string
	StatusCode int
	Meta       ErrorMeta
}

// Error implements the error interface.
func (e *ProviderError) Error() string { return e.Message }
