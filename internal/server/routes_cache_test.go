package server

import (
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/admin"
	"github.com/bloodf/g0router/internal/auth"
	httprouter "github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

// TestSemanticCacheRoutesRegistered proves RegisterAdminRoutes wires GET and
// DELETE /api/cache/semantic (the routes_admin serial-chain terminus,
// bf-core-2). A registered route resolves to RequireSession (401), never 404.
func TestSemanticCacheRoutesRegistered(t *testing.T) {
	st := newTestStore(t)
	sessions := auth.NewSessions(st, time.Hour)
	h := admin.New(st, sessions, nil)

	r := httprouter.New()
	r.NotFound = func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString("not found")
	}
	RegisterAdminRoutes(r, h)

	for _, method := range []string{"GET", "DELETE"} {
		var ctx fasthttp.RequestCtx
		ctx.Request.Header.SetMethod(method)
		ctx.Request.SetRequestURI("/api/cache/semantic")
		r.Handler(&ctx)
		if ctx.Response.StatusCode() == fasthttp.StatusNotFound {
			t.Fatalf("%s /api/cache/semantic not registered (404)", method)
		}
	}
}
