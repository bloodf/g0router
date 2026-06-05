package handlers

import (
	"context"

	"github.com/valyala/fasthttp"
)

// requestContext returns a context for downstream work that is fully detached
// from the pooled *fasthttp.RequestCtx.
//
// The fasthttp RequestCtx is recycled (Server.releaseCtx -> reset) once the
// handler returns. Returning it directly leaks the pooled value into downstream
// net/http clients, whose Transport wraps it in a cancel context and, from a
// background readLoop goroutine, walks ctx.Value()/UserValue() during
// cancellation. That read races with fasthttp resetting the same RequestCtx,
// producing a use-after-recycle data race.
//
// fasthttp's RequestCtx.Done() only fires on server shutdown (not per-request
// client disconnect), so passing the RequestCtx never carried request-scoped
// cancellation anyway. Detaching to context.Background() drops nothing of value
// and removes the racy Value()/cancellation chain into the recyclable ctx.
func requestContext(ctx *fasthttp.RequestCtx) context.Context {
	return context.Background()
}
