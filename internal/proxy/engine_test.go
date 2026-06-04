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
	"github.com/bloodf/g0router/internal/usage"
)

type fakeProvider struct {
	name        providers.ModelProvider
	response    *providers.ChatResponse
	responses   []*providers.ChatResponse
	stream      <-chan providers.StreamChunk
	models      []providers.Model
	err         error
	errs        []error
	called      bool
	streamed    bool
	calls       int
	received    *providers.ChatRequest
	receivedKey providers.Key
	requests    []*providers.ChatRequest
	keys        []providers.Key
}

func (f *fakeProvider) Name() providers.ModelProvider {
	return f.name
}

func (f *fakeProvider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	f.called = true
	f.calls++
	f.receivedKey = key
	f.received = req
	f.keys = append(f.keys, key)
	f.requests = append(f.requests, req)
	index := f.calls - 1
	err := f.err
	if index < len(f.errs) {
		err = f.errs[index]
	}
	if err != nil {
		return nil, err
	}
	if index < len(f.responses) {
		return f.responses[index], nil
	}
	return f.response, nil
}

func (f *fakeProvider) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	f.streamed = true
	f.receivedKey = key
	f.received = req
	f.keys = append(f.keys, key)
	f.requests = append(f.requests, req)
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

type fakeQuotaFetcher struct {
	quota  usage.Quota
	quotas []usage.Quota
	err    error
	errs   []error
	calls  int
	gotKey providers.Key
	keys   []providers.Key
}

func (f *fakeQuotaFetcher) FetchQuota(ctx context.Context, key providers.Key) (usage.Quota, error) {
	f.calls++
	f.gotKey = key
	f.keys = append(f.keys, key)
	index := f.calls - 1
	if f.err != nil {
		return usage.Quota{}, f.err
	}
	if index < len(f.errs) && f.errs[index] != nil {
		return usage.Quota{}, f.errs[index]
	}
	if index < len(f.quotas) {
		return f.quotas[index], nil
	}
	return f.quota, nil
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
	if resp.Provider != providers.ProviderOpenAI {
		t.Fatalf("response provider = %q, want openai", resp.Provider)
	}
	if resp.ConnectionID != openAI.receivedKey.ConnID {
		t.Fatalf("response connection ID = %q, want %q", resp.ConnectionID, openAI.receivedKey.ConnID)
	}
	if resp.AuthType != string(store.AuthTypeAPIKey) {
		t.Fatalf("response auth type = %q, want api_key", resp.AuthType)
	}
}

func TestDispatchUsesModelAliasProviderAndRewritesUpstreamModel(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "groq", "groq-key")
	if err := s.SetModelAlias(store.ModelAlias{
		Alias:    "fast",
		Provider: "groq",
		Model:    "llama-3.3-70b-versatile",
	}); err != nil {
		t.Fatalf("SetModelAlias: %v", err)
	}

	groq := &fakeProvider{
		name: providers.ProviderGroq,
		response: &providers.ChatResponse{
			ID:    "chatcmpl-groq",
			Model: "llama-3.3-70b-versatile",
		},
	}
	engine := NewEngine(s)
	engine.Register(groq)

	req := &providers.ChatRequest{Model: "fast"}
	resp, err := engine.Dispatch(context.Background(), req)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if resp.ID != "chatcmpl-groq" {
		t.Fatalf("response ID = %q, want chatcmpl-groq", resp.ID)
	}
	if !groq.called {
		t.Fatal("groq provider was not called")
	}
	if groq.received == req {
		t.Fatal("alias dispatch should pass a copied request")
	}
	if groq.received.Model != "llama-3.3-70b-versatile" {
		t.Fatalf("provider request model = %q, want alias target", groq.received.Model)
	}
	if req.Model != "fast" {
		t.Fatalf("original request model = %q, want alias name unchanged", req.Model)
	}
	if groq.receivedKey.Provider != providers.ProviderGroq {
		t.Fatalf("key provider = %q, want groq", groq.receivedKey.Provider)
	}
	if groq.receivedKey.Value != "groq-key" {
		t.Fatalf("key value = %q, want groq-key", groq.receivedKey.Value)
	}
	if resp.Provider != providers.ProviderGroq {
		t.Fatalf("response provider = %q, want groq", resp.Provider)
	}
	if resp.ConnectionID != groq.receivedKey.ConnID {
		t.Fatalf("response connection ID = %q, want %q", resp.ConnectionID, groq.receivedKey.ConnID)
	}
	if resp.AuthType != string(store.AuthTypeAPIKey) {
		t.Fatalf("response auth type = %q, want api_key", resp.AuthType)
	}
}

