package handlers

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
	"github.com/valyala/fasthttp"
)

type UsageStore interface {
	GetUsage(filter store.UsageFilter) ([]store.RequestLogEntry, error)
	GetUsageSummary(filter store.UsageFilter) (*store.UsageSummary, error)
	CountUsage(filter store.UsageFilter) (int, error)
	GetUsageChart(period, granularity string, now time.Time) (*store.UsageChart, error)
}

type usageListResponse struct {
	Object string             `json:"object"`
	Data   []usageLogResponse `json:"data"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
	Total  int                `json:"total"`
}

var allowedStatusClasses = map[string]struct{}{
	store.StatusClassSuccess:     {},
	store.StatusClassClientError: {},
	store.StatusClassServerError: {},
}

type usageSummaryResponse struct {
	RequestCount int64   `json:"request_count"`
	TotalTokens  int64   `json:"total_tokens"`
	TotalCostUSD float64 `json:"total_cost_usd"`
}

type usageChartResponse struct {
	Buckets      []string  `json:"buckets"`
	Requests     []int64   `json:"requests"`
	TokensInput  []int64   `json:"tokens_input"`
	TokensOutput []int64   `json:"tokens_output"`
	Costs        []float64 `json:"costs"`
}

type usageLogResponse struct {
	ID                 int64    `json:"id"`
	RequestID          string   `json:"request_id"`
	Timestamp          string   `json:"timestamp"`
	Provider           string   `json:"provider"`
	Model              string   `json:"model"`
	ConnectionID       *string  `json:"connection_id"`
	AuthType           string   `json:"auth_type"`
	InputTokens        *int     `json:"input_tokens"`
	OutputTokens       *int     `json:"output_tokens"`
	CacheReadTokens    *int     `json:"cache_read_tokens"`
	CacheWriteTokens   *int     `json:"cache_write_tokens"`
	TotalTokens        *int     `json:"total_tokens"`
	CostUSD            *float64 `json:"cost_usd"`
	LatencyMS          *int     `json:"latency_ms"`
	StatusCode         *int     `json:"status_code"`
	Error              *string  `json:"error"`
	SourceFormat       *string  `json:"source_format"`
	TargetFormat       *string  `json:"target_format"`
	RTKEnabled         *bool    `json:"rtk_enabled"`
	RTKBytesSaved      *int     `json:"rtk_bytes_saved"`
	CavemanEnabled     *bool    `json:"caveman_enabled"`
	ComboName          *string  `json:"combo_name"`
	APIKeyID           *string  `json:"api_key_id"`
	APIKeyName         *string  `json:"api_key_name"`
	ClientTool         *string  `json:"client_tool"`
	ConnectionName     *string  `json:"connection_name"`
	ConnectionProvider *string  `json:"connection_provider"`
	AccountEmail       *string  `json:"account_email"`
}

func Usage(ctx *fasthttp.RequestCtx, usageStore UsageStore) {
	if usageStore == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "usage store unavailable")
		return
	}

	filter, err := parseUsageFilter(ctx)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}

	adapter := usageStoreAdapter{store: usageStore}
	logs, total, err := usage.ListUsage(&adapter, toUsageFilter(filter))
	if err != nil {
		log.Printf("get usage: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to get usage")
		return
	}

	writeJSON(ctx, fasthttp.StatusOK, usageListResponse{
		Object: "list",
		Data:   toUsageLogResponses(logs),
		Limit:  filter.Limit,
		Offset: filter.Offset,
		Total:  total,
	})
}

func UsageSummary(ctx *fasthttp.RequestCtx, usageStore UsageStore) {
	if usageStore == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "usage store unavailable")
		return
	}

	filter, err := parseUsageFilter(ctx)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}

	adapter := usageStoreAdapter{store: usageStore}
	summary, err := usage.GetSummary(&adapter, toUsageFilter(filter))
	if err != nil {
		log.Printf("get usage summary: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to get usage summary")
		return
	}

	writeJSON(ctx, fasthttp.StatusOK, usageSummaryResponse{
		RequestCount: summary.RequestCount,
		TotalTokens:  summary.TotalTokens,
		TotalCostUSD: summary.TotalCostUSD,
	})
}

func UsageChart(ctx *fasthttp.RequestCtx, usageStore UsageStore, now time.Time) {
	if usageStore == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "usage store unavailable")
		return
	}

	period := string(ctx.QueryArgs().Peek("period"))
	if period == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "missing period")
		return
	}

	granularity := string(ctx.QueryArgs().Peek("granularity"))
	if granularity == "" {
		if period == "today" || period == "24h" {
			granularity = "hour"
		} else {
			granularity = "day"
		}
	}

	validPeriods := map[string]struct{}{"today": {}, "24h": {}, "7d": {}, "30d": {}, "60d": {}}
	if _, ok := validPeriods[period]; !ok {
		writeError(ctx, fasthttp.StatusBadRequest, fmt.Sprintf("invalid period: %q", period))
		return
	}
	validGranularities := map[string]struct{}{"hour": {}, "day": {}}
	if _, ok := validGranularities[granularity]; !ok {
		writeError(ctx, fasthttp.StatusBadRequest, fmt.Sprintf("invalid granularity: %q", granularity))
		return
	}

	chart, err := usageStore.GetUsageChart(period, granularity, now)
	if err != nil {
		log.Printf("get usage chart: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to get usage chart")
		return
	}

	writeJSON(ctx, fasthttp.StatusOK, usageChartResponse{
		Buckets:      chart.Buckets,
		Requests:     chart.Requests,
		TokensInput:  chart.TokensInput,
		TokensOutput: chart.TokensOutput,
		Costs:        chart.Costs,
	})
}

type quotaStore interface {
	GetActiveConnections(string) ([]*store.Connection, error)
}

func UsageQuota(ctx *fasthttp.RequestCtx, s quotaStore, fetchers map[providers.ModelProvider]usage.QuotaFetcher, key providers.Key) {
	provider := providers.ModelProvider(strings.TrimPrefix(string(ctx.Path()), "/api/usage/quota/"))
	if provider == "" || string(provider) == string(ctx.Path()) {
		writeError(ctx, fasthttp.StatusBadRequest, "missing provider")
		return
	}

	fetcher := fetchers[provider]
	if fetcher == nil {
		writeError(ctx, fasthttp.StatusNotFound, "quota fetcher not found")
		return
	}

	key = quotaKeyForProvider(s, provider, key)
	quota, err := fetcher.FetchQuota(requestContext(ctx), key)
	if err != nil {
		if errors.Is(err, usage.ErrQuotaUnsupported) {
			writeError(ctx, fasthttp.StatusNotImplemented, "quota fetching is not supported for this provider")
			return
		}
		log.Printf("fetch quota: %v", err)
		writeError(ctx, fasthttp.StatusBadGateway, "failed to fetch quota")
		return
	}

	writeJSON(ctx, fasthttp.StatusOK, quota)
}

func quotaKeyForProvider(s quotaStore, provider providers.ModelProvider, fallback providers.Key) providers.Key {
	if !isStoreNil(s) {
		connections, err := s.GetActiveConnections(string(provider))
		if err == nil {
			for _, conn := range connections {
				if key, ok := quotaKeyFromConnection(provider, conn); ok {
					return key
				}
			}
		}
	}
	fallback.Provider = provider
	return fallback
}

func quotaKeyFromConnection(provider providers.ModelProvider, conn *store.Connection) (providers.Key, bool) {
	if conn == nil {
		return providers.Key{}, false
	}
	key := providers.Key{
		Provider: provider,
		ConnID:   conn.ID,
		AuthType: string(conn.AuthType),
	}
	if conn.AuthType == store.AuthTypeAPIKey && conn.APIKey != nil && *conn.APIKey != "" {
		key.Value = *conn.APIKey
		return key, true
	}
	if conn.AccessToken != nil && *conn.AccessToken != "" {
		key.Value = *conn.AccessToken
		return key, true
	}
	return providers.Key{}, false
}

func parseUsageFilter(ctx *fasthttp.RequestCtx) (store.UsageFilter, error) {
	args := ctx.QueryArgs()
	filter := store.UsageFilter{
		Provider:     queryString(args, "provider"),
		Model:        queryString(args, "model"),
		AuthType:     queryString(args, "auth_type"),
		APIKeyID:     queryString(args, "api_key_id"),
		SourceFormat: queryString(args, "source_format"),
		Search:       string(args.Peek("search")),
	}

	statusClass := string(args.Peek("status_class"))
	if statusClass != "" {
		if _, ok := allowedStatusClasses[statusClass]; !ok {
			return store.UsageFilter{}, fmt.Errorf("invalid status_class: %q", statusClass)
		}
		filter.StatusClass = statusClass
	}

	if err := parseTimeArg(args, "from", &filter.From); err != nil {
		return store.UsageFilter{}, err
	}
	if err := parseTimeArg(args, "to", &filter.To); err != nil {
		return store.UsageFilter{}, err
	}
	if err := parseTimeArg(args, "start", &filter.Start); err != nil {
		return store.UsageFilter{}, err
	}
	if err := parseTimeArg(args, "end", &filter.End); err != nil {
		return store.UsageFilter{}, err
	}

	limit, err := parseNonNegativeIntArg(args, "limit")
	if err != nil {
		return store.UsageFilter{}, err
	}
	offset, err := parseNonNegativeIntArg(args, "offset")
	if err != nil {
		return store.UsageFilter{}, err
	}
	filter.Limit = limit
	filter.Offset = offset

	return filter, nil
}

func queryString(args *fasthttp.Args, name string) *string {
	value := string(args.Peek(name))
	if value == "" {
		return nil
	}
	return &value
}

func parseTimeArg(args *fasthttp.Args, name string, target **time.Time) error {
	value := string(args.Peek(name))
	if value == "" {
		return nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return fmt.Errorf("invalid %s: %w", name, err)
	}
	*target = &parsed
	return nil
}

func parseNonNegativeIntArg(args *fasthttp.Args, name string) (int, error) {
	value := string(args.Peek(name))
	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}
	if parsed < 0 {
		return 0, fmt.Errorf("invalid %s: must be non-negative", name)
	}
	return parsed, nil
}

func usageLogResponses(entries []store.RequestLogEntry) []usageLogResponse {
	responses := make([]usageLogResponse, 0, len(entries))
	for _, entry := range entries {
		responses = append(responses, usageLogResponse{
			ID:                 entry.ID,
			RequestID:          entry.RequestID,
			Timestamp:          entry.Timestamp.Format(time.RFC3339),
			Provider:           entry.Provider,
			Model:              entry.Model,
			ConnectionID:       entry.ConnectionID,
			AuthType:           entry.AuthType,
			InputTokens:        entry.InputTokens,
			OutputTokens:       entry.OutputTokens,
			CacheReadTokens:    entry.CacheReadTokens,
			CacheWriteTokens:   entry.CacheWriteTokens,
			TotalTokens:        entry.TotalTokens,
			CostUSD:            entry.CostUSD,
			LatencyMS:          entry.LatencyMS,
			StatusCode:         entry.StatusCode,
			Error:              entry.Error,
			SourceFormat:       entry.SourceFormat,
			TargetFormat:       entry.TargetFormat,
			RTKEnabled:         entry.RTKEnabled,
			RTKBytesSaved:      entry.RTKBytesSaved,
			CavemanEnabled:     entry.CavemanEnabled,
			ComboName:          entry.ComboName,
			APIKeyID:           entry.APIKeyID,
			APIKeyName:         entry.APIKeyName,
			ClientTool:         entry.ClientTool,
			ConnectionName:     entry.ConnectionName,
			ConnectionProvider: entry.ConnectionProvider,
			AccountEmail:       entry.AccountEmail,
		})
	}
	return responses
}

type usageStoreAdapter struct {
	store UsageStore
}

func (a *usageStoreAdapter) GetUsage(filter usage.UsageFilter) ([]usage.UsageLog, error) {
	entries, err := a.store.GetUsage(toStoreFilter(filter))
	if err != nil {
		return nil, fmt.Errorf("adapter get usage: %w", err)
	}
	logs := make([]usage.UsageLog, len(entries))
	for i, e := range entries {
		logs[i] = toUsageLog(e)
	}
	return logs, nil
}

func (a *usageStoreAdapter) CountUsage(filter usage.UsageFilter) (int, error) {
	return a.store.CountUsage(toStoreFilter(filter))
}

func (a *usageStoreAdapter) GetUsageSummary(filter usage.UsageFilter) (*usage.UsageSummary, error) {
	summary, err := a.store.GetUsageSummary(toStoreFilter(filter))
	if err != nil {
		return nil, fmt.Errorf("adapter get usage summary: %w", err)
	}
	return &usage.UsageSummary{
		RequestCount: summary.RequestCount,
		TotalTokens:  summary.TotalTokens,
		TotalCostUSD: summary.TotalCostUSD,
	}, nil
}

func toUsageFilter(f store.UsageFilter) usage.UsageFilter {
	return usage.UsageFilter{
		Provider:     f.Provider,
		Model:        f.Model,
		AuthType:     f.AuthType,
		APIKeyID:     f.APIKeyID,
		SourceFormat: f.SourceFormat,
		StatusClass:  f.StatusClass,
		Search:       f.Search,
		From:         f.From,
		To:           f.To,
		Start:        f.Start,
		End:          f.End,
		Limit:        f.Limit,
		Offset:       f.Offset,
	}
}

func toStoreFilter(f usage.UsageFilter) store.UsageFilter {
	return store.UsageFilter{
		Provider:     f.Provider,
		Model:        f.Model,
		AuthType:     f.AuthType,
		APIKeyID:     f.APIKeyID,
		SourceFormat: f.SourceFormat,
		StatusClass:  f.StatusClass,
		Search:       f.Search,
		From:         f.From,
		To:           f.To,
		Start:        f.Start,
		End:          f.End,
		Limit:        f.Limit,
		Offset:       f.Offset,
	}
}

func toUsageLog(e store.RequestLogEntry) usage.UsageLog {
	return usage.UsageLog{
		ID:                 e.ID,
		RequestID:          e.RequestID,
		Timestamp:          e.Timestamp,
		Provider:           e.Provider,
		Model:              e.Model,
		ConnectionID:       e.ConnectionID,
		AuthType:           e.AuthType,
		InputTokens:        e.InputTokens,
		OutputTokens:       e.OutputTokens,
		CacheReadTokens:    e.CacheReadTokens,
		CacheWriteTokens:   e.CacheWriteTokens,
		TotalTokens:        e.TotalTokens,
		CostUSD:            e.CostUSD,
		LatencyMS:          e.LatencyMS,
		StatusCode:         e.StatusCode,
		Error:              e.Error,
		SourceFormat:       e.SourceFormat,
		TargetFormat:       e.TargetFormat,
		RTKEnabled:         e.RTKEnabled,
		RTKBytesSaved:      e.RTKBytesSaved,
		CavemanEnabled:     e.CavemanEnabled,
		ComboName:          e.ComboName,
		APIKeyID:           e.APIKeyID,
		APIKeyName:         e.APIKeyName,
		ClientTool:         e.ClientTool,
		ConnectionName:     e.ConnectionName,
		ConnectionProvider: e.ConnectionProvider,
		AccountEmail:       e.AccountEmail,
	}
}

func toUsageLogResponses(entries []usage.UsageLog) []usageLogResponse {
	responses := make([]usageLogResponse, 0, len(entries))
	for _, entry := range entries {
		responses = append(responses, usageLogResponse{
			ID:                 entry.ID,
			RequestID:          entry.RequestID,
			Timestamp:          entry.Timestamp.Format(time.RFC3339),
			Provider:           entry.Provider,
			Model:              entry.Model,
			ConnectionID:       entry.ConnectionID,
			AuthType:           entry.AuthType,
			InputTokens:        entry.InputTokens,
			OutputTokens:       entry.OutputTokens,
			CacheReadTokens:    entry.CacheReadTokens,
			CacheWriteTokens:   entry.CacheWriteTokens,
			TotalTokens:        entry.TotalTokens,
			CostUSD:            entry.CostUSD,
			LatencyMS:          entry.LatencyMS,
			StatusCode:         entry.StatusCode,
			Error:              entry.Error,
			SourceFormat:       entry.SourceFormat,
			TargetFormat:       entry.TargetFormat,
			RTKEnabled:         entry.RTKEnabled,
			RTKBytesSaved:      entry.RTKBytesSaved,
			CavemanEnabled:     entry.CavemanEnabled,
			ComboName:          entry.ComboName,
			APIKeyID:           entry.APIKeyID,
			APIKeyName:         entry.APIKeyName,
			ClientTool:         entry.ClientTool,
			ConnectionName:     entry.ConnectionName,
			ConnectionProvider: entry.ConnectionProvider,
			AccountEmail:       entry.AccountEmail,
		})
	}
	return responses
}
