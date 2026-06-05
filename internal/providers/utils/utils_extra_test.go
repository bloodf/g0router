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

// --- ProviderError.Error() ---

func TestProviderErrorNoBody(t *testing.T) {
	err := &ProviderError{StatusCode: 500}
	got := err.Error()
	if !strings.Contains(got, "500") {
		t.Fatalf("error = %q, want status code", got)
	}
	if strings.Contains(got, ": ") && strings.Contains(got, ":  ") {
		t.Fatalf("error = %q, should not have trailing colon when body is empty", got)
	}
}

func TestProviderErrorWithBody(t *testing.T) {
	err := &ProviderError{StatusCode: 429, Body: "too many requests"}
	got := err.Error()
	if !strings.Contains(got, "429") || !strings.Contains(got, "too many requests") {
		t.Fatalf("error = %q, want status+body", got)
	}
}

// --- parseRetryAfter ---

func TestParseRetryAfterEmpty(t *testing.T) {
	d := parseRetryAfter("", time.Now())
	if d != 0 {
		t.Fatalf("expected 0 for empty, got %v", d)
	}
}

func TestParseRetryAfterSeconds(t *testing.T) {
	d := parseRetryAfter("30", time.Now())
	if d != 30*time.Second {
		t.Fatalf("expected 30s, got %v", d)
	}
}

func TestParseRetryAfterHTTPDate(t *testing.T) {
	future := time.Now().Add(60 * time.Second)
	header := future.UTC().Format(http.TimeFormat)
	d := parseRetryAfter(header, time.Now())
	// Should be approximately 60s (allow ±2s for test timing)
	if d < 58*time.Second || d > 62*time.Second {
		t.Fatalf("expected ~60s, got %v", d)
	}
}

func TestParseRetryAfterHTTPDateInPast(t *testing.T) {
	past := time.Now().Add(-10 * time.Second)
	header := past.UTC().Format(http.TimeFormat)
	d := parseRetryAfter(header, time.Now())
	if d != 0 {
		t.Fatalf("expected 0 for past date, got %v", d)
	}
}

func TestParseRetryAfterInvalidString(t *testing.T) {
	d := parseRetryAfter("not-a-date-or-number", time.Now())
	if d != 0 {
		t.Fatalf("expected 0 for invalid, got %v", d)
	}
}

// --- sleepBeforeRetry ---

func TestSleepBeforeRetryZeroDelay(t *testing.T) {
	// Zero/negative delay returns immediately
	err := sleepBeforeRetry(context.Background(), 0)
	if err != nil {
		t.Fatalf("sleepBeforeRetry(0): %v", err)
	}
}

func TestSleepBeforeRetryNegativeDelay(t *testing.T) {
	err := sleepBeforeRetry(context.Background(), -time.Second)
	if err != nil {
		t.Fatalf("sleepBeforeRetry(-1s): %v", err)
	}
}

func TestSleepBeforeRetryContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancelled

	err := sleepBeforeRetry(ctx, time.Hour) // long delay
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestSleepBeforeRetryPositiveDelay(t *testing.T) {
	start := time.Now()
	err := sleepBeforeRetry(context.Background(), 20*time.Millisecond)
	if err != nil {
		t.Fatalf("sleepBeforeRetry: %v", err)
	}
	if time.Since(start) < 15*time.Millisecond {
		t.Fatal("sleep returned too early")
	}
}

// --- DoRequest: nil client/request ---

func TestDoRequestNilClient(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	_, err := DoRequest(context.Background(), nil, req, RequestOptions{})
	if err == nil {
		t.Fatal("expected error for nil client")
	}
	if !strings.Contains(err.Error(), "nil http client") {
		t.Fatalf("error = %q, want nil client context", err.Error())
	}
}

func TestDoRequestNilRequest(t *testing.T) {
	_, err := DoRequest(context.Background(), &http.Client{}, nil, RequestOptions{})
	if err == nil {
		t.Fatal("expected error for nil request")
	}
	if !strings.Contains(err.Error(), "nil request") {
		t.Fatalf("error = %q, want nil request context", err.Error())
	}
}

