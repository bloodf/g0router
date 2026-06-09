package api

import (
	"errors"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

// TestWriteErrorMarshalFailureFallsBackToPlainText verifies that when
// jsonMarshal fails, writeError falls back to a plain-text 500 response
// instead of silently dropping the error (AUD-009/010/011/012).
func TestWriteErrorMarshalFailureFallsBackToPlainText(t *testing.T) {
	prev := jsonMarshal
	t.Cleanup(func() { jsonMarshal = prev })

	jsonMarshal = func(v any) ([]byte, error) {
		return nil, errors.New("simulated marshal failure")
	}

	var ctx fasthttp.RequestCtx
	writeError(&ctx, fasthttp.StatusBadRequest, "invalid_request_error", "test message", nil)

	if got := ctx.Response.StatusCode(); got != fasthttp.StatusInternalServerError {
		t.Errorf("status = %d, want %d", got, fasthttp.StatusInternalServerError)
	}
	if got := string(ctx.Response.Body()); got != "internal error" {
		t.Errorf("body = %q, want %q", got, "internal error")
	}
	if got := string(ctx.Response.Header.ContentType()); !strings.HasPrefix(got, "text/plain") {
		t.Errorf("content-type = %q, want text/plain prefix", got)
	}
}

// TestWriteErrorSuccessWritesJSONEnvelope verifies the happy path still
// produces a JSON envelope with the caller-supplied status and body.
func TestWriteErrorSuccessWritesJSONEnvelope(t *testing.T) {
	var ctx fasthttp.RequestCtx
	writeError(&ctx, fasthttp.StatusBadRequest, "invalid_request_error", "test message", nil)

	if got := ctx.Response.StatusCode(); got != fasthttp.StatusBadRequest {
		t.Errorf("status = %d, want %d", got, fasthttp.StatusBadRequest)
	}
	body := string(ctx.Response.Body())
	if !strings.Contains(body, `"error"`) || !strings.Contains(body, `"test message"`) {
		t.Errorf("body = %q, want it to contain error envelope", body)
	}
	if got := string(ctx.Response.Header.ContentType()); !strings.HasPrefix(got, "application/json") {
		t.Errorf("content-type = %q, want application/json prefix", got)
	}
}
