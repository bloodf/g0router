package openai

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
)

// ---- Name ----

func TestName(t *testing.T) {
	p := New("")
	if p.Name() != providers.ProviderOpenAI {
		t.Fatalf("Name = %q", p.Name())
	}
}

// ---- RateLimitError ----

func TestRateLimitErrorMessage(t *testing.T) {
	e := &RateLimitError{Message: "slow", RetryAfter: 5}
	if !strings.Contains(e.Error(), "slow") || !strings.Contains(e.Error(), "5") {
		t.Fatalf("error = %q", e.Error())
	}
	e2 := &RateLimitError{}
	if e2.Error() != ErrRateLimit.Error() {
		t.Fatalf("empty = %q", e2.Error())
	}
	e3 := &RateLimitError{Message: "slow"}
	if !strings.Contains(e3.Error(), "slow") {
		t.Fatalf("no retry = %q", e3.Error())
	}
}

func TestRateLimitErrorIs(t *testing.T) {
	e := &RateLimitError{Message: "x"}
	if !errors.Is(e, ErrRateLimit) {
		t.Fatal("Is(ErrRateLimit) should be true")
	}
}

// ---- mapStatusError paths ----

func TestMapStatusErrorForbidden(t *testing.T) {
	err := mapStatusError(http.StatusForbidden, []byte(`{"error":{"message":"forbidden"}}`), "")
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

func TestMapStatusError400(t *testing.T) {
	err := mapStatusError(400, []byte(`{"error":{"message":"bad"}}`), "")
	if errors.Is(err, ErrAuth) || errors.Is(err, ErrServer) {
		t.Fatalf("unexpected sentinel: %v", err)
	}
	if !strings.Contains(err.Error(), "400") {
		t.Fatalf("error = %v", err)
	}
}

func TestMapStatusErrorEmptyBody(t *testing.T) {
	err := mapStatusError(500, []byte(""), "")
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestMapStatusErrorPlainBody(t *testing.T) {
	err := mapStatusError(500, []byte("internal error"), "")
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
}

func TestMapStatusError429NoRetryAfter(t *testing.T) {
	err := mapStatusError(429, []byte(`{"error":{"message":"limit"}}`), "")
	var rle *RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("expected RateLimitError, got %T", err)
	}
	if rle.RetryAfter != 0 {
		t.Fatalf("RetryAfter = %d", rle.RetryAfter)
	}
}

func TestMapStatusError429NonNumericRetryAfter(t *testing.T) {
	err := mapStatusError(429, []byte(`{"error":{"message":"x"}}`), "abc")
	var rle *RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("expected RateLimitError, got %T", err)
	}
	if rle.RetryAfter != 0 {
		t.Fatalf("RetryAfter = %d", rle.RetryAfter)
	}
}

// ---- parseErrorMessage ----

func TestParseErrorMessageEmptyBody(t *testing.T) {
	if parseErrorMessage(nil) != "empty response" {
		t.Fatalf("want empty response")
	}
}

func TestParseErrorMessagePlainText(t *testing.T) {
	msg := parseErrorMessage([]byte("plain text"))
	if msg != "plain text" {
		t.Fatalf("msg = %q", msg)
	}
}

func TestParseErrorMessageJSON(t *testing.T) {
	msg := parseErrorMessage([]byte(`{"error":{"message":"structured"}}`))
	if msg != "structured" {
		t.Fatalf("msg = %q", msg)
	}
}

// ---- retryAfterSeconds ----

func TestRetryAfterSecondsEmpty(t *testing.T) {
	if retryAfterSeconds("") != 0 {
		t.Fatal("empty should be 0")
	}
}

func TestRetryAfterSecondsValid(t *testing.T) {
	if retryAfterSeconds("30") != 30 {
		t.Fatal("30 should be 30")
	}
}

func TestRetryAfterSecondsInvalid(t *testing.T) {
	if retryAfterSeconds("abc") != 0 {
		t.Fatal("invalid should be 0")
	}
}

// ---- newJSONRequest marshal failure ----

