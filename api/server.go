package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bloodf/g0router"
	"github.com/bloodf/g0router/api/handlers"
	"github.com/bloodf/g0router/internal/logging"
	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/metrics"
	"github.com/bloodf/g0router/internal/modelcatalog"
	"github.com/bloodf/g0router/internal/notify"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/proxy"
	"github.com/bloodf/g0router/internal/ratelimit"
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

// logRetentionInterval is how often the background cleanup job runs.
const logRetentionInterval = time.Hour

// connectionRefreshInterval is how often the proactive OAuth refresh job runs.
const connectionRefreshInterval = time.Minute

// ConnectionRefresher proactively refreshes OAuth connections whose tokens are
// near expiry. It is satisfied by *proxy.Engine.
type ConnectionRefresher interface {
	RefreshExpiringConnections(ctx context.Context, now time.Time) []proxy.RefreshOutcome
}

type Server struct {
	config ServerConfig
	server *fasthttp.Server
	uiFS   fs.FS
	uiErr  error

	settingsMu    sync.RWMutex
	settingsCache *store.Settings

	limiter *ratelimit.Limiter

	metrics *metrics.Collector

	logRetentionInterval time.Duration
	// runRetention performs a single retention pass. It is a field so tests can
	// inject a panicking pass and assert the loop survives; it defaults to
	// runLogRetentionOnce.
	runRetention func(time.Time)

	connectionRefreshInterval time.Duration
	// connRefresher performs proactive OAuth refreshes. Nil when no engine is
	// available (the refresh job becomes a no-op).
	connRefresher ConnectionRefresher
	// notifierFor builds a Notifier from a webhook URL. A field so tests can
	// inject a capturing notifier; defaults to notify.NewNotifier.
	notifierFor func(url string) notify.Notifier
	// runConnectionRefresh performs a single refresh pass. A field so tests can
	// inject a panicking pass; defaults to runConnectionRefreshOnce.
	runConnectionRefresh func(time.Time)
	// notifiedStale tracks connection IDs already notified about for the current
	// stale episode, so we notify once per episode rather than every tick.
	notifiedMu    sync.Mutex
	notifiedStale map[string]bool
}

// UpdateSettings writes settings to the store and invalidates the cache.
func (s *Server) UpdateSettings(settings store.Settings) error {
	if s.config.Store == nil {
		return fmt.Errorf("store unavailable")
	}
	if err := s.config.Store.UpdateSettings(settings); err != nil {
		return err
	}
	s.settingsMu.Lock()
	s.settingsCache = &settings
	s.settingsMu.Unlock()
	return nil
}

func NewServer(config ServerConfig) *Server {
	uiFS, err := g0router.UI()
	srv := &Server{
		config:                    config,
		uiFS:                      uiFS,
		uiErr:                     err,
		limiter:                   ratelimit.NewLimiter(),
		metrics:                   metrics.NewCollector(),
		logRetentionInterval:      logRetentionInterval,
		connectionRefreshInterval: connectionRefreshInterval,
		notifiedStale:             make(map[string]bool),
	}
	srv.runRetention = srv.runLogRetentionOnce
	srv.runConnectionRefresh = srv.runConnectionRefreshOnce
	srv.notifierFor = func(url string) notify.Notifier { return notify.NewNotifier(url, nil) }
	if refresher, ok := config.InferenceEngine.(ConnectionRefresher); ok {
		srv.connRefresher = refresher
	}
	srv.server = &fasthttp.Server{
		Handler: srv.handle,
	}
	return srv
}

// StartLogRetention launches the background request-log cleanup job. It runs
// once immediately, then on every logRetentionInterval tick, and stops when ctx
// is cancelled. Safe to call once during server startup.
func (s *Server) StartLogRetention(ctx context.Context) {
	if s.config.Store == nil {
		return
	}
	interval := s.logRetentionInterval
	if interval <= 0 {
		interval = logRetentionInterval
	}
	run := s.runRetention
	if run == nil {
		run = s.runLogRetentionOnce
	}
	go func() {
		s.runRetentionGuarded(run, time.Now().UTC())
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runRetentionGuarded(run, time.Now().UTC())
			}
		}
	}()
}

// runRetentionGuarded runs one retention pass and recovers from any panic so a
// single failed cycle cannot kill the background loop.
func (s *Server) runRetentionGuarded(run func(time.Time), now time.Time) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("log retention cleanup panic recovered: %v", r)
		}
	}()
	run(now)
}

