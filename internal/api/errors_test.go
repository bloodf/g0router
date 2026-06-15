package api

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
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

// TestWriteErrorWithParamSurfacesParam verifies the additive param-surface
// (PAR-BF-OAI-302 variant-augment): a non-nil param is emitted in the OpenAI
// error object; the existing envelope shape is otherwise unchanged.
func TestWriteErrorWithParamSurfacesParam(t *testing.T) {
	var ctx fasthttp.RequestCtx
	param := "model"
	writeErrorWithParam(&ctx, fasthttp.StatusBadRequest, "invalid_request_error", "bad model", nil, &param)

	var got struct {
		Error struct {
			Message string  `json:"message"`
			Type    string  `json:"type"`
			Param   *string `json:"param"`
		} `json:"error"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &got); err != nil {
		t.Fatalf("unmarshal: %v; body=%s", err, ctx.Response.Body())
	}
	if got.Error.Param == nil || *got.Error.Param != "model" {
		t.Errorf("error.param = %v, want \"model\"; body=%s", got.Error.Param, ctx.Response.Body())
	}
}

// TestWriteErrorWithParamOmitsNilParam verifies a nil param produces no "param"
// key (omitempty parity with the OpenAI error object).
func TestWriteErrorWithParamOmitsNilParam(t *testing.T) {
	var ctx fasthttp.RequestCtx
	writeErrorWithParam(&ctx, fasthttp.StatusBadRequest, "invalid_request_error", "oops", nil, nil)

	if strings.Contains(string(ctx.Response.Body()), `"param"`) {
		t.Errorf("body must not contain param key when nil; body=%s", ctx.Response.Body())
	}
}

// TestWriteProviderErrorForwardsParam verifies writeProviderError forwards a
// ProviderError.Param into the emitted error object.
func TestWriteProviderErrorForwardsParam(t *testing.T) {
	var ctx fasthttp.RequestCtx
	param := "input"
	writeProviderError(&ctx, &schemas.ProviderError{StatusCode: 400, Type: "invalid_request_error", Message: "bad input", Param: &param})

	if !strings.Contains(string(ctx.Response.Body()), `"param":"input"`) {
		t.Errorf("body must surface provider param; body=%s", ctx.Response.Body())
	}
}
