package proxy

import (
	"context"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

func TestGetProviderNotFound(t *testing.T) {
	engine := NewEngine(nil)
	_, ok := engine.GetProvider(providers.ProviderOpenAI)
	if ok {
		t.Fatal("expected provider not found")
	}
}

type fakeIsModelDisabledStore struct {
	*store.Store
	err error
}

func (f *fakeIsModelDisabledStore) IsModelDisabled(provider, model string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.Store.IsModelDisabled(provider, model)
}

func TestFilterDisabledModelsError(t *testing.T) {
	s := openProxyTestStore(t)
	key := "sk-openai"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai", Name: "p", AuthType: store.AuthTypeAPIKey, APIKey: &key, IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	openAI := &fakeProvider{
		name:   providers.ProviderOpenAI,
		models: []providers.Model{{ID: "gpt-4o", Provider: providers.ProviderOpenAI}},
	}

	fakeStore := &fakeIsModelDisabledStore{Store: s, err: errors.New("db error")}
	engine := NewEngine(fakeStore)
	engine.Register(openAI)

	models, err := engine.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}

	if len(models) != 0 {
		t.Fatalf("expected 0 models when IsModelDisabled errors, got %d", len(models))
	}
}

type fakeListCustomStore struct {
	*store.Store
	err error
}

func (f *fakeListCustomStore) ListCustomModels() ([]store.CustomModel, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.Store.ListCustomModels()
}

func TestListCustomModelsError(t *testing.T) {
	s := openProxyTestStore(t)
	key := "sk-openai"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai", Name: "p", AuthType: store.AuthTypeAPIKey, APIKey: &key, IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	openAI := &fakeProvider{
		name:   providers.ProviderOpenAI,
		models: []providers.Model{{ID: "gpt-4o", Provider: providers.ProviderOpenAI}},
	}

	fakeStore := &fakeListCustomStore{Store: s, err: errors.New("db error")}
	engine := NewEngine(fakeStore)
	engine.Register(openAI)

	models, err := engine.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}

	if len(models) == 0 {
		t.Fatal("expected catalog models even when ListCustomModels errors")
	}
}

type fakeProxyProvider struct {
	fakeProvider
	withProxyPoolCalled bool
	receivedPool        *store.ProxyPool
}

func (f *fakeProxyProvider) WithProxyPool(pool *store.ProxyPool) providers.Provider {
	f.withProxyPoolCalled = true
	f.receivedPool = pool
	return f
}

type fakeProxyEngineStore struct {
	*store.Store
	getConnectionProxyPoolID func(string) (*string, error)
	getProxyPool             func(string) (*store.ProxyPool, error)
}

func (f *fakeProxyEngineStore) GetConnectionProxyPoolID(connID string) (*string, error) {
	if f.getConnectionProxyPoolID != nil {
		return f.getConnectionProxyPoolID(connID)
	}
	return f.Store.GetConnectionProxyPoolID(connID)
}

func (f *fakeProxyEngineStore) GetProxyPool(id string) (*store.ProxyPool, error) {
	if f.getProxyPool != nil {
		return f.getProxyPool(id)
	}
	return f.Store.GetProxyPool(id)
}

func TestProviderWithProxyPoolIDError(t *testing.T) {
	s := openProxyTestStore(t)
	engine := NewEngine(&fakeProxyEngineStore{
		Store: s,
		getConnectionProxyPoolID: func(string) (*string, error) {
			return nil, errors.New("db error")
		},
	})

	provider := &fakeProxyProvider{}
	conn := &store.Connection{ID: "conn1"}

	result := engine.providerWithProxy(provider, conn)
	if result != provider {
		t.Fatal("expected same provider when GetConnectionProxyPoolID errors")
	}
}

func TestProviderWithProxyPoolIDNil(t *testing.T) {
	s := openProxyTestStore(t)
	engine := NewEngine(s)

	provider := &fakeProxyProvider{}
	conn := &store.Connection{ID: "conn1"}

	result := engine.providerWithProxy(provider, conn)
	if result != provider {
		t.Fatal("expected same provider when proxy pool id is nil")
	}
}

func TestProviderWithProxyNotConfigurable(t *testing.T) {
	s := openProxyTestStore(t)
	engine := NewEngine(&fakeProxyEngineStore{
		Store: s,
		getConnectionProxyPoolID: func(string) (*string, error) {
			id := "some-pool"
			return &id, nil
		},
	})

	provider := &fakeProvider{name: providers.ProviderOpenAI}
	conn := &store.Connection{ID: "conn1"}

	result := engine.providerWithProxy(provider, conn)
	if result != provider {
		t.Fatal("expected same provider when not proxyConfigurable")
	}
}