// runLogRetentionOnce performs a single retention pass: it reads the current
// retention setting and, when it is positive, deletes logs older than
// now-retention. A retention of 0 keeps logs forever.
func (s *Server) runLogRetentionOnce(now time.Time) {
	if s.config.Store == nil {
		return
	}
	retentionDays := s.runtimeSettings().LogRetentionDays
	if retentionDays <= 0 {
		return
	}
	cutoff := now.Add(-time.Duration(retentionDays) * 24 * time.Hour)
	deleted, err := s.config.Store.DeleteRequestLogsOlderThan(cutoff)
	if err != nil {
		log.Printf("log retention cleanup: %v", err)
		return
	}
	if deleted > 0 {
		log.Printf("log retention: deleted %d request log(s) older than %s", deleted, cutoff.Format(time.RFC3339))
	}
}

// StartConnectionRefresh launches the background proactive OAuth refresh job. It
// runs once immediately, then on every connectionRefreshInterval tick, and stops
// when ctx is cancelled. Safe to call once during server startup. It is a no-op
// when no refresher is configured.
func (s *Server) StartConnectionRefresh(ctx context.Context) {
	if s.connRefresher == nil {
		return
	}
	interval := s.connectionRefreshInterval
	if interval <= 0 {
		interval = connectionRefreshInterval
	}
	run := s.runConnectionRefresh
	if run == nil {
		run = s.runConnectionRefreshOnce
	}
	go func() {
		s.runConnectionRefreshGuarded(run, time.Now().UTC())
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runConnectionRefreshGuarded(run, time.Now().UTC())
			}
		}
	}()
}

// runConnectionRefreshGuarded runs one refresh pass and recovers from any panic
// so a single failed cycle cannot kill the background loop.
func (s *Server) runConnectionRefreshGuarded(run func(time.Time), now time.Time) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("connection refresh panic recovered: %v", r)
		}
	}()
	run(now)
}

// runConnectionRefreshOnce performs a single proactive refresh pass: it refreshes
// expiring OAuth connections and, for each that newly went stale (failed to
// refresh), sends a notification when NotifyOnReauth is enabled. Notifications
// are throttled to once per stale episode and cleared when a connection
// recovers.
func (s *Server) runConnectionRefreshOnce(now time.Time) {
	if s.connRefresher == nil {
		return
	}
	outcomes := s.connRefresher.RefreshExpiringConnections(context.Background(), now)

	settings := s.runtimeSettings()

	s.notifiedMu.Lock()
	defer s.notifiedMu.Unlock()

	for _, outcome := range outcomes {
		if outcome.Refreshed {
			// Connection recovered: clear throttle so a future failure notifies.
			delete(s.notifiedStale, outcome.ConnectionID)
			continue
		}
		if !outcome.Failed {
			continue
		}
		s.metrics.IncRefreshFailure()
		if s.notifiedStale[outcome.ConnectionID] {
			// Already notified for this stale episode.
			continue
		}
		s.notifiedStale[outcome.ConnectionID] = true
		if !settings.NotifyOnReauth {
			continue
		}
		s.notifyStale(settings.NotifyWebhookURL, outcome)
	}
}

func (s *Server) notifyStale(webhookURL string, outcome proxy.RefreshOutcome) {
	notifier := s.notifierFor(webhookURL)
	if notifier == nil {
		return
	}
	event := notify.Event{
		Title:   "Connection needs re-authentication",
		Message: fmt.Sprintf("%s connection %q failed to refresh: %s", outcome.Provider, outcome.Name, outcome.Reason),
		Level:   "warning",
	}
	if err := notifier.Notify(context.Background(), event); err != nil {
		log.Printf("stale connection notification failed for %s: %v", outcome.ConnectionID, err)
	}
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
	case "/metrics":
		s.handleMetrics(ctx)
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
	case "/v1/embeddings":
		s.handleExtra(ctx, handlers.Embeddings)
	case "/v1/images/generations":
		s.handleExtra(ctx, handlers.Images)
	case "/v1/audio/transcriptions":
		s.handleExtra(ctx, handlers.AudioTranscription)
	case "/v1/audio/speech":
		s.handleExtra(ctx, handlers.Speech)
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
		s.recordAuditIfMutation(ctx)
	}
}

func (s *Server) handleInference(ctx *fasthttp.RequestCtx) {
	s.handleLoggedInference(ctx, "openai", handlers.Inference)
}