// --- DoRequest: 4xx non-429 → ErrProvider ---

func TestDoRequest4xxNonRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	_, err := DoRequest(context.Background(), srv.Client(), req, RequestOptions{})
	if !errors.Is(err, ErrProvider) {
		t.Fatalf("expected ErrProvider for 400, got %v", err)
	}
	var pe *ProviderError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ProviderError, got %T", err)
	}
	if pe.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want 400", pe.StatusCode)
	}
}

// --- DoRequest: 5xx exhausts retries ---

func TestDoRequest5xxExhaustsRetries(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	_, err := DoRequest(context.Background(), srv.Client(), req, RequestOptions{MaxRetries: 2})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("calls = %d, want 3", got)
	}
	// After exhausting retries on 5xx, returns providerError
	var pe *ProviderError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ProviderError after retries, got %T: %v", err, err)
	}
}

// --- DoRequest: context cancelled mid-retry (non-ctx error on first attempt) ---

func TestDoRequestNetworkErrorExhaustedWithRetries(t *testing.T) {
	// Server closes immediately
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	_, err := DoRequest(context.Background(), srv.Client(), req, RequestOptions{MaxRetries: 1})
	if err == nil {
		t.Fatal("expected error for closed server")
	}
	if !strings.Contains(err.Error(), "provider request") {
		t.Fatalf("error = %q, want provider request context", err.Error())
	}
}

// --- DoRequest: context error during network call ---

func TestDoRequestContextCancelledDuringNetworkCall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately so the request fails with ctx error
	cancel()

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	_, err := DoRequest(ctx, srv.Client(), req, RequestOptions{MaxRetries: 1})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// --- cloneRequest: first attempt with body returns original ---

func TestDoRequestFirstAttemptWithBodySucceeds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	body := strings.NewReader("payload")
	req, _ := http.NewRequest(http.MethodPost, srv.URL, body)
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("payload")), nil
	}

	resp, err := DoRequest(context.Background(), srv.Client(), req, RequestOptions{MaxRetries: 0})
	if err != nil {
		t.Fatalf("DoRequest: %v", err)
	}
	defer resp.Body.Close()
}

// --- providerError: body read (internal) via DoRequest ---

func TestDoRequestProviderErrorBodyRead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		http.Error(w, `{"error":"rate limit"}`, http.StatusTooManyRequests)
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	_, err := DoRequest(context.Background(), srv.Client(), req, RequestOptions{})
	if !errors.Is(err, ErrRateLimit) {
		t.Fatalf("expected ErrRateLimit, got %v", err)
	}
	var pe *ProviderError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ProviderError, got %T", err)
	}
	if pe.RetryAfter != 5*time.Second {
		t.Fatalf("RetryAfter = %v, want 5s", pe.RetryAfter)
	}
	if !strings.Contains(pe.Body, "rate limit") {
		t.Fatalf("Body = %q, want rate limit text", pe.Body)
	}
}

// --- DoRequest: retry with context cancel during sleep ---

