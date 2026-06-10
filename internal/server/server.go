package server

import (
	"io/fs"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/store"
	httprouter "github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

// New creates a fasthttp server with API routes and UI fallback.
// st backs the management API; pass nil to serve only the OpenAI-compatible
// surface (no admin routes).
func New(uiFS fs.FS, st *store.Store, allowedOrigins []string) *fasthttp.Server {
	infRouter := inference.NewRouter()

	r := httprouter.New()
	r.NotFound = uiHandler(uiFS)

	// Health check (public, no auth)
	r.GET("/api/health", healthHandler())

	// OpenAI-compatible API routes
	RegisterOpenAIRoutes(r, infRouter)

	// Management API routes
	if st != nil {
		RegisterAdminRoutes(r, NewAdminHandlers(st))
	}

	handler := Chain(r.Handler,
		RequestIDMiddleware,
		CORSMiddleware(allowedOrigins),
	)

	return &fasthttp.Server{
		Handler:            handler,
		ReadTimeout:        0,
		WriteTimeout:       0,
		MaxRequestBodySize: 1 << 30, // 1 GiB
	}
}
