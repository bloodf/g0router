package handlers

import (
	"context"
	"strings"

	"github.com/bloodf/g0router/internal/proxy"
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
	return buildRequestContext(context.Background(), ctx)
}

// streamContext returns a cancellable context for streaming requests that
// preserves API key ID and routing headers from the fasthttp request.
func streamContext(ctx *fasthttp.RequestCtx) (context.Context, context.CancelFunc) {
	return context.WithCancel(buildRequestContext(context.Background(), ctx))
}

func buildRequestContext(base context.Context, ctx *fasthttp.RequestCtx) context.Context {
	c := base
	if id, ok := ctx.UserValue("g0router.api_key_id").(string); ok && id != "" {
		c = proxy.WithAPIKeyID(c, id)
	}
	headers := make(map[string]string)
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		headers[strings.ToLower(string(key))] = string(value)
	})
	c = proxy.WithRoutingHeaders(c, headers)
	return c
}
