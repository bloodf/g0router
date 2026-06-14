package store

// ProxyPool is a configured outbound proxy. Password is plaintext in memory and
// encrypted at rest in the password_enc column.
type ProxyPool struct {
	ID              string
	Name            string
	Protocol        string
	Host            string
	Port            int
	Username        string
	Password        string
	IsActive        bool
	LastCheckStatus string
	LastCheckAt     string // ISO-8601 (RFC3339)
	CreatedAt       int64
	UpdatedAt       int64
}

// CreateProxyPool inserts a proxy pool. Stub: real impl lands in T-proxypools STEP(b).
func (s *Store) CreateProxyPool(p *ProxyPool) (*ProxyPool, error) { return nil, nil }

// ListProxyPools returns proxy pools, optionally filtered by active state.
// Stub: real impl lands in T-proxypools STEP(b).
func (s *Store) ListProxyPools(filterActive *bool) ([]*ProxyPool, error) { return nil, nil }

// GetProxyPoolByID returns the pool with the given id. Stub.
func (s *Store) GetProxyPoolByID(id string) (*ProxyPool, error) { return nil, ErrNotFound }

// UpdateProxyPool persists mutable fields. Stub.
func (s *Store) UpdateProxyPool(p *ProxyPool) error { return nil }

// DeleteProxyPool removes the pool with the given id. Stub.
func (s *Store) DeleteProxyPool(id string) error { return nil }

// SetProxyPoolCheck records the result of a connectivity test. Stub.
func (s *Store) SetProxyPoolCheck(id, status, atRFC3339 string) error { return nil }

// CountConnectionsUsingProxyPool counts connections bound to the pool. Stub.
func (s *Store) CountConnectionsUsingProxyPool(id string) (int, error) { return 0, nil }
