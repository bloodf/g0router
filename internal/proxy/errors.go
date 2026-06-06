package proxy

import (
	"errors"
	"net/http"

	"github.com/bloodf/g0router/internal/guardrails"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/providers/anthropic"
	"github.com/bloodf/g0router/internal/providers/azure"
	"github.com/bloodf/g0router/internal/providers/bedrock"
	"github.com/bloodf/g0router/internal/providers/gemini"
	"github.com/bloodf/g0router/internal/providers/openai"
	"github.com/bloodf/g0router/internal/providers/openaicompat"
	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/providers/vertex"
)

type DispatchErrorClass struct {
	StatusCode int
	Message    string
	Type       string
	Code       string
}

func ClassifyDispatchError(err error) DispatchErrorClass {
	switch {
	case errors.Is(err, ErrProviderNotFound):
		return DispatchErrorClass{StatusCode: http.StatusNotFound, Message: "provider not found", Type: "invalid_request_error", Code: "provider_not_found"}
	case errors.Is(err, ErrProviderInferenceUnavailable):
		return DispatchErrorClass{StatusCode: http.StatusNotFound, Message: "provider inference unavailable", Type: "invalid_request_error", Code: "provider_inference_unavailable"}
	case errors.Is(err, ErrModelDisabled):
		return DispatchErrorClass{StatusCode: http.StatusBadRequest, Message: err.Error(), Type: "invalid_request_error", Code: "model_disabled"}
	case errors.Is(err, guardrails.ErrBlocklistMatch):
		return DispatchErrorClass{StatusCode: http.StatusBadRequest, Message: err.Error(), Type: "invalid_request_error", Code: "guardrails_blocklist"}
	case errors.Is(err, ErrNoConnections):
		return DispatchErrorClass{StatusCode: http.StatusServiceUnavailable, Message: "no active provider connections", Type: "server_error", Code: "no_active_connections"}
	case errors.Is(err, ErrQuotaExhausted):
		return DispatchErrorClass{StatusCode: http.StatusTooManyRequests, Message: "quota exhausted", Type: "rate_limit_error", Code: "quota_exhausted"}
	case errors.Is(err, ErrCapabilityUnsupported):
		return DispatchErrorClass{StatusCode: http.StatusNotImplemented, Message: "capability unsupported for provider", Type: "invalid_request_error", Code: "capability_unsupported"}
	case isStreamingUnsupportedError(err):
		return DispatchErrorClass{StatusCode: http.StatusNotImplemented, Message: "streaming unsupported for provider", Type: "invalid_request_error", Code: "streaming_unsupported"}
	case isUpstreamAuthError(err):
		return DispatchErrorClass{StatusCode: http.StatusUnauthorized, Message: "upstream provider authentication failed", Type: "invalid_request_error", Code: "upstream_auth_error"}
	case isUpstreamRateLimitError(err):
		return DispatchErrorClass{StatusCode: http.StatusTooManyRequests, Message: "upstream provider rate limit", Type: "rate_limit_error", Code: "upstream_rate_limit"}
	case isUpstreamServerError(err):
		return DispatchErrorClass{StatusCode: http.StatusBadGateway, Message: "upstream provider server error", Type: "server_error", Code: "upstream_server_error"}
	default:
		return DispatchErrorClass{StatusCode: http.StatusBadGateway, Message: "upstream provider error", Type: "server_error", Code: "upstream_error"}
	}
}

func isStreamingUnsupportedError(err error) bool {
	return errors.Is(err, providers.ErrStreamingUnsupported) ||
		errors.Is(err, gemini.ErrUnsupported) ||
		errors.Is(err, vertex.ErrUnsupported)
}

func isUpstreamAuthError(err error) bool {
	return errors.Is(err, anthropic.ErrAuth) ||
		errors.Is(err, azure.ErrAuth) ||
		errors.Is(err, bedrock.ErrAuth) ||
		errors.Is(err, gemini.ErrAuth) ||
		errors.Is(err, openai.ErrAuth) ||
		errors.Is(err, openaicompat.ErrAuth) ||
		errors.Is(err, vertex.ErrAuth)
}

func isUpstreamRateLimitError(err error) bool {
	return errors.Is(err, anthropic.ErrRateLimit) ||
		errors.Is(err, azure.ErrRateLimit) ||
		errors.Is(err, gemini.ErrRateLimit) ||
		errors.Is(err, openai.ErrRateLimit) ||
		errors.Is(err, openaicompat.ErrRateLimit) ||
		errors.Is(err, utils.ErrRateLimit) ||
		errors.Is(err, vertex.ErrRateLimit)
}

func isUpstreamServerError(err error) bool {
	return errors.Is(err, anthropic.ErrServer) ||
		errors.Is(err, azure.ErrServer) ||
		errors.Is(err, bedrock.ErrServer) ||
		errors.Is(err, gemini.ErrServer) ||
		errors.Is(err, openai.ErrServer) ||
		errors.Is(err, openaicompat.ErrServer) ||
		errors.Is(err, vertex.ErrServer)
}
