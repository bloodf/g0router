package proxy

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
)

func TestComboResolvePreservesStoredStepOrder(t *testing.T) {
	s := openProxyTestStore(t)
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

	resolver := NewComboResolver(s)
	steps, err := resolver.Resolve("fast-fallback")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("step count = %d, want 2", len(steps))
	}
	if steps[0].Provider != providers.ProviderGroq || steps[0].Model != "llama-3.3-70b-versatile" {
		t.Fatalf("first step = %#v, want groq llama-3.3-70b-versatile", steps[0])
	}
	if steps[1].Provider != providers.ProviderOpenAI || steps[1].Model != "gpt-4o-mini" {
		t.Fatalf("second step = %#v, want openai gpt-4o-mini", steps[1])
	}
}

func TestComboDispatchFallsBackSequentially(t *testing.T) {
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

	groq := &fakeProvider{
		name: providers.ProviderGroq,
		err:  errors.New("rate limited"),
	}
	openAI := &fakeProvider{
		name: providers.ProviderOpenAI,
		response: &providers.ChatResponse{
			ID:    "chatcmpl-fallback",
			Model: "gpt-4o-mini",
		},
	}
	engine := NewEngine(s)
	engine.Register(groq)
	engine.Register(openAI)

	req := &providers.ChatRequest{Model: "combo/fast-fallback"}
	resp, err := NewComboResolver(s).Dispatch(context.Background(), engine, "fast-fallback", req)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if resp.ID != "chatcmpl-fallback" {
		t.Fatalf("response ID = %q, want chatcmpl-fallback", resp.ID)
	}
	if !groq.called {
		t.Fatal("first combo step was not called")
	}
	if !openAI.called {
		t.Fatal("second combo step was not called")
	}
	if groq.received.Model != "llama-3.3-70b-versatile" {
		t.Fatalf("groq request model = %q, want llama-3.3-70b-versatile", groq.received.Model)
	}
	if openAI.received.Model != "gpt-4o-mini" {
		t.Fatalf("openai request model = %q, want gpt-4o-mini", openAI.received.Model)
	}
	if req.Model != "combo/fast-fallback" {
		t.Fatalf("original request model = %q, want combo/fast-fallback", req.Model)
	}
	if openAI.receivedKey.Value != "openai-key" {
		t.Fatalf("openai key = %q, want openai-key", openAI.receivedKey.Value)
	}
}

