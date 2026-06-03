package proxy

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

type fakeProvider struct {
	name        providers.ModelProvider
	response    *providers.ChatResponse
	stream      <-chan providers.StreamChunk
	models      []providers.Model
	err         error
	called      bool
	streamed    bool
	received    *providers.ChatRequest
	receivedKey providers.Key
}

func (f *fakeProvider) Name() providers.ModelProvider {
	return f.name
}

func (f *fakeProvider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	f.called = true
	f.receivedKey = key
	f.received = req
	return f.response, f.err
}

func (f *fakeProvider) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	f.streamed = true
	f.receivedKey = key
	f.received = req
	return f.stream, f.err
}

func (f *fakeProvider) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	return f.models, f.err
}

type fakeOAuthRefresher struct {
	token                oauth.TokenResult
	err                  error
	calls                int
	receivedRefreshToken string
}

func (f *fakeOAuthRefresher) Refresh(ctx context.Context, refreshToken string) (oauth.TokenResult, error) {
	f.calls++
	f.receivedRefreshToken = refreshToken
	if f.err != nil {
		return oauth.TokenResult{}, f.err
	}
	return f.token, nil
}

func TestDispatchRoutesToCorrectProvider(t *testing.T) {
	s := openProxyTestStore(t)
	openAIKey := "sk-openai"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "primary",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &openAIKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection openai: %v", err)
	}
	anthropicKey := "sk-anthropic"
	if err := s.CreateConnection(&store.Connection{
		Provider: "anthropic",
		Name:     "backup",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &anthropicKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection anthropic: %v", err)
	}

	openAI := &fakeProvider{
		name: providers.ProviderOpenAI,
		response: &providers.ChatResponse{
			ID:    "chatcmpl-1",
			Model: "gpt-4o",
		},
	}
	anthropic := &fakeProvider{name: providers.ProviderAnthropic}
	engine := NewEngine(s)
	engine.Register(openAI)
	engine.Register(anthropic)

	req := &providers.ChatRequest{Model: "gpt-4o"}
	resp, err := engine.Dispatch(context.Background(), req)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if resp.ID != "chatcmpl-1" {
		t.Fatalf("response ID = %q, want chatcmpl-1", resp.ID)
	}
	if !openAI.called {
		t.Fatal("openai provider was not called")
	}
	if anthropic.called {
		t.Fatal("anthropic provider should not be called")
	}
	if openAI.received != req {
		t.Fatal("provider should receive original request")
	}
	if openAI.receivedKey.Provider != providers.ProviderOpenAI {
		t.Fatalf("key provider = %q, want openai", openAI.receivedKey.Provider)
	}
	if openAI.receivedKey.Value != openAIKey {
		t.Fatalf("key value = %q, want %q", openAI.receivedKey.Value, openAIKey)
	}
	if openAI.receivedKey.ConnID == "" {
		t.Fatal("connection ID should be set")
	}
	if openAI.receivedKey.AuthType != string(store.AuthTypeAPIKey) {
		t.Fatalf("auth type = %q, want api_key", openAI.receivedKey.AuthType)
	}
}