func TestDispatchUsesCatalogForPublicOpenAICompatibleProviders(t *testing.T) {
	cases := []struct {
		name     string
		provider providers.ModelProvider
		model    string
		key      string
	}{
		{name: "groq", provider: providers.ProviderGroq, model: "llama-3.3-70b-versatile", key: "groq-key"},
		{name: "cerebras", provider: providers.ProviderCerebras, model: "llama3.1-8b", key: "cerebras-key"},
		{name: "cohere", provider: providers.ProviderCohere, model: "command-r-08-2024", key: "cohere-key"},
		{name: "fireworks", provider: providers.ProviderFireworks, model: "accounts/fireworks/models/llama-v3p1-70b-instruct", key: "fireworks-key"},
		{name: "huggingface", provider: providers.ProviderHuggingFace, model: "meta-llama/Llama-3.3-70B-Instruct:groq", key: "huggingface-key"},
		{name: "mistral", provider: providers.ProviderMistral, model: "mistral-small-latest", key: "mistral-key"},
		{name: "minimax", provider: providers.ProviderMiniMax, model: "MiniMax-M3", key: "minimax-key"},
		{name: "nebius", provider: providers.ProviderNebius, model: "meta-llama/Llama-3.3-70B-Instruct", key: "nebius-key"},
		{name: "deepseek", provider: providers.ProviderDeepSeek, model: "deepseek-chat", key: "deepseek-key"},
		{name: "openrouter", provider: providers.ProviderOpenRouter, model: "openai/gpt-4o-mini", key: "openrouter-key"},
		{name: "perplexity", provider: providers.ProviderPerplexity, model: "sonar-pro", key: "perplexity-key"},
		{name: "qwen", provider: providers.ProviderQwen, model: "qwen3.6-plus", key: "qwen-key"},
		{name: "together", provider: providers.ProviderTogether, model: "meta-llama/Llama-3.3-70B-Instruct-Turbo", key: "together-key"},
		{name: "xai", provider: providers.ProviderXAI, model: "grok-4.3", key: "xai-key"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := openProxyTestStore(t)
			createProxyConnection(t, s, tc.provider.String(), tc.key)

			runtime := &fakeProvider{
				name: tc.provider,
				response: &providers.ChatResponse{
					ID:    "chatcmpl-" + tc.name,
					Model: tc.model,
				},
			}
			engine := NewEngine(s)
			engine.Register(runtime)

			req := &providers.ChatRequest{Model: tc.model}
			resp, err := engine.Dispatch(context.Background(), req)
			if err != nil {
				t.Fatalf("Dispatch: %v", err)
			}
			if !runtime.called {
				t.Fatal("provider was not called")
			}
			if runtime.received != req {
				t.Fatal("catalog dispatch should pass the original request")
			}
			if runtime.receivedKey.Provider != tc.provider {
				t.Fatalf("key provider = %q, want %q", runtime.receivedKey.Provider, tc.provider)
			}
			if runtime.receivedKey.Value != tc.key {
				t.Fatalf("key value = %q, want %q", runtime.receivedKey.Value, tc.key)
			}
			if resp.Provider != tc.provider {
				t.Fatalf("response provider = %q, want %q", resp.Provider, tc.provider)
			}
		})
	}
}

