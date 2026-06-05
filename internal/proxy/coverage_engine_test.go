package proxy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
)

func TestListModelsAggregatesProvidersAndCatalogFallback(t *testing.T) {
	s := openProxyTestStore(t)
	key := "sk-openai"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai", Name: "p", AuthType: store.AuthTypeAPIKey, APIKey: &key, IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	// openai returns models directly.
	openAI := &fakeProvider{
		name:   providers.ProviderOpenAI,
		models: []providers.Model{{ID: "gpt-4o", Provider: providers.ProviderOpenAI}},
	}
	// anthropic has no connection -> catalog fallback path.
	anthropic := &fakeProvider{name: providers.ProviderAnthropic}

	engine := NewEngine(s)
	engine.Register(openAI)
	engine.Register(anthropic)

	models, err := engine.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	var sawOpenAI, sawAnthropic bool
	for _, m := range models {
		if m.ID == "gpt-4o" {
			sawOpenAI = true
		}
		if m.Provider == providers.ProviderAnthropic {
			sawAnthropic = true
		}
	}
	if !sawOpenAI {
		t.Error("expected openai model in aggregation")
	}
	if !sawAnthropic {
		t.Error("expected anthropic catalog models in aggregation")
	}
}

func TestProviderModelsFallsBackOnProviderError(t *testing.T) {
	s := openProxyTestStore(t)
	key := "sk-anthropic"
	if err := s.CreateConnection(&store.Connection{
		Provider: "anthropic", Name: "p", AuthType: store.AuthTypeAPIKey, APIKey: &key, IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	// Provider errors on ListModels -> catalog fallback.
	prov := &fakeProvider{name: providers.ProviderAnthropic, err: errors.New("list failed")}
	engine := NewEngine(s)
	engine.Register(prov)

	models, err := engine.providerModels(context.Background(), providers.ProviderAnthropic)
	if err != nil {
		t.Fatalf("providerModels: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("expected catalog fallback models")
	}
}

func TestProviderModelsUnknownProvider(t *testing.T) {
	s := openProxyTestStore(t)
	engine := NewEngine(s)
	if _, err := engine.providerModels(context.Background(), providers.ProviderAnthropic); !errors.Is(err, ErrProviderNotFound) {
		t.Fatalf("err = %v, want ErrProviderNotFound", err)
	}
}

func TestProviderForUnknownModel(t *testing.T) {
	s := openProxyTestStore(t)
	engine := NewEngine(s)
	if _, _, _, _, err := engine.providerFor(context.Background(), "totally-unknown-model"); err == nil {
		t.Fatal("providerFor unknown model: want error")
	}
}

func TestMCPToolManagerGetter(t *testing.T) {
	s := openProxyTestStore(t)
	engine := NewEngine(s)
	if engine.MCPToolManager() != nil {
		t.Fatal("expected nil tool manager initially")
	}
	tm := mcp.NewToolManager()
	engine.RegisterMCPToolManager(tm)
	if engine.MCPToolManager() != tm {
		t.Fatal("MCPToolManager getter mismatch")
	}
}

func TestRegisterQuotaFetcherNilDeletes(t *testing.T) {
	s := openProxyTestStore(t)
	engine := NewEngine(s)
	fetcher := &fakeQuotaFetcher{quota: usage.Quota{Unlimited: true}}
	engine.RegisterQuotaFetcher(providers.ProviderOpenAI, fetcher)
	if engine.quotaFetcherFor(providers.ProviderOpenAI) == nil {
		t.Fatal("expected fetcher registered")
	}
	engine.RegisterQuotaFetcher(providers.ProviderOpenAI, nil)
	if engine.quotaFetcherFor(providers.ProviderOpenAI) != nil {
		t.Fatal("expected fetcher deleted on nil")
	}
}

func TestFallbackWorthyError(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{ErrQuotaExhausted, true},
		{context.DeadlineExceeded, true},
		{errors.New("provider rate limit hit"), true},
		{errors.New("503 service unavailable"), true},
		{errors.New("bad gateway"), true},
		{errors.New("gateway timeout"), true},
		{errors.New("request timeout"), true},
		{errors.New("invalid request"), false},
	}
	for _, tc := range cases {
		if got := fallbackWorthyError(tc.err); got != tc.want {
			t.Errorf("fallbackWorthyError(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}

func TestRecordProviderFailureNilConn(t *testing.T) {
	s := openProxyTestStore(t)
	engine := NewEngine(s)
	// Should not panic with nil connection.
	engine.recordProviderFailure(nil, "model")
	engine.recordProviderSuccess(nil, "model")
}

func TestOAuthProviderForConnection(t *testing.T) {
	s := openProxyTestStore(t)
	engine := NewEngine(s)

	// Explicit oauth_provider in provider-specific data wins.
	conn := &store.Connection{ProviderSpecificData: map[string]any{"oauth_provider": " codex "}}
	if got := engine.oauthProviderForConnection(providers.ProviderAnthropic, conn); got != oauth.ProviderID("codex") {
		t.Fatalf("explicit oauth_provider = %q", got)
	}
	// OpenAI runtime maps to codex by default.
	if got := engine.oauthProviderForConnection(providers.ProviderOpenAI, &store.Connection{}); got != oauth.ProviderID("codex") {
		t.Fatalf("openai default = %q", got)
	}
	// Other providers map to their own name.
	if got := engine.oauthProviderForConnection(providers.ProviderAnthropic, &store.Connection{}); got != oauth.ProviderID("anthropic") {
		t.Fatalf("anthropic default = %q", got)
	}
	// Empty oauth_provider string falls through to runtime default.
	conn2 := &store.Connection{ProviderSpecificData: map[string]any{"oauth_provider": "  "}}
	if got := engine.oauthProviderForConnection(providers.ProviderAnthropic, conn2); got != oauth.ProviderID("anthropic") {
		t.Fatalf("empty oauth_provider = %q", got)
	}
}

func TestConnectionNeedsRefresh(t *testing.T) {
	s := openProxyTestStore(t)
	engine := NewEngine(s)
	now := time.Unix(1700000000, 0)
	engine.now = func() time.Time { return now }

	refresh := "r"
	soon := now.Add(time.Second).Unix()
	later := now.Add(time.Hour).Unix()

	// API key connection never needs refresh.
	if engine.connectionNeedsRefresh(&store.Connection{AuthType: store.AuthTypeAPIKey}) {
		t.Fatal("api key should not need refresh")
	}
	// OAuth without refresh token.
	if engine.connectionNeedsRefresh(&store.Connection{AuthType: store.AuthTypeOAuth}) {
		t.Fatal("missing refresh token should not need refresh")
	}
	// OAuth without expiry.
	if engine.connectionNeedsRefresh(&store.Connection{AuthType: store.AuthTypeOAuth, RefreshToken: &refresh}) {
		t.Fatal("missing expiry should not need refresh")
	}
	// Expiring soon -> needs refresh.
	if !engine.connectionNeedsRefresh(&store.Connection{AuthType: store.AuthTypeOAuth, RefreshToken: &refresh, ExpiresAt: &soon}) {
		t.Fatal("expiring soon should need refresh")
	}
	// Far future -> no refresh.
	if engine.connectionNeedsRefresh(&store.Connection{AuthType: store.AuthTypeOAuth, RefreshToken: &refresh, ExpiresAt: &later}) {
		t.Fatal("far future should not need refresh")
	}
}

func TestRefreshConnectionFailurePropagates(t *testing.T) {
	s := openProxyTestStore(t)
	now := time.Unix(1700000000, 0)
	oldExpires := now.Add(time.Second).Unix()
	token := "old-access"
	refresh := "old-refresh"
	if err := s.CreateConnection(&store.Connection{
		Provider:     "openai",
		Name:         "oauth",
		AuthType:     store.AuthTypeOAuth,
		AccessToken:  &token,
		RefreshToken: &refresh,
		ExpiresAt:    &oldExpires,
		IsActive:     true,
		ProviderSpecificData: map[string]any{"oauth_provider": "codex"},
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "x"}}
	refresher := &fakeOAuthRefresher{err: errors.New("refresh boom")}
	engine := NewEngine(s)
	engine.now = func() time.Time { return now }
	engine.Register(openAI)
	engine.RegisterOAuthRefresher(oauth.ProviderID("codex"), refresher)

	if _, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"}); err == nil {
		t.Fatal("dispatch with failing refresh: want error")
	}
}
