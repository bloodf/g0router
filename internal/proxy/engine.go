package proxy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/modelcatalog"
	providercore "github.com/bloodf/g0router/internal/provider"
	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
)

var (
	ErrProviderNotFound             = errors.New("provider not found")
	ErrProviderInferenceUnavailable = errors.New("provider inference unavailable")
	ErrNoConnections                = errors.New("no active connections")
	ErrQuotaExhausted               = errors.New("quota exhausted")
	// ErrCapabilityUnsupported is returned when the resolved provider does not
	// implement the optional capability (embeddings, images, audio) requested.
	ErrCapabilityUnsupported = errors.New("capability unsupported")
)

// tokenLikePattern matches long alphanumeric/base64 blobs that look like tokens.
var tokenLikePattern = regexp.MustCompile(`[A-Za-z0-9+/\-_]{20,}`)

const defaultRefreshWindow = 5 * time.Minute

type oauthRefresher interface {
	Refresh(ctx context.Context, refreshToken string) (oauth.TokenResult, error)
}

type modelRoute struct {
	Provider providers.ModelProvider
	Model    string
}

type engineClock struct {
	engine *Engine
}

func (c engineClock) Now() time.Time {
	return c.engine.now()
}

type Engine struct {
	store          *store.Store
	pool           providerPool
	registryMu     sync.RWMutex
	refreshers     map[oauth.ProviderID]oauthRefresher
	refreshManager *providercore.RefreshManager
	fallback       *providercore.FallbackManager
	quotaFetchers  map[providers.ModelProvider]usage.QuotaFetcher
	mcpTools       *mcp.ToolManager
	aliasCache     *aliasCache
	comboResolver  *ComboResolver
	refreshWindow  time.Duration
	now            func() time.Time
}

func NewEngine(s *store.Store) *Engine {
	engine := &Engine{
		store:          s,
		pool:           newProviderPool(),
		refreshers:     make(map[oauth.ProviderID]oauthRefresher),
		refreshManager: providercore.NewRefreshManager(),
		quotaFetchers:  make(map[providers.ModelProvider]usage.QuotaFetcher),
		aliasCache:     newAliasCache(defaultAliasCacheTTL),
		comboResolver:  NewComboResolver(s),
		refreshWindow:  defaultRefreshWindow,
		now:            time.Now,
	}
	engine.fallback = providercore.NewFallbackManagerWithClock(s, engineClock{engine: engine})
	return engine
}

func (e *Engine) Register(provider providers.Provider) {
	e.pool.register(provider)
}

func (e *Engine) RegisterOAuthRefresher(provider oauth.ProviderID, refresher oauthRefresher) {
	e.registryMu.Lock()
	defer e.registryMu.Unlock()
	e.refreshers[provider] = refresher
}

func (e *Engine) RegisterQuotaFetcher(provider providers.ModelProvider, fetcher usage.QuotaFetcher) {
	e.registryMu.Lock()
	defer e.registryMu.Unlock()
	if fetcher == nil {
		delete(e.quotaFetchers, provider)
		return
	}
	e.quotaFetchers[provider] = fetcher
}

func (e *Engine) refresherFor(provider oauth.ProviderID) (oauthRefresher, bool) {
	e.registryMu.RLock()
	defer e.registryMu.RUnlock()
	refresher, ok := e.refreshers[provider]
	return refresher, ok
}

func (e *Engine) quotaFetcherFor(provider providers.ModelProvider) usage.QuotaFetcher {
	e.registryMu.RLock()
	defer e.registryMu.RUnlock()
	return e.quotaFetchers[provider]
}

func (e *Engine) RegisterMCPToolManager(tools *mcp.ToolManager) {
	e.mcpTools = tools
}

func (e *Engine) MCPToolManager() *mcp.ToolManager {
	return e.mcpTools
}

func (e *Engine) RegisteredProviders() []providers.ModelProvider {
	return e.pool.names()
}

func (e *Engine) Dispatch(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	if comboName, ok := comboModelName(req.Model); ok {
		return e.comboResolver.Dispatch(ctx, e, comboName, req)
	}

	route, err := e.resolveModelRoute(req.Model)
	if err != nil {
		return nil, err
	}
	return e.dispatchRoute(ctx, route, req)
}