func TestDispatchUsesCatalogForOllamaNoAuthProvider(t *testing.T) {
	s := openProxyTestStore(t)
	if err := s.CreateConnection(&store.Connection{
		Provider: "ollama",
		Name:     "local-ollama",
		AuthType: store.AuthTypeNoAuth,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection ollama: %v", err)
	}

	ollama := &fakeProvider{
		name: providers.ProviderOllama,
		response: &providers.ChatResponse{
			ID:    "chatcmpl-ollama",
			Model: "llama3.1:8b",
		},
	}
	engine := NewEngine(s)
	engine.Register(ollama)

	resp, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "llama3.1:8b"})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if !ollama.called {
		t.Fatal("ollama provider was not called")
	}
	if ollama.receivedKey.Provider != providers.ProviderOllama {
		t.Fatalf("key provider = %q, want ollama", ollama.receivedKey.Provider)
	}
	if ollama.receivedKey.AuthType != string(store.AuthTypeNoAuth) {
		t.Fatalf("auth type = %q, want noauth", ollama.receivedKey.AuthType)
	}
	if ollama.receivedKey.Value != "" {
		t.Fatalf("key value = %q, want empty noauth key", ollama.receivedKey.Value)
	}
	if resp.Provider != providers.ProviderOllama {
		t.Fatalf("response provider = %q, want ollama", resp.Provider)
	}
	if resp.AuthType != string(store.AuthTypeNoAuth) {
		t.Fatalf("response auth type = %q, want noauth", resp.AuthType)
	}
}

func TestDispatchUsesCatalogForGeminiProvider(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "gemini", "gemini-key")

	gemini := &fakeProvider{
		name: providers.ProviderGemini,
		response: &providers.ChatResponse{
			ID:    "chatcmpl-gemini",
			Model: "gemini-2.5-flash",
		},
	}
	engine := NewEngine(s)
	engine.Register(gemini)

	resp, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gemini-2.5-flash"})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if !gemini.called {
		t.Fatal("gemini provider was not called")
	}
	if gemini.receivedKey.Provider != providers.ProviderGemini {
		t.Fatalf("key provider = %q, want gemini", gemini.receivedKey.Provider)
	}
	if gemini.receivedKey.AuthType != string(store.AuthTypeAPIKey) {
		t.Fatalf("auth type = %q, want api_key", gemini.receivedKey.AuthType)
	}
	if resp.Provider != providers.ProviderGemini {
		t.Fatalf("response provider = %q, want gemini", resp.Provider)
	}
}

func TestDispatchBlocksAliasToProviderWithoutInference(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "bedrock", "bedrock-key")
	if err := s.SetModelAlias(store.ModelAlias{
		Alias:    "bedrock-alias",
		Provider: "bedrock",
		Model:    "anthropic.claude-3-haiku-20240307-v1:0",
	}); err != nil {
		t.Fatalf("SetModelAlias: %v", err)
	}

	bedrock := &fakeProvider{name: providers.ProviderBedrock, response: &providers.ChatResponse{ID: "bedrock-should-not-run"}}
	engine := NewEngine(s)
	engine.Register(bedrock)

	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "bedrock-alias"})
	if !errors.Is(err, ErrProviderInferenceUnavailable) {
		t.Fatalf("Dispatch error = %v, want ErrProviderInferenceUnavailable", err)
	}
	if bedrock.called {
		t.Fatal("bedrock provider should not be called through an alias")
	}
}

