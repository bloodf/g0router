package server

import (
	"testing"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/translation"
	httprouter "github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

func TestResponsesRouteRegistered(t *testing.T) {
	r := httprouter.New()
	r.NotFound = func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString("not found")
	}
	RegisterOpenAIRoutes(r, inference.NewRouter(translation.NewRegistry()), nil)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetRequestURI("/v1/responses")
	ctx.Request.SetBody([]byte(`{"model":"gpt-4"}`))
	r.Handler(&ctx)

	if ctx.Response.StatusCode() == fasthttp.StatusNotFound {
		t.Fatalf("/v1/responses returned 404 — route not registered")
	}
}
