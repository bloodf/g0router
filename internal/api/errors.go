package api

import (
	"encoding/json"

	"github.com/valyala/fasthttp"
)

// writeError writes an OpenAI-compatible error response.
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
	b, _ := json.Marshal(resp)
	ctx.SetStatusCode(status)
	ctx.SetContentTypeBytes([]byte("application/json"))
	ctx.SetBody(b)
}
