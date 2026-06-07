package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
	"github.com/valyala/fasthttp"
)

type fakeQuotaAggregateStore struct {
	connections []*store.Connection
	err         error
}

func (f *fakeQuotaAggregateStore) ListConnections() ([]*store.Connection, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.connections, nil
}

type fakeAggQuotaFetcher struct {
	quota usage.Quota
	err   error
}

func (f *fakeAggQuotaFetcher) FetchQuota(ctx context.Context, key providers.Key) (usage.Quota, error) {
	if f.err != nil {
		return usage.Quota{}, f.err
	}
	return f.quota, nil
}

func TestQuotaAggregateReturnsQuotas(t *testing.T) {
	apiKey := "sk-test"
	s := &fakeQuotaAggregateStore{
		connections: []*store.Connection{
			{ID: "conn-1", Provider: "openai", Name: "OpenAI Prod", AuthType: store.AuthTypeAPIKey, APIKey: &apiKey, IsActive: true, Email: strPtr("user@example.com")},
		},
	}
	fetchers := map[providers.ModelProvider]usage.QuotaFetcher{
		providers.ProviderOpenAI: &fakeAggQuotaFetcher{
			quota: usage.Quota{Provider: providers.ProviderOpenAI, Limit: 10000, Used: 5000, Remaining: 5000, Unit: "tokens"},
		},
	}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		QuotaAggregate(ctx, s, fetchers)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded []struct {
		ConnectionID   string  `json:"connection_id"`
		Provider       string  `json:"provider"`
		ConnectionName string  `json:"connection_name"`
		AccountLabel   *string `json:"account_label"`
		Plan           string  `json:"plan"`
		Used           float64 `json:"used"`
		Limit          float64 `json:"limit"`
		Unit           string  `json:"unit"`
		ResetAt        *string `json:"reset_at"`
		IsActive       bool    `json:"is_active"`
		Message        *string `json:"message"`
		Error          *string `json:"error"`
	}
	decodeJSON(t, body, &decoded)
	if len(decoded) != 1 {
		t.Fatalf("len = %d, want 1", len(decoded))
	}

	row := decoded[0]
	if row.ConnectionID != "conn-1" {
		t.Errorf("connection_id = %q, want conn-1", row.ConnectionID)
	}
	if row.Provider != "openai" {
		t.Errorf("provider = %q, want openai", row.Provider)
	}
	if row.ConnectionName != "OpenAI Prod" {
		t.Errorf("connection_name = %q, want OpenAI Prod", row.ConnectionName)
	}
	if row.AccountLabel == nil || *row.AccountLabel != "user@example.com" {
		t.Errorf("account_label = %v, want user@example.com", row.AccountLabel)
	}
	if row.Used != 5000 {
		t.Errorf("used = %v, want 5000", row.Used)
	}
	if row.Limit != 10000 {
		t.Errorf("limit = %v, want 10000", row.Limit)
	}
	if row.Unit != "tokens" {
		t.Errorf("unit = %q, want tokens", row.Unit)
	}
	if !row.IsActive {
		t.Error("is_active should be true")
	}
	if row.Error != nil {
		t.Errorf("error = %v, want nil", row.Error)
	}
}

