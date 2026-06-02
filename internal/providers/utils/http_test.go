package utils

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDoRequestSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp, err := DoRequest(context.Background(), srv.Client(), req, RequestOptions{MaxRetries: 2})
	if err != nil {
		t.Fatalf("DoRequest: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(body) != "ok" {
		t.Fatalf("body = %q, want ok", body)
	}
}

func TestDoRequestRetry5xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := atomic.AddInt32(&calls, 1)
		if call < 3 {
			http.Error(w, "try again", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp, err := DoRequest(context.Background(), srv.Client(), req, RequestOptions{MaxRetries: 2})
	if err != nil {
		t.Fatalf("DoRequest: %v", err)
	}
	defer resp.Body.Close()

	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("calls = %d, want 3", got)
	}
}

func TestDoRequest429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "2")
		http.Error(w, "too many requests", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	_, err = DoRequest(context.Background(), srv.Client(), req, RequestOptions{MaxRetries: 2})
	if !errors.Is(err, ErrRateLimit) {
		t.Fatalf("expected ErrRateLimit, got: %v", err)
	}

	var providerErr *ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("expected ProviderError, got: %T", err)
	}
	if providerErr.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("StatusCode = %d, want 429", providerErr.StatusCode)
	}
	if providerErr.RetryAfter != 2*time.Second {
		t.Fatalf("RetryAfter = %s, want 2s", providerErr.RetryAfter)
	}
}

func TestDoRequestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	_, err = DoRequest(ctx, srv.Client(), req, RequestOptions{MaxRetries: 2})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got: %v", err)
	}
}

func TestProviderErrorUnwrap(t *testing.T) {
	err := &ProviderError{
		StatusCode: http.StatusTooManyRequests,
		Body:       "rate limited",
		Err:        ErrRateLimit,
	}

	if !errors.Is(err, ErrRateLimit) {
		t.Fatal("errors.Is should match ErrRateLimit")
	}
}

func TestDoRequestRejectsRetryWithoutReusableBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "try again", http.StatusInternalServerError)
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.GetBody = nil

	_, err = DoRequest(context.Background(), srv.Client(), req, RequestOptions{MaxRetries: 1})
	if err == nil {
		t.Fatal("expected retry body error")
	}
	if !strings.Contains(err.Error(), "request body is not reusable") {
		t.Fatalf("error = %q", err)
	}
}