func TestDispatchUsesComboModelThroughEngine(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "groq", "groq-key")
	createProxyConnection(t, s, "openai", "openai-key")
	if err := s.CreateCombo(&store.Combo{
		Name: "fast-fallback",
		Steps: []store.ComboStep{
			{Provider: "groq", Model: "llama-3.3-70b-versatile"},
			{Provider: "openai", Model: "gpt-4o-mini"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	groq := &fakeProvider{name: providers.ProviderGroq, err: errors.New("rate limited")}
	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-combo"}}
	engine := NewEngine(s)
	engine.Register(groq)
	engine.Register(openAI)

	resp, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "combo/fast-fallback"})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if resp.ID != "chatcmpl-combo" {
		t.Fatalf("response ID = %q, want chatcmpl-combo", resp.ID)
	}
	if !groq.called || !openAI.called {
		t.Fatalf("combo providers called groq=%v openai=%v", groq.called, openAI.called)
	}
	if groq.received.Model != "llama-3.3-70b-versatile" {
		t.Fatalf("groq model = %q, want combo step model", groq.received.Model)
	}
	if openAI.received.Model != "gpt-4o-mini" {
		t.Fatalf("openai model = %q, want combo step model", openAI.received.Model)
	}
}

func TestDispatchRoundRobinsActiveConnections(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "openai", "openai-key-1")
	createProxyConnection(t, s, "openai", "openai-key-2")
	openAI := &fakeProvider{
		name: providers.ProviderOpenAI,
		responses: []*providers.ChatResponse{
			{ID: "chatcmpl-1"},
			{ID: "chatcmpl-2"},
		},
	}
	engine := NewEngine(s)
	engine.Register(openAI)

	if _, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"}); err != nil {
		t.Fatalf("Dispatch first: %v", err)
	}
	if _, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"}); err != nil {
		t.Fatalf("Dispatch second: %v", err)
	}
	if len(openAI.keys) != 2 {
		t.Fatalf("keys = %+v", openAI.keys)
	}
	if openAI.keys[0].Value == openAI.keys[1].Value {
		t.Fatalf("selected keys = %q, %q; want different round-robin keys", openAI.keys[0].Value, openAI.keys[1].Value)
	}
	selected := map[string]bool{openAI.keys[0].Value: true, openAI.keys[1].Value: true}
	if !selected["openai-key-1"] || !selected["openai-key-2"] {
		t.Fatalf("selected keys = %+v; want both configured keys", openAI.keys)
	}
}

func TestDispatchRecordsFailureAndSkipsBackedOffConnection(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "openai", "openai-key-1")
	createProxyConnection(t, s, "openai", "openai-key-2")
	now := time.Unix(1_700_000_000, 0)
	upstreamErr := errors.New("rate limited")
	openAI := &fakeProvider{
		name:      providers.ProviderOpenAI,
		errs:      []error{upstreamErr, nil},
		responses: []*providers.ChatResponse{nil, &providers.ChatResponse{ID: "chatcmpl-recovered"}},
	}
	engine := NewEngine(s)
	engine.now = func() time.Time { return now }
	engine.Register(openAI)

	resp, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if resp.ID != "chatcmpl-recovered" {
		t.Fatalf("response ID = %q", resp.ID)
	}
	if len(openAI.keys) != 2 {
		t.Fatalf("selected keys = %+v", openAI.keys)
	}
	if openAI.keys[0].Value == openAI.keys[1].Value {
		t.Fatalf("selected keys = %+v; want retry on a different connection", openAI.keys)
	}

	firstConn, err := s.GetConnection(openAI.keys[0].ConnID)
	if err != nil {
		t.Fatalf("GetConnection first: %v", err)
	}
	if firstConn.BackoffLevel != 1 {
		t.Fatalf("backoff level = %d, want 1", firstConn.BackoffLevel)
	}
	if firstConn.ModelLocks["gpt-4o"] != now.Add(time.Second).Unix() {
		t.Fatalf("model locks = %+v", firstConn.ModelLocks)
	}
}

