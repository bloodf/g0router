package utils

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrProvider  = errors.New("provider: error")
	ErrRateLimit = errors.New("provider: rate limit")
)

type ProviderError struct {
	StatusCode int
	Body       string
	RetryAfter time.Duration
	Err        error
}

func (e *ProviderError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("provider request failed: status %d", e.StatusCode)
	}
	return fmt.Sprintf("provider request failed: status %d: %s", e.StatusCode, e.Body)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}