func TestDispatchRefreshesOAuthConnectionBeforeProviderCall(t *testing.T) {
	s := openProxyTestStore(t)
	now := time.Unix(1700000000, 0)
	oldExpires := now.Add(time.Minute).Unix()
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
		ProviderSpecificData: map[string]any{
			"oauth_provider": "codex",
		},
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-1"}}
	refresher := &fakeOAuthRefresher{token: oauth.TokenResult{
		Provider:     oauth.ProviderID("codex"),
		AccessToken:  "new-access",
		RefreshToken: "new-refresh",
		TokenType:    "bearer",
		ExpiresAt:    now.Add(time.Hour),
	}}
	engine := NewEngine(s)
	engine.now = func() time.Time { return now }
	engine.Register(openAI)
	engine.RegisterOAuthRefresher(oauth.ProviderID("codex"), refresher)

	if _, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	if refresher.calls != 1 {
		t.Fatalf("refresh calls = %d, want 1", refresher.calls)
	}
	if refresher.receivedRefreshToken != "old-refresh" {
		t.Fatalf("refresh token = %q, want old-refresh", refresher.receivedRefreshToken)
	}
	if openAI.receivedKey.Value != "new-access" {
		t.Fatalf("provider key = %q, want refreshed access token", openAI.receivedKey.Value)
	}

	connections, err := s.GetActiveConnections("openai")
	if err != nil {
		t.Fatalf("GetActiveConnections: %v", err)
	}
	if len(connections) != 1 {
		t.Fatalf("connections = %d, want 1", len(connections))
	}
	if connections[0].AccessToken == nil || *connections[0].AccessToken != "new-access" {
		t.Fatalf("stored access token = %v, want new-access", connections[0].AccessToken)
	}
	if connections[0].RefreshToken == nil || *connections[0].RefreshToken != "new-refresh" {
		t.Fatalf("stored refresh token = %v, want new-refresh", connections[0].RefreshToken)
	}
	wantExpires := now.Add(time.Hour).Unix()
	if connections[0].ExpiresAt == nil || *connections[0].ExpiresAt != wantExpires {
		t.Fatalf("stored expires at = %v, want %d", connections[0].ExpiresAt, wantExpires)
	}
}

