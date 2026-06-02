package handlers

import (
	"context"
	"encoding/json"
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
	quota  usage.Quota
	err    error
}

func (f *fakeQuotaFetcher) FetchQuota(ctx context.Context, key providers.Key) (usage.Quota, error) {
	f.gotKey = key
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
		TotalCostUSD float64 `json:"total_cost_usd"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &decoded); err != nil {
		t.Fatalf("unmarshal summary: %v", err)
	}
	if decoded.RequestCount != 2 || decoded.TotalTokens != 30 || decoded.TotalCostUSD != 1.0 {
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

	UsageQuota(ctx, map[providers.ModelProvider]usage.QuotaFetcher{
		providers.ProviderOpenAI: fetcher,
	}, providers.Key{Value: "sk-test", AuthType: "api_key"})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if fetcher.gotKey.Provider != providers.ProviderOpenAI || fetcher.gotKey.Value != "sk-test" {
		t.Fatalf("key = %+v, want openai/sk-test", fetcher.gotKey)
	}
	var decoded usage.Quota
	if err := json.Unmarshal(ctx.Response.Body(), &decoded); err != nil {
		t.Fatalf("unmarshal quota: %v", err)
	}
	if decoded.Limit != 1000 || decoded.Used != 125 || decoded.Remaining != 875 {
		t.Fatalf("quota = %+v, want 1000/125/875", decoded)
	}
}

func TestUsageQuotaHandlesUnsupportedProvider(t *testing.T) {
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage/quota/openai")

	UsageQuota(ctx, nil, providers.Key{})

	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestUsageQuotaMapsUnsupportedFetcherError(t *testing.T) {
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage/quota/ollama")

	UsageQuota(ctx, map[providers.ModelProvider]usage.QuotaFetcher{
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
