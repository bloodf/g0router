package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type RequestOptions struct {
	MaxRetries int
	RetryDelay time.Duration
}

func DoRequest(ctx context.Context, client *http.Client, req *http.Request, opts RequestOptions) (*http.Response, error) {
	if client == nil {
		return nil, fmt.Errorf("provider request: nil http client")
	}
	if req == nil {
		return nil, fmt.Errorf("provider request: nil request")
	}

	attempts := opts.MaxRetries + 1
	for attempt := 0; attempt < attempts; attempt++ {
		attemptReq, err := cloneRequest(ctx, req, attempt)
		if err != nil {
			return nil, err
		}

		resp, err := client.Do(attemptReq)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, fmt.Errorf("provider request: %w", ctxErr)
			}
			if attempt < attempts-1 {
				if err := sleepBeforeRetry(ctx, opts.RetryDelay); err != nil {
					return nil, err
				}
				continue
			}
			return nil, fmt.Errorf("provider request: %w", err)
		}

		if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
			return resp, nil
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, providerError(resp, ErrRateLimit)
		}
		if resp.StatusCode >= 500 && attempt < attempts-1 {
			resp.Body.Close()
			if err := sleepBeforeRetry(ctx, opts.RetryDelay); err != nil {
				return nil, err
			}
			continue
		}

		return nil, providerError(resp, ErrProvider)
	}

	return nil, fmt.Errorf("provider request: exhausted retries")
}

func cloneRequest(ctx context.Context, req *http.Request, attempt int) (*http.Request, error) {
	cloned := req.Clone(ctx)
	if req.Body == nil {
		return cloned, nil
	}
	if attempt == 0 {
		return cloned, nil
	}
	if req.GetBody == nil {
		return nil, fmt.Errorf("provider request retry: request body is not reusable")
	}

	body, err := req.GetBody()
	if err != nil {
		return nil, fmt.Errorf("provider request retry: get body: %w", err)
	}
	cloned.Body = body
	return cloned, nil
}

func providerError(resp *http.Response, cause error) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		body = []byte("failed to read error response")
	}

	return &ProviderError{
		StatusCode: resp.StatusCode,
		Body:       strings.TrimSpace(string(body)),
		RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After"), time.Now()),
		Err:        cause,
	}
}

func parseRetryAfter(value string, now time.Time) time.Duration {
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}
	retryAt, err := http.ParseTime(value)
	if err != nil {
		return 0
	}
	if retryAt.Before(now) {
		return 0
	}
	return retryAt.Sub(now)
}

func sleepBeforeRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("provider request retry: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}
