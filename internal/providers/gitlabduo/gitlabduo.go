package gitlabduo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/providers/anthropic"
	"github.com/bloodf/g0router/internal/providers/openaicompat"
	"github.com/bloodf/g0router/internal/store"
)

const (
	defaultGitLabURL  = "https://gitlab.com"
	defaultGatewayURL = "https://cloud.gitlab.com"
	directAccessTTL   = 25 * time.Minute
)

type Config struct {
	GitLabURL  string
	GatewayURL string
	HTTPClient *http.Client
	Now        func() time.Time
	ProxyPool  *store.ProxyPool
}

type Provider struct {
	gitLabURL  string
	gatewayURL string
	client     *http.Client
	now        func() time.Time
	mu         sync.Mutex
	cache      map[string]directAccessToken
	proxyPool  *store.ProxyPool
}

type directAccessToken struct {
	Token     string            `json:"token"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt time.Time
}

type modelMapping struct {
	Provider string
	Model    string
}

type modelAlias struct {
	ID      string
	Mapping modelMapping
}

var modelAliases = [...]modelAlias{
	{ID: "duo-chat-opus-4-6", Mapping: modelMapping{Provider: "anthropic", Model: "claude-opus-4-6"}},
	{ID: "duo-chat-sonnet-4-6", Mapping: modelMapping{Provider: "anthropic", Model: "claude-sonnet-4-6"}},
	{ID: "duo-chat-opus-4-5", Mapping: modelMapping{Provider: "anthropic", Model: "claude-opus-4-5-20251101"}},
	{ID: "duo-chat-sonnet-4-5", Mapping: modelMapping{Provider: "anthropic", Model: "claude-sonnet-4-5-20250929"}},
	{ID: "duo-chat-haiku-4-5", Mapping: modelMapping{Provider: "anthropic", Model: "claude-haiku-4-5-20251001"}},
	{ID: "duo-chat-gpt-5-1", Mapping: modelMapping{Provider: "openai", Model: "gpt-5.1-2025-11-13"}},
	{ID: "duo-chat-gpt-5-2", Mapping: modelMapping{Provider: "openai", Model: "gpt-5.2-2025-12-11"}},
	{ID: "duo-chat-gpt-5-mini", Mapping: modelMapping{Provider: "openai", Model: "gpt-5-mini-2025-08-07"}},
	{ID: "duo-chat-gpt-5-codex", Mapping: modelMapping{Provider: "openai", Model: "gpt-5-codex"}},
	{ID: "duo-chat-gpt-5-2-codex", Mapping: modelMapping{Provider: "openai", Model: "gpt-5.2-codex"}},
}

func New(config Config) *Provider {
	gitLabURL := config.GitLabURL
	if gitLabURL == "" {
		gitLabURL = defaultGitLabURL
	}
	gatewayURL := config.GatewayURL
	if gatewayURL == "" {
		gatewayURL = defaultGatewayURL
	}
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	return &Provider{
		gitLabURL:  strings.TrimRight(gitLabURL, "/"),
		gatewayURL: strings.TrimRight(gatewayURL, "/"),
		client:     client,
		now:        now,
		cache:      make(map[string]directAccessToken),
		proxyPool:  config.ProxyPool,
	}
}

func NewDefault() *Provider {
	return New(Config{})
}

func (p *Provider) WithProxyPool(pool *store.ProxyPool) providers.Provider {
	return New(Config{
		GitLabURL:  p.gitLabURL,
		GatewayURL: p.gatewayURL,
		HTTPClient: p.client,
		Now:        p.now,
		ProxyPool:  pool,
	})
}

func (p *Provider) Name() providers.ModelProvider {
	return providers.ProviderGitLabDuo
}

func (p *Provider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	mapping, err := mappedRequest(req)
	if err != nil {
		return nil, err
	}
	directAccess, err := p.directAccess(ctx, key.Value)
	if err != nil {
		return nil, err
	}
	dispatchReq := requestWithModel(req, mapping.Model)
	dispatchKey := providers.Key{Value: directAccess.Token, Provider: providers.ProviderGitLabDuo, ConnID: key.ConnID, AuthType: "oauth"}
	switch mapping.Provider {
	case "openai":
		provider, err := openaicompat.New(openaicompat.Config{
			Provider:  providers.ProviderGitLabDuo,
			BaseURL:   p.gatewayURL + "/ai/v1/proxy/openai/v1",
			Headers:   directAccess.Headers,
			ProxyPool: p.proxyPool,
		})
		if err != nil {
			return nil, err
		}
		return provider.ChatCompletion(ctx, dispatchKey, dispatchReq)
	case "anthropic":
		provider := anthropic.NewForProviderWithHeaders(providers.ProviderGitLabDuo, p.gatewayURL+"/ai/v1/proxy/anthropic", directAccess.Headers, p.proxyPool)
		return provider.ChatCompletion(ctx, dispatchKey, dispatchReq)
	default:
		return nil, fmt.Errorf("gitlab-duo unsupported mapped provider %q", mapping.Provider)
	}
}

func (p *Provider) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	mapping, err := mappedRequest(req)
	if err != nil {
		return nil, err
	}
	directAccess, err := p.directAccess(ctx, key.Value)
	if err != nil {
		return nil, err
	}
	dispatchReq := requestWithModel(req, mapping.Model)
	dispatchKey := providers.Key{Value: directAccess.Token, Provider: providers.ProviderGitLabDuo, ConnID: key.ConnID, AuthType: "oauth"}
	switch mapping.Provider {
	case "openai":
		provider, err := openaicompat.New(openaicompat.Config{
			Provider:  providers.ProviderGitLabDuo,
			BaseURL:   p.gatewayURL + "/ai/v1/proxy/openai/v1",
			Headers:   directAccess.Headers,
			ProxyPool: p.proxyPool,
		})
		if err != nil {
			return nil, err
		}
		return provider.ChatCompletionStream(ctx, dispatchKey, dispatchReq)
	case "anthropic":
		provider := anthropic.NewForProviderWithHeaders(providers.ProviderGitLabDuo, p.gatewayURL+"/ai/v1/proxy/anthropic", directAccess.Headers, p.proxyPool)
		return provider.ChatCompletionStream(ctx, dispatchKey, dispatchReq)
	default:
		return nil, fmt.Errorf("gitlab-duo unsupported mapped provider %q", mapping.Provider)
	}
}

func (p *Provider) ListModels(context.Context, providers.Key) ([]providers.Model, error) {
	ids := make([]string, 0, len(modelAliases))
	for _, alias := range modelAliases {
		ids = append(ids, alias.ID)
	}
	sort.Strings(ids)

	models := make([]providers.Model, 0, len(ids))
	for _, id := range ids {
		models = append(models, providers.Model{ID: id, Object: "model", OwnedBy: "gitlab-duo", Provider: providers.ProviderGitLabDuo})
	}
	return models, nil
}

func (p *Provider) directAccess(ctx context.Context, gitLabToken string) (directAccessToken, error) {
	p.mu.Lock()
	cached, ok := p.cache[gitLabToken]
	if ok && cached.ExpiresAt.After(p.now()) {
		p.mu.Unlock()
		return cached, nil
	}
	p.mu.Unlock()

	body := []byte(`{"feature_flags":{"DuoAgentPlatformNext":true}}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.gitLabURL+"/api/v4/ai/third_party_agents/direct_access", bytes.NewReader(body))
	if err != nil {
		return directAccessToken{}, fmt.Errorf("gitlab-duo direct access request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+gitLabToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return directAccessToken{}, fmt.Errorf("gitlab-duo direct access: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return directAccessToken{}, fmt.Errorf("gitlab-duo direct access response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return directAccessToken{}, fmt.Errorf("gitlab-duo direct access status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var decoded struct {
		Token   string            `json:"token"`
		Headers map[string]string `json:"headers"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return directAccessToken{}, fmt.Errorf("parse gitlab-duo direct access response: %w", err)
	}
	if decoded.Token == "" {
		return directAccessToken{}, fmt.Errorf("gitlab-duo direct access response missing token")
	}
	if decoded.Headers == nil {
		decoded.Headers = map[string]string{}
	}
	token := directAccessToken{Token: decoded.Token, Headers: decoded.Headers, ExpiresAt: p.now().Add(directAccessTTL)}
	p.mu.Lock()
	p.cache[gitLabToken] = token
	p.mu.Unlock()
	return token, nil
}

func mappedRequest(req *providers.ChatRequest) (modelMapping, error) {
	if req == nil {
		return modelMapping{}, fmt.Errorf("gitlab-duo request: nil chat request")
	}
	for _, alias := range modelAliases {
		if alias.ID == req.Model {
			return alias.Mapping, nil
		}
	}
	return modelMapping{}, fmt.Errorf("unsupported gitlab-duo model: %s", req.Model)
}

func requestWithModel(req *providers.ChatRequest, model string) *providers.ChatRequest {
	copyReq := *req
	copyReq.Model = model
	return &copyReq
}