func TestDispatchSuccessClearsExpiredModelBackoff(t *testing.T) {
	s := openProxyTestStore(t)
	now := time.Unix(1_700_000_000, 0)
	lockedUntil := now.Add(-time.Minute).Unix()
	key := "openai-key"
	conn := &store.Connection{
		Provider:     "openai",
		Name:         "primary",
		AuthType:     store.AuthTypeAPIKey,
		APIKey:       &key,
		IsActive:     true,
		BackoffLevel: 3,
		ModelLocks:   map[string]int64{"gpt-4o": lockedUntil},
	}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-ok"}}
	engine := NewEngine(s)
	engine.now = func() time.Time { return now }
	engine.Register(openAI)

	if _, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	updated, err := s.GetConnection(conn.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if updated.BackoffLevel != 0 {
		t.Fatalf("backoff level = %d, want 0", updated.BackoffLevel)
	}
	if _, ok := updated.ModelLocks["gpt-4o"]; ok {
		t.Fatalf("model lock was not cleared: %+v", updated.ModelLocks)
	}
}

func TestDispatchDoesNotBackoffNonFallbackError(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "openai", "openai-key-1")
	createProxyConnection(t, s, "openai", "openai-key-2")
	badRequestErr := errors.New("invalid request body")
	openAI := &fakeProvider{name: providers.ProviderOpenAI, err: badRequestErr}
	engine := NewEngine(s)
	engine.Register(openAI)

	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
	if !errors.Is(err, badRequestErr) {
		t.Fatalf("Dispatch error = %v, want bad request error", err)
	}
	if len(openAI.keys) != 1 {
		t.Fatalf("selected keys = %+v", openAI.keys)
	}
	conn, err := s.GetConnection(openAI.keys[0].ConnID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if conn.BackoffLevel != 0 || len(conn.ModelLocks) != 0 {
		t.Fatalf("connection was backed off for non-fallback error: %+v", conn)
	}
}

func TestDispatchQuotaExhaustionBlocksProviderCall(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "openai", "openai-key")
	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-should-not-run"}}
	quota := &fakeQuotaFetcher{quota: usage.Quota{
		Provider:  providers.ProviderOpenAI,
		Limit:     100,
		Used:      100,
		Remaining: 0,
	}}
	engine := NewEngine(s)
	engine.Register(openAI)
	engine.RegisterQuotaFetcher(providers.ProviderOpenAI, quota)

	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
	if !errors.Is(err, ErrQuotaExhausted) {
		t.Fatalf("Dispatch error = %v, want ErrQuotaExhausted", err)
	}
	if openAI.called {
		t.Fatal("provider should not be called when quota is exhausted")
	}
	if quota.calls != 1 {
		t.Fatalf("quota calls = %d, want 1", quota.calls)
	}
	if quota.gotKey.Value != "openai-key" || quota.gotKey.Provider != providers.ProviderOpenAI {
		t.Fatalf("quota key = %+v", quota.gotKey)
	}
}

func TestDispatchExplicitZeroRemainingQuotaBlocksProviderCall(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "openai", "openai-key")
	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-should-not-run"}}
	quota := &fakeQuotaFetcher{quota: usage.Quota{
		Provider:  providers.ProviderOpenAI,
		Remaining: 0,
	}}
	engine := NewEngine(s)
	engine.Register(openAI)
	engine.RegisterQuotaFetcher(providers.ProviderOpenAI, quota)

	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
	if !errors.Is(err, ErrQuotaExhausted) {
		t.Fatalf("Dispatch error = %v, want ErrQuotaExhausted", err)
	}
	if openAI.called {
		t.Fatal("provider should not be called when quota reports zero remaining")
	}
}

func TestDispatchPrefixModelQuotaExhaustionBlocksProviderCall(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "openai", "openai-key")
	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-should-not-run"}}
	quota := &fakeQuotaFetcher{quota: usage.Quota{
		Provider:  providers.ProviderOpenAI,
		Remaining: 0,
	}}
	engine := NewEngine(s)
	engine.Register(openAI)
	engine.RegisterQuotaFetcher(providers.ProviderOpenAI, quota)

	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-prefix-only-model"})
	if !errors.Is(err, ErrQuotaExhausted) {
		t.Fatalf("Dispatch error = %v, want ErrQuotaExhausted", err)
	}
	if openAI.called {
		t.Fatal("provider should not be called when prefix route quota is exhausted")
	}
	if quota.gotKey.Provider != providers.ProviderOpenAI || quota.gotKey.Value != "openai-key" {
		t.Fatalf("quota key = %+v", quota.gotKey)
	}
}

