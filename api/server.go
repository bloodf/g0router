package api

import (
	"context"
	"encoding/json"
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
	"github.com/bloodf/g0router/internal/logging"
	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/modelcatalog"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/proxy"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
	"github.com/valyala/fasthttp"
)

type ServerConfig struct {
	Port               int
	Version            string
	EnableRequestLogs  bool
	RequireAPIKey      bool
	APIKeySecret       string
	APIKeyValidator    APIKeyValidator
	InferenceEngine    handlers.InferenceEngine
	Store              *store.Store
	ModelSource        handlers.ManagementModelSource
	OAuthFlows         handlers.OAuthFlows
	UsageStore         handlers.UsageStore
	QuotaFetchers      map[providers.ModelProvider]usage.QuotaFetcher
	QuotaKey           providers.Key
	MCPClientManager   *mcp.ClientManager
	MCPToolManager     *mcp.ToolManager
	MCPInstanceRuntime handlers.MCPInstanceRuntime
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
		captured = &capturingInferenceEngine{
			base: engine,
			onStreamComplete: func(req *providers.ChatRequest, model string, providerUsage *providers.Usage) {
				s.logInferenceUsage(ctx, started, req, nil, nil, model, providerUsage, fasthttp.StatusOK)
			},
		}
		engine = captured
	}
	handle(ctx, engine)
	if captured != nil && captured.streamed {
		return
	}
	var resp *providers.ChatResponse
	var req *providers.ChatRequest
	var dispatchErr error
	if captured != nil {
		resp = captured.response
		req = captured.request
		dispatchErr = captured.err
	}
	s.logInferenceUsage(ctx, started, req, resp, dispatchErr, "", nil, ctx.Response.StatusCode())
}

func (s *Server) logInferenceUsage(ctx *fasthttp.RequestCtx, started time.Time, request *providers.ChatRequest, response *providers.ChatResponse, dispatchErr error, streamModel string, streamUsage *providers.Usage, statusCode int) {
	usageStore, ok := s.config.UsageStore.(logging.RequestStore)
	if !ok || usageStore == nil || !s.requestLogsEnabled() {
		return
	}

	if response == nil && request == nil && streamModel == "" && statusCode < 400 {
		return
	}

	metadata := inferenceLogMetadata(ctx, request, response, streamModel)
	var extractedUsage *usage.Usage
	if response != nil {
		if value, ok := usage.FromChatResponse(*response); ok {
			extractedUsage = &value
		}
	} else if streamUsage != nil {
		if value, ok := usage.FromChatResponse(providers.ChatResponse{Usage: streamUsage}); ok {
			extractedUsage = &value
		}
	}

	var costUSD *float64
	if extractedUsage != nil {
		costUSD = costForUsage(s.config.Store, metadata.provider, metadata.model, extractedUsage)
	}
	entry := logging.RequestLog{
		RequestID:    string(ctx.Response.Header.Peek(requestIDHeader)),
		Timestamp:    started.UTC(),
		Provider:     metadata.provider,
		Model:        metadata.model,
		ConnectionID: metadata.connectionID,
		AuthType:     metadata.authType,
		Usage:        extractedUsage,
		CostUSD:      costUSD,
		Latency:      time.Since(started),
		StatusCode:   statusCode,
		Error:        sanitizedLogError(ctx, dispatchErr, statusCode),
		APIKeyID:     metadata.apiKeyID,
	}
	_ = logging.NewLogger(usageStore).Log(entry)
}

type capturingInferenceEngine struct {
	base             handlers.InferenceEngine
	response         *providers.ChatResponse
	request          *providers.ChatRequest
	err              error
	streamed         bool
	onStreamComplete func(*providers.ChatRequest, string, *providers.Usage)
}

func (e *capturingInferenceEngine) Dispatch(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	e.request = req
	resp, err := e.base.Dispatch(ctx, req)
	e.err = err
	if err == nil {
		e.response = resp
	}
	return resp, err
}

func (e *capturingInferenceEngine) DispatchStream(ctx context.Context, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	e.request = req
	stream, err := e.base.DispatchStream(ctx, req)
	e.err = err
	if err != nil || stream == nil {
		return stream, err
	}
	e.streamed = true
	return e.captureStream(req, stream), nil
}

func (e *capturingInferenceEngine) ListModels(ctx context.Context) ([]providers.Model, error) {
	return e.base.ListModels(ctx)
}