func TestComboDispatchResolvesCatalogProviderForStepModel(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "openai", "openai-key")
	createProxyConnection(t, s, "anthropic", "anthropic-key")
	if err := s.CreateCombo(&store.Combo{
		Name: "catalog-step",
		Steps: []store.ComboStep{
			{Provider: "openai", Model: "claude-sonnet-4"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "wrong-provider"}}
	anthropic := &fakeProvider{name: providers.ProviderAnthropic, response: &providers.ChatResponse{ID: "chatcmpl-anthropic"}}
	engine := NewEngine(s)
	engine.Register(openAI)
	engine.Register(anthropic)

	resp, err := NewComboResolver(s).Dispatch(context.Background(), engine, "catalog-step", &providers.ChatRequest{Model: "combo/catalog-step"})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if resp.ID != "chatcmpl-anthropic" {
		t.Fatalf("response ID = %q, want anthropic response", resp.ID)
	}
	if openAI.called {
		t.Fatal("stored openai provider should not be called for catalog-owned Anthropic model")
	}
	if !anthropic.called || anthropic.received.Model != "claude-sonnet-4" {
		t.Fatalf("anthropic request = %+v", anthropic.received)
	}
}

func TestComboDispatchResolvesAliasStepAndRewritesModel(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "openai", "openai-key")
	createProxyConnection(t, s, "groq", "groq-key")
	if err := s.SetModelAlias(store.ModelAlias{
		Alias:    "fast-step",
		Provider: "groq",
		Model:    "llama-3.3-70b-versatile",
	}); err != nil {
		t.Fatalf("SetModelAlias: %v", err)
	}
	if err := s.CreateCombo(&store.Combo{
		Name: "alias-step",
		Steps: []store.ComboStep{
			{Provider: "openai", Model: "fast-step"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "wrong-provider"}}
	groq := &fakeProvider{name: providers.ProviderGroq, response: &providers.ChatResponse{ID: "chatcmpl-groq"}}
	engine := NewEngine(s)
	engine.Register(openAI)
	engine.Register(groq)

	resp, err := NewComboResolver(s).Dispatch(context.Background(), engine, "alias-step", &providers.ChatRequest{Model: "combo/alias-step"})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if resp.ID != "chatcmpl-groq" {
		t.Fatalf("response ID = %q, want groq response", resp.ID)
	}
	if openAI.called {
		t.Fatal("stored openai provider should not be called for alias-owned Groq model")
	}
	if !groq.called || groq.received.Model != "llama-3.3-70b-versatile" {
		t.Fatalf("groq request = %+v", groq.received)
	}
}

func TestComboDispatchUsesBedrockAdapterOnlyStep(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "bedrock", "bedrock-key")
	createProxyConnection(t, s, "openai", "openai-key")
	if err := s.CreateCombo(&store.Combo{
		Name: "bedrock-then-openai",
		Steps: []store.ComboStep{
			{Provider: "bedrock", Model: "anthropic.claude-3-haiku-20240307-v1:0"},
			{Provider: "openai", Model: "gpt-4o-mini"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	bedrock := &fakeProvider{name: providers.ProviderBedrock, response: &providers.ChatResponse{ID: "chatcmpl-bedrock"}}
	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-openai"}}
	engine := NewEngine(s)
	engine.Register(bedrock)
	engine.Register(openAI)

	resp, err := NewComboResolver(s).Dispatch(context.Background(), engine, "bedrock-then-openai", &providers.ChatRequest{Model: "combo/bedrock-then-openai"})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if !bedrock.called || resp.ID != "chatcmpl-bedrock" {
		t.Fatalf("bedrock called=%v resp=%+v", bedrock.called, resp)
	}
	if openAI.called {
		t.Fatal("openai fallback should not run after Bedrock succeeds")
	}
}

func TestComboDispatchRetriesNextAccountBeforeNextStep(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "groq", "groq-key-1")
	createProxyConnection(t, s, "groq", "groq-key-2")
	createProxyConnection(t, s, "openai", "openai-key")
	if err := s.CreateCombo(&store.Combo{
		Name: "account-fallback",
		Steps: []store.ComboStep{
			{Provider: "groq", Model: "llama-3.3-70b-versatile"},
			{Provider: "openai", Model: "gpt-4o-mini"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	groq := &fakeProvider{
		name:      providers.ProviderGroq,
		errs:      []error{errors.New("rate limited"), nil},
		responses: []*providers.ChatResponse{nil, &providers.ChatResponse{ID: "chatcmpl-groq-second-account"}},
	}
	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-openai"}}
	engine := NewEngine(s)
	engine.Register(groq)
	engine.Register(openAI)

	resp, err := NewComboResolver(s).Dispatch(context.Background(), engine, "account-fallback", &providers.ChatRequest{Model: "combo/account-fallback"})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if resp.ID != "chatcmpl-groq-second-account" {
		t.Fatalf("response ID = %q, want second Groq account", resp.ID)
	}
	if groq.calls != 2 {
		t.Fatalf("groq calls = %d, want 2", groq.calls)
	}
	if len(groq.keys) != 2 || groq.keys[0].Value == groq.keys[1].Value {
		t.Fatalf("groq keys = %+v; want two different accounts", groq.keys)
	}
	if openAI.called {
		t.Fatal("second combo step should not run before retrying next Groq account")
	}
}

func TestComboDispatchRefreshesOAuthConnectionBeforeProviderCall(t *testing.T) {
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
	if err := s.CreateCombo(&store.Combo{
		Name: "openai-only",
		Steps: []store.ComboStep{
			{Provider: "openai", Model: "gpt-4o-mini"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-combo"}}
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

	_, err := NewComboResolver(s).Dispatch(
		context.Background(),
		engine,
		"openai-only",
		&providers.ChatRequest{Model: "combo/openai-only"},
	)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if refresher.calls != 1 {
		t.Fatalf("refresh calls = %d, want 1", refresher.calls)
	}
	if openAI.receivedKey.Value != "new-access" {
		t.Fatalf("combo key = %q, want refreshed access token", openAI.receivedKey.Value)
	}
}

func TestComboDispatchCanonicalizesLegacyProviderStep(t *testing.T) {
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
	if err := s.CreateCombo(&store.Combo{
		Name: "legacy-openai",
		Steps: []store.ComboStep{
			{Provider: "codex", Model: "gpt-4o-mini"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-combo"}}
	engine := NewEngine(s)
	engine.Register(openAI)

	_, err := NewComboResolver(s).Dispatch(
		context.Background(),
		engine,
		"legacy-openai",
		&providers.ChatRequest{Model: "combo/legacy-openai"},
	)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if openAI.receivedKey.Value != "legacy-codex-key" {
		t.Fatalf("combo key = %q, want legacy codex key", openAI.receivedKey.Value)
	}
	if openAI.received.Model != "gpt-4o-mini" {
		t.Fatalf("combo model = %q, want gpt-4o-mini", openAI.received.Model)
	}
}

func TestComboDispatchReturnsLastStepError(t *testing.T) {
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

	lastErr := errors.New("quota exhausted")
	engine := NewEngine(s)
	engine.Register(&fakeProvider{name: providers.ProviderGroq, err: errors.New("rate limited")})
	engine.Register(&fakeProvider{name: providers.ProviderOpenAI, err: lastErr})

	_, err := NewComboResolver(s).Dispatch(
		context.Background(),
		engine,
		"fast-fallback",
		&providers.ChatRequest{Model: "combo/fast-fallback"},
	)
	if !errors.Is(err, lastErr) {
		t.Fatalf("Dispatch error = %v, want wrapped last step error", err)
	}
}

func TestComboDispatchQuotaUsesStepProviderConnection(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "groq", "groq-key")
	if err := s.CreateCombo(&store.Combo{
		Name: "groq-only",
		Steps: []store.ComboStep{
			{Provider: "groq", Model: "llama-3.3-70b-versatile"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	groq := &fakeProvider{name: providers.ProviderGroq, response: &providers.ChatResponse{ID: "chatcmpl-groq"}}
	quota := &fakeQuotaFetcher{quota: usage.Quota{Provider: providers.ProviderGroq, Remaining: 5}}
	engine := NewEngine(s)
	engine.Register(groq)
	engine.RegisterQuotaFetcher(providers.ProviderGroq, quota)

	resp, err := NewComboResolver(s).Dispatch(
		context.Background(),
		engine,
		"groq-only",
		&providers.ChatRequest{Model: "combo/groq-only"},
	)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if resp.ID != "chatcmpl-groq" {
		t.Fatalf("response ID = %q, want chatcmpl-groq", resp.ID)
	}
	if quota.gotKey.Provider != providers.ProviderGroq {
		t.Fatalf("quota provider = %q, want groq", quota.gotKey.Provider)
	}
	if quota.gotKey.ConnID == "" || quota.gotKey.ConnID != groq.receivedKey.ConnID {
		t.Fatalf("quota connection = %q, provider connection = %q", quota.gotKey.ConnID, groq.receivedKey.ConnID)
	}
	if quota.gotKey.Value != "groq-key" {
		t.Fatalf("quota key = %q, want groq-key", quota.gotKey.Value)
	}
}

func TestComboDispatchQuotaExhaustedAccountStopsBeforeNextStep(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "groq", "groq-key-1")
	createProxyConnection(t, s, "groq", "groq-key-2")
	createProxyConnection(t, s, "openai", "openai-key")
	if err := s.CreateCombo(&store.Combo{
		Name: "quota-account-fallback",
		Steps: []store.ComboStep{
			{Provider: "groq", Model: "llama-3.3-70b-versatile"},
			{Provider: "openai", Model: "gpt-4o-mini"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	groq := &fakeProvider{name: providers.ProviderGroq, response: &providers.ChatResponse{ID: "chatcmpl-groq-second-account"}}
	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-openai"}}
	quota := &fakeQuotaFetcher{quotas: []usage.Quota{
		{Provider: providers.ProviderGroq, Remaining: 0},
		{Provider: providers.ProviderGroq, Remaining: 12},
	}}
	engine := NewEngine(s)
	engine.Register(groq)
	engine.Register(openAI)
	engine.RegisterQuotaFetcher(providers.ProviderGroq, quota)

	_, err := NewComboResolver(s).Dispatch(
		context.Background(),
		engine,
		"quota-account-fallback",
		&providers.ChatRequest{Model: "combo/quota-account-fallback"},
	)
	if !errors.Is(err, ErrQuotaExhausted) {
		t.Fatalf("Dispatch error = %v, want ErrQuotaExhausted", err)
	}
	if groq.calls != 0 {
		t.Fatalf("groq calls = %d, want 0", groq.calls)
	}
	if len(quota.keys) != 1 {
		t.Fatalf("quota keys = %+v; want only one selected Groq account", quota.keys)
	}
	if openAI.called {
		t.Fatal("second combo step should not run after quota exhaustion")
	}
}

func TestComboDispatchAllQuotaExhaustedStepsReturnQuotaError(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "groq", "groq-key")
	createProxyConnection(t, s, "openai", "openai-key")
	if err := s.CreateCombo(&store.Combo{
		Name: "all-exhausted",
		Steps: []store.ComboStep{
			{Provider: "groq", Model: "llama-3.3-70b-versatile"},
			{Provider: "openai", Model: "gpt-4o-mini"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	groq := &fakeProvider{name: providers.ProviderGroq, response: &providers.ChatResponse{ID: "groq-should-not-run"}}
	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "openai-should-not-run"}}
	engine := NewEngine(s)
	engine.Register(groq)
	engine.Register(openAI)
	engine.RegisterQuotaFetcher(providers.ProviderGroq, &fakeQuotaFetcher{quota: usage.Quota{Provider: providers.ProviderGroq, Remaining: 0}})
	engine.RegisterQuotaFetcher(providers.ProviderOpenAI, &fakeQuotaFetcher{quota: usage.Quota{Provider: providers.ProviderOpenAI, Remaining: 0}})

	_, err := NewComboResolver(s).Dispatch(
		context.Background(),
		engine,
		"all-exhausted",
		&providers.ChatRequest{Model: "combo/all-exhausted"},
	)
	if !errors.Is(err, ErrQuotaExhausted) {
		t.Fatalf("Dispatch error = %v, want ErrQuotaExhausted", err)
	}
	if groq.called || openAI.called {
		t.Fatalf("providers should not be called when quotas are exhausted: groq=%v openai=%v", groq.called, openAI.called)
	}
}

func TestComboDispatchStreamQuotaUsesStepProviderConnection(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "anthropic", "anthropic-key")
	if err := s.CreateCombo(&store.Combo{
		Name: "anthropic-stream",
		Steps: []store.ComboStep{
			{Provider: "anthropic", Model: "claude-sonnet-4"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	chunks := make(chan providers.StreamChunk, 1)
	chunks <- providers.StreamChunk{ID: "chunk-anthropic", Model: "claude-sonnet-4"}
	close(chunks)
	anthropic := &fakeProvider{name: providers.ProviderAnthropic, stream: chunks}
	quota := &fakeQuotaFetcher{quota: usage.Quota{Provider: providers.ProviderAnthropic, Remaining: 3}}
	engine := NewEngine(s)
	engine.Register(anthropic)
	engine.RegisterQuotaFetcher(providers.ProviderAnthropic, quota)

	stream, err := NewComboResolver(s).DispatchStream(
		context.Background(),
		engine,
		"anthropic-stream",
		&providers.ChatRequest{Model: "combo/anthropic-stream"},
	)
	if err != nil {
		t.Fatalf("DispatchStream: %v", err)
	}
	got, ok := <-stream
	if !ok {
		t.Fatal("stream closed before first chunk")
	}
	if got.ID != "chunk-anthropic" {
		t.Fatalf("chunk ID = %q, want chunk-anthropic", got.ID)
	}
	if quota.gotKey.Provider != providers.ProviderAnthropic {
		t.Fatalf("quota provider = %q, want anthropic", quota.gotKey.Provider)
	}
	if quota.gotKey.ConnID == "" || quota.gotKey.ConnID != anthropic.receivedKey.ConnID {
		t.Fatalf("quota connection = %q, provider connection = %q", quota.gotKey.ConnID, anthropic.receivedKey.ConnID)
	}
	if quota.gotKey.Value != "anthropic-key" {
		t.Fatalf("quota key = %q, want anthropic-key", quota.gotKey.Value)
	}
}

func TestComboDispatchStreamQuotaExhaustionBlocksProviderCall(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "anthropic", "anthropic-key")
	if err := s.CreateCombo(&store.Combo{
		Name: "anthropic-stream-exhausted",
		Steps: []store.ComboStep{
			{Provider: "anthropic", Model: "claude-sonnet-4"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	chunks := make(chan providers.StreamChunk)
	close(chunks)
	anthropic := &fakeProvider{name: providers.ProviderAnthropic, stream: chunks}
	quota := &fakeQuotaFetcher{quota: usage.Quota{Provider: providers.ProviderAnthropic, Remaining: 0}}
	engine := NewEngine(s)
	engine.Register(anthropic)
	engine.RegisterQuotaFetcher(providers.ProviderAnthropic, quota)

	_, err := NewComboResolver(s).DispatchStream(
		context.Background(),
		engine,
		"anthropic-stream-exhausted",
		&providers.ChatRequest{Model: "combo/anthropic-stream-exhausted"},
	)
	if !errors.Is(err, ErrQuotaExhausted) {
		t.Fatalf("DispatchStream error = %v, want ErrQuotaExhausted", err)
	}
	if anthropic.streamed {
		t.Fatal("stream provider should not be called when combo quota is exhausted")
	}
}

func TestComboDispatchStreamQuotaExhaustedStepStopsBeforeFallback(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "groq", "groq-key")
	createProxyConnection(t, s, "openai", "openai-key")
	if err := s.CreateCombo(&store.Combo{
		Name: "quota-stream-fallback",
		Steps: []store.ComboStep{
			{Provider: "groq", Model: "llama-3.3-70b-versatile"},
			{Provider: "openai", Model: "gpt-4o-mini"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	groqChunks := make(chan providers.StreamChunk)
	close(groqChunks)
	openAIChunks := make(chan providers.StreamChunk, 1)
	openAIChunks <- providers.StreamChunk{ID: "chunk-openai", Model: "gpt-4o-mini"}
	close(openAIChunks)
	groq := &fakeProvider{name: providers.ProviderGroq, stream: groqChunks}
	openAI := &fakeProvider{name: providers.ProviderOpenAI, stream: openAIChunks}
	engine := NewEngine(s)
	engine.Register(groq)
	engine.Register(openAI)
	engine.RegisterQuotaFetcher(providers.ProviderGroq, &fakeQuotaFetcher{quota: usage.Quota{Provider: providers.ProviderGroq, Remaining: 0}})
	engine.RegisterQuotaFetcher(providers.ProviderOpenAI, &fakeQuotaFetcher{quota: usage.Quota{Provider: providers.ProviderOpenAI, Remaining: 8}})

	_, err := NewComboResolver(s).DispatchStream(
		context.Background(),
		engine,
		"quota-stream-fallback",
		&providers.ChatRequest{Model: "combo/quota-stream-fallback"},
	)
	if !errors.Is(err, ErrQuotaExhausted) {
		t.Fatalf("DispatchStream error = %v, want ErrQuotaExhausted", err)
	}
	if groq.streamed {
		t.Fatal("quota-exhausted combo stream step should not open provider stream")
	}
	if openAI.streamed {
		t.Fatal("fallback combo stream step should not open after quota exhaustion")
	}
}

func TestSelectAutoStepIndex(t *testing.T) {
	steps := []ComboStep{
		{Provider: providers.ProviderAnthropic, Model: "claude-sonnet-4"},
		{Provider: providers.ProviderOpenAI, Model: "gpt-4o-mini"},
		{Provider: providers.ProviderGroq, Model: "llama-3.3-70b-versatile"},
	}

	small := &providers.ChatRequest{
		Messages: []providers.Message{{Role: "user", Content: "hi"}},
	}
	if got := selectAutoStepIndex(steps, small); got != len(steps)-1 {
		t.Fatalf("small request index = %d, want %d (cheapest/last)", got, len(steps)-1)
	}

	withTools := &providers.ChatRequest{
		Messages: []providers.Message{{Role: "user", Content: "hi"}},
		Tools:    []providers.Tool{{Type: "function"}},
	}
	if got := selectAutoStepIndex(steps, withTools); got != 0 {
		t.Fatalf("tool request index = %d, want 0 (most capable/first)", got)
	}

	bigContent := make([]byte, 9000)
	for i := range bigContent {
		bigContent[i] = 'x'
	}
	large := &providers.ChatRequest{
		Messages: []providers.Message{{Role: "user", Content: string(bigContent)}},
	}
	if got := selectAutoStepIndex(steps, large); got != 0 {
		t.Fatalf("large request index = %d, want 0 (most capable/first)", got)
	}
}

func TestComboDispatchRoundRobinRotatesFirstStep(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "groq", "groq-key")
	createProxyConnection(t, s, "openai", "openai-key")
	createProxyConnection(t, s, "anthropic", "anthropic-key")
	if err := s.CreateCombo(&store.Combo{
		Name:     "rr",
		Strategy: store.ComboStrategyRoundRobin,
		Steps: []store.ComboStep{
			{Provider: "groq", Model: "llama-3.3-70b-versatile"},
			{Provider: "openai", Model: "gpt-4o-mini"},
			{Provider: "anthropic", Model: "claude-sonnet-4"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	groq := &fakeProvider{name: providers.ProviderGroq, response: &providers.ChatResponse{ID: "groq"}}
	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "openai"}}
	anthropic := &fakeProvider{name: providers.ProviderAnthropic, response: &providers.ChatResponse{ID: "anthropic"}}
	engine := NewEngine(s)
	engine.Register(groq)
	engine.Register(openAI)
	engine.Register(anthropic)

	resolver := NewComboResolver(s)
	var firstIDs []string
	for i := 0; i < 3; i++ {
		resp, err := resolver.Dispatch(context.Background(), engine, "rr", &providers.ChatRequest{Model: "combo/rr"})
		if err != nil {
			t.Fatalf("Dispatch %d: %v", i, err)
		}
		firstIDs = append(firstIDs, resp.ID)
	}
	if firstIDs[0] == firstIDs[1] || firstIDs[1] == firstIDs[2] || firstIDs[0] == firstIDs[2] {
		t.Fatalf("round_robin did not rotate first step: %v", firstIDs)
	}
}

func TestComboDispatchLeastUsedPicksLowestCount(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "groq", "groq-key")
	createProxyConnection(t, s, "openai", "openai-key")
	if err := s.CreateCombo(&store.Combo{
		Name:     "lu",
		Strategy: store.ComboStrategyLeastUsed,
		Steps: []store.ComboStep{
			{Provider: "groq", Model: "llama-3.3-70b-versatile"},
			{Provider: "openai", Model: "gpt-4o-mini"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	groq := &fakeProvider{name: providers.ProviderGroq, response: &providers.ChatResponse{ID: "groq"}}
	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "openai"}}
	engine := NewEngine(s)
	engine.Register(groq)
	engine.Register(openAI)

	resolver := NewComboResolver(s)
	// First call goes to step 0 (groq) on a tie, skewing its count.
	if _, err := resolver.Dispatch(context.Background(), engine, "lu", &providers.ChatRequest{Model: "combo/lu"}); err != nil {
		t.Fatalf("Dispatch first: %v", err)
	}
	// Second call must now pick the least-used step (openai).
	resp, err := resolver.Dispatch(context.Background(), engine, "lu", &providers.ChatRequest{Model: "combo/lu"})
	if err != nil {
		t.Fatalf("Dispatch second: %v", err)
	}
	if resp.ID != "openai" {
		t.Fatalf("least_used second pick = %q, want openai", resp.ID)
	}
}

func TestComboDispatchAutoPicksCapableForTools(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "anthropic", "anthropic-key")
	createProxyConnection(t, s, "groq", "groq-key")
	if err := s.CreateCombo(&store.Combo{
		Name:     "auto",
		Strategy: store.ComboStrategyAuto,
		Steps: []store.ComboStep{
			{Provider: "anthropic", Model: "claude-sonnet-4"},
			{Provider: "groq", Model: "llama-3.3-70b-versatile"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	anthropic := &fakeProvider{name: providers.ProviderAnthropic, response: &providers.ChatResponse{ID: "anthropic"}}
	groq := &fakeProvider{name: providers.ProviderGroq, response: &providers.ChatResponse{ID: "groq"}}
	engine := NewEngine(s)
	engine.Register(anthropic)
	engine.Register(groq)

	resolver := NewComboResolver(s)
	toolResp, err := resolver.Dispatch(context.Background(), engine, "auto", &providers.ChatRequest{
		Model: "combo/auto",
		Tools: []providers.Tool{{Type: "function"}},
	})
	if err != nil {
		t.Fatalf("Dispatch tools: %v", err)
	}
	if toolResp.ID != "anthropic" {
		t.Fatalf("auto tool pick = %q, want anthropic (most capable)", toolResp.ID)
	}

	plainResp, err := resolver.Dispatch(context.Background(), engine, "auto", &providers.ChatRequest{
		Model:    "combo/auto",
		Messages: []providers.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Dispatch plain: %v", err)
	}
	if plainResp.ID != "groq" {
		t.Fatalf("auto plain pick = %q, want groq (cheapest/last)", plainResp.ID)
	}
}

func TestComboDispatchStrategyFallbackOnError(t *testing.T) {
	for _, strategy := range []string{
		store.ComboStrategyRoundRobin,
		store.ComboStrategyLeastUsed,
		store.ComboStrategyAuto,
	} {
		t.Run(strategy, func(t *testing.T) {
			s := openProxyTestStore(t)
			createProxyConnection(t, s, "groq", "groq-key")
			createProxyConnection(t, s, "openai", "openai-key")
			if err := s.CreateCombo(&store.Combo{
				Name:     "fb",
				Strategy: strategy,
				Steps: []store.ComboStep{
					{Provider: "groq", Model: "llama-3.3-70b-versatile"},
					{Provider: "openai", Model: "gpt-4o-mini"},
				},
				IsActive: true,
			}); err != nil {
				t.Fatalf("CreateCombo: %v", err)
			}

			// Both steps error to force trying every step regardless of which is first.
			groq := &fakeProvider{name: providers.ProviderGroq, err: errors.New("rate limited")}
			openAI := &fakeProvider{name: providers.ProviderOpenAI, err: errors.New("rate limited")}
			engine := NewEngine(s)
			engine.Register(groq)
			engine.Register(openAI)

			_, err := NewComboResolver(s).Dispatch(context.Background(), engine, "fb", &providers.ChatRequest{Model: "combo/fb"})
			if err == nil {
				t.Fatal("expected error when all steps fail")
			}
			if !groq.called || !openAI.called {
				t.Fatalf("both steps should be attempted as fallbacks: groq=%v openai=%v", groq.called, openAI.called)
			}
		})
	}
}

func TestComboDispatchConcurrentStrategiesRaceFree(t *testing.T) {
	for _, strategy := range []string{store.ComboStrategyRoundRobin, store.ComboStrategyLeastUsed} {
		t.Run(strategy, func(t *testing.T) {
			s := openProxyTestStore(t)
			createProxyConnection(t, s, "groq", "groq-key")
			createProxyConnection(t, s, "openai", "openai-key")
			createProxyConnection(t, s, "anthropic", "anthropic-key")
			if err := s.CreateCombo(&store.Combo{
				Name:     "conc",
				Strategy: strategy,
				Steps: []store.ComboStep{
					{Provider: "groq", Model: "llama-3.3-70b-versatile"},
					{Provider: "openai", Model: "gpt-4o-mini"},
					{Provider: "anthropic", Model: "claude-sonnet-4"},
				},
				IsActive: true,
			}); err != nil {
				t.Fatalf("CreateCombo: %v", err)
			}

			// Exercise the selector's shared cursor/counts state directly under
			// concurrency; this is the only mutable state the strategies add.
			steps, st, err := NewComboResolver(s).resolveWithStrategy("conc")
			if err != nil {
				t.Fatalf("resolveWithStrategy: %v", err)
			}
			resolver := NewComboResolver(s)
			req := &providers.ChatRequest{Model: "combo/conc"}
			var wg sync.WaitGroup
			for i := 0; i < 50; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					ordered := resolver.orderComboSteps("conc", st, steps, req)
					if len(ordered) != len(steps) {
						t.Errorf("ordered len = %d, want %d", len(ordered), len(steps))
					}
				}()
			}
			wg.Wait()
		})
	}
}

func createProxyConnection(t *testing.T, s *store.Store, provider string, key string) {
	t.Helper()

	if err := s.CreateConnection(&store.Connection{
		Provider: provider,
		Name:     provider + "-primary",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &key,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection %s: %v", provider, err)
	}
}
