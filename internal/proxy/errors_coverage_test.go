package proxy

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/providers/anthropic"
	"github.com/bloodf/g0router/internal/providers/gemini"
	"github.com/bloodf/g0router/internal/providers/openai"
)

func TestClassifyDispatchError(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode string
		wantHTTP int
	}{
		{"provider not found", ErrProviderNotFound, "provider_not_found", http.StatusNotFound},
		{"inference unavailable", ErrProviderInferenceUnavailable, "provider_inference_unavailable", http.StatusNotFound},
		{"no connections", ErrNoConnections, "no_active_connections", http.StatusServiceUnavailable},
		{"quota exhausted", ErrQuotaExhausted, "quota_exhausted", http.StatusTooManyRequests},
		{"capability unsupported", ErrCapabilityUnsupported, "capability_unsupported", http.StatusNotImplemented},
		{"streaming unsupported", providers.ErrStreamingUnsupported, "streaming_unsupported", http.StatusNotImplemented},
		{"gemini unsupported", gemini.ErrUnsupported, "streaming_unsupported", http.StatusNotImplemented},
		{"upstream auth", anthropic.ErrAuth, "upstream_auth_error", http.StatusUnauthorized},
		{"openai auth", openai.ErrAuth, "upstream_auth_error", http.StatusUnauthorized},
		{"upstream rate limit", anthropic.ErrRateLimit, "upstream_rate_limit", http.StatusTooManyRequests},
		{"upstream server", anthropic.ErrServer, "upstream_server_error", http.StatusBadGateway},
		{"wrapped auth", fmt.Errorf("dispatch: %w", openai.ErrAuth), "upstream_auth_error", http.StatusUnauthorized},
		{"unknown", errors.New("boom"), "upstream_error", http.StatusBadGateway},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyDispatchError(tc.err)
			if got.Code != tc.wantCode {
				t.Errorf("code = %q, want %q", got.Code, tc.wantCode)
			}
			if got.StatusCode != tc.wantHTTP {
				t.Errorf("status = %d, want %d", got.StatusCode, tc.wantHTTP)
			}
			if got.Message == "" || got.Type == "" {
				t.Errorf("message/type empty: %+v", got)
			}
		})
	}
}