// handleExtra serves the optional OpenAI-compatible endpoints (embeddings,
// images, audio). They share the /v1/* auth and source policy enforced by
// applyMiddleware plus the API-key policy gate. The inference engine is
// type-asserted to handlers.ExtraEngine; a nil assertion yields a clean 501
// through the handler's nil-engine path.
func (s *Server) handleExtra(ctx *fasthttp.RequestCtx, handle func(*fasthttp.RequestCtx, handlers.ExtraEngine)) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}
	if !s.enforceKeyPolicy(ctx) {
		return
	}
	engine, _ := s.config.InferenceEngine.(handlers.ExtraEngine)
	handle(ctx, engine)
}

func (s *Server) handleMessages(ctx *fasthttp.RequestCtx) {
	s.handleLoggedInference(ctx, "anthropic", handlers.Messages)
}

func (s *Server) handleResponses(ctx *fasthttp.RequestCtx) {
	s.handleLoggedInference(ctx, "responses", handlers.Responses)
}

func (s *Server) handleLoggedInference(ctx *fasthttp.RequestCtx, sourceFormat string, handle func(*fasthttp.RequestCtx, handlers.InferenceEngine)) {
	if !s.enforceKeyPolicy(ctx) {
		return
	}
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
				s.recordKeyUsage(snapshot.apiKeyID, model, req, nil, providerUsage)
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
	s.recordKeyUsage(userValueStringPtr(ctx, requestAPIKeyIDKey), "", req, resp, nil)
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
	metadata.clientTool = snapshot.clientTool
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
		RTKBytesSaved:  rtkBytesSaved(settings, request),
		CavemanEnabled: boolPtr(settings.CavemanEnabled),
		ComboName:      s.comboNameForModel(request),
		ClientTool:     metadata.clientTool,
	}
	if err := logging.NewLogger(usageStore).Log(entry); err != nil {
		log.Printf("write inference log: %v", err)
	}

	s.observeRequestMetric(metadata, extractedUsage, costUSD, statusCode, time.Since(started))
}

// observeRequestMetric records the just-logged inference into the Prometheus
// collector. It mirrors the values written to the request log so /metrics and
// /api/usage stay consistent. Cost and tokens default to zero when unknown.
func (s *Server) observeRequestMetric(metadata requestLogMetadata, extracted *usage.Usage, costUSD *float64, statusCode int, dur time.Duration) {
	if s.metrics == nil {
		return
	}
	var inputTok, outputTok int
	if extracted != nil {
		inputTok = extracted.InputTokens
		outputTok = extracted.OutputTokens
	}
	var cost float64
	if costUSD != nil {
		cost = *costUSD
	}
	s.metrics.ObserveRequest(metadata.provider, metadata.model, statusClassFor(statusCode), inputTok, outputTok, cost, dur)
}

// statusClassFor maps an HTTP status code to a Prometheus-friendly class label.
func statusClassFor(statusCode int) string {
	switch {
	case statusCode >= 500:
		return "5xx"
	case statusCode >= 400:
		return "4xx"
	case statusCode >= 300:
		return "3xx"
	case statusCode >= 200:
		return "2xx"
	default:
		return "other"
	}
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
	return e.captureStream(ctx, req, stream), nil
}

func (e *capturingInferenceEngine) ListModels(ctx context.Context) ([]providers.Model, error) {
	return e.base.ListModels(ctx)
}

func (e *capturingInferenceEngine) captureStream(ctx context.Context, req *providers.ChatRequest, stream <-chan providers.StreamChunk) <-chan providers.StreamChunk {
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
			// Abandon the send if the consumer has gone (client disconnect);
			// ctx is cancelled when the body-stream writer loop exits, so this
			// goroutine unwinds instead of blocking forever on out <- chunk.
			select {
			case out <- chunk:
			case <-ctx.Done():
				return
			}
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
	return s.runtimeSettings().EnableRequestLogs
}

