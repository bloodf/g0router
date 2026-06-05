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
	"github.com/bloodf/g0router/internal/rtk"
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
	s.handleLoggedInference(ctx, "openai", handlers.Inference)
}

func (s *Server) handleMessages(ctx *fasthttp.RequestCtx) {
	s.handleLoggedInference(ctx, "anthropic", handlers.Messages)
}

func (s *Server) handleResponses(ctx *fasthttp.RequestCtx) {
	s.handleLoggedInference(ctx, "responses", handlers.Responses)
}

func (s *Server) handleLoggedInference(ctx *fasthttp.RequestCtx, sourceFormat string, handle func(*fasthttp.RequestCtx, handlers.InferenceEngine)) {
	started := time.Now()
	engine := s.config.InferenceEngine
	var captured *capturingInferenceEngine
	if engine != nil {
		engine = preprocessingInferenceEngine{
			base:     engine,
			settings: s.runtimeSettings,
			tools:    s.config.MCPToolManager,
		}
		// Snapshot request-scoped data from the pooled ctx on the request
		// goroutine. The streaming-complete callback fires from the capture
		// goroutine concurrently with fasthttp's body-stream writer and ctx
		// recycling, so it must never read ctx itself.
		snapshot := newStreamLogSnapshot(ctx)
		captured = &capturingInferenceEngine{
			base: engine,
			onStreamComplete: func(req *providers.ChatRequest, model string, providerUsage *providers.Usage) {
				s.logStreamingInferenceUsage(snapshot, started, sourceFormat, req, model, providerUsage)
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
	s.logInferenceUsage(ctx, started, sourceFormat, req, resp, dispatchErr, "", nil, ctx.Response.StatusCode())
}

func (s *Server) logInferenceUsage(ctx *fasthttp.RequestCtx, started time.Time, sourceFormat string, request *providers.ChatRequest, response *providers.ChatResponse, dispatchErr error, streamModel string, streamUsage *providers.Usage, statusCode int) {
	usageStore, ok := s.config.UsageStore.(logging.RequestStore)
	if !ok || usageStore == nil || !s.requestLogsEnabled() {
		return
	}

	if response == nil && request == nil && streamModel == "" && statusCode < 400 {
		return
	}

	metadata := inferenceLogMetadata(ctx, request, response, streamModel)
	s.writeInferenceLog(usageStore, metadata, string(ctx.Response.Header.Peek(requestIDHeader)), started, sourceFormat, request, response, streamUsage, sanitizedLogError(ctx, dispatchErr, statusCode), statusCode)
}

// logStreamingInferenceUsage logs a completed streaming inference using values
// captured before streaming began. It deliberately takes no *fasthttp.RequestCtx:
// it runs from the capture goroutine after the handler returned, so touching the
// pooled ctx would race with fasthttp's body-stream writer and ctx recycling.
func (s *Server) logStreamingInferenceUsage(snapshot streamLogSnapshot, started time.Time, sourceFormat string, request *providers.ChatRequest, streamModel string, streamUsage *providers.Usage) {
	usageStore, ok := s.config.UsageStore.(logging.RequestStore)
	if !ok || usageStore == nil || !s.requestLogsEnabled() {
		return
	}
	if request == nil && streamModel == "" {
		return
	}
	metadata := inferenceLogMetadataWithAuth(request, nil, streamModel, snapshot.authType, snapshot.authTypeSet, snapshot.apiKeyID)
	s.writeInferenceLog(usageStore, metadata, snapshot.requestID, started, sourceFormat, request, nil, streamUsage, nil, fasthttp.StatusOK)
}

func (s *Server) writeInferenceLog(usageStore logging.RequestStore, metadata requestLogMetadata, requestID string, started time.Time, sourceFormat string, request *providers.ChatRequest, response *providers.ChatResponse, streamUsage *providers.Usage, logError *string, statusCode int) {
	settings := s.runtimeSettings()
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
		RequestID:      requestID,
		Timestamp:      started.UTC(),
		Provider:       metadata.provider,
		Model:          metadata.model,
		ConnectionID:   metadata.connectionID,
		AuthType:       metadata.authType,
		Usage:          extractedUsage,
		CostUSD:        costUSD,
		Latency:        time.Since(started),
		StatusCode:     statusCode,
		Error:          logError,
		APIKeyID:       metadata.apiKeyID,
		SourceFormat:   stringPtrIfNotEmpty(sourceFormat),
		TargetFormat:   stringPtrIfNotEmpty(metadata.provider),
		RTKEnabled:     boolPtr(settings.RTKEnabled),
		CavemanEnabled: boolPtr(settings.CavemanEnabled),
	}
	_ = logging.NewLogger(usageStore).Log(entry)
}

type preprocessingInferenceEngine struct {
	base     handlers.InferenceEngine
	settings func() store.Settings
	tools    *mcp.ToolManager
}

func (e preprocessingInferenceEngine) Dispatch(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	return e.base.Dispatch(ctx, e.preprocess(ctx, req))
}

func (e preprocessingInferenceEngine) DispatchStream(ctx context.Context, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return e.base.DispatchStream(ctx, e.preprocess(ctx, req))
}

func (e preprocessingInferenceEngine) ListModels(ctx context.Context) ([]providers.Model, error) {
	return e.base.ListModels(ctx)
}

func (e preprocessingInferenceEngine) preprocess(ctx context.Context, req *providers.ChatRequest) *providers.ChatRequest {
	if req == nil {
		return nil
	}
	settings := e.settings()
	processed := *req
	if settings.RTKEnabled {
		processed = rtk.CompressRequest(processed)
	}
	if settings.CavemanEnabled {
		processed = rtk.InjectCaveman(processed, rtk.CavemanLevel(settings.CavemanLevel))
	}
	if len(processed.Tools) == 0 && e.tools != nil {
		processed.Tools = e.tools.CompactToolsForRequest(ctx)
	}
	return &processed
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

func (s *Server) runtimeSettings() store.Settings {
	if s.config.Store != nil {
		settings, err := s.config.Store.GetSettings()
		if err == nil {
			return settings
		}
	}
	return store.Settings{
		RTKEnabled:     true,
		CavemanEnabled: false,
		CavemanLevel:   "full",
	}
}

type requestLogMetadata struct {
	provider     string
	model        string
	connectionID *string
	authType     string
	apiKeyID     *string
}

// streamLogSnapshot captures the request-scoped values needed to log a
// streaming inference after the handler returns. It is taken on the request
// goroutine so the streaming-complete callback never reads the pooled
// *fasthttp.RequestCtx (which races with the body-stream writer and is recycled
// once the handler returns).
type streamLogSnapshot struct {
	requestID   string
	authType    string
	apiKeyID    *string
	authTypeSet bool
}

func newStreamLogSnapshot(ctx *fasthttp.RequestCtx) streamLogSnapshot {
	snapshot := streamLogSnapshot{
		requestID: string(ctx.Response.Header.Peek(requestIDHeader)),
		apiKeyID:  userValueStringPtr(ctx, requestAPIKeyIDKey),
	}
	if value, ok := ctx.UserValue(requestAuthTypeKey).(string); ok && value != "" {
		snapshot.authType = value
		snapshot.authTypeSet = true
	}
	return snapshot
}

func inferenceLogMetadata(ctx *fasthttp.RequestCtx, request *providers.ChatRequest, response *providers.ChatResponse, streamModel string) requestLogMetadata {
	authType := authTypeForRequest(false)
	authTypeSet := false
	if value, ok := ctx.UserValue(requestAuthTypeKey).(string); ok && value != "" {
		authType = value
		authTypeSet = true
	}
	return inferenceLogMetadataWithAuth(request, response, streamModel, authType, authTypeSet, userValueStringPtr(ctx, requestAPIKeyIDKey))
}

func inferenceLogMetadataWithAuth(request *providers.ChatRequest, response *providers.ChatResponse, streamModel, snapshotAuthType string, snapshotAuthTypeSet bool, apiKeyID *string) requestLogMetadata {
	model := streamModel
	requestModel := ""
	provider := ""
	var connectionID *string
	authType := authTypeForRequest(false)

	if request != nil {
		requestModel = request.Model
		if model == "" {
			model = request.Model
		}
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
	if requestModel != "" && provider != "" {
		if route, ok := modelcatalog.NewCatalog().RouteForModel(requestModel); ok && route.Provider.String() == provider {
			model = requestModel
		}
	}
	if snapshotAuthTypeSet {
		authType = snapshotAuthType
	} else if response == nil || response.AuthType == "" {
		authType = authTypeForRequest(false)
	}

	return requestLogMetadata{
		provider:     provider,
		model:        model,
		connectionID: connectionID,
		authType:     authType,
		apiKeyID:     apiKeyID,
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

func boolPtr(value bool) *bool {
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
	case len(parts) == 4 && parts[0] == "api" && parts[1] == "connections" && parts[3] == "test":
		handlers.ConnectionTest(ctx, s.config.Store, parts[2])
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
	case path == "/api/aliases":
		handlers.Aliases(ctx, s.config.Store, "")
	case len(parts) == 3 && parts[0] == "api" && parts[1] == "aliases":
		handlers.Aliases(ctx, s.config.Store, parts[2])
	case path == "/api/pricing":
		handlers.Pricing(ctx, s.config.Store, "", "")
	case len(parts) == 4 && parts[0] == "api" && parts[1] == "pricing":
		handlers.Pricing(ctx, s.config.Store, parts[2], parts[3])
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
		handlers.UsageQuota(ctx, s.config.Store, s.config.QuotaFetchers, s.config.QuotaKey)
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