func TestDoRequestRetrySleepContextCancelled(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	go func() {
		// Cancel context shortly after first attempt
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := DoRequest(ctx, srv.Client(), req, RequestOptions{MaxRetries: 5, RetryDelay: 200 * time.Millisecond})
	if err == nil {
		t.Fatal("expected error after context cancel")
	}
	// Should have been cancelled during sleep
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// --- ParseSSE scanner error path ---

func TestParseSSEReaderError(t *testing.T) {
	// errReader always returns error
	err := ParseSSE(errSSEReader{}, func(data string) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error from scanner")
	}
	if !strings.Contains(err.Error(), "parse sse") {
		t.Fatalf("error = %q, want parse sse context", err.Error())
	}
}

type errSSEReader struct{}

func (e errSSEReader) Read(p []byte) (int, error) {
	return 0, &testReadError{}
}

type testReadError struct{}

func (e *testReadError) Error() string { return "test read error" }

// --- ParseSSE: dispatch at end (no trailing blank line) ---

func TestParseSSEDispatchAtEndWithDONE(t *testing.T) {
	// [DONE] at end without trailing blank line dispatches via post-loop call
	input := strings.NewReader("data: payload1\n\ndata: [DONE]")
	var got []string
	err := ParseSSE(input, func(data string) error {
		got = append(got, data)
		return nil
	})
	if err != nil {
		t.Fatalf("ParseSSE: %v", err)
	}
	if len(got) != 1 || got[0] != "payload1" {
		t.Fatalf("got %v, want [payload1]", got)
	}
}

// --- ParseSSE: non-data line ignored ---

func TestParseSSENonDataLineIgnored(t *testing.T) {
	input := strings.NewReader("event: update\ndata: payload\n\n")
	var got []string
	err := ParseSSE(input, func(data string) error {
		got = append(got, data)
		return nil
	})
	if err != nil {
		t.Fatalf("ParseSSE: %v", err)
	}
	if len(got) != 1 || got[0] != "payload" {
		t.Fatalf("got %v, want [payload]", got)
	}
}

// Verify RetryAfter is set correctly in ProviderError for 429 response
func TestDoRequestRetryAfterHTTPDate(t *testing.T) {
	future := time.Now().Add(30 * time.Second)
	header := future.UTC().Format(http.TimeFormat)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", header)
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	_, err := DoRequest(context.Background(), srv.Client(), req, RequestOptions{})
	if !errors.Is(err, ErrRateLimit) {
		t.Fatalf("expected ErrRateLimit, got %v", err)
	}
	var pe *ProviderError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ProviderError, got %T", err)
	}
	if pe.RetryAfter < 28*time.Second || pe.RetryAfter > 32*time.Second {
		t.Fatalf("RetryAfter = %v, want ~30s", pe.RetryAfter)
	}
}

// --- cloneRequest: successful GetBody on retry ---

func TestDoRequestRetry5xxWithReusableBody(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 2 {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	bodyContent := "request body"
	req, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(bodyContent))
	// Provide GetBody so retry can re-read the body
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(bodyContent)), nil
	}

	resp, err := DoRequest(context.Background(), srv.Client(), req, RequestOptions{MaxRetries: 1})
	if err != nil {
		t.Fatalf("DoRequest with reusable body: %v", err)
	}
	defer resp.Body.Close()
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

// --- cloneRequest: GetBody returns error on retry ---

func TestDoRequestRetryGetBodyError(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	bodyContent := "request body"
	req, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(bodyContent))
	req.GetBody = func() (io.ReadCloser, error) {
		return nil, errors.New("body unavailable")
	}

	_, err := DoRequest(context.Background(), srv.Client(), req, RequestOptions{MaxRetries: 1})
	if err == nil {
		t.Fatal("expected error for GetBody failure")
	}
	if !strings.Contains(err.Error(), "get body") {
		t.Fatalf("error = %q, want get body context", err.Error())
	}
}

// --- providerError: body read failure ---
// Call providerError directly (same package) with a response whose body errors on Read.

type errBodyReader struct{}

func (e errBodyReader) Read(p []byte) (int, error) { return 0, errors.New("body read failed") }
func (e errBodyReader) Close() error               { return nil }

func TestProviderErrorBodyReadError(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Header:     make(http.Header),
		Body:       errBodyReader{},
	}
	err := providerError(resp, ErrProvider)
	if err == nil {
		t.Fatal("expected error")
	}
	pe, ok := err.(*ProviderError)
	if !ok {
		t.Fatalf("expected *ProviderError, got %T", err)
	}
	if pe.Body != "failed to read error response" {
		t.Fatalf("Body = %q, want fallback message", pe.Body)
	}
	if !errors.Is(err, ErrProvider) {
		t.Fatal("expected ErrProvider wrapped")
	}
}
