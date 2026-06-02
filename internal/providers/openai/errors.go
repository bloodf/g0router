package openai

import (
	"errors"
	"fmt"
)

var (
	ErrAuth      = errors.New("openai auth error")
	ErrRateLimit = errors.New("openai rate limit")
	ErrServer    = errors.New("openai server error")
)

type RateLimitError struct {
	Message    string
	RetryAfter int
}

func (e *RateLimitError) Error() string {
	if e.Message == "" {
		return ErrRateLimit.Error()
	}
	if e.RetryAfter > 0 {
		return fmt.Sprintf("%s: %s (retry after %ds)", ErrRateLimit, e.Message, e.RetryAfter)
	}
	return fmt.Sprintf("%s: %s", ErrRateLimit, e.Message)
}

func (e *RateLimitError) Is(target error) bool {
	return target == ErrRateLimit
}
