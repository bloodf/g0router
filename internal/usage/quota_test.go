package usage

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

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