func TestDispatchQuotaErrorsFailOpen(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{name: "unsupported", err: usage.ErrQuotaUnsupported},
		{name: "transient", err: errors.New("quota API temporarily unavailable")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := openProxyTestStore(t)
			createProxyConnection(t, s, "openai", "openai-key")
			openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-ok"}}
			quota := &fakeQuotaFetcher{err: tc.err}
			engine := NewEngine(s)
			engine.Register(openAI)
			engine.RegisterQuotaFetcher(providers.ProviderOpenAI, quota)

			resp, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
			if err != nil {
				t.Fatalf("Dispatch: %v", err)
			}
			if resp.ID != "chatcmpl-ok" {
				t.Fatalf("response ID = %q, want chatcmpl-ok", resp.ID)
			}
			if !openAI.called {
				t.Fatal("provider should be called when quota check fails open")
			}
		})
	}
}

func TestDispatchExplicitQuotaErrorStopsBeforeProviderAndFallback(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "openai", "openai-key-1")
	createProxyConnection(t, s, "openai", "openai-key-2")
	now := time.Unix(1_700_000_000, 0)
	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-second-account"}}
	quota := &fakeQuotaFetcher{
		errs: []error{ErrQuotaExhausted, nil},
		quotas: []usage.Quota{
			{},
			{Provider: providers.ProviderOpenAI, Remaining: 10},
		},
	}
	engine := NewEngine(s)
	engine.now = func() time.Time { return now }
	engine.Register(openAI)
	engine.RegisterQuotaFetcher(providers.ProviderOpenAI, quota)

	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
	if !errors.Is(err, ErrQuotaExhausted) {
		t.Fatalf("Dispatch error = %v, want ErrQuotaExhausted", err)
	}
	if openAI.calls != 0 {
		t.Fatalf("provider calls = %d, want 0", openAI.calls)
	}
	if len(quota.keys) != 1 {
		t.Fatalf("quota keys = %+v; want only one selected account", quota.keys)
	}
	firstConn, err := s.GetConnection(quota.keys[0].ConnID)
	if err != nil {
		t.Fatalf("GetConnection first: %v", err)
	}
	if firstConn.BackoffLevel != 0 || len(firstConn.ModelLocks) != 0 {
		t.Fatalf("quota exhaustion should not create fallback backoff: level=%d locks=%+v", firstConn.BackoffLevel, firstConn.ModelLocks)
	}
}

func TestDispatchQuotaExhaustedConnectionStopsBeforeProviderAndFallback(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "openai", "openai-key-1")
	createProxyConnection(t, s, "openai", "openai-key-2")
	now := time.Unix(1_700_000_000, 0)
	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-second-account"}}
	quota := &fakeQuotaFetcher{quotas: []usage.Quota{
		{Provider: providers.ProviderOpenAI, Remaining: 0},
		{Provider: providers.ProviderOpenAI, Remaining: 10},
	}}
	engine := NewEngine(s)
	engine.now = func() time.Time { return now }
	engine.Register(openAI)
	engine.RegisterQuotaFetcher(providers.ProviderOpenAI, quota)

	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
	if !errors.Is(err, ErrQuotaExhausted) {
		t.Fatalf("Dispatch error = %v, want ErrQuotaExhausted", err)
	}
	if openAI.calls != 0 {
		t.Fatalf("provider calls = %d, want 0", openAI.calls)
	}
	if len(quota.keys) != 1 {
		t.Fatalf("quota keys = %+v; want only one selected account", quota.keys)
	}
	firstConn, err := s.GetConnection(quota.keys[0].ConnID)
	if err != nil {
		t.Fatalf("GetConnection first: %v", err)
	}
	if firstConn.BackoffLevel != 0 || len(firstConn.ModelLocks) != 0 {
		t.Fatalf("quota exhaustion should not create fallback backoff: level=%d locks=%+v", firstConn.BackoffLevel, firstConn.ModelLocks)
	}
}

