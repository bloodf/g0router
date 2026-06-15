package api

import (
	"encoding/json"

	"github.com/valyala/fasthttp"
)

// jsonMarshal is a seam for injecting JSON marshal failures in tests
// (AUD-009/010/011/012). All response serialization in the api package
// must go through this variable instead of json.Marshal directly so
// that tests can exercise the error path.
var jsonMarshal = json.Marshal

// writeError writes an OpenAI-compatible error response.
// If JSON marshaling fails (e.g. an unrepresentable value), falls back
// to a plain-text 500 response instead of silently dropping the error.
func writeError(ctx *fasthttp.RequestCtx, status int, errType, message string, code *string) {
	writeErrorWithParam(ctx, status, errType, message, code, nil)
}

// writeErrorWithParam is writeError plus the optional OpenAI error `param` field.
// When param is non-nil it is surfaced under error.param (the upstream openai
// error object includes param on validation errors); writeError delegates here
// with a nil param so all existing call sites keep their stable signature
// (PAR-BF-OAI-302 variant-augment — the existing APIError.Param surfaced; the
// envelope shape is otherwise unchanged).
func writeErrorWithParam(ctx *fasthttp.RequestCtx, status int, errType, message string, code, param *string) {
	resp := map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    errType,
		},
	}
	if code != nil {
		resp["error"].(map[string]any)["code"] = *code
	}
	if param != nil {
		resp["error"].(map[string]any)["param"] = *param
	}
	b, err := jsonMarshal(resp)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentTypeBytes([]byte("text/plain"))
		ctx.SetBodyString("internal error")
		return
	}
	ctx.SetStatusCode(status)
	ctx.SetContentTypeBytes([]byte("application/json"))
	ctx.SetBody(b)
}
