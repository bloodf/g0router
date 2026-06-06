package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bloodf/g0router/api/handlers"
	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/cache"
	"github.com/bloodf/g0router/internal/console"
	"github.com/bloodf/g0router/internal/governance"
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
	"github.com/bloodf/g0router/internal/traffic"
	"github.com/bloodf/g0router/internal/usage"
	"github.com/valyala/fasthttp"
)

type ServerConfig struct {
	Port                int
	Version             string
	EnableRequestLogs   bool
	RequireAPIKey       bool
	APIKeySecret        string
	APIKeyValidator     APIKeyValidator
	InferenceEngine     handlers.InferenceEngine
	Store               *store.Store
	ModelSource         handlers.ManagementModelSource
	ProviderAdapterSource handlers.ProviderAdapterSource
	OAuthFlows          handlers.OAuthFlows
	UsageStore          handlers.UsageStore
	QuotaFetchers       map[providers.ModelProvider]usage.QuotaFetcher
	QuotaKey            providers.Key
	MCPClientManager    *mcp.ClientManager
	MCPToolManager      *mcp.ToolManager
	MCPInstanceRuntime  handlers.MCPInstanceRuntime
	TunnelManager       handlers.TunnelManager
	ConsoleBroker       *console.Broker
	Governance          *governance.Governance
}

// logRetentionInterval is how often the background cleanup job runs.
const logRetentionInterval = time.Hour

// responseCacheMaxEntries bounds the optional non-streaming response cache.
const responseCacheMaxEntries = 1000

// connectionRefreshInterval is how often the proactive OAuth refresh job runs.
const connectionRefreshInterval = time.Minute

// tunnelHealthInterval is how often the background tunnel health check job runs.
const tunnelHealthInterval = 60 * time.Second

// proxyPoolHealthInterval is how often the background proxy pool health check job runs.
const proxyPoolHealthInterval = 5 * time.Minute

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

	loginRateLimiter *auth.LoginRateLimiter

	metrics *metrics.Collector

	trafficBroker *traffic.Broker
	consoleBroker *console.Broker

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

	tunnelHealthInterval    time.Duration
	proxyPoolHealthInterval time.Duration

	// responseCacheMu guards the optional non-streaming response cache and the
	// TTL it was built with. The cache TTL is fixed at construction, so when the
	// operator changes cache_ttl_seconds we rebuild the cache to honor it.
	responseCacheMu  sync.Mutex
	responseCache    *cache.Cache
	responseCacheTTL time.Duration

	// stopCh is closed when Stop() is called. Long-running handlers such as the
	// SSE traffic stream select on it so they can exit before the server
	// drains its connection pool.
	stopCh chan struct{}
	stopOnce sync.Once
}

// UpdateSettings writes settings to the store and invalidates the cache.
func (s *Server) UpdateSettings(settings store.Settings) error {
	if s.config.Store == nil {
		return fmt.Errorf("store unavailable")
	}
	if err := s.config.Store.UpdateSettings(settings); err != nil {
		return fmt.Errorf("update settings: %w", err)
	}
	s.settingsMu.Lock()
	s.settingsCache = &settings
	s.settingsMu.Unlock()
	return nil
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

// StartTunnelHealth launches the background tunnel health check job.
// It runs every 60s, stopping when ctx is cancelled.
func (s *Server) StartTunnelHealth(ctx context.Context) {
	if s.config.Store == nil {
		return
	}
	interval := s.tunnelHealthInterval
	if interval <= 0 {
		interval = tunnelHealthInterval
	}
	go func() {
		s.runTunnelHealthGuarded()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runTunnelHealthGuarded()
			}
		}
	}()
}

// runTunnelHealthGuarded runs one tunnel health pass and recovers from any panic
// so a single failed cycle cannot kill the background loop.
func (s *Server) runTunnelHealthGuarded() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("tunnel health panic recovered: %v", r)
		}
	}()
	s.runTunnelHealthOnce()
}