func TestProviderWithProxyCacheHit(t *testing.T) {
	s := openProxyTestStore(t)
	engine := NewEngine(s)

	provider := &fakeProxyProvider{fakeProvider: fakeProvider{name: providers.ProviderOpenAI}}
	conn := &store.Connection{ID: "conn1"}

	pool := &store.ProxyPool{Name: "test", Protocol: "http", Host: "host", Port: 8080}
	created, err := s.CreateProxyPool(*pool)
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}

	key := "sk-test"
	dbConn := &store.Connection{
		Provider: "openai", Name: "p", AuthType: store.AuthTypeAPIKey, APIKey: &key, IsActive: true,
	}
	if err := s.CreateConnection(dbConn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if err := s.UpdateConnectionProxyPool(dbConn.ID, &created.ID); err != nil {
		t.Fatalf("UpdateConnectionProxyPool: %v", err)
	}

	conn.ID = dbConn.ID
	result1 := engine.providerWithProxy(provider, conn)
	if !provider.withProxyPoolCalled {
		t.Fatal("expected WithProxyPool to be called on first call")
	}
	if result1 != provider {
		t.Fatal("expected same provider instance returned")
	}

	provider.withProxyPoolCalled = false

	result2 := engine.providerWithProxy(provider, conn)
	if provider.withProxyPoolCalled {
		t.Fatal("expected WithProxyPool NOT to be called on cache hit")
	}
	if result2 != result1 {
		t.Fatal("expected cached provider on second call")
	}
}

func TestProviderWithProxyPoolNotFound(t *testing.T) {
	s := openProxyTestStore(t)
	engine := NewEngine(&fakeProxyEngineStore{
		Store: s,
		getConnectionProxyPoolID: func(string) (*string, error) {
			id := "missing-pool"
			return &id, nil
		},
		getProxyPool: func(string) (*store.ProxyPool, error) {
			return nil, store.ErrNotFound
		},
	})

	provider := &fakeProxyProvider{fakeProvider: fakeProvider{name: providers.ProviderOpenAI}}
	conn := &store.Connection{ID: "conn1"}

	result := engine.providerWithProxy(provider, conn)
	if result != provider {
		t.Fatal("expected same provider when GetProxyPool returns error")
	}
}

func TestProviderWithProxySuccess(t *testing.T) {
	s := openProxyTestStore(t)
	engine := NewEngine(s)

	provider := &fakeProxyProvider{fakeProvider: fakeProvider{name: providers.ProviderOpenAI}}
	conn := &store.Connection{ID: "conn1"}

	pool := &store.ProxyPool{Name: "test", Protocol: "http", Host: "host", Port: 8080}
	created, err := s.CreateProxyPool(*pool)
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}

	key := "sk-test"
	dbConn := &store.Connection{
		Provider: "openai", Name: "p", AuthType: store.AuthTypeAPIKey, APIKey: &key, IsActive: true,
	}
	if err := s.CreateConnection(dbConn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if err := s.UpdateConnectionProxyPool(dbConn.ID, &created.ID); err != nil {
		t.Fatalf("UpdateConnectionProxyPool: %v", err)
	}

	conn.ID = dbConn.ID
	result := engine.providerWithProxy(provider, conn)
	if result != provider {
		t.Fatal("expected same provider instance (WithProxyPool returns itself)")
	}
	if !provider.withProxyPoolCalled {
		t.Fatal("expected WithProxyPool to be called")
	}
	if provider.receivedPool == nil {
		t.Fatal("expected received pool to be set")
	}
}

func TestRoutableModelRouteIsModelDisabledError(t *testing.T) {
	s := openProxyTestStore(t)
	fakeStore := &fakeIsModelDisabledStore{Store: s, err: errors.New("db error")}
	engine := NewEngine(fakeStore)

	_, err := engine.routableModelRoute(modelRoute{Provider: providers.ProviderOpenAI, Model: "gpt-4o"})
	if err == nil {
		t.Fatal("expected error when IsModelDisabled fails")
	}
}

func TestDispatchIsModelDisabledError(t *testing.T) {
	s := openProxyTestStore(t)
	key := "sk-openai"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai", Name: "p", AuthType: store.AuthTypeAPIKey, APIKey: &key, IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	openAI := &fakeProvider{name: providers.ProviderOpenAI}
	fakeStore := &fakeIsModelDisabledStore{Store: s, err: errors.New("db error")}
	engine := NewEngine(fakeStore)
	engine.Register(openAI)

	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
	if err == nil {
		t.Fatal("expected error when IsModelDisabled fails during dispatch")
	}
}

func TestProviderWithProxyPoolIDErrorWrapped(t *testing.T) {
	// Cover the GetConnectionProxyPoolID error branch with a wrapped error
	s := openProxyTestStore(t)
	engine := NewEngine(&fakeProxyEngineStore{
		Store: s,
		getConnectionProxyPoolID: func(string) (*string, error) {
			return nil, errors.New("wrapped: db connection failed")
		},
	})

	provider := &fakeProxyProvider{fakeProvider: fakeProvider{name: providers.ProviderOpenAI}}
	conn := &store.Connection{ID: "conn1"}

	result := engine.providerWithProxy(provider, conn)
	if result != provider {
		t.Fatal("expected same provider when GetConnectionProxyPoolID returns any error")
	}
}
