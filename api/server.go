package api

import (
	"context"
	"fmt"
	"io/fs"
	"mime"
	"net"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/bloodf/g0router"
	"github.com/bloodf/g0router/api/handlers"
	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/modelcatalog"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
	"github.com/valyala/fasthttp"
)

type ServerConfig struct {
	Port             int
	Version          string
	RequireAPIKey    bool
	APIKeySecret     string
	APIKeyValidator  APIKeyValidator
	InferenceEngine  handlers.InferenceEngine
	Store            *store.Store
	ModelSource      handlers.ManagementModelSource
	OAuthFlows       handlers.OAuthFlows
	UsageStore       handlers.UsageStore
	QuotaFetchers    map[providers.ModelProvider]usage.QuotaFetcher
	QuotaKey         providers.Key
	MCPClientManager *mcp.ClientManager
	MCPToolManager   *mcp.ToolManager
}

type Server struct {
	config ServerConfig
	server *fasthttp.Server
	uiFS   fs.FS
	uiErr  error
}

func NewServer(config ServerConfig) *Server {
	uiFS, err := g0router.UI()
	srv := &Server{config: config, uiFS: uiFS, uiErr: err}
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
			s.handleInference(ctx)
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNotFound)
	case "/v1/messages":
		if string(ctx.Method()) == fasthttp.MethodPost {
			s.handleMessages(ctx)
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNotFound)
	case "/v1/responses":
		if string(ctx.Method()) == fasthttp.MethodPost {
			s.handleResponses(ctx)
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
		if strings.HasPrefix(string(ctx.Path()), "/v1/") {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			return
		}
		s.handleAPI(ctx)
	}
}

func (s *Server) handleInference(ctx *fasthttp.RequestCtx) {
	s.handleLoggedInference(ctx, handlers.Inference)
}

func (s *Server) handleMessages(ctx *fasthttp.RequestCtx) {
	s.handleLoggedInference(ctx, handlers.Messages)
}

func (s *Server) handleResponses(ctx *fasthttp.RequestCtx) {
	s.handleLoggedInference(ctx, handlers.Responses)
}

func (s *Server) handleLoggedInference(ctx *fasthttp.RequestCtx, handle func(*fasthttp.RequestCtx, handlers.InferenceEngine)) {
	started := time.Now()
	engine := s.config.InferenceEngine
	var captured *capturingInferenceEngine
	if engine != nil {
		captured = &capturingInferenceEngine{base: engine}
		engine = captured
	}
	handle(ctx, engine)
	var resp *providers.ChatResponse
	if captured != nil {
		resp = captured.response
	}
	s.logInferenceUsage(ctx, started, resp)
}

func (s *Server) logInferenceUsage(ctx *fasthttp.RequestCtx, started time.Time, response *providers.ChatResponse) {
	usageStore, ok := s.config.UsageStore.(requestLogStore)
	if !ok || usageStore == nil || response == nil || ctx.Response.StatusCode() != fasthttp.StatusOK || !s.requestLogsEnabled() {
		return
	}

	extractedUsage, ok := usage.FromChatResponse(*response)
	if !ok {
		return
	}

	provider := response.Provider.String()
	if provider == "" {
		provider = providerFromModel(response.Model)
	}
	authType := response.AuthType
	if authType == "" {
		authType = authTypeForRequest(s.config.RequireAPIKey)
	}
	costUSD := costForUsage(provider, response.Model, &extractedUsage)

	entry := store.RequestLogEntry{
		RequestID:    string(ctx.Response.Header.Peek("X-Request-ID")),
		Timestamp:    started.UTC(),
		Provider:     provider,
		Model:        response.Model,
		ConnectionID: stringPtrIfNotEmpty(response.ConnectionID),
		AuthType:     authType,
		InputTokens:  intPtr(extractedUsage.InputTokens),
		OutputTokens: intPtr(extractedUsage.OutputTokens),
		TotalTokens:  intPtr(extractedUsage.TotalTokens),
		LatencyMS:    intPtr(int(time.Since(started) / time.Millisecond)),
		StatusCode:   intPtr(ctx.Response.StatusCode()),
		CostUSD:      costUSD,
	}
	if extractedUsage.CacheReadTokens > 0 {
		entry.CacheReadTokens = intPtr(extractedUsage.CacheReadTokens)
	}
	_ = usageStore.LogRequest(&entry)
}

