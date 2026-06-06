package openaicompat

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/valyala/fasthttp"
)

// --- RateLimitError ---

func TestRateLimitErrorNoMessage(t *testing.T) {
	err := &RateLimitError{}
	if err.Error() != ErrRateLimit.Error() {
		t.Fatalf("error = %q, want default rate limit message", err.Error())
	}
}

func TestRateLimitErrorWithMessage(t *testing.T) {
	err := &RateLimitError{Message: "quota exceeded"}
	got := err.Error()
	if !strings.Contains(got, "quota exceeded") {
		t.Fatalf("error = %q, want message in output", got)
	}
	if !strings.Contains(got, ErrRateLimit.Error()) {
		t.Fatalf("error = %q, want rate limit prefix", got)
	}
}

func TestRateLimitErrorWithRetryAfter(t *testing.T) {
	err := &RateLimitError{Message: "too fast", RetryAfter: 30}
	got := err.Error()
	if !strings.Contains(got, "retry after 30s") {
		t.Fatalf("error = %q, want retry after", got)
	}
}

func TestRateLimitErrorIsErrRateLimit(t *testing.T) {
	err := &RateLimitError{Message: "slow down"}
	if !errors.Is(err, ErrRateLimit) {
		t.Fatal("errors.Is should match ErrRateLimit")
	}
	if errors.Is(err, ErrAuth) {
		t.Fatal("errors.Is should not match ErrAuth")
	}
}

// --- New validation ---

func TestNewRejectsEmptyProvider(t *testing.T) {
	_, err := New(Config{BaseURL: "http://example.com"})
	if err == nil {
		t.Fatal("expected error for empty provider")
	}
	if !errors.Is(err, ErrUnknownProvider) {
		t.Fatalf("expected ErrUnknownProvider, got %v", err)
	}
}

func TestNewRejectsPathWithoutLeadingSlash(t *testing.T) {
	_, err := New(Config{
		Provider:            providers.ProviderGroq,
		BaseURL:             "http://example.com",
		ChatCompletionsPath: "chat/completions",
	})
	if err == nil {
		t.Fatal("expected error for path without leading slash")
	}
	if !strings.Contains(err.Error(), "must start with /") {
		t.Fatalf("error = %q, want leading slash context", err.Error())
	}
}

func TestNewIgnoresBlankHeaderKeys(t *testing.T) {
	p, err := New(Config{
		Provider: providers.ProviderGroq,
		BaseURL:  "http://example.com",
		Headers:  map[string]string{"": "value", "X-Valid": "ok"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, ok := p.headers[""]; ok {
		t.Fatal("blank header key should be filtered out")
	}
	if p.headers["X-Valid"] != "ok" {
		t.Fatalf("X-Valid header missing: %+v", p.headers)
	}
}

// --- mapStatusError / mapError ---

func TestChatCompletionReturnsErrAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key","type":"auth_error"}}`))
	}))
	t.Cleanup(server.Close)

	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = p.ChatCompletion(context.Background(), testKey(providers.ProviderGroq), testChatRequest())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
	if !strings.Contains(err.Error(), "invalid api key") {
		t.Fatalf("error = %q, want message from body", err.Error())
	}
}

func TestChatCompletionReturnsErrAuthForbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"forbidden","type":"auth_error"}}`))
	}))
	t.Cleanup(server.Close)

	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = p.ChatCompletion(context.Background(), testKey(providers.ProviderGroq), testChatRequest())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth for 403, got %v", err)
	}
}

func TestChatCompletionReturnsRateLimitError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited","type":"rate_limit"}}`))
	}))
	t.Cleanup(server.Close)

	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = p.ChatCompletion(context.Background(), testKey(providers.ProviderGroq), testChatRequest())
	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected RateLimitError, got %T: %v", err, err)
	}
	if rlErr.RetryAfter != 60 {
		t.Fatalf("RetryAfter = %d, want 60", rlErr.RetryAfter)
	}
	if !errors.Is(err, ErrRateLimit) {
		t.Fatal("errors.Is should match ErrRateLimit")
	}
}

func TestChatCompletionReturnsErrServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"internal error","type":"server_error"}}`))
	}))
	t.Cleanup(server.Close)

	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = p.ChatCompletion(context.Background(), testKey(providers.ProviderGroq), testChatRequest())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
}

func TestChatCompletionReturnsGenericErrorFor4xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"bad request"}}`))
	}))
	t.Cleanup(server.Close)

	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = p.ChatCompletion(context.Background(), testKey(providers.ProviderGroq), testChatRequest())
	if err == nil {
		t.Fatal("expected error for 400")
	}
	if strings.Contains(err.Error(), "server error") {
		t.Fatalf("error = %q, should not be server error", err.Error())
	}
}

func TestChatCompletionEmptyBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		// empty body
	}))
	t.Cleanup(server.Close)

	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = p.ChatCompletion(context.Background(), testKey(providers.ProviderGroq), testChatRequest())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Fatalf("error = %q, want 'empty response' for blank body", err.Error())
	}
}

func TestListModelsReturnsAuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"bad key"}}`))
	}))
	t.Cleanup(server.Close)

	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = p.ListModels(context.Background(), testKey(providers.ProviderGroq))
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

func TestListModelsDecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(server.Close)

	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = p.ListModels(context.Background(), testKey(providers.ProviderGroq))
	if err == nil {
		t.Fatal("expected decode error")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Fatalf("error = %q, want parse context", err.Error())
	}
}

func TestChatCompletionDecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(server.Close)

	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = p.ChatCompletion(context.Background(), testKey(providers.ProviderGroq), testChatRequest())
	if err == nil {
		t.Fatal("expected decode error")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Fatalf("error = %q, want parse context", err.Error())
	}
}

// --- retryAfterSeconds ---

func TestRetryAfterSecondsEmpty(t *testing.T) {
	if retryAfterSeconds("") != 0 {
		t.Fatal("expected 0 for empty")
	}
}

func TestRetryAfterSecondsValid(t *testing.T) {
	if retryAfterSeconds("30") != 30 {
		t.Fatal("expected 30")
	}
}

func TestRetryAfterSecondsInvalid(t *testing.T) {
	if retryAfterSeconds("not-a-number") != 0 {
		t.Fatal("expected 0 for invalid")
	}
}

// --- parseErrorMessage ---

func TestParseErrorMessageValidJSON(t *testing.T) {
	body := []byte(`{"error":{"message":"bad token","type":"auth"}}`)
	got := parseErrorMessage(body)
	if got != "bad token" {
		t.Fatalf("got %q, want bad token", got)
	}
}

func TestParseErrorMessagePlainText(t *testing.T) {
	body := []byte("plain error text")
	got := parseErrorMessage(body)
	if got != "plain error text" {
		t.Fatalf("got %q, want plain error text", got)
	}
}

func TestParseErrorMessageEmpty(t *testing.T) {
	got := parseErrorMessage([]byte(""))
	if got != "empty response" {
		t.Fatalf("got %q, want empty response", got)
	}
}

func TestParseErrorMessageJSONNoMessage(t *testing.T) {
	body := []byte(`{"error":{"type":"auth"}}`)
	got := parseErrorMessage(body)
	// Falls back to raw text since message is empty
	if got != `{"error":{"type":"auth"}}` {
		t.Fatalf("got %q, want raw body fallback", got)
	}
}

// --- do() with cancelled/expired context ---

func TestDoContextAlreadyCancelled(t *testing.T) {
	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: "http://127.0.0.1:1"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = p.ChatCompletion(ctx, testKey(providers.ProviderGroq), testChatRequest())
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestDoContextExpiredDeadline(t *testing.T) {
	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: "http://127.0.0.1:1"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Deadline already in the past
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	_, err = p.ChatCompletion(ctx, testKey(providers.ProviderGroq), testChatRequest())
	if err == nil {
		t.Fatal("expected error for expired deadline")
	}
}

// --- ChatCompletionStream error paths ---

func TestChatCompletionStreamHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	t.Cleanup(server.Close)

	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = p.ChatCompletionStream(context.Background(), testKey(providers.ProviderGroq), testChatRequest())
	if err == nil {
		t.Fatal("expected error for non-2xx stream response")
	}
	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestChatCompletionStreamNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = p.ChatCompletionStream(context.Background(), testKey(providers.ProviderGroq), testChatRequest())
	if err == nil {
		t.Fatal("expected network error")
	}
	if !strings.Contains(err.Error(), "chat completion stream") {
		t.Fatalf("error = %q, missing stream context", err.Error())
	}
}

// --- parseSSE edge cases ---

func TestParseSSENoDataLines(t *testing.T) {
	// Only blank lines — nothing dispatched
	body := strings.NewReader("\n\n\n")
	chunks := make(chan providers.StreamChunk, 10)
	parseSSE(body, chunks)
	close(chunks)
	var got []providers.StreamChunk
	for c := range chunks {
		got = append(got, c)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 chunks for blank-only input, got %d", len(got))
	}
}

func TestParseSSETrailingDataNoBlankLine(t *testing.T) {
	// Data at end without trailing blank line — handleSSEData called after loop
	body := strings.NewReader("data: " + streamChunkContentJSON)
	chunks := make(chan providers.StreamChunk, 10)
	parseSSE(body, chunks)
	close(chunks)
	var got []providers.StreamChunk
	for c := range chunks {
		got = append(got, c)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 chunk for trailing data, got %d", len(got))
	}
}

func TestParseSSEDoneInTrailingPosition(t *testing.T) {
	// [DONE] at end without trailing blank line
	body := strings.NewReader("data: [DONE]")
	chunks := make(chan providers.StreamChunk, 10)
	parseSSE(body, chunks)
	close(chunks)
	var got []providers.StreamChunk
	for c := range chunks {
		got = append(got, c)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 chunks for trailing [DONE], got %d", len(got))
	}
}

// --- mapError via fasthttp response (through ChatCompletion) ---

func TestMapErrorViaFasthttpCallsMapStatusError(t *testing.T) {
	// mapError is called for ChatCompletion non-2xx; already covered above
	// but let's cover mapError directly via a 500 response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`service down`))
	}))
	t.Cleanup(server.Close)

	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = p.ChatCompletion(context.Background(), testKey(providers.ProviderGroq), testChatRequest())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer for 503, got %v", err)
	}
}

// --- newHTTPJSONRequest marshal error (nil body skips marshal) ---

func TestChatCompletionStreamMarshalError(t *testing.T) {
	// We can't easily cause a marshal error via the public API since ChatRequest
	// is always marshalable. Cover newJSONRequest marshal path via newHTTPJSONRequest
	// indirectly: stream request always has a valid body, but we can verify
	// the stream sets Content-Type when body is non-nil.
	var gotCT string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	t.Cleanup(server.Close)

	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ch, err := p.ChatCompletionStream(context.Background(), testKey(providers.ProviderGroq), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}
	for range ch {
	}
	if gotCT != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", gotCT)
	}
}

// --- do() with deadline that expires before the request times out ---

func TestDoWithValidDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(chatResponseJSON))
	}))
	t.Cleanup(server.Close)

	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = p.ChatCompletion(ctx, testKey(providers.ProviderGroq), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletion with deadline: %v", err)
	}
}

// --- mapError via fasthttp.Response directly ---

func TestMapErrorDirectly(t *testing.T) {
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	resp.SetStatusCode(fasthttp.StatusUnauthorized)
	resp.SetBodyString(`{"error":{"message":"unauthorized"}}`)

	err := mapError(providers.ProviderGroq, resp)
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

// --- handleSSEData edge: empty dataLines slice ---

func TestHandleSSEDataEmpty(t *testing.T) {
	chunks := make(chan providers.StreamChunk, 10)
	done, failed := handleSSEData(nil, chunks)
	if done || failed {
		t.Fatalf("empty dataLines should return false, false; got done=%v failed=%v", done, failed)
	}
}

// Verify that parseErrorMessage handles non-JSON body correctly
func TestParseErrorMessageNonJSON(t *testing.T) {
	got := parseErrorMessage([]byte("  connection reset  "))
	if got != "connection reset" {
		t.Fatalf("got %q, want trimmed text", got)
	}
}

// Verify retryAfterSeconds with string value that is integer
func TestRetryAfterSecondsZero(t *testing.T) {
	if retryAfterSeconds("0") != 0 {
		t.Fatal("expected 0 for '0'")
	}
}

// mapStatusError with retry-after header via stream path
func TestChatCompletionStreamRateLimitWithRetryAfter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"slow down"}}`))
	}))
	t.Cleanup(server.Close)

	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = p.ChatCompletionStream(context.Background(), testKey(providers.ProviderGroq), testChatRequest())
	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected RateLimitError, got %T: %v", err, err)
	}
	if rlErr.RetryAfter != 120 {
		t.Fatalf("RetryAfter = %d, want 120", rlErr.RetryAfter)
	}
}

