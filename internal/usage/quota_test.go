package usage

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
)

func TestHTTPQuotaFetcherFetchesQuota(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Fatalf("authorization = %q, want bearer token", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"limit":1000,"used":125,"remaining":875}`))
	}))
	defer server.Close()

	fetcher := NewHTTPQuotaFetcher(providers.ProviderOpenAI, server.URL, server.Client())
	got, err := fetcher.FetchQuota(context.Background(), providers.Key{
		Value:    "sk-test",
		Provider: providers.ProviderOpenAI,
	})
	if err != nil {
		t.Fatalf("FetchQuota: %v", err)
	}
	if got.Provider != providers.ProviderOpenAI {
		t.Fatalf("provider = %s, want openai", got.Provider)
	}
	if got.Limit != 1000 || got.Used != 125 || got.Remaining != 875 {
		t.Fatalf("quota = %+v, want 1000/125/875", got)
	}
}

func TestHTTPQuotaFetcherReturnsErrorForProviderFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "quota unavailable", http.StatusTooManyRequests)
	}))
	defer server.Close()

	fetcher := NewHTTPQuotaFetcher(providers.ProviderOpenAI, server.URL, server.Client())
	_, err := fetcher.FetchQuota(context.Background(), providers.Key{Value: "sk-test"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUnsupportedQuotaFetcherReturnsExplicitError(t *testing.T) {
	fetcher := NewUnsupportedQuotaFetcher(providers.ProviderOllama)
	_, err := fetcher.FetchQuota(context.Background(), providers.Key{Provider: providers.ProviderOllama})
	if !errors.Is(err, ErrQuotaUnsupported) {
		t.Fatalf("error = %v, want ErrQuotaUnsupported", err)
	}
}

func TestCachingQuotaFetcherReturnsCachedQuotaWithinTTL(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	inner := &fakeQuotaFetcher{
		quotas: []Quota{
			{Provider: providers.ProviderOpenAI, Limit: 1000, Used: 100, Remaining: 900},
			{Provider: providers.ProviderOpenAI, Limit: 1000, Used: 900, Remaining: 100},
		},
	}
	fetcher := NewCachingQuotaFetcher(inner, 5*time.Minute)
	fetcher.now = func() time.Time { return now }
	key := providers.Key{Provider: providers.ProviderOpenAI, Value: "sk-test", ConnID: "conn-1"}

	first, err := fetcher.FetchQuota(context.Background(), key)
	if err != nil {
		t.Fatalf("first FetchQuota: %v", err)
	}
	now = now.Add(4 * time.Minute)
	second, err := fetcher.FetchQuota(context.Background(), key)
	if err != nil {
		t.Fatalf("second FetchQuota: %v", err)
	}

	if inner.calls != 1 {
		t.Fatalf("inner calls = %d, want 1", inner.calls)
	}
	if second != first {
		t.Fatalf("second quota = %+v, want cached %+v", second, first)
	}
}

func TestCachingQuotaFetcherRefreshesAfterTTL(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	inner := &fakeQuotaFetcher{
		quotas: []Quota{
			{Provider: providers.ProviderOpenAI, Limit: 1000, Used: 100, Remaining: 900},
			{Provider: providers.ProviderOpenAI, Limit: 1000, Used: 900, Remaining: 100},
		},
	}
	fetcher := NewCachingQuotaFetcher(inner, 5*time.Minute)
	fetcher.now = func() time.Time { return now }
	key := providers.Key{Provider: providers.ProviderOpenAI, Value: "sk-test", ConnID: "conn-1"}

	if _, err := fetcher.FetchQuota(context.Background(), key); err != nil {
		t.Fatalf("first FetchQuota: %v", err)
	}
	now = now.Add(5*time.Minute + time.Nanosecond)
	got, err := fetcher.FetchQuota(context.Background(), key)
	if err != nil {
		t.Fatalf("second FetchQuota: %v", err)
	}

	if inner.calls != 2 {
		t.Fatalf("inner calls = %d, want 2", inner.calls)
	}
	if got.Remaining != 100 {
		t.Fatalf("remaining = %d, want refreshed quota", got.Remaining)
	}
}

func TestCachingQuotaFetcherDoesNotCacheErrors(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	wantErr := errors.New("quota API unavailable")
	inner := &fakeQuotaFetcher{
		errs:   []error{wantErr, nil},
		quotas: []Quota{{Provider: providers.ProviderOpenAI, Limit: 1000, Used: 200, Remaining: 800}},
	}
	fetcher := NewCachingQuotaFetcher(inner, 5*time.Minute)
	fetcher.now = func() time.Time { return now }
	key := providers.Key{Provider: providers.ProviderOpenAI, Value: "sk-test", ConnID: "conn-1"}

	_, err := fetcher.FetchQuota(context.Background(), key)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	got, err := fetcher.FetchQuota(context.Background(), key)
	if err != nil {
		t.Fatalf("second FetchQuota: %v", err)
	}

	if inner.calls != 2 {
		t.Fatalf("inner calls = %d, want retry after error", inner.calls)
	}
	if got.Remaining != 800 {
		t.Fatalf("remaining = %d, want successful retry quota", got.Remaining)
	}
}

type fakeQuotaFetcher struct {
	calls  int
	quotas []Quota
	errs   []error
}

func (f *fakeQuotaFetcher) FetchQuota(ctx context.Context, key providers.Key) (Quota, error) {
	f.calls++
	index := f.calls - 1
	if index < len(f.errs) && f.errs[index] != nil {
		return Quota{}, f.errs[index]
	}
	if index < len(f.quotas) {
		return f.quotas[index], nil
	}
	return f.quotas[len(f.quotas)-1], nil
}
