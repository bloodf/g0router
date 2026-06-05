package usage

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/modelcatalog"
	"github.com/bloodf/g0router/internal/providers"
)

type erroringOverrides struct{ err error }

func (e erroringOverrides) PricingOverride(provider, model string) (PricingOverride, bool, error) {
	return PricingOverride{}, false, e.err
}

func TestCalculateCostUSDOverrideResolverError(t *testing.T) {
	usage := Usage{InputTokens: 10, OutputTokens: 5}
	wantErr := errors.New("db down")
	_, err := CalculateCostUSDWithOverrides(modelcatalog.NewCatalog(), erroringOverrides{wantErr}, providers.ProviderOpenAI, "gpt-4o", &usage)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}

func TestFromStreamChunksFindsLastUsage(t *testing.T) {
	chunks := []providers.StreamChunk{
		{Usage: nil},
		{Usage: &providers.Usage{PromptTokens: 5, CompletionTokens: 1, TotalTokens: 6}},
		{Usage: nil},
	}
	got, ok := FromStreamChunks(chunks)
	if !ok {
		t.Fatal("want usage found")
	}
	if got.InputTokens != 5 || got.OutputTokens != 1 {
		t.Fatalf("usage = %+v", got)
	}
}

func TestFromStreamChunksNoUsage(t *testing.T) {
	if _, ok := FromStreamChunks([]providers.StreamChunk{{Usage: nil}}); ok {
		t.Fatal("want no usage")
	}
	if _, ok := FromStreamChunks(nil); ok {
		t.Fatal("want no usage for nil")
	}
}

func TestIsOpenRouterQuotaFetcherAllBranches(t *testing.T) {
	or := NewOpenRouterQuotaFetcher("", nil)
	if !IsOpenRouterQuotaFetcher(or) {
		t.Fatal("direct OpenRouter fetcher should match")
	}
	cached := NewCachingQuotaFetcher(or, 0)
	if !IsOpenRouterQuotaFetcher(cached) {
		t.Fatal("cached OpenRouter fetcher should match")
	}
	http := NewHTTPQuotaFetcher(providers.ProviderOpenAI, "http://x", nil)
	if IsOpenRouterQuotaFetcher(http) {
		t.Fatal("HTTP fetcher should not match")
	}
	if IsOpenRouterQuotaFetcher(NewCachingQuotaFetcher(http, 0)) {
		t.Fatal("cached HTTP fetcher should not match")
	}
}

func TestNewQuotaFetchersDefaultClient(t *testing.T) {
	if NewHTTPQuotaFetcher(providers.ProviderOpenAI, "http://x", nil).client != http.DefaultClient {
		t.Fatal("HTTP fetcher should default client")
	}
	or := NewOpenRouterQuotaFetcher("", nil)
	if or.client != http.DefaultClient {
		t.Fatal("OpenRouter fetcher should default client")
	}
	if or.endpoint != "https://openrouter.ai/api/v1/key" {
		t.Fatalf("default endpoint = %q", or.endpoint)
	}
}

func TestHTTPQuotaFetcherRequestCreationError(t *testing.T) {
	// Control character in URL forces http.NewRequestWithContext to fail.
	f := NewHTTPQuotaFetcher(providers.ProviderOpenAI, "http://\x7f.example", nil)
	if _, err := f.FetchQuota(context.Background(), providers.Key{}); err == nil {
		t.Fatal("want request creation error")
	}
	or := NewOpenRouterQuotaFetcher("http://\x7f.example", nil)
	if _, err := or.FetchQuota(context.Background(), providers.Key{}); err == nil {
		t.Fatal("want openrouter request creation error")
	}
}

func TestHTTPQuotaFetcherDecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()
	f := NewHTTPQuotaFetcher(providers.ProviderOpenAI, server.URL, server.Client())
	if _, err := f.FetchQuota(context.Background(), providers.Key{}); err == nil {
		t.Fatal("want decode error")
	}
	or := NewOpenRouterQuotaFetcher(server.URL, server.Client())
	if _, err := or.FetchQuota(context.Background(), providers.Key{}); err == nil {
		t.Fatal("want openrouter decode error")
	}
}

func TestHTTPQuotaFetcherNoAuthHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Fatal("expected no auth header for empty key")
		}
		_, _ = w.Write([]byte(`{"limit":1,"used":0,"remaining":1}`))
	}))
	defer server.Close()
	f := NewHTTPQuotaFetcher(providers.ProviderOpenAI, server.URL, server.Client())
	if _, err := f.FetchQuota(context.Background(), providers.Key{}); err != nil {
		t.Fatalf("FetchQuota: %v", err)
	}
}

func TestHTTPQuotaFetcherDoError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	client := server.Client()
	server.Close() // connection refused on next request
	f := NewHTTPQuotaFetcher(providers.ProviderOpenAI, url, client)
	if _, err := f.FetchQuota(context.Background(), providers.Key{Value: "k"}); err == nil {
		t.Fatal("want client.Do error")
	}
}

func TestCachingQuotaFetcherTTLZeroPassthrough(t *testing.T) {
	inner := &fakeQuotaFetcher{quotas: []Quota{{Provider: providers.ProviderOpenAI, Limit: 1}}}
	f := NewCachingQuotaFetcher(inner, 0)
	if _, err := f.FetchQuota(context.Background(), providers.Key{}); err != nil {
		t.Fatalf("FetchQuota: %v", err)
	}
	if _, err := f.FetchQuota(context.Background(), providers.Key{}); err != nil {
		t.Fatalf("FetchQuota: %v", err)
	}
	if inner.calls != 2 {
		t.Fatalf("calls = %d, want 2 (no caching)", inner.calls)
	}
}
