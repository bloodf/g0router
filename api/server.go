package api

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/bloodf/g0router/api/handlers"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
	"github.com/valyala/fasthttp"
)

type ServerConfig struct {
	Port            int
	Version         string
	RequireAPIKey   bool
	APIKeySecret    string
	APIKeyValidator APIKeyValidator
	InferenceEngine handlers.InferenceEngine
	Store           *store.Store
	ModelSource     handlers.ManagementModelSource
	OAuthFlows      handlers.OAuthFlows
	UsageStore      handlers.UsageStore
	QuotaFetchers   map[providers.ModelProvider]usage.QuotaFetcher
	QuotaKey        providers.Key
}

type Server struct {
	config ServerConfig
	server *fasthttp.Server
}

func NewServer(config ServerConfig) *Server {
	srv := &Server{config: config}
	srv.server = &fasthttp.Server{
		Handler: srv.handle,
	}
	return srv
}

func (s *Server) Serve(ln net.Listener) error {
	if err := s.server.Serve(ln); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

func (s *Server) Stop() error {
	if err := s.server.Shutdown(); err != nil {
		return fmt.Errorf("stop server: %w", err)
	}
	return nil
}

func (s *Server) listener() net.Listener {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(s.config.Port))
	if err != nil {
		return nil
	}
	return ln
}

func (s *Server) handle(ctx *fasthttp.RequestCtx) {
	if !s.applyMiddleware(ctx) {
		return
	}

	switch string(ctx.Path()) {
	case "/healthz":
		handlers.Health(ctx, s.config.Version)
	case "/v1/chat/completions":
		if string(ctx.Method()) == fasthttp.MethodPost {
			handlers.Inference(ctx, s.config.InferenceEngine)
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNotFound)
	case "/v1/models":
		if string(ctx.Method()) == fasthttp.MethodGet {
			handlers.Models(ctx, s.config.InferenceEngine)
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNotFound)
	default:
		s.handleAPI(ctx)
	}
}

func (s *Server) handleAPI(ctx *fasthttp.RequestCtx) {
	path := strings.TrimRight(string(ctx.Path()), "/")
	parts := pathParts(path)

	switch {
	case path == "/api/providers":
		handlers.Providers(ctx, s.config.ModelSource, "")
	case len(parts) == 3 && parts[0] == "api" && parts[1] == "providers":
		handlers.Providers(ctx, s.config.ModelSource, parts[2])
	case path == "/api/connections":
		handlers.Connections(ctx, s.config.Store, "")
	case len(parts) == 3 && parts[0] == "api" && parts[1] == "connections":
		handlers.Connections(ctx, s.config.Store, parts[2])
	case path == "/api/settings":
		handlers.Settings(ctx, s.config.Store)
	case path == "/api/keys":
		handlers.APIKeys(ctx, s.config.Store, s.config.APIKeySecret, "")
	case len(parts) == 3 && parts[0] == "api" && parts[1] == "keys":
		handlers.APIKeys(ctx, s.config.Store, s.config.APIKeySecret, parts[2])
	case path == "/api/combos":
		handlers.Combos(ctx, s.config.Store, "")
	case len(parts) == 3 && parts[0] == "api" && parts[1] == "combos":
		handlers.Combos(ctx, s.config.Store, parts[2])
	case path == "/api/oauth/callback":
		handlers.OAuthCallback(ctx, s.config.OAuthFlows)
	case len(parts) == 4 && parts[0] == "api" && parts[1] == "oauth" && parts[3] == "start":
		handlers.OAuthStart(ctx, s.config.OAuthFlows)
	case len(parts) == 4 && parts[0] == "api" && parts[1] == "oauth" && parts[3] == "poll":
		handlers.OAuthPoll(ctx, s.config.OAuthFlows)
	case len(parts) == 4 && parts[0] == "api" && parts[1] == "oauth" && parts[3] == "exchange":
		handlers.OAuthExchange(ctx, s.config.OAuthFlows)
	case path == "/api/usage":
		handlers.Usage(ctx, s.config.UsageStore)
	case path == "/api/usage/summary":
		handlers.UsageSummary(ctx, s.config.UsageStore)
	case strings.HasPrefix(path, "/api/usage/quota/"):
		handlers.UsageQuota(ctx, s.config.QuotaFetchers, s.config.QuotaKey)
	case path == "/api/logs":
		handlers.Logs(ctx, s.config.UsageStore)
	default:
		ctx.SetStatusCode(fasthttp.StatusNotFound)
	}
}

func pathParts(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}
