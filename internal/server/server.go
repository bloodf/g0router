package server

import (
	"io/fs"
	"time"

	"github.com/bloodf/g0router/internal/admin"
	"github.com/bloodf/g0router/internal/api"
	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/platform"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/bloodf/g0router/internal/usage"
	httprouter "github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

// New creates a fasthttp server with API routes and UI fallback.
// st backs the management API; pass nil to serve only the OpenAI-compatible
// surface (no admin routes).
func New(uiFS fs.FS, st *store.Store, allowedOrigins []string) *fasthttp.Server {
	return NewWithShutdown(uiFS, st, allowedOrigins).Server
}

// NewWithShutdown constructs a fasthttp server plus the shutdown hook for
// the request_details writer. Production callers should use this so the
// observability buffer is flushed on graceful shutdown (PAR-USAGE-026).
func NewWithShutdown(uiFS fs.FS, st *store.Store, allowedOrigins []string) *Server {
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
	var recorder api.UsageRecorder
	var tracker api.PendingTracker
	var detail api.DetailCapture
	var usageDeps admin.UsageDeps
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
		// Provider-node prefix-routing override (w7-platnodes, PAR-ROUTE-009/040):
		// a model "prefix/bare" whose prefix matches a registered node routes to
		// that node's provider + base URL before static alias/catalog resolution.
		infRouter.SetNodeResolver(platform.NewProviderNodeService(st))

		cd := inference.NewCooldownEngine(st, time.Now)
		sel := inference.NewSelectionEngine(st, st, cd, time.Now)
		runner := inference.NewAccountRunner(sel)
		comboEngine := inference.NewComboEngine(st, st, runner, time.Now, time.Sleep)
		comboDisp = newComboDispatcher(st, comboEngine)

		// Usage glue (w5-b/c adapters bridged into the api layer's
		// consumer interfaces — the api package must not import
		// internal/usage directly per AGENTS.md layered DDD).
		events := usage.NewEvents()
		clock := time.Now
		timerFactory := func(d time.Duration, fn func()) func() {
			t := time.AfterFunc(d, fn)
			return func() { t.Stop() }
		}
		usageTracker := usage.NewTracker(clock, timerFactory, events)
		usageRing := usage.NewRing(50)
		usageDeps = admin.UsageDeps{
			Events:  events,
			Tracker: usageTracker,
			Ring:    usageRing,
		}
		recorder = newUsageRecorderAdapter(usage.NewRecorder(
			usage.NewResolver(st, func() int64 { return clock().UnixNano() }),
			st,
			clock,
			events,
		))
		tracker = newPendingTrackerAdapter(usageTracker)
		detail = newDetailCaptureAdapter(usage.NewDetailWriter(
			st,
			usage.NewObsConfigLoader(st, func(string) string { return "" }, clock),
			clock,
			nil,
			nil,
		))
	}

	// OpenAI-compatible API routes
	RegisterOpenAIRoutes(r, infRouter, st, refresher, comboDisp, recorder, tracker, detail)

	// Management API routes and central guard (only when a store is present).
	var guard Middleware = func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return next
	}
	var adminHandlers *admin.Handlers
	if st != nil {
		sessions := auth.NewSessions(st, sessionTTL)
		adminHandlers = NewAdminHandlers(st, usageDeps)
		RegisterAdminRoutes(r, adminHandlers)
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

	return &Server{
		Server: &fasthttp.Server{
			Handler:            handler,
			ReadTimeout:        0,
			WriteTimeout:       0,
			MaxRequestBodySize: 1 << 30, // 1 GiB
		},
		detail: detail,
		admin:  adminHandlers,
	}
}

// Server wraps a fasthttp.Server with a Close() that also flushes the
// request_details buffer (PAR-USAGE-026). The fasthttp.Server itself does
// not expose this hook, so we carry the writer on the wrapper.
type Server struct {
	*fasthttp.Server
	detail api.DetailCapture
	admin  *admin.Handlers
}

// SetVersionInfo forwards the binary's version/build date to the admin handlers
// so GET /api/version can report them (PAR-UI-102). No-op when no store/admin
// surface is present.
func (s *Server) SetVersionInfo(version, buildDate string) {
	if s == nil || s.admin == nil {
		return
	}
	s.admin.SetVersionInfo(version, buildDate)
}

// SetShutdownFunc forwards the graceful-shutdown hook to the admin handlers so
// POST /api/version/shutdown can trigger it (PAR-UI-103). No-op when no
// store/admin surface is present.
func (s *Server) SetShutdownFunc(fn func()) {
	if s == nil || s.admin == nil {
		return
	}
	s.admin.SetShutdownFunc(fn)
}

// Close shuts down the underlying fasthttp server and flushes the
// observability buffer. The detail.Close() failure is returned to the
// caller; the fasthttp shutdown error is reported when the detail flush
// succeeds first (so a noisy writer doesn't mask a shutdown error).
func (s *Server) Close() error {
	if s == nil {
		return nil
	}
	if s.detail != nil {
		if err := s.detail.Close(); err != nil {
			return err
		}
	}
	if s.Server != nil {
		return s.Server.Shutdown()
	}
	return nil
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
