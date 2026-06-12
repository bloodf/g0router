package server

import (
	"io/fs"
	"time"

	"github.com/bloodf/g0router/internal/admin"
	"github.com/bloodf/g0router/internal/api"
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

	// Credential refresher and combo dispatcher for OAuth/combo paths
	// (only when a store is present).
	var refresher api.CredentialRefresher
	var comboDisp api.ComboDispatcher
	var flows map[string]*auth.OAuthFlow
	if st != nil {
		flows = map[string]*auth.OAuthFlow{
			"anthropic": auth.NewOAuthFlow(auth.AnthropicOAuth(), st, nil),
			"gemini":    auth.NewOAuthFlow(auth.GeminiOAuth(), st, nil),
			"xai":       auth.NewOAuthFlow(auth.XaiOAuth(), st, nil),
		}
		resolver := auth.NewCredentialResolver(st, flows)
		refresher = resolver
		infRouter.SetKeyResolver(resolver)
		infRouter.SetAliasStore(st)

		cd := inference.NewCooldownEngine(st, time.Now)
		sel := inference.NewSelectionEngine(st, st, cd, time.Now)
		runner := inference.NewAccountRunner(sel)
		comboEngine := inference.NewComboEngine(st, st, runner, time.Now, time.Sleep)
		comboDisp = newComboDispatcher(st, comboEngine)
	}

	// OpenAI-compatible API routes
	RegisterOpenAIRoutes(r, infRouter, st, refresher, comboDisp)

	// Management API routes and central guard (only when a store is present).
	var guard Middleware = func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return next
	}
	if st != nil {
		sessions := auth.NewSessions(st, sessionTTL)
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

// comboDispatcher adapts the inference.ComboEngine to the api.ComboDispatcher
// interface so the api layer stays store-free. It surfaces the last model error
// when all combo models fail, matching combo.js behavior.
type comboDispatcher struct {
	cs inference.ComboStore
	ce *inference.ComboEngine
}

func newComboDispatcher(cs inference.ComboStore, ce *inference.ComboEngine) *comboDispatcher {
	return &comboDispatcher{cs: cs, ce: ce}
}

func (d *comboDispatcher) IsCombo(name string) bool {
	_, err := d.cs.GetCombo(name)
	return err == nil
}

func (d *comboDispatcher) ExecuteCombo(name string, fn func(model, connID, credential string) (inference.Verdict, error)) error {
	var lastErr error
	err := d.ce.ExecuteCombo(name, func(model string, conn *store.Connection) (inference.Verdict, error) {
		credential := conn.AccessToken
		if credential == "" {
			credential = conn.Secret
		}
		verdict, fnErr := fn(model, conn.ID, credential)
		if fnErr != nil {
			lastErr = fnErr
		}
		return verdict, fnErr
	})
	if err != nil {
		if lastErr != nil {
			return lastErr
		}
		return err
	}
	return nil
}