func (e *Engine) DispatchStream(ctx context.Context, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	if comboName, ok := comboModelName(req.Model); ok {
		return e.comboResolver.DispatchStream(ctx, e, comboName, req)
	}

	route, err := e.resolveModelRoute(req.Model)
	if err != nil {
		return nil, err
	}
	return e.dispatchStreamRoute(ctx, route, req)
}

func (e *Engine) dispatchRoute(ctx context.Context, route modelRoute, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	var lastErr error
	for attempt := 0; attempt < e.maxConnectionAttempts(route.Provider); attempt++ {
		provider, key, conn, upstreamModel, err := e.providerForRoute(ctx, route)
		if err != nil {
			if lastErr != nil && errors.Is(err, ErrNoConnections) {
				return nil, lastErr
			}
			return nil, err
		}

		if err := e.checkQuota(ctx, key); err != nil {
			return nil, err
		}

		dispatchReq := requestWithModel(req, upstreamModel)
		resp, err := e.chatCompletion(ctx, provider, key, dispatchReq)
		if err != nil {
			wrapped := fmt.Errorf("chat completion: %w", err)
			if fallbackWorthyError(err) {
				e.recordProviderFailure(conn, upstreamModel)
				lastErr = wrapped
				continue
			}
			return nil, wrapped
		}
		e.recordProviderSuccess(conn, upstreamModel)
		annotateDispatchResponse(resp, key)
		return resp, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, ErrNoConnections
}

func (e *Engine) chatCompletion(ctx context.Context, provider providers.Provider, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	if e.shouldRunMCPAgent(ctx, req) {
		return mcp.NewAgent(provider, key, e.mcpTools).Run(ctx, req)
	}
	return provider.ChatCompletion(ctx, key, req)
}

func (e *Engine) shouldRunMCPAgent(ctx context.Context, req *providers.ChatRequest) bool {
	if e.mcpTools == nil || req == nil {
		return false
	}
	if len(req.Tools) == 0 {
		return len(e.mcpTools.CompactToolsForRequest(ctx)) > 0
	}
	for _, tool := range req.Tools {
		if _, err := e.mcpTools.Lookup(tool.Function.Name); err == nil {
			return true
		}
	}
	return false
}

func (e *Engine) dispatchStreamRoute(ctx context.Context, route modelRoute, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	var lastErr error
	for attempt := 0; attempt < e.maxConnectionAttempts(route.Provider); attempt++ {
		provider, key, conn, upstreamModel, err := e.providerForRoute(ctx, route)
		if err != nil {
			if lastErr != nil && errors.Is(err, ErrNoConnections) {
				return nil, lastErr
			}
			return nil, err
		}

		if err := e.checkQuota(ctx, key); err != nil {
			return nil, err
		}

		stream, err := provider.ChatCompletionStream(ctx, key, requestWithModel(req, upstreamModel))
		if err != nil {
			wrapped := fmt.Errorf("chat completion stream: %w", err)
			if fallbackWorthyError(err) {
				e.recordProviderFailure(conn, upstreamModel)
				lastErr = wrapped
				continue
			}
			return nil, wrapped
		}

		// Success is only recorded once a non-error chunk flows. A stream that
		// errors at the first chunk records failure and rotates to the next
		// connection, so mid-stream failures still penalize backoff.
		first, ok := <-stream
		if !ok {
			// Clean completion with no chunks: treat as success.
			e.recordProviderSuccess(conn, upstreamModel)
			return closedStream(), nil
		}
		if streamErr := chunkError(first); streamErr != nil {
			wrapped := fmt.Errorf("chat completion stream: %w", streamErr)
			if fallbackWorthyError(streamErr) {
				e.recordProviderFailure(conn, upstreamModel)
				lastErr = wrapped
				continue
			}
			return nil, wrapped
		}
		e.recordProviderSuccess(conn, upstreamModel)
		return prependChunk(ctx, first, stream), nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, ErrNoConnections
}

// chunkError reports a stream-level error carried by a chunk, if any.
func chunkError(chunk providers.StreamChunk) error {
	if chunk.Error == nil {
		return nil
	}
	msg := chunk.Error.Message
	if msg == "" {
		msg = chunk.Error.Code
	}
	if msg == "" {
		msg = "stream error"
	}
	return errors.New(msg)
}

// closedStream returns an already-closed empty stream channel.
func closedStream() <-chan providers.StreamChunk {
	ch := make(chan providers.StreamChunk)
	close(ch)
	return ch
}

// prependChunk re-emits first followed by the remaining chunks of rest. Sends
// select on ctx.Done() so the goroutine abandons (rather than blocking forever)
// when the consumer disconnects mid-stream; ctx is cancelled once the request's
// body-stream writer loop exits.
func prependChunk(ctx context.Context, first providers.StreamChunk, rest <-chan providers.StreamChunk) <-chan providers.StreamChunk {
	out := make(chan providers.StreamChunk)
	go func() {
		defer close(out)
		select {
		case out <- first:
		case <-ctx.Done():
			return
		}
		for chunk := range rest {
			select {
			case out <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func (e *Engine) ListModels(ctx context.Context) ([]providers.Model, error) {
	var models []providers.Model
	for _, providerName := range e.pool.names() {
		providerModels, err := e.providerModels(ctx, providerName)
		if err != nil {
			log.Printf("proxy: list models for provider %s: %v", providerName, err)
			continue
		}
		models = append(models, providerModels...)
	}
	return models, nil
}

func (e *Engine) providerModels(ctx context.Context, providerName providers.ModelProvider) ([]providers.Model, error) {
	provider, ok := e.pool.get(providerName)
	if !ok {
		return nil, ErrProviderNotFound
	}

	key, err := e.keyFor(ctx, providerName)
	if errors.Is(err, ErrNoConnections) {
		return catalogModels(providerName), nil
	}
	if err != nil {
		return nil, err
	}

	models, err := provider.ListModels(ctx, key)
	if err == nil && len(models) > 0 {
		return models, nil
	}
	if err != nil {
		log.Printf("proxy: provider %s list models, falling back to catalog: %v", providerName, err)
	}
	return catalogModels(providerName), nil
}

func catalogModels(providerName providers.ModelProvider) []providers.Model {
	prices := modelcatalog.NewCatalog().Models(providerName)
	modelIDs := make([]string, 0, len(prices))
	for modelID := range prices {
		modelIDs = append(modelIDs, modelID)
	}
	sort.Strings(modelIDs)

	models := make([]providers.Model, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		models = append(models, providers.Model{
			ID:       modelID,
			Object:   "model",
			OwnedBy:  providerName.String(),
			Provider: providerName,
		})
	}
	return models
}

func (e *Engine) providerFor(ctx context.Context, model string) (providers.Provider, providers.Key, *store.Connection, string, error) {
	route, err := e.resolveModelRoute(model)
	if err != nil {
		return nil, providers.Key{}, nil, "", err
	}
	return e.providerForRoute(ctx, route)
}

func (e *Engine) providerForRoute(ctx context.Context, route modelRoute) (providers.Provider, providers.Key, *store.Connection, string, error) {
	provider, ok := e.pool.get(route.Provider)
	if !ok {
		return nil, providers.Key{}, nil, "", ErrProviderNotFound
	}

	key, conn, err := e.keyForModel(ctx, route.Provider, route.Model)
	if err != nil {
		return nil, providers.Key{}, nil, "", err
	}
	return provider, key, conn, route.Model, nil
}

func (e *Engine) resolveModelRoute(model string) (modelRoute, error) {
	alias, ok, err := e.resolveModelAlias(model)
	if err != nil {
		return modelRoute{}, err
	}
	if ok {
		return routableModelRoute(modelRoute{
			Provider: providers.ModelProvider(providercore.CanonicalProviderID(alias.Provider)),
			Model:    alias.Model,
		})
	}

	if route, ok := modelcatalog.NewCatalog().RouteForModel(model); ok {
		return routableModelRoute(modelRoute{Provider: route.Provider, Model: route.UpstreamModel})
	}

	if route, ok := providerQualifiedDynamicRoute(model); ok {
		return routableModelRoute(route)
	}

	if provider, ok := resolveProvider(model); ok {
		return routableModelRoute(modelRoute{Provider: provider, Model: model})
	}

	return modelRoute{}, ErrProviderNotFound
}

func (e *Engine) resolveModelAlias(model string) (store.ModelAlias, bool, error) {
	now := e.now()
	if alias, ok := e.aliasCache.get(model, now); ok {
		return alias, true, nil
	}

	alias, err := e.store.ResolveModelAlias(model)
	if errors.Is(err, store.ErrNotFound) {
		return store.ModelAlias{}, false, nil
	}
	if err != nil {
		return store.ModelAlias{}, false, fmt.Errorf("resolve model alias: %w", err)
	}

	e.aliasCache.set(model, alias, now)
	return alias, true, nil
}

func (e *Engine) resolveComboStepRoute(step ComboStep) (modelRoute, error) {
	route, err := e.resolveModelRoute(step.Model)
	if err == nil {
		return route, nil
	}
	if err != nil && !errors.Is(err, ErrProviderNotFound) {
		return modelRoute{}, err
	}
	return routableModelRoute(modelRoute{Provider: step.Provider, Model: step.Model})
}

func routableModelRoute(route modelRoute) (modelRoute, error) {
	entry, ok := providercore.ProviderMatrix().Provider(route.Provider.String())
	if ok && !entry.Inference {
		return modelRoute{}, fmt.Errorf("%w: %s", ErrProviderInferenceUnavailable, route.Provider)
	}
	return route, nil
}

func requestWithModel(req *providers.ChatRequest, model string) *providers.ChatRequest {
	if req.Model == model {
		return req
	}
	copied := *req
	copied.Model = model
	return &copied
}

func annotateDispatchResponse(resp *providers.ChatResponse, key providers.Key) {
	if resp == nil {
		return
	}
	resp.Provider = key.Provider
	resp.ConnectionID = key.ConnID
	resp.AuthType = key.AuthType
}

func (e *Engine) keyFor(ctx context.Context, provider providers.ModelProvider) (providers.Key, error) {
	key, _, err := e.keyForModel(ctx, provider, "")
	return key, err
}

func (e *Engine) keyForModel(ctx context.Context, provider providers.ModelProvider, model string) (providers.Key, *store.Connection, error) {
	conn, err := e.connectionForModel(provider, model)
	if err != nil {
		if errors.Is(err, ErrNoConnections) && providerSupportsNoAuth(provider) {
			return providers.Key{
				Provider: provider,
				AuthType: string(store.AuthTypeNoAuth),
			}, nil, nil
		}
		return providers.Key{}, nil, err
	}
	conn, err = e.refreshConnectionIfNeeded(ctx, provider, conn)
	if err != nil {
		return providers.Key{}, nil, err
	}

	key := providers.Key{
		Provider: provider,
		ConnID:   conn.ID,
		AuthType: string(conn.AuthType),
	}
	if conn.APIKey != nil {
		key.Value = *conn.APIKey
	} else if conn.AccessToken != nil {
		key.Value = *conn.AccessToken
	}
	if conn.AccountID != nil {
		key.AccountID = *conn.AccountID
	}

	return key, conn, nil
}

func providerSupportsNoAuth(provider providers.ModelProvider) bool {
	entry, ok := providercore.ProviderMatrix().Provider(provider.String())
	if !ok {
		return false
	}
	for _, authType := range entry.AuthTypes {
		if authType == string(store.AuthTypeNoAuth) {
			return true
		}
	}
	return false
}

func (e *Engine) connectionForModel(provider providers.ModelProvider, model string) (*store.Connection, error) {
	var lastErr error
	for _, providerID := range providercore.ProviderAliases(provider.String()) {
		conn, err := e.fallback.Next(providerID, model)
		if err == nil {
			return conn, nil
		}
		if errors.Is(err, providercore.ErrNoActiveConnections) {
			lastErr = err
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("get active connections: %w", err)
		}
	}
	if lastErr != nil {
		return nil, ErrNoConnections
	}
	return nil, ErrNoConnections
}

func (e *Engine) maxConnectionAttempts(provider providers.ModelProvider) int {
	total := 0
	for _, providerID := range providercore.ProviderAliases(provider.String()) {
		connections, err := e.store.GetActiveConnections(providerID)
		if err != nil {
			continue
		}
		total += len(connections)
	}
	if total == 0 {
		return 1
	}
	return total
}

func (e *Engine) checkQuota(ctx context.Context, key providers.Key) error {
	fetcher := e.quotaFetcherFor(key.Provider)
	if fetcher == nil {
		return nil
	}
	quota, err := fetcher.FetchQuota(ctx, key)
	if err != nil {
		if errors.Is(err, ErrQuotaExhausted) {
			return fmt.Errorf("%s quota exhausted: %w", key.Provider, ErrQuotaExhausted)
		}
		if errors.Is(err, usage.ErrQuotaUnsupported) {
			return nil
		}
		return nil
	}
	if quotaExhausted(quota) {
		return fmt.Errorf("%s quota exhausted: %w", key.Provider, ErrQuotaExhausted)
	}
	return nil
}

func quotaExhausted(quota usage.Quota) bool {
	if quota.Unlimited {
		return false
	}
	return quota.Remaining <= 0
}

// RefreshOutcome reports the result of a proactive refresh attempt for a single
// OAuth connection.
type RefreshOutcome struct {
	ConnectionID string
	Provider     string
	Name         string
	Refreshed    bool
	Failed       bool
	Reason       string
}

// RefreshExpiringConnections lists all OAuth connections whose tokens fall
// within the refresh window and, for each that has a registered refresher,
// attempts a refresh via the existing refresh-before-dispatch path. Success
// persists new tokens and clears needs_reauth; failure marks needs_reauth. It
// returns one outcome per attempted connection so the caller can notify on new
// failures. Connections that are healthy or lack a refresher are skipped.
func (e *Engine) RefreshExpiringConnections(ctx context.Context, now time.Time) []RefreshOutcome {
	conns, err := e.store.ListConnections()
	if err != nil {
		log.Printf("proactive refresh: list connections: %v", err)
		return nil
	}

	var outcomes []RefreshOutcome
	for _, conn := range conns {
		if conn == nil || !conn.IsActive {
			continue
		}
		if !e.connectionNeedsRefresh(conn) {
			continue
		}
		runtimeProvider := providers.ModelProvider(conn.Provider)
		oauthProvider := e.oauthProviderForConnection(runtimeProvider, conn)
		if _, ok := e.refresherFor(oauthProvider); !ok {
			continue
		}

		outcome := RefreshOutcome{
			ConnectionID: conn.ID,
			Provider:     conn.Provider,
			Name:         conn.Name,
		}
		if _, err := e.refreshConnectionIfNeeded(ctx, runtimeProvider, conn); err != nil {
			outcome.Failed = true
			outcome.Reason = sanitizeRefreshReason(err)
		} else {
			outcome.Refreshed = true
		}
		outcomes = append(outcomes, outcome)
	}
	return outcomes
}

func (e *Engine) refreshConnectionIfNeeded(ctx context.Context, provider providers.ModelProvider, conn *store.Connection) (*store.Connection, error) {
	if !e.connectionNeedsRefresh(conn) {
		return conn, nil
	}
	oauthProvider := e.oauthProviderForConnection(provider, conn)
	refresher, ok := e.refresherFor(oauthProvider)
	if !ok {
		return conn, nil
	}

	token, err := e.refreshManager.Refresh(ctx, conn, func(ctx context.Context, conn *store.Connection) (oauth.TokenResult, error) {
		return refresher.Refresh(ctx, *conn.RefreshToken)
	})
	if err != nil {
		_ = e.store.MarkConnectionRefreshFailure(conn.ID, sanitizeRefreshReason(err))
		return nil, fmt.Errorf("refresh oauth credentials: %w", err)
	}
	if token.AccessToken == "" {
		_ = e.store.MarkConnectionRefreshFailure(conn.ID, "refresh failed: empty access token")
		return nil, errors.New("refresh oauth credentials: access token is required")
	}

	accessToken := token.AccessToken
	refreshToken := conn.RefreshToken
	if token.RefreshToken != "" {
		newRefreshToken := token.RefreshToken
		refreshToken = &newRefreshToken
	}
	expiresAt := conn.ExpiresAt
	if !token.ExpiresAt.IsZero() {
		newExpiresAt := token.ExpiresAt.Unix()
		expiresAt = &newExpiresAt
	}

	if err := e.store.UpdateConnectionCredentials(conn.ID, &accessToken, refreshToken, expiresAt); err != nil {
		return nil, fmt.Errorf("update refreshed credentials: %w", err)
	}
	_ = e.store.ClearConnectionRefreshFailure(conn.ID)

	updated := *conn
	updated.AccessToken = &accessToken
	updated.RefreshToken = refreshToken
	updated.ExpiresAt = expiresAt
	updated.NeedsReauth = false
	updated.LastRefreshError = nil
	return &updated, nil
}

// sanitizeRefreshReason extracts a non-secret classification from a refresh
// error. It strips anything that looks like a token (long alphanumeric blobs)
// and returns at most the first 200 runes.
func sanitizeRefreshReason(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	// Redact token-like substrings: 20+ char alphanumeric/base64 sequences.
	msg = tokenLikePattern.ReplaceAllString(msg, "[redacted]")
	r := []rune(msg)
	if len(r) > 200 {
		r = r[:200]
	}
	return string(r)
}

func (e *Engine) oauthProviderForConnection(runtimeProvider providers.ModelProvider, conn *store.Connection) oauth.ProviderID {
	if conn.ProviderSpecificData != nil {
		if value, ok := conn.ProviderSpecificData["oauth_provider"].(string); ok {
			if value = strings.TrimSpace(value); value != "" {
				return oauth.ProviderID(value)
			}
		}
	}
	if runtimeProvider == providers.ProviderOpenAI {
		return oauth.ProviderID("codex")
	}
	return oauth.ProviderID(runtimeProvider.String())
}

func (e *Engine) connectionNeedsRefresh(conn *store.Connection) bool {
	if conn.AuthType != store.AuthTypeOAuth {
		return false
	}
	if conn.RefreshToken == nil || *conn.RefreshToken == "" {
		return false
	}
	if conn.ExpiresAt == nil {
		return false
	}
	return time.Unix(*conn.ExpiresAt, 0).Before(e.now().Add(e.refreshWindow))
}

func resolveProvider(model string) (providers.ModelProvider, bool) {
	switch {
	case strings.HasPrefix(model, "gpt-"):
		return providers.ProviderOpenAI, true
	case strings.HasPrefix(model, "claude-"):
		return providers.ProviderAnthropic, true
	default:
		return "", false
	}
}

func providerQualifiedDynamicRoute(model string) (modelRoute, bool) {
	providerID, upstreamModel, ok := strings.Cut(model, "/")
	if !ok || upstreamModel == "" {
		return modelRoute{}, false
	}

	provider := providers.ModelProvider(providercore.CanonicalProviderID(providerID))
	switch provider {
	case providers.ProviderAlibaba,
		providers.ProviderAzure,
		providers.ProviderCloudflare,
		providers.ProviderGitHubCopilot,
		providers.ProviderGitLabDuo,
		providers.ProviderKilo,
		providers.ProviderKimi,
		providers.ProviderLiteLLM,
		providers.ProviderLMStudio,
		providers.ProviderOllamaCloud,
		providers.ProviderOpenCode,
		providers.ProviderQianfan,
		providers.ProviderReplicate,
		providers.ProviderVLLM,
		providers.ProviderXiaomi,
		providers.ProviderZhipu:
		return modelRoute{Provider: provider, Model: upstreamModel}, true
	default:
		return modelRoute{}, false
	}
}

func comboModelName(model string) (string, bool) {
	name := strings.TrimPrefix(model, "combo/")
	if name == model || name == "" {
		return "", false
	}
	return name, true
}

func (e *Engine) recordProviderFailure(conn *store.Connection, model string) {
	if conn == nil {
		return
	}
	_ = e.fallback.RecordFailure(conn, model)
}

func (e *Engine) recordProviderSuccess(conn *store.Connection, model string) {
	if conn == nil {
		return
	}
	_ = e.fallback.RecordSuccess(conn, model)
}

func fallbackWorthyError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrQuotaExhausted) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	text := strings.ToLower(err.Error())
	for _, marker := range []string{
		"rate limit",
		"rate limited",
		"quota",
		"server error",
		"temporarily unavailable",
		"service unavailable",
		"bad gateway",
		"gateway timeout",
		"timeout",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}
