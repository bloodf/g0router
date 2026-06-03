package usage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/bloodf/g0router/internal/providers"
)

var ErrQuotaUnsupported = errors.New("usage: quota unsupported")

type Quota struct {
	Provider  providers.ModelProvider
	Limit     int64
	Used      int64
	Remaining int64
}

type QuotaFetcher interface {
	FetchQuota(ctx context.Context, key providers.Key) (Quota, error)
}

type CachingQuotaFetcher struct {
	fetcher QuotaFetcher
	ttl     time.Duration
	now     func() time.Time

	mu      sync.Mutex
	entries map[providers.Key]cachedQuota
}

type cachedQuota struct {
	quota     Quota
	expiresAt time.Time
}

func NewCachingQuotaFetcher(fetcher QuotaFetcher, ttl time.Duration) *CachingQuotaFetcher {
	return &CachingQuotaFetcher{
		fetcher: fetcher,
		ttl:     ttl,
		now:     time.Now,
		entries: make(map[providers.Key]cachedQuota),
	}
}

func (f *CachingQuotaFetcher) FetchQuota(ctx context.Context, key providers.Key) (Quota, error) {
	if f.ttl <= 0 {
		return f.fetcher.FetchQuota(ctx, key)
	}

	now := f.now()
	f.mu.Lock()
	entry, ok := f.entries[key]
	if ok && now.Before(entry.expiresAt) {
		f.mu.Unlock()
		return entry.quota, nil
	}
	f.mu.Unlock()

	quota, err := f.fetcher.FetchQuota(ctx, key)
	if err != nil {
		return Quota{}, err
	}

	f.mu.Lock()
	f.entries[key] = cachedQuota{
		quota:     quota,
		expiresAt: f.now().Add(f.ttl),
	}
	f.mu.Unlock()

	return quota, nil
}

type HTTPQuotaFetcher struct {
	provider providers.ModelProvider
	endpoint string
	client   *http.Client
}

func NewHTTPQuotaFetcher(provider providers.ModelProvider, endpoint string, client *http.Client) *HTTPQuotaFetcher {
	if client == nil {
		client = http.DefaultClient
	}

	return &HTTPQuotaFetcher{
		provider: provider,
		endpoint: endpoint,
		client:   client,
	}
}

func (f *HTTPQuotaFetcher) FetchQuota(ctx context.Context, key providers.Key) (Quota, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.endpoint, nil)
	if err != nil {
		return Quota{}, fmt.Errorf("create quota request: %w", err)
	}
	if key.Value != "" {
		req.Header.Set("Authorization", "Bearer "+key.Value)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return Quota{}, fmt.Errorf("fetch quota: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return Quota{}, fmt.Errorf("fetch quota: provider returned %s", resp.Status)
	}

	var quota Quota
	if err := json.NewDecoder(resp.Body).Decode(&quota); err != nil {
		return Quota{}, fmt.Errorf("decode quota: %w", err)
	}
	quota.Provider = f.provider

	return quota, nil
}

type UnsupportedQuotaFetcher struct {
	provider providers.ModelProvider
}

func NewUnsupportedQuotaFetcher(provider providers.ModelProvider) *UnsupportedQuotaFetcher {
	return &UnsupportedQuotaFetcher{provider: provider}
}

func (f *UnsupportedQuotaFetcher) FetchQuota(ctx context.Context, key providers.Key) (Quota, error) {
	return Quota{}, fmt.Errorf("%s quota: %w", f.provider, ErrQuotaUnsupported)
}
