package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
	"github.com/valyala/fasthttp"
)

type fakeQuotaFetcher struct {
	gotKey providers.Key
	gotCtx context.Context
	quota  usage.Quota
	err    error
}

func (f *fakeQuotaFetcher) FetchQuota(ctx context.Context, key providers.Key) (usage.Quota, error) {
	f.gotKey = key
	f.gotCtx = ctx
	if f.err != nil {
		return usage.Quota{}, f.err
	}
	return f.quota, nil
}

func TestUsageListsFilteredEntries(t *testing.T) {
	s := openHandlerTestStore(t)
	first := handlerUsageEntry("req-1", "openai", "gpt-4o", time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC))
	first.TotalTokens = intPtr(15)
	first.CostUSD = floatPtr(0.0015)
	second := handlerUsageEntry("req-2", "anthropic", "claude-sonnet-4", time.Date(2026, 6, 2, 10, 1, 0, 0, time.UTC))
	logHandlerEntries(t, s, []store.RequestLogEntry{first, second})

	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage?provider=openai&limit=10")
	Usage(ctx, s)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var decoded struct {
		Object string `json:"object"`
		Data   []struct {
			RequestID   string   `json:"request_id"`
			Provider    string   `json:"provider"`
			Model       string   `json:"model"`
			TotalTokens *int     `json:"total_tokens"`
			CostUSD     *float64 `json:"cost_usd"`
		} `json:"data"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &decoded); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if decoded.Object != "list" {
		t.Fatalf("object = %q, want list", decoded.Object)
	}
	if len(decoded.Data) != 1 {
		t.Fatalf("data len = %d, want 1", len(decoded.Data))
	}
	if decoded.Data[0].RequestID != "req-1" || decoded.Data[0].Provider != "openai" {
		t.Fatalf("entry = %+v, want req-1/openai", decoded.Data[0])
	}
	if decoded.Data[0].TotalTokens == nil || *decoded.Data[0].TotalTokens != 15 {
		t.Fatalf("total tokens = %v, want 15", decoded.Data[0].TotalTokens)
	}
	if decoded.Data[0].CostUSD == nil || *decoded.Data[0].CostUSD != 0.0015 {
		t.Fatalf("cost = %v, want 0.0015", decoded.Data[0].CostUSD)
	}
	if decoded.Limit != 10 || decoded.Offset != 0 {
		t.Fatalf("pagination = %d/%d, want 10/0", decoded.Limit, decoded.Offset)
	}
}

func TestUsageRejectsInvalidDateFilter(t *testing.T) {
	s := openHandlerTestStore(t)
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage?from=not-a-date")

	Usage(ctx, s)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestUsageSummaryReturnsAggregate(t *testing.T) {
	s := openHandlerTestStore(t)
	first := handlerUsageEntry("req-1", "openai", "gpt-4o", time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC))
	first.TotalTokens = intPtr(10)
	first.CostUSD = floatPtr(0.25)
	second := handlerUsageEntry("req-2", "openai", "gpt-4o-mini", time.Date(2026, 6, 2, 10, 1, 0, 0, time.UTC))
	second.TotalTokens = intPtr(20)
	second.CostUSD = floatPtr(0.75)
	logHandlerEntries(t, s, []store.RequestLogEntry{first, second})

	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage/summary?provider=openai")
	UsageSummary(ctx, s)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var decoded struct {
		RequestCount int64   `json:"request_count"`
		TotalTokens  int64   `json:"total_tokens"`
		TotalCost    float64 `json:"total_cost"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &decoded); err != nil {
		t.Fatalf("unmarshal summary: %v", err)
	}
	if decoded.RequestCount != 2 || decoded.TotalTokens != 30 || decoded.TotalCost != 1.0 {
		t.Fatalf("summary = %+v, want 2/30/1.0", decoded)
	}
}