// Confirm do() returns error when fasthttp client fails (network error)
func TestDoNetworkError(t *testing.T) {
	// Port 1 is not listening — connection refused
	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: "http://127.0.0.1:1"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = p.ChatCompletion(context.Background(), testKey(providers.ProviderGroq), testChatRequest())
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
	if !strings.Contains(err.Error(), "chat completion") {
		t.Fatalf("error = %q, want chat completion context", err.Error())
	}
}

// Verify the fmt.Errorf path for provider-level list models error
func TestListModelsNetworkError(t *testing.T) {
	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: "http://127.0.0.1:1"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = p.ListModels(context.Background(), testKey(providers.ProviderGroq))
	if err == nil {
		t.Fatal("expected network error")
	}
	if !strings.Contains(err.Error(), "list models") {
		t.Fatalf("error = %q, want list models context", err.Error())
	}
}

// --- parseSSE scanner error (bad reader) ---

type errReader struct{}

func (e errReader) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("read error")
}

func TestParseSSEScannerError(t *testing.T) {
	chunks := make(chan providers.StreamChunk, 10)
	parseSSE(errReader{}, chunks)
	close(chunks)

	var got []providers.StreamChunk
	for c := range chunks {
		got = append(got, c)
	}
	if len(got) != 1 || got[0].Error == nil {
		t.Fatalf("expected one error chunk for scanner error, got %+v", got)
	}
	if got[0].Error.Code != "upstream_stream_error" {
		t.Fatalf("error code = %q, want upstream_stream_error", got[0].Error.Code)
	}
}

