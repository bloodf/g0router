package server

import (
	"io/fs"

	"github.com/bloodf/g0router/internal/admin"
	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/translation"
	httprouter "github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

// New creates a fasthttp server with API routes and UI fallback.
// st backs the management API; pass nil to serve only the OpenAI-compatible
// surface (no admin routes).
func New(uiFS fs.FS, st *store.Store, allowedOrigins []string) *fasthttp.Server {
	infRouter := inference.NewRouter(translation.NewRegistry())

	r := httprouter.New()
	r.NotFound = uiHandler(uiFS)

	// Health check (public, no auth)
	r.GET("/api/health", healthHandler())

	// OpenAI-compatible API routes
	RegisterOpenAIRoutes(r, infRouter)

	// Management API routes and central guard (only when a store is present).
	var guard Middleware = func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return next
	}
	if st != nil {
		sessions := auth.NewSessions(st, sessionTTL)
		flows := map[string]*auth.OAuthFlow{
			"anthropic": auth.NewOAuthFlow(auth.AnthropicOAuth(), st, nil),
			"gemini":    auth.NewOAuthFlow(auth.GeminiOAuth(), st, nil),
			"xai":       auth.NewOAuthFlow(auth.XaiOAuth(), st, nil),
		}
		infRouter.SetKeyResolver(auth.NewCredentialResolver(st, flows))
		RegisterAdminRoutes(r, admin.New(st, sessions, flows))
		guard = (&Guard{
			Sessions:          sessions,
			Settings:          st,
			CLITokenValidator: auth.NewCLITokenValidator(st.DataDir()),
			APIKeyValidator: auth.NewAPIKeyValidator(func(key string) (string, bool, error) {
				rec, err := st.GetAPIKeyByKey(key)
				if err != nil {
					return "", false, err
				}
				return rec.MachineID, rec.IsActive, nil
			}),
		}).Wrap
	}

	handler := Chain(r.Handler,
		CORSMiddleware(allowedOrigins),
		RequestIDMiddleware,
		guard,
	)

	return &fasthttp.Server{
		Handler:            handler,
		ReadTimeout:        0,
		WriteTimeout:       0,
		MaxRequestBodySize: 1 << 30, // 1 GiB
	}
}