func (e *capturingInferenceEngine) captureStream(req *providers.ChatRequest, stream <-chan providers.StreamChunk) <-chan providers.StreamChunk {
	out := make(chan providers.StreamChunk)
	go func() {
		defer close(out)
		var lastModel string
		var lastUsage *providers.Usage
		for chunk := range stream {
			if chunk.Model != "" {
				lastModel = chunk.Model
			}
			if chunk.Usage != nil {
				usageCopy := *chunk.Usage
				lastUsage = &usageCopy
			}
			out <- chunk
		}
		if e.onStreamComplete != nil {
			e.onStreamComplete(req, lastModel, lastUsage)
		}
	}()
	return out
}

func (s *Server) requestLogsEnabled() bool {
	if s.config.EnableRequestLogs {
		return true
	}
	if s.config.Store == nil {
		return false
	}
	settings, err := s.config.Store.GetSettings()
	if err != nil {
		return false
	}
	return settings.EnableRequestLogs
}

type requestLogMetadata struct {
	provider     string
	model        string
	connectionID *string
	authType     string
	apiKeyID     *string
}

func inferenceLogMetadata(ctx *fasthttp.RequestCtx, request *providers.ChatRequest, response *providers.ChatResponse, streamModel string) requestLogMetadata {
	model := streamModel
	provider := ""
	var connectionID *string
	authType := authTypeForRequest(false)

	if request != nil && model == "" {
		model = request.Model
	}
	if response != nil {
		if response.Model != "" {
			model = response.Model
		}
		if response.Provider != "" {
			provider = response.Provider.String()
		}
		connectionID = stringPtrIfNotEmpty(response.ConnectionID)
		if response.AuthType != "" {
			authType = response.AuthType
		}
	}
	if provider == "" {
		provider = providerFromModel(model)
	}
	if value, ok := ctx.UserValue(requestAuthTypeKey).(string); ok && value != "" {
		authType = value
	} else if response == nil || response.AuthType == "" {
		authType = authTypeForRequest(false)
	}

	return requestLogMetadata{
		provider:     provider,
		model:        model,
		connectionID: connectionID,
		authType:     authType,
		apiKeyID:     userValueStringPtr(ctx, requestAPIKeyIDKey),
	}
}

func sanitizedLogError(ctx *fasthttp.RequestCtx, dispatchErr error, statusCode int) *string {
	if dispatchErr != nil {
		classification := proxy.ClassifyDispatchError(dispatchErr)
		value := classification.Code + ": " + classification.Message
		return &value
	}
	if statusCode < 400 {
		return nil
	}

	var openAIError struct {
		Error struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &openAIError); err == nil && openAIError.Error.Message != "" {
		code := openAIError.Error.Code
		if code == "" {
			code = "request_error"
		}
		value := code + ": " + openAIError.Error.Message
		return &value
	}

	var simpleError struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &simpleError); err == nil && simpleError.Error != "" {
		value := "request_error: " + simpleError.Error
		return &value
	}

	value := fmt.Sprintf("request_error: status %d", statusCode)
	return &value
}

func userValueStringPtr(ctx *fasthttp.RequestCtx, key string) *string {
	value, ok := ctx.UserValue(key).(string)
	if !ok || value == "" {
		return nil
	}
	return &value
}

func costForUsage(overrides usage.PricingOverrideResolver, provider, model string, extractedUsage *usage.Usage) *float64 {
	if provider == "" || provider == "unknown" {
		return nil
	}
	cost, err := usage.CalculateCostUSDWithOverrides(modelcatalog.NewCatalog(), overrides, providers.ModelProvider(provider), model, extractedUsage)
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
		handlers.MCPInstances(ctx, s.config.Store, s.config.MCPInstanceRuntime, "")
	case len(parts) == 4 && parts[0] == "api" && parts[1] == "mcp" && parts[2] == "instances":
		handlers.MCPInstances(ctx, s.config.Store, s.config.MCPInstanceRuntime, parts[3])
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
		handlers.MCPOAuthCallback(ctx, mcp.NewOAuthEngine(s.config.Store, nil), s.config.MCPInstanceRuntime, s.config.Store)
	case len(parts) == 6 && parts[0] == "api" && parts[1] == "mcp" && parts[2] == "instances" && parts[4] == "oauth" && parts[5] == "complete":
		if !requireMethod(ctx, fasthttp.MethodPost) {
			return
		}
		handlers.MCPOAuthComplete(ctx, mcp.NewOAuthEngine(s.config.Store, nil), s.config.MCPInstanceRuntime, s.config.Store, parts[3])
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