func TestUsageQuotaFetchesProviderQuota(t *testing.T) {
	fetcher := &fakeQuotaFetcher{
		quota: usage.Quota{
			Provider:  providers.ProviderOpenAI,
			Limit:     1000,
			Used:      125,
			Remaining: 875,
		},
	}
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage/quota/openai")

	UsageQuota(ctx, nil, map[providers.ModelProvider]usage.QuotaFetcher{
		providers.ProviderOpenAI: fetcher,
	}, providers.Key{Value: "sk-test", AuthType: "api_key"})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if fetcher.gotKey.Provider != providers.ProviderOpenAI || fetcher.gotKey.Value != "sk-test" {
		t.Fatalf("key = %+v, want openai/sk-test", fetcher.gotKey)
	}
	if fetcher.gotCtx == nil {
		t.Fatal("quota context is nil, want non-nil request-scoped context")
	}
	if _, ok := fetcher.gotCtx.(*fasthttp.RequestCtx); ok {
		t.Fatalf("quota context must be detached from the pooled *fasthttp.RequestCtx to avoid use-after-recycle, got %T", fetcher.gotCtx)
	}
	var decoded usage.Quota
	if err := json.Unmarshal(ctx.Response.Body(), &decoded); err != nil {
		t.Fatalf("unmarshal quota: %v", err)
	}
	if decoded.Limit != 1000 || decoded.Used != 125 || decoded.Remaining != 875 {
		t.Fatalf("quota = %+v, want 1000/125/875", decoded)
	}
}

func TestUsageQuotaRawJSONContract(t *testing.T) {
	fetcher := &fakeQuotaFetcher{
		quota: usage.Quota{
			Provider:  providers.ProviderOpenAI,
			Limit:     1000,
			Used:      125,
			Remaining: 875,
			Unit:      "credits",
		},
	}
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage/quota/openai")

	UsageQuota(ctx, nil, map[providers.ModelProvider]usage.QuotaFetcher{
		providers.ProviderOpenAI: fetcher,
	}, providers.Key{Value: "sk-test", AuthType: "api_key"})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var raw map[string]any
	if err := json.Unmarshal(ctx.Response.Body(), &raw); err != nil {
		t.Fatalf("unmarshal raw quota: %v", err)
	}
	for _, key := range []string{"Provider", "Limit", "Used", "Remaining", "Unit"} {
		if _, ok := raw[key]; !ok {
			t.Fatalf("quota JSON missing %q: %s", key, ctx.Response.Body())
		}
	}
	for _, key := range []string{"provider", "limit", "used", "remaining", "unit"} {
		if _, ok := raw[key]; ok {
			t.Fatalf("quota JSON should not expose lower-case key %q: %s", key, ctx.Response.Body())
		}
	}
	if raw["Provider"] != string(providers.ProviderOpenAI) || raw["Unit"] != "credits" {
		t.Fatalf("quota JSON = %+v, want Provider openai and Unit credits", raw)
	}
}

func TestUsageQuotaUsesActiveStoredProviderConnection(t *testing.T) {
	s := openHandlerTestStore(t)
	apiKey := "sk-from-store"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "primary",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	fetcher := &fakeQuotaFetcher{
		quota: usage.Quota{Provider: providers.ProviderOpenAI, Limit: 10, Used: 2, Remaining: 8},
	}
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage/quota/openai")

	UsageQuota(ctx, s, map[providers.ModelProvider]usage.QuotaFetcher{
		providers.ProviderOpenAI: fetcher,
	}, providers.Key{Value: "static-key", AuthType: "api_key"})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if fetcher.gotKey.Value != apiKey || fetcher.gotKey.ConnID == "" || fetcher.gotKey.AuthType != "api_key" {
		t.Fatalf("key = %+v, want stored api key connection", fetcher.gotKey)
	}
}

