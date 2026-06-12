package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// RetryEntry is a single status-code retry configuration: how many times to
// retry and how long to wait between attempts. Mirrors the entries in
// open-sse/config/runtimeConfig.js:52-57.
type RetryEntry struct {
	Attempts int
	DelayMs  int
}

// newDefaultRetryConfig returns the default per-status retry policy ported from
// open-sse/config/runtimeConfig.js:52-57. Returned as a new map on each call so
// callers cannot mutate shared package state.
func newDefaultRetryConfig() map[int]RetryEntry {
	return map[int]RetryEntry{
		429: {Attempts: 0, DelayMs: 0},
		502: {Attempts: 3, DelayMs: 3000},
		503: {Attempts: 3, DelayMs: 2000},
		504: {Attempts: 2, DelayMs: 3000},
	}
}

// Provider exposes retry configuration for a backend. It is implemented by
// catalog.ProviderConfig.
type Provider interface {
	RetryOverride() map[int]int
}

// Settings is the minimal key-value store used by PR-1626 auto-learning.
// It is implemented by *store.Store.
type Settings interface {
	GetSetting(key string) (string, error)
	SetSetting(key, value string) error
}

// Token parameter names involved in PR-1626 auto-learning.
const (
	TokenParamMaxTokens           = "max_tokens"
	TokenParamMaxCompletionTokens = "max_completion_tokens"
)

// learnedTokenParamKey returns the settings key for a learned token parameter.
func learnedTokenParamKey(providerID, modelID string) string {
	return "learned_token_param:" + providerID + ":" + modelID
}

// WithRetry wraps a provider call with classification-driven retries. It honors
// per-status attempt counts and delays, provider-specific overrides, rate-limit
// resets_at sleeps (capped at 30 min), and converts connect-timeouts to 502
// without retry (PAR-ROUTE-022).
func WithRetry(ctx context.Context, provider Provider, call func() (int, []byte, error), retryConfig map[int]RetryEntry) (int, []byte, error) {
	effective := buildEffectiveRetryConfig(retryConfig, provider)
	attemptsByStatus := make(map[int]int)

	for {
		status, body, err := call()
		if err != nil {
			// Client-side abort must propagate, not be treated as upstream failure.
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return 0, nil, err
			}

			// Connect timeout (dial timeout) is internal: return 502 immediately.
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() && !netErr.Temporary() {
				return 502, connectTimeoutBody(), nil
			}

			// All other transport errors map to the 502 retry path.
			status = 502
			body = nil
		}

		if status >= 200 && status < 300 {
			return status, body, nil
		}

		class := Classify(status, body)
		if !class.Retryable {
			return status, body, nil
		}

		entry, ok := effective[status]
		if !ok {
			return status, body, nil
		}
		if entry.Attempts <= 0 {
			return status, body, nil
		}
		if attemptsByStatus[status] >= entry.Attempts {
			return status, body, nil
		}
		attemptsByStatus[status]++

		// Determine sleep duration. Rate-limit resets_at takes precedence.
		delay := time.Duration(entry.DelayMs) * time.Millisecond
		if class.Class == ClassRateLimit && !class.ResetsAt.IsZero() {
			retryAfter := time.Until(class.ResetsAt)
			if retryAfter > delay {
				delay = retryAfter
			}
		}

		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return 0, nil, ctx.Err()
			case <-timer.C:
			}
		}
	}
}

// buildEffectiveRetryConfig merges the default config with provider-specific
// attempt overrides. Provider overrides replace only the Attempts field.
func buildEffectiveRetryConfig(base map[int]RetryEntry, provider Provider) map[int]RetryEntry {
	out := make(map[int]RetryEntry, len(base))
	for k, v := range base {
		out[k] = v
	}
	if provider == nil {
		return out
	}
	for status, attempts := range provider.RetryOverride() {
		entry := out[status]
		entry.Attempts = attempts
		out[status] = entry
	}
	return out
}

// connectTimeoutBody returns a stable OpenAI-shaped error body for connect timeouts.
func connectTimeoutBody() []byte {
	b, _ := json.Marshal(map[string]any{
		"error": map[string]any{
			"message": "Upstream connect timeout",
			"type":    "server_error",
			"code":    "bad_gateway",
		},
	})
	return b
}


// AutoLearnTokenParam executes a provider call with PR-1626 token-parameter
// auto-learning. It reads any previously-learned preference for
// provider+model, applies it to the request body, and if the provider rejects
// the active parameter as unsupported, retries once with the alternate
// parameter and persists the learned preference on success.
func AutoLearnTokenParam(
	ctx context.Context,
	providerID, modelID string,
	body map[string]any,
	settings Settings,
	call func(body map[string]any) (int, []byte, error),
) (int, []byte, error) {
	currentParam := TokenParamMaxTokens

	// Apply any previously learned preference.
	key := learnedTokenParamKey(providerID, modelID)
	if learned, err := settings.GetSetting(key); err == nil && learned != "" {
		if learned == TokenParamMaxCompletionTokens || learned == TokenParamMaxTokens {
			currentParam = applyTokenParam(body, learned)
		}
	}

	status, respBody, err := call(body)
	if err != nil {
		return status, respBody, err
	}
	if status >= 200 && status < 300 {
		return status, respBody, nil
	}

	class := Classify(status, respBody)
	if class.Class != ClassUnsupportedParam || !isUnsupportedTokenParamMismatch(respBody, currentParam) {
		return status, respBody, nil
	}

	// Retry once with the alternate parameter.
	switchedTo := switchTokenParam(body, currentParam)
	if switchedTo == "" {
		return status, respBody, nil
	}

	status2, respBody2, err2 := call(body)
	if err2 != nil {
		return status2, respBody2, err2
	}
	if status2 >= 200 && status2 < 300 {
		if err := settings.SetSetting(key, switchedTo); err != nil {
			return status2, respBody2, fmt.Errorf("persist learned token param: %w", err)
		}
	}
	return status2, respBody2, nil
}

// applyTokenParam ensures only the preferred token parameter is present in body.
// It returns the parameter that is now active.
func applyTokenParam(body map[string]any, preferred string) string {
	other := TokenParamMaxTokens
	if preferred == TokenParamMaxTokens {
		other = TokenParamMaxCompletionTokens
	}

	if _, ok := body[preferred]; ok {
		delete(body, other)
		return preferred
	}
	if value, ok := body[other]; ok {
		delete(body, other)
		body[preferred] = value
		return preferred
	}
	return preferred
}

// switchTokenParam moves the value from the current token parameter to the
// alternate one. It returns the new active parameter, or empty if neither was
// present.
func switchTokenParam(body map[string]any, current string) string {
	alternate := TokenParamMaxCompletionTokens
	if current == TokenParamMaxCompletionTokens {
		alternate = TokenParamMaxTokens
	}

	value, ok := body[current]
	if !ok {
		return ""
	}
	delete(body, current)
	body[alternate] = value
	return alternate
}

// isUnsupportedTokenParamMismatch reports whether the response body indicates
// the active token parameter is unsupported.
func isUnsupportedTokenParamMismatch(body []byte, param string) bool {
	msg := strings.ToLower(extractMessage(body))
	return strings.Contains(msg, "unsupported") && strings.Contains(msg, param)
}
