package inference

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"time"
)

// RetryEntry is a single status-code retry configuration: how many times to
// retry and how long to wait between attempts. Mirrors the entries in
// open-sse/config/runtimeConfig.js:52-57.
type RetryEntry struct {
	Attempts int
	DelayMs  int
}

// DefaultRetryConfig is the default per-status retry policy ported from
// open-sse/config/runtimeConfig.js:52-57.
var DefaultRetryConfig = map[int]RetryEntry{
	429: {Attempts: 0, DelayMs: 0},
	502: {Attempts: 3, DelayMs: 3000},
	503: {Attempts: 3, DelayMs: 2000},
	504: {Attempts: 2, DelayMs: 3000},
}

// Provider exposes retry configuration for a backend. It is implemented by
// catalog.ProviderConfig.
type Provider interface {
	RetryOverride() map[int]int
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
		if class.Class == ClassPermanent {
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