func TestDispatchStreamRefreshesOAuthConnectionBeforeProviderCall(t *testing.T) {
	s := openProxyTestStore(t)
	now := time.Unix(1700000000, 0)
	oldExpires := now.Add(time.Minute).Unix()
	token := "old-access"
	refresh := "old-refresh"
	if err := s.CreateConnection(&store.Connection{
		Provider:     "anthropic",
		Name:         "oauth",
		AuthType:     store.AuthTypeOAuth,
		AccessToken:  &token,
		RefreshToken: &refresh,
		ExpiresAt:    &oldExpires,
		IsActive:     true,
		ProviderSpecificData: map[string]any{
			"oauth_provider": "anthropic",
		},
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	chunks := make(chan providers.StreamChunk)
	close(chunks)
	anthropic := &fakeProvider{name: providers.ProviderAnthropic, stream: chunks}
	refresher := &fakeOAuthRefresher{token: oauth.TokenResult{
		Provider:     oauth.ProviderID("anthropic"),
		AccessToken:  "new-access",
		RefreshToken: "new-refresh",
		TokenType:    "bearer",
		ExpiresAt:    now.Add(time.Hour),
	}}
	engine := NewEngine(s)
	engine.now = func() time.Time { return now }
	engine.Register(anthropic)
	engine.RegisterOAuthRefresher(oauth.ProviderID("anthropic"), refresher)

	if _, err := engine.DispatchStream(context.Background(), &providers.ChatRequest{Model: "claude-3-5-sonnet"}); err != nil {
		t.Fatalf("DispatchStream: %v", err)
	}
	if refresher.calls != 1 {
		t.Fatalf("refresh calls = %d, want 1", refresher.calls)
	}
	if anthropic.receivedKey.Value != "new-access" {
		t.Fatalf("stream key = %q, want refreshed access token", anthropic.receivedKey.Value)
	}
}

func TestDispatchDoesNotRefreshFreshOAuthConnection(t *testing.T) {
	s := openProxyTestStore(t)
	now := time.Unix(1700000000, 0)
	expires := now.Add(time.Hour).Unix()
	token := "current-access"
	refresh := "current-refresh"
	if err := s.CreateConnection(&store.Connection{
		Provider:     "openai",
		Name:         "oauth",
		AuthType:     store.AuthTypeOAuth,
		AccessToken:  &token,
		RefreshToken: &refresh,
		ExpiresAt:    &expires,
		IsActive:     true,
		ProviderSpecificData: map[string]any{
			"oauth_provider": "codex",
		},
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-1"}}
	refresher := &fakeOAuthRefresher{token: oauth.TokenResult{Provider: oauth.ProviderID("codex"), AccessToken: "new-access"}}
	engine := NewEngine(s)
	engine.now = func() time.Time { return now }
	engine.Register(openAI)
	engine.RegisterOAuthRefresher(oauth.ProviderID("codex"), refresher)

	if _, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if refresher.calls != 0 {
		t.Fatalf("refresh calls = %d, want 0", refresher.calls)
	}
	if openAI.receivedKey.Value != "current-access" {
		t.Fatalf("provider key = %q, want current access token", openAI.receivedKey.Value)
	}
}

func TestDispatchUnknownModel(t *testing.T) {
	engine := NewEngine(openProxyTestStore(t))
	engine.Register(&fakeProvider{name: providers.ProviderOpenAI})

	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "unknown-model"})
	if !errors.Is(err, ErrProviderNotFound) {
		t.Fatalf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestDispatchNoConnections(t *testing.T) {
	engine := NewEngine(openProxyTestStore(t))
	engine.Register(&fakeProvider{name: providers.ProviderOpenAI})

	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
	if !errors.Is(err, ErrNoConnections) {
		t.Fatalf("expected ErrNoConnections, got %v", err)
	}
}

func TestListModelsReturnsCatalogWithoutConnections(t *testing.T) {
	engine := NewEngine(openProxyTestStore(t))
	engine.Register(&fakeProvider{name: providers.ProviderOpenAI})
	engine.Register(&fakeProvider{name: providers.ProviderAnthropic})

	models, err := engine.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("models should not be empty for a fresh registered engine")
	}

	foundOpenAI := false
	foundAnthropic := false
	for _, model := range models {
		if model.ID == "gpt-4o" && model.Provider == providers.ProviderOpenAI {
			foundOpenAI = true
		}
		if model.ID == "claude-sonnet-4" && model.Provider == providers.ProviderAnthropic {
			foundAnthropic = true
		}
	}
	if !foundOpenAI || !foundAnthropic {
		t.Fatalf("models = %+v, want openai and anthropic catalog models", models)
	}
}

func TestListModelsFallsBackToCatalogWhenProviderListFails(t *testing.T) {
	s := openProxyTestStore(t)
	apiKey := "sk-openai"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "primary",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	engine := NewEngine(s)
	engine.Register(&fakeProvider{name: providers.ProviderOpenAI, err: errors.New("upstream unavailable")})

	models, err := engine.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	for _, model := range models {
		if model.ID == "gpt-4o" && model.Provider == providers.ProviderOpenAI {
			return
		}
	}
	t.Fatalf("models = %+v, want openai catalog fallback", models)
}

func TestDispatchStreamReturnsChannel(t *testing.T) {
	s := openProxyTestStore(t)
	token := "token-anthropic"
	if err := s.CreateConnection(&store.Connection{
		Provider:    "anthropic",
		Name:        "oauth",
		AuthType:    store.AuthTypeOAuth,
		AccessToken: &token,
		IsActive:    true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	content := "hello"
	chunks := make(chan providers.StreamChunk, 1)
	chunks <- providers.StreamChunk{
		ID:    "chunk-1",
		Model: "claude-3-5-sonnet",
		Choices: []providers.StreamChoice{
			{Delta: providers.StreamDelta{Content: &content}},
		},
	}
	close(chunks)

	anthropic := &fakeProvider{name: providers.ProviderAnthropic, stream: chunks}
	engine := NewEngine(s)
	engine.Register(anthropic)

	stream, err := engine.DispatchStream(context.Background(), &providers.ChatRequest{Model: "claude-3-5-sonnet"})
	if err != nil {
		t.Fatalf("DispatchStream: %v", err)
	}
	got, ok := <-stream
	if !ok {
		t.Fatal("stream closed before first chunk")
	}
	if got.ID != "chunk-1" {
		t.Fatalf("chunk ID = %q, want chunk-1", got.ID)
	}
	if !anthropic.streamed {
		t.Fatal("anthropic stream provider was not called")
	}
	if anthropic.receivedKey.Value != token {
		t.Fatalf("key value = %q, want %q", anthropic.receivedKey.Value, token)
	}
	if anthropic.receivedKey.AuthType != string(store.AuthTypeOAuth) {
		t.Fatalf("auth type = %q, want oauth", anthropic.receivedKey.AuthType)
	}
}

func openProxyTestStore(t *testing.T) *store.Store {
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