func TestUsageQuotaHandlesUnsupportedProvider(t *testing.T) {
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage/quota/openai")

	UsageQuota(ctx, nil, nil, providers.Key{})

	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestUsageQuotaMapsUnsupportedFetcherError(t *testing.T) {
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage/quota/ollama")

	UsageQuota(ctx, nil, map[providers.ModelProvider]usage.QuotaFetcher{
		providers.ProviderOllama: &fakeQuotaFetcher{err: usage.ErrQuotaUnsupported},
	}, providers.Key{})

	if ctx.Response.StatusCode() != fasthttp.StatusNotImplemented {
		t.Fatalf("status = %d, want 501; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func openHandlerTestStore(t *testing.T) *store.Store {
	t.Helper()

	s, err := store.NewStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	return s
}

func newHandlerCtx(method, uri string) *fasthttp.RequestCtx {
	var req fasthttp.Request
	req.Header.SetMethod(method)
	req.SetRequestURI(uri)

	ctx := &fasthttp.RequestCtx{}
	ctx.Init(&req, &net.TCPAddr{}, nil)
	return ctx
}

func handlerUsageEntry(requestID, provider, model string, timestamp time.Time) store.RequestLogEntry {
	return store.RequestLogEntry{
		RequestID: requestID,
		Timestamp: timestamp,
		Provider:  provider,
		Model:     model,
		AuthType:  "api_key",
	}
}

func logHandlerEntries(t *testing.T, s *store.Store, entries []store.RequestLogEntry) {
	t.Helper()

	for i := range entries {
		if err := s.LogRequest(&entries[i]); err != nil {
			t.Fatalf("LogRequest %q: %v", entries[i].RequestID, err)
		}
	}
}

func intPtr(value int) *int {
	return &value
}

func floatPtr(value float64) *float64 {
	return &value
}

func TestUsageResponseIncludesAttributionFields(t *testing.T) {
	s := openHandlerTestStore(t)

	key, _, err := s.CreateAPIKey("handler-key", "testsecret")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	email := "handler@example.com"
	conn := &store.Connection{
		Provider: "openai",
		Name:     "handler-conn",
		AuthType: store.AuthTypeAPIKey,
		IsActive: true,
		Email:    &email,
	}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	conns, err := s.GetConnections("openai")
	if err != nil || len(conns) == 0 {
		t.Fatalf("GetConnections: %v (len=%d)", err, len(conns))
	}
	connID := conns[0].ID

	entry := store.RequestLogEntry{
		RequestID:    "req-attr",
		Timestamp:    time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC),
		Provider:     "openai",
		Model:        "gpt-4o",
		AuthType:     "api_key",
		APIKeyID:     &key.ID,
		ConnectionID: &connID,
	}
	logHandlerEntries(t, s, []store.RequestLogEntry{entry})

	ctx := newHandlerCtx("GET", "/api/usage")
	Usage(ctx, s)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}

	var decoded struct {
		Data []struct {
			APIKeyID           *string `json:"api_key_id"`
			APIKeyName         *string `json:"api_key_name"`
			ConnectionID       *string `json:"connection_id"`
			ConnectionName     *string `json:"connection_name"`
			ConnectionProvider *string `json:"connection_provider"`
			AccountEmail       *string `json:"account_email"`
		} `json:"data"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(decoded.Data) != 1 {
		t.Fatalf("data len = %d, want 1", len(decoded.Data))
	}
	row := decoded.Data[0]
	if row.APIKeyID == nil || *row.APIKeyID != key.ID {
		t.Fatalf("api_key_id = %v, want %s", row.APIKeyID, key.ID)
	}
	if row.APIKeyName == nil || *row.APIKeyName != "handler-key" {
		t.Fatalf("api_key_name = %v, want handler-key", row.APIKeyName)
	}
	if row.ConnectionID == nil || *row.ConnectionID != connID {
		t.Fatalf("connection_id = %v, want %s", row.ConnectionID, connID)
	}
	if row.ConnectionName == nil || *row.ConnectionName != "handler-conn" {
		t.Fatalf("connection_name = %v, want handler-conn", row.ConnectionName)
	}
	if row.ConnectionProvider == nil || *row.ConnectionProvider != "openai" {
		t.Fatalf("connection_provider = %v, want openai", row.ConnectionProvider)
	}
	if row.AccountEmail == nil || *row.AccountEmail != "handler@example.com" {
		t.Fatalf("account_email = %v, want handler@example.com", row.AccountEmail)
	}
}

func TestUsageFilterByAPIKeyID(t *testing.T) {
	s := openHandlerTestStore(t)

	key, _, err := s.CreateAPIKey("filter-key", "testsecret")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	base := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	withKey := handlerUsageEntry("req-keyed", "openai", "gpt-4o", base)
	withKey.APIKeyID = &key.ID
	noKey := handlerUsageEntry("req-nokey", "openai", "gpt-4o", base.Add(time.Minute))
	logHandlerEntries(t, s, []store.RequestLogEntry{withKey, noKey})

	ctx := newHandlerCtx("GET", "/api/usage?api_key_id="+key.ID)
	Usage(ctx, s)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var decoded struct {
		Data  []struct{ RequestID string `json:"request_id"` } `json:"data"`
		Total int                                               `json:"total"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(decoded.Data) != 1 || decoded.Data[0].RequestID != "req-keyed" {
		t.Fatalf("data = %+v, want only req-keyed", decoded.Data)
	}
	if decoded.Total != 1 {
		t.Fatalf("total = %d, want 1", decoded.Total)
	}
}

func TestUsageNullAttributionFieldsReturnNull(t *testing.T) {
	s := openHandlerTestStore(t)
	entry := handlerUsageEntry("req-null-attr", "openai", "gpt-4o", time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC))
	logHandlerEntries(t, s, []store.RequestLogEntry{entry})

	ctx := newHandlerCtx("GET", "/api/usage")
	Usage(ctx, s)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var decoded struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(decoded.Data) != 1 {
		t.Fatalf("data len = %d, want 1", len(decoded.Data))
	}
	row := decoded.Data[0]
	for _, field := range []string{"api_key_name", "connection_name", "connection_provider", "account_email"} {
		if v, ok := row[field]; ok && v != nil {
			t.Fatalf("field %q = %v, want null", field, v)
		}
	}
}

func TestUsageChartReturnsAggregatedData(t *testing.T) {
	s := openHandlerTestStore(t)
	now := time.Date(2026, 6, 6, 14, 30, 0, 0, time.UTC)

	base := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	entries := []store.RequestLogEntry{
		handlerChartEntry("req-a", base, 10, 5, 0.50),
		handlerChartEntry("req-b", base.Add(30*time.Minute), 20, 10, 1.00),
		handlerChartEntry("req-c", base.Add(2*time.Hour), 5, 2, 0.25),
	}
	logHandlerEntries(t, s, entries)

	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage/chart?period=today&granularity=hour")
	UsageChart(ctx, s, now)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}

	var decoded struct {
		Buckets      []string  `json:"buckets"`
		Requests     []int64   `json:"requests"`
		TokensInput  []int64   `json:"tokens_input"`
		TokensOutput []int64   `json:"tokens_output"`
		Costs        []float64 `json:"costs"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.Buckets) != 15 {
		t.Fatalf("buckets len = %d, want 15", len(decoded.Buckets))
	}
	if len(decoded.Buckets) != len(decoded.Requests) || len(decoded.Buckets) != len(decoded.TokensInput) ||
		len(decoded.Buckets) != len(decoded.TokensOutput) || len(decoded.Buckets) != len(decoded.Costs) {
		t.Fatalf("array lengths misaligned")
	}

	idx := make(map[string]int)
	for i, b := range decoded.Buckets {
		idx[b] = i
	}

	if decoded.Requests[idx["2026-06-06T10:00"]] != 2 {
		t.Fatalf("10:00 requests = %d, want 2", decoded.Requests[idx["2026-06-06T10:00"]])
	}
	if decoded.TokensInput[idx["2026-06-06T10:00"]] != 30 {
		t.Fatalf("10:00 tokens_input = %d, want 30", decoded.TokensInput[idx["2026-06-06T10:00"]])
	}
	if decoded.Requests[idx["2026-06-06T11:00"]] != 0 {
		t.Fatalf("11:00 requests = %d, want 0", decoded.Requests[idx["2026-06-06T11:00"]])
	}
}

func TestUsageChartInvalidPeriod(t *testing.T) {
	s := openHandlerTestStore(t)
	now := time.Date(2026, 6, 6, 14, 30, 0, 0, time.UTC)

	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage/chart?period=invalid&granularity=hour")
	UsageChart(ctx, s, now)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestUsageChartInvalidGranularity(t *testing.T) {
	s := openHandlerTestStore(t)
	now := time.Date(2026, 6, 6, 14, 30, 0, 0, time.UTC)

	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage/chart?period=7d&granularity=week")
	UsageChart(ctx, s, now)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestUsageChartDefaultsGranularityForToday(t *testing.T) {
	s := openHandlerTestStore(t)
	now := time.Date(2026, 6, 6, 14, 30, 0, 0, time.UTC)

	base := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	logHandlerEntries(t, s, []store.RequestLogEntry{handlerChartEntry("req-a", base, 10, 5, 0.50)})

	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage/chart?period=today")
	UsageChart(ctx, s, now)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var decoded struct {
		Buckets []string `json:"buckets"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(decoded.Buckets) != 15 {
		t.Fatalf("today default granularity = %d buckets, want 15 (hour)", len(decoded.Buckets))
	}
}

func TestUsageChartDefaultsGranularityFor7d(t *testing.T) {
	s := openHandlerTestStore(t)
	now := time.Date(2026, 6, 6, 14, 30, 0, 0, time.UTC)

	base := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	logHandlerEntries(t, s, []store.RequestLogEntry{handlerChartEntry("req-a", base, 10, 5, 0.50)})

	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage/chart?period=7d")
	UsageChart(ctx, s, now)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var decoded struct {
		Buckets []string `json:"buckets"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(decoded.Buckets) != 8 {
		t.Fatalf("7d default granularity = %d buckets, want 8 (day)", len(decoded.Buckets))
	}
}

func handlerChartEntry(requestID string, ts time.Time, inputTokens, outputTokens int, cost float64) store.RequestLogEntry {
	return store.RequestLogEntry{
		RequestID:    requestID,
		Timestamp:    ts,
		Provider:     "openai",
		Model:        "gpt-4o",
		AuthType:     "api_key",
		InputTokens:  &inputTokens,
		OutputTokens: &outputTokens,
		CostUSD:      &cost,
	}
}

func TestUsageChartNilStoreReturns503(t *testing.T) {
	now := time.Date(2026, 6, 6, 14, 30, 0, 0, time.UTC)
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage/chart?period=today")
	UsageChart(ctx, nil, now)
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestUsageChartMissingPeriodReturns400(t *testing.T) {
	s := openHandlerTestStore(t)
	now := time.Date(2026, 6, 6, 14, 30, 0, 0, time.UTC)
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage/chart")
	UsageChart(ctx, s, now)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestUsageChartStoreErrorReturns500(t *testing.T) {
	now := time.Date(2026, 6, 6, 14, 30, 0, 0, time.UTC)
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage/chart?period=today&granularity=hour")
	UsageChart(ctx, failingChartStore{}, now)
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

type failingChartStore struct{}

func (f failingChartStore) GetUsage(filter store.UsageFilter) ([]store.RequestLogEntry, error) {
	return nil, errors.New("db error")
}
func (f failingChartStore) GetUsageSummary(filter store.UsageFilter) (*store.UsageSummary, error) {
	return nil, errors.New("db error")
}
func (f failingChartStore) CountUsage(filter store.UsageFilter) (int, error) {
	return 0, errors.New("db error")
}
func (f failingChartStore) GetUsageChart(period, granularity string, now time.Time) (*store.UsageChart, error) {
	return nil, errors.New("db error")
}