func TestNewJSONRequestMarshalFailure(t *testing.T) {
	p := New("http://example.com")
	_, err := p.newJSONRequest("POST", "/v1/chat/completions", testKey(), make(chan int))
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

// ---- ChatCompletion: nil request → toAnthropicRequest-style failure doesn't apply;
//      test bad JSON response ----

func TestChatCompletionBadJSONResponse(t *testing.T) {
	server := jsonServer(t, http.StatusOK, `not json`, nil)
	p := New(server.URL)
	_, err := p.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

// ---- ChatCompletion: do() error (network) ----

func TestChatCompletionDoError(t *testing.T) {
	p := New("http://127.0.0.1:1")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := p.ChatCompletion(ctx, testKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected network error")
	}
}

// ---- ListModels: bad JSON ----

func TestListModelsBadJSON(t *testing.T) {
	server := jsonServer(t, http.StatusOK, `not json`, nil)
	p := New(server.URL)
	_, err := p.ListModels(context.Background(), testKey())
	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

// ---- ListModels: error status ----

func TestListModelsError401(t *testing.T) {
	server := jsonServer(t, http.StatusUnauthorized, `{"error":{"message":"unauth"}}`, nil)
	p := New(server.URL)
	_, err := p.ListModels(context.Background(), testKey())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

// ---- ListModels: do() error ----

func TestListModelsDoError(t *testing.T) {
	p := New("http://127.0.0.1:1")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := p.ListModels(ctx, testKey())
	if err == nil {
		t.Fatal("expected network error")
	}
}

// ---- do: cancelled context ----

func TestDoContextCancelledBeforeRequest(t *testing.T) {
	p := New("http://127.0.0.1:1")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := p.ChatCompletion(ctx, testKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// ---- do: deadline path ----

func TestDoWithDeadlineReachesServer(t *testing.T) {
	server := jsonServer(t, http.StatusOK, chatResponseJSON, nil)
	p := New(server.URL)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer cancel()
	resp, err := p.ChatCompletion(ctx, testKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if resp.ID != "chatcmpl-123" {
		t.Fatalf("resp.ID = %q", resp.ID)
	}
}

func TestDoWithExpiredDeadline(t *testing.T) {
	p := New("http://127.0.0.1:1")
	deadline := time.Now().Add(-1 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	_, err := p.ChatCompletion(ctx, testKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected error for expired deadline")
	}
}

// ---- ChatCompletionStream: error status ----

func TestChatCompletionStreamErrorStatus(t *testing.T) {
	server := jsonServer(t, http.StatusUnauthorized, `{"error":{"message":"bad key"}}`, nil)
	p := New(server.URL)
	_, err := p.ChatCompletionStream(context.Background(), testKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---- parseSSE: [DONE] terminates ----

func TestParseSSEDoneTerminates(t *testing.T) {
	body := "data: [DONE]\n\ndata: " + streamChunkContentJSON + "\n\n"
	chunks := make(chan providers.StreamChunk, 10)
	parseSSE(strings.NewReader(body), chunks)
	close(chunks)
	var got []providers.StreamChunk
	for c := range chunks {
		got = append(got, c)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 chunks after [DONE], got %d", len(got))
	}
}

// ---- parseSSE: flush at EOF (no trailing blank line) ----

func TestParseSSEFlushAtEOF(t *testing.T) {
	body := "data: [DONE]" // no trailing \n\n
	chunks := make(chan providers.StreamChunk, 10)
	parseSSE(strings.NewReader(body), chunks)
	close(chunks)
	var got []providers.StreamChunk
	for c := range chunks {
		got = append(got, c)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 after [DONE] flush, got %d", len(got))
	}
}

// ---- parseSSE: scanner error ----

type openaiScanErrReader struct {
	payload string
	sent    bool
}

func (r *openaiScanErrReader) Read(p []byte) (int, error) {
	if !r.sent {
		r.sent = true
		n := copy(p, r.payload)
		return n, nil
	}
	return 0, &openaiReadErr{}
}

type openaiReadErr struct{}

func (e *openaiReadErr) Error() string { return "injected openai read error" }

func TestParseSSEScannerError(t *testing.T) {
	r := &openaiScanErrReader{payload: "data: " + streamChunkContentJSON + "\n\n"}
	chunks := make(chan providers.StreamChunk, 10)
	parseSSE(r, chunks)
	close(chunks)
	var got []providers.StreamChunk
	for c := range chunks {
		got = append(got, c)
	}
	_ = got
}

// ---- handleSSEData: empty lines ----

func TestHandleSSEDataEmpty(t *testing.T) {
	chunks := make(chan providers.StreamChunk, 1)
	done, failed := handleSSEData(nil, chunks)
	if done || failed {
		t.Fatal("empty should return false, false")
	}
}

// ---- Responses: nil request ----

func TestResponsesNilRequest(t *testing.T) {
	p := New("http://example.com")
	_, err := p.Responses(context.Background(), testKey(), nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

// ---- Responses: bad JSON response ----

func TestResponsesBadJSONResponse(t *testing.T) {
	server := jsonServer(t, http.StatusOK, `not json`, nil)
	p := New(server.URL)
	_, err := p.Responses(context.Background(), testKey(), &ResponseRequest{Model: "gpt-4o"})
	if err == nil {
		t.Fatal("expected parse error")
	}
}

// ---- Responses: error status ----

func TestResponsesErrorStatus(t *testing.T) {
	server := jsonServer(t, http.StatusUnauthorized, `{"error":{"message":"bad"}}`, nil)
	p := New(server.URL)
	_, err := p.Responses(context.Background(), testKey(), &ResponseRequest{Model: "gpt-4o"})
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

// ---- Responses: do() error ----

func TestResponsesDoError(t *testing.T) {
	p := New("http://127.0.0.1:1")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := p.Responses(ctx, testKey(), &ResponseRequest{Model: "gpt-4o"})
	if err == nil {
		t.Fatal("expected network error")
	}
}

// ---- ResponsesStream: nil request ----

func TestResponsesStreamNilRequest(t *testing.T) {
	p := New("http://example.com")
	_, err := p.ResponsesStream(context.Background(), testKey(), nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

// ---- ResponsesStream: error status ----

func TestResponsesStreamErrorStatus(t *testing.T) {
	server := jsonServer(t, http.StatusUnauthorized, `{"error":{"message":"bad"}}`, nil)
	p := New(server.URL)
	_, err := p.ResponsesStream(context.Background(), testKey(), &ResponseRequest{Model: "gpt-4o"})
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

// ---- ResponsesStream: do() error ----

func TestResponsesStreamDoError(t *testing.T) {
	p := New("http://127.0.0.1:1")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := p.ResponsesStream(ctx, testKey(), &ResponseRequest{Model: "gpt-4o"})
	if err == nil {
		t.Fatal("expected network error")
	}
}

// ---- parseResponsesSSE: malformed JSON event is silently skipped ----

func TestParseResponsesSSEMalformedEvent(t *testing.T) {
	body := strings.Join([]string{
		`data: {bad json`,
		"",
		`data: {"type":"response.done","sequence_number":1}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
	events := make(chan ResponseEvent, 10)
	parseResponsesSSE(strings.NewReader(body), events)
	close(events)
	var got []ResponseEvent
	for e := range events {
		got = append(got, e)
	}
	// malformed silently skipped; valid event delivered
	if len(got) != 1 {
		t.Fatalf("events = %d", len(got))
	}
}

// ---- handleResponsesData: empty lines ----

func TestHandleResponsesDataEmpty(t *testing.T) {
	done := handleResponsesData(nil, func(string) bool { return false })
	if done {
		t.Fatal("empty should return false")
	}
}

// ---- handleResponsesData: [DONE] ----

func TestHandleResponsesDataDone(t *testing.T) {
	done := handleResponsesData([]string{"[DONE]"}, func(string) bool { return false })
	if !done {
		t.Fatal("[DONE] should return true")
	}
}

// ---- parseSSEData: flush at EOF ----

func TestParseSSEDataFlushAtEOF(t *testing.T) {
	body := `data: {"type":"response.done","sequence_number":1}` // no trailing blank
	var got []string
	parseSSEData(strings.NewReader(body), func(data string) bool {
		got = append(got, data)
		return false
	})
	if len(got) != 1 {
		t.Fatalf("got = %v", got)
	}
}

// ---- newHTTPJSONRequest: nil body ----

func TestNewHTTPJSONRequestNilBody(t *testing.T) {
	p := New("http://example.com")
	req, err := p.newHTTPJSONRequest(context.Background(), "GET", "/v1/models", testKey(), nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if req.Header.Get("Content-Type") != "" {
		t.Fatal("Content-Type should not be set for nil body")
	}
}

// ---- newHTTPJSONRequest: marshal failure ----

func TestNewHTTPJSONRequestMarshalFailure(t *testing.T) {
	p := New("http://example.com")
	_, err := p.newHTTPJSONRequest(context.Background(), "POST", "/v1/chat/completions", testKey(), make(chan int))
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

// ---- ChatCompletion: newHTTPJSONRequest marshal fail for stream ----

func TestChatCompletionStreamNewHTTPRequestFail(t *testing.T) {
	// ChatCompletionStream calls newHTTPJSONRequest; give it a request
	// that will marshal fine — test the error status path via 401.
	server := jsonServer(t, http.StatusForbidden, `{"error":{"message":"no"}}`, nil)
	p := New(server.URL)
	_, err := p.ChatCompletionStream(context.Background(), testKey(), testChatRequest())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

// ---- parseSSEData: handler returns true early ----

func TestParseSSEDataHandlerReturnsTrue(t *testing.T) {
	body := strings.Join([]string{
		`data: {"type":"a"}`,
		"",
		`data: {"type":"b"}`,
		"",
	}, "\n")
	var got []string
	parseSSEData(strings.NewReader(body), func(data string) bool {
		got = append(got, data)
		return true // stop after first
	})
	if len(got) != 1 {
		t.Fatalf("expected 1 handled event, got %d", len(got))
	}
}

// ---- do: ctx.Err after successful response — hard to hit, but test deadline ----

func TestDoDeadlineNotYetExpired(t *testing.T) {
	server := jsonServer(t, http.StatusOK, chatResponseJSON, nil)
	p := New(server.URL)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()
	_, err := p.ListModels(ctx, testKey())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---- do: expired deadline (timeout <= 0) ----

func TestDoExpiredDeadline(t *testing.T) {
	p := New("http://127.0.0.1:1")
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	_, err := p.ChatCompletion(ctx, testKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected error for expired deadline")
	}
}

// ---- newHTTPJSONRequest: invalid method causes NewRequestWithContext error ----

func TestNewHTTPJSONRequestInvalidMethod(t *testing.T) {
	p := New("http://example.com")
	// A method with a space is invalid per net/http
	_, err := p.newHTTPJSONRequest(context.Background(), "INVALID METHOD", "/v1/chat/completions", testKey(), nil)
	if err == nil {
		t.Fatal("expected error for invalid HTTP method")
	}
}

// ---- ChatCompletion: newJSONRequest body marshal error ----
// The ChatRequest itself always marshals fine; test via ChatCompletionStream
// with a newHTTPJSONRequest marshal failure (chan in stream request body isn't reachable
// through public API, but the error path in newJSONRequest is reachable).

// Covered already by TestNewJSONRequestMarshalFailure which calls newJSONRequest directly.

// ---- ListModels: modelsResponseJSON with empty data ----

func TestListModelsEmptyData(t *testing.T) {
	server := jsonServer(t, http.StatusOK, `{"object":"list","data":[]}`, nil)
	p := New(server.URL)
	models, err := p.ListModels(context.Background(), testKey())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != 0 {
		t.Fatalf("models = %+v", models)
	}
}

// ---- parseSSE: flush path with malformed event triggers failed=true ----

func TestParseSSEFlushMalformed(t *testing.T) {
	// No trailing blank line — goes through the flush path at EOF.
	// Malformed JSON triggers streamErrorChunk (failed=true in handleSSEData).
	body := `data: {bad json` // no trailing \n\n → EOF flush path
	chunks := make(chan providers.StreamChunk, 10)
	parseSSE(strings.NewReader(body), chunks)
	close(chunks)
	var got []providers.StreamChunk
	for c := range chunks {
		got = append(got, c)
	}
	if len(got) != 1 || got[0].Error == nil {
		t.Fatalf("expected 1 error chunk from malformed flush, got %+v", got)
	}
	if got[0].Error.Code != "upstream_stream_malformed" {
		t.Fatalf("code = %q", got[0].Error.Code)
	}
}

// ---- ChatCompletionStream: stream client network error ----

func TestChatCompletionStreamNetworkError(t *testing.T) {
	p := New("http://127.0.0.1:1")
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := p.ChatCompletionStream(ctx, testKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected network error from stream client")
	}
}

// ---- Responses: newJSONRequest marshal fail via chan ----

func TestResponsesMarshalFailure(t *testing.T) {
	p := New("http://example.com")
	// ResponseRequest always marshals fine via public API; test newJSONRequest directly
	_, err := p.newJSONRequest("POST", "/v1/responses", testKey(), make(chan int))
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

// ---- unused bytes import guard ----
var _ = bytes.NewReader
