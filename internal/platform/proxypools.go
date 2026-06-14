package platform

import "github.com/bloodf/g0router/internal/store"

// ProxyPoolService is the domain service for proxy pools (transport→domain→
// repository). It owns CRUD wrappers over the store, the connectivity test, and
// per-connection proxy resolution. Stub bodies land in T-proxypools/T-conntest.
type ProxyPoolService struct {
	st *store.Store
}

// NewProxyPoolService constructs the service over a store.
func NewProxyPoolService(st *store.Store) *ProxyPoolService {
	return &ProxyPoolService{st: st}
}

// Create inserts a proxy pool. Stub.
func (s *ProxyPoolService) Create(p *store.ProxyPool) (*store.ProxyPool, error) {
	return s.st.CreateProxyPool(p)
}

// List returns proxy pools, optionally filtered by active state. Stub.
func (s *ProxyPoolService) List(filterActive *bool) ([]*store.ProxyPool, error) {
	return s.st.ListProxyPools(filterActive)
}

// Get returns the pool with the given id. Stub.
func (s *ProxyPoolService) Get(id string) (*store.ProxyPool, error) {
	return s.st.GetProxyPoolByID(id)
}

// Update persists mutable fields. Stub.
func (s *ProxyPoolService) Update(p *store.ProxyPool) error {
	return s.st.UpdateProxyPool(p)
}

// Delete removes the pool with the given id. Stub.
func (s *ProxyPoolService) Delete(id string) error {
	return s.st.DeleteProxyPool(id)
}

// CountBoundConnections returns the number of connections bound to the pool. Stub.
func (s *ProxyPoolService) CountBoundConnections(id string) (int, error) {
	return s.st.CountConnectionsUsingProxyPool(id)
}