func TestDispatchAllQuotaExhaustedConnectionsReturnQuotaError(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "openai", "openai-key-1")
	createProxyConnection(t, s, "openai", "openai-key-2")
	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-should-not-run"}}
	quota := &fakeQuotaFetcher{quota: usage.Quota{
		Provider:  providers.ProviderOpenAI,
		Remaining: 0,
	}}
	engine := NewEngine(s)
	engine.Register(openAI)
	engine.RegisterQuotaFetcher(providers.ProviderOpenAI, quota)

	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
	if !errors.Is(err, ErrQuotaExhausted) {
		t.Fatalf("Dispatch error = %v, want ErrQuotaExhausted", err)
	}
	if openAI.called {
		t.Fatal("provider should not be called when all connections are quota exhausted")
	}
	if quota.calls != 1 {
		t.Fatalf("quota calls = %d, want 1", quota.calls)
	}
}

func TestDispatchAliasQuotaUsesTargetProviderConnection(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "groq", "groq-key")
	if err := s.SetModelAlias(store.ModelAlias{
		Alias:    "fast",
		Provider: "groq",
		Model:    "llama-3.3-70b-versatile",
	}); err != nil {
		t.Fatalf("SetModelAlias: %v", err)
	}
	groq := &fakeProvider{name: providers.ProviderGroq, response: &providers.ChatResponse{ID: "chatcmpl-groq"}}
	quota := &fakeQuotaFetcher{quota: usage.Quota{Provider: providers.ProviderGroq, Remaining: 42}}
	engine := NewEngine(s)
	engine.Register(groq)
	engine.RegisterQuotaFetcher(providers.ProviderGroq, quota)

	if _, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "fast"}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if quota.gotKey.Provider != providers.ProviderGroq {
		t.Fatalf("quota provider = %q, want groq", quota.gotKey.Provider)
	}
	if quota.gotKey.ConnID == "" {
		t.Fatal("quota key should include selected connection ID")
	}
	if quota.gotKey.ConnID != groq.receivedKey.ConnID {
		t.Fatalf("quota connection = %q, provider connection = %q", quota.gotKey.ConnID, groq.receivedKey.ConnID)
	}
	if quota.gotKey.Value != "groq-key" {
		t.Fatalf("quota key value = %q, want groq-key", quota.gotKey.Value)
	}
}

func TestDispatchPrefixModelQuotaUsesResolvedProviderConnection(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "openai", "openai-prefix-key")
	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-prefix"}}
	quota := &fakeQuotaFetcher{quota: usage.Quota{Provider: providers.ProviderOpenAI, Remaining: 8}}
	engine := NewEngine(s)
	engine.Register(openAI)
	engine.RegisterQuotaFetcher(providers.ProviderOpenAI, quota)

	resp, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-experimental-prefix"})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if resp.ID != "chatcmpl-prefix" {
		t.Fatalf("response ID = %q, want chatcmpl-prefix", resp.ID)
	}
	if quota.gotKey.Provider != providers.ProviderOpenAI {
		t.Fatalf("quota provider = %q, want openai", quota.gotKey.Provider)
	}
	if quota.gotKey.ConnID == "" || quota.gotKey.ConnID != openAI.receivedKey.ConnID {
		t.Fatalf("quota connection = %q, provider connection = %q", quota.gotKey.ConnID, openAI.receivedKey.ConnID)
	}
	if openAI.received.Model != "gpt-experimental-prefix" {
		t.Fatalf("provider model = %q, want prefix model unchanged", openAI.received.Model)
	}
}

