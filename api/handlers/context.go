package handlers

import (
	"context"

	"github.com/valyala/fasthttp"
)

func requestContext(ctx *fasthttp.RequestCtx) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
