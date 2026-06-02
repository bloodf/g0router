package usage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

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
