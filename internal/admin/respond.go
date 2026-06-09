package admin

import (
	"encoding/json"

	"github.com/valyala/fasthttp"
)

// envelope is the management API response shape: {data, error}.
type envelope struct {
	Data  any            `json:"data"`
	Error *envelopeError `json:"error"`
}

type envelopeError struct {
	Message string `json:"message"`
}

func writeData(ctx *fasthttp.RequestCtx, status int, data any) {
	writeEnvelope(ctx, status, envelope{Data: data})
}

func writeError(ctx *fasthttp.RequestCtx, status int, message string) {
	writeEnvelope(ctx, status, envelope{Error: &envelopeError{Message: message}})
}

func writeEnvelope(ctx *fasthttp.RequestCtx, status int, env envelope) {
	b, err := json.Marshal(env)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentType("application/json")
		ctx.SetBodyString(`{"data":null,"error":{"message":"encode response"}}`)
		return
	}
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json")
	ctx.SetBody(b)
}