// --- newJSONRequest marshal error ---
// We call newJSONRequest directly (same package) with an unmarshalable body.
func TestNewJSONRequestMarshalError(t *testing.T) {
	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: "http://example.com"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// channels cannot be JSON-marshaled
	body := map[string]any{"ch": make(chan int)}
	req, err := p.newJSONRequest("POST", "/v1/chat/completions", testKey(providers.ProviderGroq), body)
	if req != nil {
		fasthttp.ReleaseRequest(req)
		t.Fatal("expected nil request for marshal error")
	}
	if err == nil {
		t.Fatal("expected marshal error")
	}
	if !strings.Contains(err.Error(), "marshal") {
		t.Fatalf("error = %q, want marshal context", err.Error())
	}
}

// --- newJSONRequest headers range ---
// Provider with headers, newJSONRequest called internally via ChatCompletion.
func TestNewJSONRequestSetsCustomHeaders(t *testing.T) {
	var gotHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Custom")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(chatResponseJSON))
	}))
	t.Cleanup(server.Close)

	p, err := New(Config{
		Provider: providers.ProviderGroq,
		BaseURL:  server.URL,
		Headers:  map[string]string{"X-Custom": "my-value"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = p.ChatCompletion(context.Background(), testKey(providers.ProviderGroq), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if gotHeader != "my-value" {
		t.Fatalf("X-Custom = %q, want my-value", gotHeader)
	}
}

// --- newHTTPJSONRequest bad URL ---
// Provider with a bad base URL triggers http.NewRequestWithContext error.
func TestNewHTTPJSONRequestBadURL(t *testing.T) {
	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: "://bad-url"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = p.ChatCompletionStream(context.Background(), testKey(providers.ProviderGroq), testChatRequest())
	if err == nil {
		t.Fatal("expected error for bad URL")
	}
	if !strings.Contains(err.Error(), "create") {
		t.Fatalf("error = %q, want create context", err.Error())
	}
}

// --- newHTTPJSONRequest headers range ---
// Provider with headers, stream path hits the range loop.
func TestNewHTTPJSONRequestSetsCustomHeaders(t *testing.T) {
	var gotHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Stream-Custom")
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	t.Cleanup(server.Close)

	p, err := New(Config{
		Provider: providers.ProviderGroq,
		BaseURL:  server.URL,
		Headers:  map[string]string{"X-Stream-Custom": "stream-val"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ch, err := p.ChatCompletionStream(context.Background(), testKey(providers.ProviderGroq), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}
	for range ch {
	}
	if gotHeader != "stream-val" {
		t.Fatalf("X-Stream-Custom = %q, want stream-val", gotHeader)
	}
}

// --- do() timeout <= 0 (deadline already expired for fasthttp DoTimeout) ---
func TestDoExpiredDeadlineForFasthttp(t *testing.T) {
	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: "http://127.0.0.1:1"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Deadline in the past → timeout = time.Until(past) <= 0
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Hour))
	defer cancel()
	_, err = p.ChatCompletion(ctx, testKey(providers.ProviderGroq), testChatRequest())
	if err == nil {
		t.Fatal("expected error for expired fasthttp deadline")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}

// --- parseSSE failed path in outer loop ---
// When handleSSEData returns failed=true (malformed JSON on blank-line dispatch),
// parseSSE returns immediately without processing further lines.
func TestParseSSEStopsAfterMalformedEvent(t *testing.T) {
	// First event: malformed JSON → failed=true → loop exits
	// Second event would normally be valid, but should not appear
	input := strings.NewReader("data: {bad json\n\ndata: " + streamChunkContentJSON + "\n\n")
	chunks := make(chan providers.StreamChunk, 10)
	parseSSE(input, chunks)
	close(chunks)

	var got []providers.StreamChunk
	for c := range chunks {
		got = append(got, c)
	}
	if len(got) != 1 || got[0].Error == nil {
		t.Fatalf("expected exactly one error chunk, got %+v", got)
	}
	if got[0].Error.Code != "upstream_stream_malformed" {
		t.Fatalf("error code = %q, want upstream_stream_malformed", got[0].Error.Code)
	}
}

// --- newHTTPJSONRequest marshal error ---
// Trigger via ChatCompletionStream with an unmarshalable request body.
// We achieve this by directly calling newHTTPJSONRequest with a bad body.
func TestNewHTTPJSONRequestMarshalError(t *testing.T) {
	p, err := New(Config{Provider: providers.ProviderGroq, BaseURL: "http://example.com"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	body := map[string]any{"ch": make(chan int)}
	req, err := p.newHTTPJSONRequest(context.Background(), "POST", "/v1/chat/completions", testKey(providers.ProviderGroq), body)
	if req != nil {
		t.Fatal("expected nil request for marshal error")
	}
	if err == nil {
		t.Fatal("expected marshal error")
	}
	if !strings.Contains(err.Error(), "marshal") {
		t.Fatalf("error = %q, want marshal context", err.Error())
	}
}

// --- parseSSE failed in trailing position (line 287-288) ---
// Malformed JSON as the last event without a trailing blank line.
func TestParseSSETrailingMalformedData(t *testing.T) {
	// No trailing \n\n — dispatched by post-loop handleSSEData call
	input := strings.NewReader("data: {bad json no closing brace")
	chunks := make(chan providers.StreamChunk, 10)
	parseSSE(input, chunks)
	close(chunks)

	var got []providers.StreamChunk
	for c := range chunks {
		got = append(got, c)
	}
	if len(got) != 1 || got[0].Error == nil {
		t.Fatalf("expected one error chunk for trailing malformed data, got %+v", got)
	}
	if got[0].Error.Code != "upstream_stream_malformed" {
		t.Fatalf("error code = %q, want upstream_stream_malformed", got[0].Error.Code)
	}
}

func TestProviderWithProxyPool(t *testing.T) {
	p, err := New(Config{Provider: "test", BaseURL: "https://example.com"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got := p.WithProxyPool(nil)
	if got == nil {
		t.Fatal("WithProxyPool returned nil")
	}
	if got.Name() != "test" {
		t.Fatalf("name = %q, want test", got.Name())
	}
}