// runTunnelHealthOnce performs a single tunnel health pass: it lists enabled
// tunnels with non-empty URLs and checks /healthz on each.
func (s *Server) runTunnelHealthOnce() {
	if s.config.Store == nil {
		return
	}
	configs, err := s.config.Store.ListTunnelConfigs()
	if err != nil {
		log.Printf("tunnel health: list configs: %v", err)
		return
	}
	var wg sync.WaitGroup
	for _, cfg := range configs {
		if !cfg.IsEnabled || cfg.URL == "" {
			continue
		}
		wg.Add(1)
		go func(tunnelType, url string) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("tunnel health check panic for %s recovered: %v", tunnelType, r)
				}
			}()
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Get(url + "/healthz")
			if err != nil {
				if updateErr := s.config.Store.UpdateTunnelStatus(tunnelType, "error", err.Error()); updateErr != nil {
					log.Printf("tunnel health: update status for %s: %v", tunnelType, updateErr)
				}
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				if updateErr := s.config.Store.UpdateTunnelStatus(tunnelType, "active", ""); updateErr != nil {
					log.Printf("tunnel health: update status for %s: %v", tunnelType, updateErr)
				}
			} else {
				if updateErr := s.config.Store.UpdateTunnelStatus(tunnelType, "error", fmt.Sprintf("status %d", resp.StatusCode)); updateErr != nil {
					log.Printf("tunnel health: update status for %s: %v", tunnelType, updateErr)
				}
			}
		}(cfg.Type, cfg.URL)
	}
	wg.Wait()
}

// StartProxyPoolHealth launches the background proxy pool health check job.
// It runs every 5min, stopping when ctx is cancelled.
func (s *Server) StartProxyPoolHealth(ctx context.Context) {
	if s.config.Store == nil {
		return
	}
	interval := s.proxyPoolHealthInterval
	if interval <= 0 {
		interval = proxyPoolHealthInterval
	}
	go func() {
		s.runProxyPoolHealthGuarded()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runProxyPoolHealthGuarded()
			}
		}
	}()
}

// runProxyPoolHealthGuarded runs one proxy pool health pass and recovers from any panic
// so a single failed cycle cannot kill the background loop.
func (s *Server) runProxyPoolHealthGuarded() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("proxy pool health panic recovered: %v", r)
		}
	}()
	s.runProxyPoolHealthOnce()
}

// runProxyPoolHealthOnce performs a single proxy pool health pass: it lists
// active pools and attempts a TCP dial to host:port on each.
func (s *Server) runProxyPoolHealthOnce() {
	if s.config.Store == nil {
		return
	}
	pools, err := s.config.Store.ListProxyPools()
	if err != nil {
		log.Printf("proxy pool health: list pools: %v", err)
		return
	}
	var wg sync.WaitGroup
	for _, pool := range pools {
		if !pool.IsActive {
			continue
		}
		wg.Add(1)
		go func(id, host string, port int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("proxy pool health check panic for %s recovered: %v", id, r)
				}
			}()
			addr := net.JoinHostPort(host, strconv.Itoa(port))
			conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
			if err != nil {
				if updateErr := s.config.Store.UpdateProxyPoolStatus(id, "error", err.Error()); updateErr != nil {
					log.Printf("proxy pool health: update status for %s: %v", id, updateErr)
				}
				return
			}
			conn.Close()
			if updateErr := s.config.Store.UpdateProxyPoolStatus(id, "ok", ""); updateErr != nil {
				log.Printf("proxy pool health: update status for %s: %v", id, updateErr)
			}
		}(pool.ID, pool.Host, pool.Port)
	}
	wg.Wait()
}