func TestDispatchStreamUsesModelAliasProviderAndRewritesUpstreamModel(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "groq", "groq-key")
	if err := s.SetModelAlias(store.ModelAlias{
		Alias:    "fast-stream",
		Provider: "groq",
		Model:    "llama-3.3-70b-versatile",
	}); err != nil {
		t.Fatalf("SetModelAlias: %v", err)
	}

	content := "hello"
	chunks := make(chan providers.StreamChunk, 1)
	chunks <- providers.StreamChunk{
		ID:    "chunk-groq",
		Model: "llama-3.3-70b-versatile",
		Choices: []providers.StreamChoice{
			{Delta: providers.StreamDelta{Content: &content}},
		},
	}
	close(chunks)

	groq := &fakeProvider{name: providers.ProviderGroq, stream: chunks}
	engine := NewEngine(s)
	engine.Register(groq)

	req := &providers.ChatRequest{Model: "fast-stream"}
	stream, err := engine.DispatchStream(context.Background(), req)
	if err != nil {
		t.Fatalf("DispatchStream: %v", err)
	}
	got, ok := <-stream
	if !ok {
		t.Fatal("stream closed before first chunk")
	}
	if got.ID != "chunk-groq" {
		t.Fatalf("chunk ID = %q, want chunk-groq", got.ID)
	}
	if !groq.streamed {
		t.Fatal("groq stream provider was not called")
	}
	if groq.received == req {
		t.Fatal("alias stream dispatch should pass a copied request")
	}
	if groq.received.Model != "llama-3.3-70b-versatile" {
		t.Fatalf("provider request model = %q, want alias target", groq.received.Model)
	}
	if req.Model != "fast-stream" {
		t.Fatalf("original request model = %q, want alias name unchanged", req.Model)
	}
	if groq.receivedKey.Value != "groq-key" {
		t.Fatalf("key value = %q, want groq-key", groq.receivedKey.Value)
	}
}

func TestDispatchStreamUsesComboModelThroughEngine(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "anthropic", "anthropic-key")
	if err := s.CreateCombo(&store.Combo{
		Name: "streaming",
		Steps: []store.ComboStep{
			{Provider: "anthropic", Model: "claude-sonnet-4"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	content := "hello"
	chunks := make(chan providers.StreamChunk, 1)
	chunks <- providers.StreamChunk{
		ID:    "chunk-combo",
		Model: "claude-sonnet-4",
		Choices: []providers.StreamChoice{
			{Delta: providers.StreamDelta{Content: &content}},
		},
	}
	close(chunks)

	anthropic := &fakeProvider{name: providers.ProviderAnthropic, stream: chunks}
	engine := NewEngine(s)
	engine.Register(anthropic)

	stream, err := engine.DispatchStream(context.Background(), &providers.ChatRequest{Model: "combo/streaming"})
	if err != nil {
		t.Fatalf("DispatchStream: %v", err)
	}
	got, ok := <-stream
	if !ok {
		t.Fatal("stream closed before first chunk")
	}
	if got.ID != "chunk-combo" {
		t.Fatalf("chunk ID = %q, want chunk-combo", got.ID)
	}
	if !anthropic.streamed {
		t.Fatal("anthropic stream provider was not called")
	}
	if anthropic.received.Model != "claude-sonnet-4" {
		t.Fatalf("combo stream model = %q", anthropic.received.Model)
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

func TestDispatchUsesLegacyCodexConnectionForOpenAI(t *testing.T) {
	s := openProxyTestStore(t)
	key := "legacy-codex-key"
	if err := s.CreateConnection(&store.Connection{
		Provider: "codex",
		Name:     "legacy",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &key,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-1"}}
	engine := NewEngine(s)
	engine.Register(openAI)

	if _, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if openAI.receivedKey.Value != "legacy-codex-key" {
		t.Fatalf("provider key = %q, want legacy codex key", openAI.receivedKey.Value)
	}
	if openAI.receivedKey.Provider != providers.ProviderOpenAI {
		t.Fatalf("key provider = %q, want openai", openAI.receivedKey.Provider)
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
