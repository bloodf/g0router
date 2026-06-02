package handlers

import (
	"encoding/json"

	"github.com/valyala/fasthttp"
)

func Health(ctx *fasthttp.RequestCtx, version string) {
	body, err := json.Marshal(map[string]string{
		"status":  "ok",
		"version": version,
	})
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(body)
}