func (s *Server) runtimeSettings() store.Settings {
	s.settingsMu.RLock()
	cached := s.settingsCache
	s.settingsMu.RUnlock()
	if cached != nil {
		return *cached
	}
	if s.config.Store != nil {
		settings, err := s.config.Store.GetSettings()
		if err == nil {
			s.settingsMu.Lock()
			s.settingsCache = &settings
			s.settingsMu.Unlock()
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
	clientTool   *string
}

// clientToolFromCtx resolves the operational client-tool label for a request.
// It prefers the explicit X-Client-Tool header and falls back to User-Agent.
// It must be called on the request goroutine before the pooled ctx is recycled.
func clientToolFromCtx(ctx *fasthttp.RequestCtx) *string {
	if value := strings.TrimSpace(string(ctx.Request.Header.Peek("X-Client-Tool"))); value != "" {
		value = truncateClientTool(value)
		return &value
	}
	if value := strings.TrimSpace(string(ctx.Request.Header.Peek("User-Agent"))); value != "" {
		value = truncateClientTool(value)
		return &value
	}
	return nil
}

// clientToolMaxBytes bounds the stored client-tool label so a hostile or
// oversized header cannot bloat log rows.
const clientToolMaxBytes = 512

func truncateClientTool(value string) string {
	if len(value) > clientToolMaxBytes {
		return value[:clientToolMaxBytes]
	}
	return value
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
	clientTool  *string
}

func newStreamLogSnapshot(ctx *fasthttp.RequestCtx) streamLogSnapshot {
	snapshot := streamLogSnapshot{
		requestID:  string(ctx.Response.Header.Peek(requestIDHeader)),
		apiKeyID:   userValueStringPtr(ctx, requestAPIKeyIDKey),
		clientTool: clientToolFromCtx(ctx),
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
	metadata := inferenceLogMetadataWithAuth(request, response, streamModel, authType, authTypeSet, userValueStringPtr(ctx, requestAPIKeyIDKey))
	metadata.clientTool = clientToolFromCtx(ctx)
	return metadata
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

// rtkBytesSaved returns the JSON byte count saved by RTK compression for the
// request, or nil when RTK is disabled, the request is absent, or no bytes were
// saved. It recomputes from the original request so it never reads the pooled
// fasthttp ctx.
func rtkBytesSaved(settings store.Settings, request *providers.ChatRequest) *int {
	if !settings.RTKEnabled || request == nil {
		return nil
	}
	before, err := json.Marshal(request)
	if err != nil {
		return nil
	}
	compressed := rtk.CompressRequest(*request)
	after, err := json.Marshal(&compressed)
	if err != nil {
		return nil
	}
	saved := len(before) - len(after)
	if saved <= 0 {
		return nil
	}
	return &saved
}

// comboNameForModel returns the active combo name when the requested model
// matches a stored active combo, or nil otherwise.
func (s *Server) comboNameForModel(request *providers.ChatRequest) *string {
	if s.config.Store == nil || request == nil || request.Model == "" {
		return nil
	}
	combo, err := s.config.Store.GetActiveCombo(request.Model)
	if err != nil || combo == nil {
		return nil
	}
	name := combo.Name
	return &name
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
	case path == "/api/audit":
		handlers.Audit(ctx, s.config.Store)
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

// handleMetrics serves GET /metrics in Prometheus text exposition format.
// /metrics is treated as a management path: it is gated by RequireAPIKey (via
// requiresAuth/isProtectedManagementPath) and subject to the source policy, so
// a scraper authenticates with the same bearer API key used for /api/*.
func (s *Server) handleMetrics(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}
	ctx.SetContentType("text/plain; version=0.0.4; charset=utf-8")
	if s.metrics == nil {
		return
	}
	_, _ = ctx.WriteString(s.metrics.Render())
}

// auditedResources are the top-level /api/{resource} segments whose successful
// mutations (POST/PUT/DELETE) are recorded in the admin audit log.
var auditedResources = map[string]bool{
	"settings":    true,
	"keys":        true,
	"connections": true,
	"combos":      true,
	"aliases":     true,
	"pricing":     true,
}

// recordAuditIfMutation appends an audit-log entry after a successful mutating
// management request. It runs after handleAPI so it can read the response
// status. It deliberately never logs the request body (which may contain
// secrets); details is a short, non-secret note. The actor is the
// authenticated API key id captured by the auth middleware (may be empty when
// RequireAPIKey is disabled).
func (s *Server) recordAuditIfMutation(ctx *fasthttp.RequestCtx) {
	if s.config.Store == nil {
		return
	}
	method := string(ctx.Method())
	switch method {
	case fasthttp.MethodPost, fasthttp.MethodPut, fasthttp.MethodDelete:
	default:
		return
	}
	if status := ctx.Response.StatusCode(); status < 200 || status >= 400 {
		return
	}

	path := strings.TrimRight(string(ctx.Path()), "/")
	parts := pathParts(path)
	if len(parts) < 2 || parts[0] != "api" || !auditedResources[parts[1]] {
		return
	}

	target := parts[len(parts)-1]
	var actor string
	if id := userValueStringPtr(ctx, requestAPIKeyIDKey); id != nil {
		actor = *id
	}

	entry := store.AuditEntry{
		ActorAPIKeyID: actor,
		Action:        method + " " + path,
		Target:        target,
		Details:       method + " " + parts[1],
	}
	if err := s.config.Store.AppendAudit(entry); err != nil {
		log.Printf("append audit log: %v", err)
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
