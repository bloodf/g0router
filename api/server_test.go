package api

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/bloodf/g0router/api/handlers"
	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
)

func TestHealthz(t *testing.T) {
	srv := NewServer(ServerConfig{Port: 0, Version: "test-version"})
	ln := srv.listener()
	if ln == nil {
		t.Fatal("listener failed")
	}

	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Stop() })

	resp, err := httpClient().Get("http://" + localhostAddr(t, ln) + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("status: %q", result["status"])
	}
	if result["version"] != "test-version" {
		t.Errorf("version: %q", result["version"])
	}
}

func TestUnknownRoute(t *testing.T) {
	srv := NewServer(ServerConfig{Port: 0, Version: "test"})
	ln := srv.listener()
	if ln == nil {
		t.Fatal("listener failed")
	}

	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Stop() })

	resp, err := httpClient().Get("http://" + localhostAddr(t, ln) + "/nope")
	if err != nil {
		t.Fatalf("GET /nope: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestManagementRoutesDispatchThroughServer(t *testing.T) {
	store := newAPITestStore(t)
	_, baseURL := startTestServer(t, ServerConfig{
		Port:          0,
		Version:       "test",
		Store:         store,
		APIKeySecret:  "test-secret",
		ModelSource:   routeModelSource{},
		OAuthFlows:    handlers.OAuthFlows{"minimax": routeOAuthFlow{}},
		UsageStore:    store,
		QuotaFetchers: map[providers.ModelProvider]usage.QuotaFetcher{providers.ProviderOpenAI: routeQuotaFetcher{}},
		QuotaKey:      providers.Key{Value: "sk-test", AuthType: "api_key"},
	})

	tests := []struct {
		path string
		want int
	}{
		{path: "/api/providers", want: http.StatusOK},
		{path: "/api/providers/openai", want: http.StatusOK},
		{path: "/api/connections", want: http.StatusOK},
		{path: "/api/settings", want: http.StatusOK},
		{path: "/api/keys", want: http.StatusOK},
		{path: "/api/combos", want: http.StatusOK},
		{path: "/api/oauth/minimax/start", want: http.StatusOK},
		{path: "/api/usage", want: http.StatusOK},
		{path: "/api/usage/summary", want: http.StatusOK},
		{path: "/api/usage/quota/openai", want: http.StatusOK},
		{path: "/api/logs", want: http.StatusOK},
	}

	for _, tc := range tests {
		resp, err := httpClient().Get(baseURL + tc.path)
		if err != nil {
			t.Fatalf("GET %s: %v", tc.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != tc.want {
			t.Fatalf("GET %s status = %d, want %d", tc.path, resp.StatusCode, tc.want)
		}
	}
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 2 * time.Second}
}

func localhostAddr(t *testing.T, ln net.Listener) string {
	t.Helper()

	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener addr is %T, want *net.TCPAddr", ln.Addr())
	}
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(tcpAddr.Port))
}

func newAPITestStore(t *testing.T) *store.Store {
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

type routeModelSource struct{}

func (routeModelSource) ListModels(ctx context.Context) ([]providers.Model, error) {
	return []providers.Model{
		{ID: "gpt-4o", Object: "model", Provider: providers.ProviderOpenAI},
	}, nil
}

type routeOAuthFlow struct{}

func (routeOAuthFlow) ProviderID() oauth.ProviderID {
	return "minimax"
}

func (routeOAuthFlow) Start(ctx context.Context) (oauth.AuthSession, error) {
	return oauth.AuthSession{Provider: "minimax", SessionID: "session-1"}, nil
}

func (routeOAuthFlow) Exchange(ctx context.Context, session oauth.AuthSession, code string) (oauth.TokenResult, error) {
	return oauth.TokenResult{Provider: "minimax", AccessToken: "token"}, nil
}

func (routeOAuthFlow) Poll(ctx context.Context, session oauth.AuthSession) (oauth.PollResult, error) {
	return oauth.PollResult{Status: oauth.PollStatusPending}, nil
}

type routeQuotaFetcher struct{}

func (routeQuotaFetcher) FetchQuota(ctx context.Context, key providers.Key) (usage.Quota, error) {
	return usage.Quota{Provider: key.Provider, Limit: 100, Used: 1, Remaining: 99}, nil
}