func TestQuotaAggregateNilStoreReturns503(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		QuotaAggregate(ctx, nil, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestQuotaAggregateStoreErrorReturns500(t *testing.T) {
	s := &fakeQuotaAggregateStore{err: errors.New("db error")}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		QuotaAggregate(ctx, s, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestQuotaAggregateHandlesInactiveConnections(t *testing.T) {
	s := &fakeQuotaAggregateStore{
		connections: []*store.Connection{
			{ID: "conn-1", Provider: "openai", Name: "Active", IsActive: true},
			{ID: "conn-2", Provider: "anthropic", Name: "Inactive", IsActive: false},
		},
	}
	fetchers := map[providers.ModelProvider]usage.QuotaFetcher{
		providers.ProviderOpenAI: &fakeAggQuotaFetcher{quota: usage.Quota{Limit: 100, Used: 10}},
	}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		QuotaAggregate(ctx, s, fetchers)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded []struct {
		ConnectionID string `json:"connection_id"`
		IsActive     bool   `json:"is_active"`
	}
	decodeJSON(t, body, &decoded)
	if len(decoded) != 2 {
		t.Fatalf("len = %d, want 2", len(decoded))
	}
	if !decoded[0].IsActive {
		t.Error("first should be active")
	}
	if decoded[1].IsActive {
		t.Error("second should be inactive")
	}
}

func TestQuotaAggregateHandlesFetcherError(t *testing.T) {
	apiKey := "sk-test"
	s := &fakeQuotaAggregateStore{
		connections: []*store.Connection{
			{ID: "conn-1", Provider: "openai", Name: "OpenAI", AuthType: store.AuthTypeAPIKey, APIKey: &apiKey, IsActive: true},
		},
	}
	fetchers := map[providers.ModelProvider]usage.QuotaFetcher{
		providers.ProviderOpenAI: &fakeAggQuotaFetcher{err: errors.New("quota error")},
	}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		QuotaAggregate(ctx, s, fetchers)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded []struct {
		Error *string `json:"error"`
	}
	decodeJSON(t, body, &decoded)
	if len(decoded) != 1 {
		t.Fatalf("len = %d, want 1", len(decoded))
	}
	if decoded[0].Error == nil {
		t.Fatal("expected error field for failed fetch")
	}
}

func TestQuotaAggregateHandlesMissingFetcher(t *testing.T) {
	s := &fakeQuotaAggregateStore{
		connections: []*store.Connection{
			{ID: "conn-1", Provider: "ollama", Name: "Local", IsActive: true},
		},
	}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		QuotaAggregate(ctx, s, map[providers.ModelProvider]usage.QuotaFetcher{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded []struct {
		ConnectionID string  `json:"connection_id"`
		Error        *string `json:"error"`
	}
	decodeJSON(t, body, &decoded)
	if len(decoded) != 1 {
		t.Fatalf("len = %d, want 1", len(decoded))
	}
	if decoded[0].Error == nil {
		t.Fatal("expected error field for missing fetcher")
	}
}

func TestQuotaAggregateUnsupportedProvider(t *testing.T) {
	apiKey := "sk-test"
	s := &fakeQuotaAggregateStore{
		connections: []*store.Connection{
			{ID: "conn-1", Provider: "openai", Name: "OpenAI", AuthType: store.AuthTypeAPIKey, APIKey: &apiKey, IsActive: true},
		},
	}
	fetchers := map[providers.ModelProvider]usage.QuotaFetcher{
		providers.ProviderOpenAI: &fakeAggQuotaFetcher{err: usage.ErrQuotaUnsupported},
	}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		QuotaAggregate(ctx, s, fetchers)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded []struct {
		Error *string `json:"error"`
	}
	decodeJSON(t, body, &decoded)
	if len(decoded) != 1 {
		t.Fatalf("len = %d, want 1", len(decoded))
	}
	if decoded[0].Error == nil {
		t.Fatal("expected error field for unsupported provider")
	}
}

func TestQuotaAggregateUnlimitedQuota(t *testing.T) {
	apiKey := "sk-test"
	s := &fakeQuotaAggregateStore{
		connections: []*store.Connection{
			{ID: "conn-1", Provider: "openai", Name: "OpenAI", AuthType: store.AuthTypeAPIKey, APIKey: &apiKey, IsActive: true},
		},
	}
	fetchers := map[providers.ModelProvider]usage.QuotaFetcher{
		providers.ProviderOpenAI: &fakeAggQuotaFetcher{
			quota: usage.Quota{Provider: providers.ProviderOpenAI, Unlimited: true, Used: 100, Unit: "credits"},
		},
	}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		QuotaAggregate(ctx, s, fetchers)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded []struct {
		Limit float64 `json:"limit"`
		Used  float64 `json:"used"`
	}
	decodeJSON(t, body, &decoded)
	if len(decoded) != 1 {
		t.Fatalf("len = %d, want 1", len(decoded))
	}
	if decoded[0].Limit != 0 {
		t.Errorf("limit = %v, want 0 for unlimited", decoded[0].Limit)
	}
	if decoded[0].Used != 100 {
		t.Errorf("used = %v, want 100", decoded[0].Used)
	}
}

func TestQuotaAggregateNoValidCredentials(t *testing.T) {
	s := &fakeQuotaAggregateStore{
		connections: []*store.Connection{
			{ID: "conn-1", Provider: "openai", Name: "OpenAI", AuthType: store.AuthTypeAPIKey, IsActive: true},
		},
	}
	fetchers := map[providers.ModelProvider]usage.QuotaFetcher{
		providers.ProviderOpenAI: &fakeAggQuotaFetcher{quota: usage.Quota{Limit: 100}},
	}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		QuotaAggregate(ctx, s, fetchers)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded []struct {
		Error *string `json:"error"`
	}
	decodeJSON(t, body, &decoded)
	if len(decoded) != 1 {
		t.Fatalf("len = %d, want 1", len(decoded))
	}
	if decoded[0].Error == nil {
		t.Fatal("expected error field for missing credentials")
	}
}

func TestQuotaAggregateAccountLabelFromAccountID(t *testing.T) {
	s := &fakeQuotaAggregateStore{
		connections: []*store.Connection{
			{ID: "conn-1", Provider: "openai", Name: "OpenAI", IsActive: true, AccountID: strPtr("acct-123")},
		},
	}
	fetchers := map[providers.ModelProvider]usage.QuotaFetcher{
		providers.ProviderOpenAI: &fakeAggQuotaFetcher{quota: usage.Quota{Limit: 100}},
	}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		QuotaAggregate(ctx, s, fetchers)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded []struct {
		AccountLabel *string `json:"account_label"`
	}
	decodeJSON(t, body, &decoded)
	if len(decoded) != 1 {
		t.Fatalf("len = %d, want 1", len(decoded))
	}
	if decoded[0].AccountLabel == nil || *decoded[0].AccountLabel != "acct-123" {
		t.Errorf("account_label = %v, want acct-123", decoded[0].AccountLabel)
	}
}