type requestLogStore interface {
	LogRequest(entry *store.RequestLogEntry) error
}

type capturingInferenceEngine struct {
	base     handlers.InferenceEngine
	response *providers.ChatResponse
}

func (e *capturingInferenceEngine) Dispatch(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	resp, err := e.base.Dispatch(ctx, req)
	if err == nil {
		e.response = resp
	}
	return resp, err
}

func (e *capturingInferenceEngine) DispatchStream(ctx context.Context, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return e.base.DispatchStream(ctx, req)
}

func (e *capturingInferenceEngine) ListModels(ctx context.Context) ([]providers.Model, error) {
	return e.base.ListModels(ctx)
}

func (s *Server) requestLogsEnabled() bool {
	if s.config.Store == nil {
		return false
	}
	settings, err := s.config.Store.GetSettings()
	if err != nil {
		return false
	}
	return settings.EnableRequestLogs
}

func costForUsage(provider, model string, extractedUsage *usage.Usage) *float64 {
	if provider == "" || provider == "unknown" {
		return nil
	}
	cost, err := usage.CalculateCostUSD(modelcatalog.NewCatalog(), providers.ModelProvider(provider), model, extractedUsage)
	if err != nil {
		return nil
	}
	return &cost
}

func providerFromModel(model string) string {
	if provider, ok := modelcatalog.NewCatalog().ProviderForModel(model); ok {
		return provider.String()
	}
	switch {
	case strings.HasPrefix(model, "gpt-"):
		return providers.ProviderOpenAI.String()
	case strings.HasPrefix(model, "claude-"):
		return providers.ProviderAnthropic.String()
	default:
		return "unknown"
	}
}

func authTypeForRequest(requireAPIKey bool) string {
	if requireAPIKey {
		return "api_key"
	}
	return "noauth"
}

func intPtr(value int) *int {
	return &value
}

func stringPtrIfNotEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func (s *Server) handleAPI(ctx *fasthttp.RequestCtx) {
	path := strings.TrimRight(string(ctx.Path()), "/")
	parts := pathParts(path)

	switch {
	case path == "/api/providers":
		handlers.Providers(ctx, s.config.ModelSource, "")
	case len(parts) == 4 && parts[0] == "api" && parts[1] == "providers" && parts[3] == "models":
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
		if !requireMethod(ctx, fasthttp.MethodGet) {
			return
		}
		handlers.OAuthCallback(ctx, s.config.Store, s.config.OAuthFlows)
	case len(parts) == 4 && parts[0] == "api" && parts[1] == "oauth" && parts[3] == "authorize":
		if !requireMethod(ctx, fasthttp.MethodPost) {
			return
		}
		handlers.OAuthStart(ctx, s.config.Store, s.config.OAuthFlows)
	case len(parts) == 4 && parts[0] == "api" && parts[1] == "oauth" && parts[3] == "poll":
		if !requireMethod(ctx, fasthttp.MethodGet) {
			return
		}
		handlers.OAuthPoll(ctx, s.config.Store, s.config.OAuthFlows)
	case len(parts) == 4 && parts[0] == "api" && parts[1] == "oauth" && parts[3] == "exchange":
		if !requireMethod(ctx, fasthttp.MethodPost) {
			return
		}
		handlers.OAuthExchange(ctx, s.config.Store, s.config.OAuthFlows)
	case path == "/api/usage":
		handlers.Usage(ctx, s.config.UsageStore)
	case path == "/api/usage/summary":
		handlers.UsageSummary(ctx, s.config.UsageStore)
	case strings.HasPrefix(path, "/api/usage/quota/"):
		handlers.UsageQuota(ctx, s.config.QuotaFetchers, s.config.QuotaKey)
	case path == "/api/logs":
		handlers.Logs(ctx, s.config.UsageStore)
	case path == "/api/mcp/clients":
		handlers.MCPClients(ctx, s.config.Store, s.config.MCPClientManager, s.config.MCPToolManager, "")
	case len(parts) == 4 && parts[0] == "api" && parts[1] == "mcp" && parts[2] == "clients":
		handlers.MCPClients(ctx, s.config.Store, s.config.MCPClientManager, s.config.MCPToolManager, parts[3])
	case path == "/api/mcp/instances":
		handlers.MCPInstances(ctx, s.config.Store, "")
	case len(parts) == 4 && parts[0] == "api" && parts[1] == "mcp" && parts[2] == "instances":
		handlers.MCPInstances(ctx, s.config.Store, parts[3])
	case len(parts) == 6 && parts[0] == "api" && parts[1] == "mcp" && parts[2] == "instances" && parts[4] == "auth" && parts[5] == "start":
		if !requireMethod(ctx, fasthttp.MethodPost) {
			return
		}
		handlers.MCPOAuthStart(ctx, s.config.Store, parts[3])
	case len(parts) == 5 && parts[0] == "api" && parts[1] == "mcp" && parts[2] == "instances" && parts[4] == "accounts":
		if !requireMethod(ctx, fasthttp.MethodGet) {
			return
		}
		handlers.MCPOAuthAccounts(ctx, s.config.Store, parts[3])
	case path == "/api/mcp/tools":
		handlers.MCPTools(ctx, s.config.Store, s.config.MCPToolManager, "")
	case len(parts) == 5 && parts[0] == "api" && parts[1] == "mcp" && parts[2] == "tools" && parts[4] == "execute":
		handlers.MCPTools(ctx, s.config.Store, s.config.MCPToolManager, parts[3])
	case path == "/api/mcp/oauth/callback":
		if !requireMethod(ctx, fasthttp.MethodGet) {
			return
		}
		handlers.MCPOAuthCallback(ctx, mcp.NewOAuthEngine(s.config.Store, nil))
	case len(parts) == 6 && parts[0] == "api" && parts[1] == "mcp" && parts[2] == "instances" && parts[4] == "oauth" && parts[5] == "complete":
		if !requireMethod(ctx, fasthttp.MethodPost) {
			return
		}
		handlers.MCPOAuthComplete(ctx, mcp.NewOAuthEngine(s.config.Store, nil), parts[3])
	default:
		if strings.HasPrefix(path, "/api/") || path == "/api" {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			return
		}
		s.handleUI(ctx)
	}
}

func (s *Server) handleUI(ctx *fasthttp.RequestCtx) {
	if s.uiErr != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		_, _ = ctx.WriteString(s.uiErr.Error())
		return
	}
	if string(ctx.Method()) != fasthttp.MethodGet && string(ctx.Method()) != fasthttp.MethodHead {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}

	filePath := cleanUIPath(string(ctx.Path()))
	body, err := fs.ReadFile(s.uiFS, filePath)
	if err != nil {
		if strings.HasPrefix(filePath, "assets/") {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			return
		}
		body, err = fs.ReadFile(s.uiFS, "index.html")
		filePath = "index.html"
	}
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		_, _ = ctx.WriteString(fmt.Errorf("serve ui: %w", err).Error())
		return
	}

	if contentType := mime.TypeByExtension(path.Ext(filePath)); contentType != "" {
		ctx.SetContentType(contentType)
	}
	if string(ctx.Method()) == fasthttp.MethodHead {
		return
	}
	_, _ = ctx.Write(body)
}

func cleanUIPath(requestPath string) string {
	cleaned := strings.TrimPrefix(path.Clean("/"+requestPath), "/")
	if cleaned == "" || cleaned == "." {
		return "index.html"
	}
	return cleaned
}

func pathParts(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func requireMethod(ctx *fasthttp.RequestCtx, method string) bool {
	if string(ctx.Method()) == method {
		return true
	}
	ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	return false
}
