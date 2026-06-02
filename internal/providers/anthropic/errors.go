package anthropic

import (
	"errors"
	"fmt"
)

var (
	ErrAuth      = errors.New("anthropic auth error")
	ErrRateLimit = errors.New("anthropic rate limit")
	ErrServer    = errors.New("anthropic server error")
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
