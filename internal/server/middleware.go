package server

import (
	"fmt"

	"github.com/valyala/fasthttp"
)

// Middleware is a function that wraps a fasthttp.RequestHandler.
type Middleware func(fasthttp.RequestHandler) fasthttp.RequestHandler

// Chain applies a slice of middlewares to a handler.
func Chain(h fasthttp.RequestHandler, mw ...Middleware) fasthttp.RequestHandler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

// RequestIDMiddleware injects a request ID header if not present.
func RequestIDMiddleware(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		if len(ctx.Request.Header.Peek("X-Request-ID")) == 0 {
			ctx.Response.Header.Set("X-Request-ID", fmt.Sprintf("%d", ctx.ID()))
		}
		next(ctx)
	}
}

// CORSMiddleware adds CORS headers for browser clients.
func CORSMiddleware(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		origin := string(ctx.Request.Header.Peek("Origin"))
		if origin == "" {
			origin = "*"
		}
		ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
		ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")

		if string(ctx.Method()) == "OPTIONS" {
			ctx.SetStatusCode(fasthttp.StatusNoContent)
			return
		}
		next(ctx)
	}
}