func (s *Server) Serve(ln net.Listener) error {
	if err := s.server.Serve(ln); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

func (s *Server) Stop() error {
	// Signal long-running streaming handlers (e.g. SSE) to exit before the
	// fasthttp shutdown drains the connection pool.
	s.stopOnce.Do(func() { close(s.stopCh) })
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
	rawPath := string(ctx.Path())
	method := string(ctx.Method())
	for _, r := range s.routes() {
		if r.match(rawPath, method) {
			r.handler(ctx)
			return
		}
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

	// Optional non-streaming response cache. A hit short-circuits dispatch (and
	// therefore spend/usage accounting, since a cache hit costs nothing upstream).
	respCache, cacheKey, hitBody, cacheable := s.cacheLookup(ctx)
	if cacheable && hitBody != nil {
		// These inference endpoints always emit JSON, so the cached content-type
		// is application/json.
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetContentType("application/json")
		ctx.Response.Header.Set("X-Cache", "HIT")
		ctx.SetBody(hitBody)
		return
	}

	started := time.Now()
	engine := s.config.InferenceEngine
	var captured *capturingInferenceEngine
	if engine != nil {
		var templates proxy.PromptTemplateProvider
		if s.config.Store != nil {
			templates = s.config.Store
		}
		engine = pipelineInferenceEngine{
			base:     engine,
			settings: s.runtimeSettings,
			tools:    s.config.MCPToolManager,
			guardrails: func() store.GuardrailsConfig {
				if s.config.Store == nil {
					return store.GuardrailsConfig{}
				}
				cfg, _ := s.config.Store.GetGuardrailsConfig()
				return cfg
			},
			templates: templates,
			store:     s.config.Store,
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
				s.recordVirtualKeyUsageSnapshot(snapshot, model, req, providerUsage)
			},
		}
		engine = captured
	}
	handle(ctx, engine)
	if captured != nil && captured.streamed {
		return
	}

	// Store successful non-streamed responses in the cache. The live response is
	// marked X-Cache: MISS so clients can distinguish it from a HIT.
	if cacheable && respCache != nil {
		status := ctx.Response.StatusCode()
		if status >= 200 && status < 300 {
			respCache.Set(cacheKey, ctx.Response.Body())
			ctx.Response.Header.Set("X-Cache", "MISS")
		}
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
	s.recordVirtualKeyUsage(userValueStringPtr(ctx, requestVirtualKeyIDKey), userValueStringPtr(ctx, requestVirtualKeyTeamIDKey), "", req, resp, nil)
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

	// Publish a live-traffic event for the dashboard topology stream.
	// All values are snapshotted from local variables — never from ctx — so
	// this is safe whether called from the request goroutine or the streaming
	// capture goroutine.
	if s.trafficBroker != nil {
		keyID := ""
		if metadata.apiKeyID != nil {
			keyID = *metadata.apiKeyID
		}
		s.trafficBroker.Publish(traffic.Event{
			Timestamp:   time.Now().UTC(),
			KeyID:       keyID,
			Provider:    metadata.provider,
			Model:       metadata.model,
			StatusClass: statusClassFor(statusCode),
			StatusCode:  statusCode,
			LatencyMS:   dur.Milliseconds(),
		})
	}
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

type pipelineInferenceEngine struct {
	base        handlers.InferenceEngine
	settings    func() store.Settings
	tools       *mcp.ToolManager
	guardrails  func() store.GuardrailsConfig
	templates   proxy.PromptTemplateProvider
	store       *store.Store
}

func (e pipelineInferenceEngine) Dispatch(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	ctx = e.withAllowedTools(ctx, req)
	processed, err := e.pipeline().Process(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("pipeline dispatch: %w", err)
	}
	return e.base.Dispatch(ctx, processed)
}

func (e pipelineInferenceEngine) DispatchStream(ctx context.Context, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	ctx = e.withAllowedTools(ctx, req)
	processed, err := e.pipeline().Process(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("pipeline dispatch stream: %w", err)
	}
	return e.base.DispatchStream(ctx, processed)
}

func (e pipelineInferenceEngine) ListModels(ctx context.Context) ([]providers.Model, error) {
	return e.base.ListModels(ctx)
}

func (e pipelineInferenceEngine) withAllowedTools(ctx context.Context, req *providers.ChatRequest) context.Context {
	if e.store == nil || e.tools == nil || req == nil {
		return ctx
	}
	groupName := e.resolveMCPToolGroup(ctx, req)
	if groupName == "" {
		return ctx
	}
	group, err := e.store.GetMCPToolGroupByName(groupName)
	if err != nil || group == nil || !group.IsActive {
		return ctx
	}
	return mcp.InjectAllowedTools(ctx, group.ToolIDs)
}

func (e pipelineInferenceEngine) resolveMCPToolGroup(ctx context.Context, req *providers.ChatRequest) string {
	// Check combo model name.
	if name := strings.TrimPrefix(req.Model, "combo/"); name != req.Model && name != "" {
		combo, err := e.store.GetActiveCombo(name)
		if err == nil && combo != nil && combo.MCPToolGroup != "" {
			return combo.MCPToolGroup
		}
	}
	// Check virtual key from request context.
	if fctx, ok := ctx.(*fasthttp.RequestCtx); ok {
		if vkID := userValueStringPtr(fctx, requestVirtualKeyIDKey); vkID != nil {
			if id, err := strconv.ParseInt(*vkID, 10, 64); err == nil {
				if vk, err := e.store.GetVirtualKey(id); err == nil && vk != nil && vk.MCPToolGroup != "" {
					return vk.MCPToolGroup
				}
			}
		}
	}
	return ""
}

func (e pipelineInferenceEngine) pipeline() *proxy.Pipeline {
	s := e.settings()
	var tools proxy.ToolProvider
	if e.tools != nil {
		tools = e.tools
	}
	grCfg := store.GuardrailsConfig{}
	if e.guardrails != nil {
		grCfg = e.guardrails()
	}
	return proxy.NewPipelineWithTemplates(nil, snapshotSettings{
		rtkEnabled:          s.RTKEnabled,
		cavemanEnabled:      s.CavemanEnabled,
		cavemanLevel:        s.CavemanLevel,
		guardrailsEnabled:   grCfg.GuardrailsEnabled,
		guardrailsBlocklist: grCfg.GuardrailsBlocklist,
		piiRedactionEnabled: grCfg.PIIRedactionEnabled,
		piiRedactionTypes:   grCfg.PIIRedactionTypes,
	}, tools, e.templates)
}

type snapshotSettings struct {
	rtkEnabled          bool
	cavemanEnabled      bool
	cavemanLevel        string
	guardrailsEnabled   bool
	guardrailsBlocklist []string
	piiRedactionEnabled bool
	piiRedactionTypes   []string
}

func (s snapshotSettings) RTKEnabled() bool              { return s.rtkEnabled }
func (s snapshotSettings) CavemanEnabled() bool          { return s.cavemanEnabled }
func (s snapshotSettings) CavemanLevel() string          { return s.cavemanLevel }
func (s snapshotSettings) GuardrailsEnabled() bool       { return s.guardrailsEnabled }
func (s snapshotSettings) GuardrailsBlocklist() []string { return s.guardrailsBlocklist }
func (s snapshotSettings) PIIRedactionEnabled() bool     { return s.piiRedactionEnabled }
func (s snapshotSettings) PIIRedactionTypes() []string   { return s.piiRedactionTypes }

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

// responseCacheFor returns a response cache whose TTL matches ttl, rebuilding it
// when the operator changes cache_ttl_seconds. It returns nil when ttl is
// non-positive (caching disabled). The cache is shared across requests and is
// itself thread-safe; only the (re)build is guarded here.
func (s *Server) responseCacheFor(ttl time.Duration) *cache.Cache {
	if ttl <= 0 {
		return nil
	}
	s.responseCacheMu.Lock()
	defer s.responseCacheMu.Unlock()
	if s.responseCache == nil || s.responseCacheTTL != ttl {
		s.responseCache = cache.NewCache(responseCacheMaxEntries, ttl, time.Now)
		s.responseCacheTTL = ttl
	}
	return s.responseCache
}

// cacheLookup decides whether the request is cacheable and, if so, returns the
// cache, the derived key, and any already-cached response bytes. cacheable is
// false for streaming requests, when caching is disabled, or when the body
// cannot be parsed for a model. A cached hit is signalled by hitBody != nil.
//
// It reads only ctx.PostBody(), which fasthttp keeps valid for the duration of
// the handler on the request goroutine, so this never touches the pooled ctx
// off-goroutine.
func (s *Server) cacheLookup(ctx *fasthttp.RequestCtx) (c *cache.Cache, key string, hitBody []byte, cacheable bool) {
	settings := s.runtimeSettings()
	if !settings.CacheEnabled || settings.CacheTTLSeconds <= 0 {
		return nil, "", nil, false
	}
	body := ctx.PostBody()
	model, stream, ok := parseCacheableRequest(body)
	if !ok || stream {
		return nil, "", nil, false
	}
	c = s.responseCacheFor(time.Duration(settings.CacheTTLSeconds) * time.Second)
	if c == nil {
		return nil, "", nil, false
	}
	key = c.Key(model, body)
	if cached, hit := c.Get(key); hit {
		return c, key, cached, true
	}
	return c, key, nil, true
}

// parseCacheableRequest extracts the model and stream flag from a chat request
// body. ok is false when the body is not a JSON object, so non-JSON or malformed
// requests simply bypass the cache.
func parseCacheableRequest(body []byte) (model string, stream bool, ok bool) {
	var parsed struct {
		Model  string `json:"model"`
		Stream *bool  `json:"stream"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", false, false
	}
	return parsed.Model, parsed.Stream != nil && *parsed.Stream, true
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
	requestID      string
	authType       string
	apiKeyID       *string
	authTypeSet    bool
	clientTool     *string
	virtualKeyID   *string
	virtualKeyTeamID *string
}

func newStreamLogSnapshot(ctx *fasthttp.RequestCtx) streamLogSnapshot {
	snapshot := streamLogSnapshot{
		requestID:        string(ctx.Response.Header.Peek(requestIDHeader)),
		apiKeyID:         userValueStringPtr(ctx, requestAPIKeyIDKey),
		clientTool:       clientToolFromCtx(ctx),
		virtualKeyID:     userValueStringPtr(ctx, requestVirtualKeyIDKey),
		virtualKeyTeamID: userValueStringPtr(ctx, requestVirtualKeyTeamIDKey),
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
	"settings":      true,
	"keys":          true,
	"connections":   true,
	"combos":        true,
	"aliases":       true,
	"pricing":       true,
	"virtual-keys":  true,
	"teams":         true,
	"routing-rules": true,
	"model-limits":  true,
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

// handleTrafficStream serves GET /api/traffic/stream as a Server-Sent Events
// feed. It replays the ring buffer for initial hydration, then delivers live
// events as they are published. A 15-second heartbeat comment keeps idle
// connections alive through proxies and load balancers.
func (s *Server) handleTrafficStream(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}

	broker := s.trafficBroker
	if broker == nil {
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		return
	}

	ctx.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.SetStatusCode(fasthttp.StatusOK)

	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		subID, ch := broker.Subscribe()
		defer broker.Unsubscribe(subID)

		// Emit an opening comment so fasthttp flushes the HTTP response
		// status line and headers to the client immediately. Without an
		// initial write the client may block waiting for headers until the
		// first real event arrives.
		if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}

		// Replay ring buffer for initial hydration.
		for _, ev := range broker.Recent() {
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
		}
		if err := w.Flush(); err != nil {
			return
		}

		heartbeat := time.NewTicker(15 * time.Second)
		defer heartbeat.Stop()

		stopCh := s.stopCh
		for {
			select {
			case <-stopCh:
				return
			case ev, ok := <-ch:
				if !ok {
					return
				}
				data, err := json.Marshal(ev)
				if err != nil {
					continue
				}
				if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			case <-heartbeat.C:
				if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})
}

func (s *Server) handleConsoleLogsStream(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}
	if s.consoleBroker == nil {
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		return
	}
	handlers.ConsoleLogsStream(ctx, s.consoleBroker, s.stopCh)
}

func (s *Server) handleConsoleLogsClear(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodDelete {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}
	if s.consoleBroker == nil {
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		return
	}
	handlers.ConsoleLogsClear(ctx, s.consoleBroker, s.config.Store)
}
