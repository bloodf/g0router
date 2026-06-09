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
	resp := map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    errType,
		},
	}
	if code != nil {
		resp["error"].(map[string]any)["code"] = *code
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
