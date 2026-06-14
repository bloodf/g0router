package platform

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

// probeTarget is the public endpoint a connectivity test reaches THROUGH the
// proxy to confirm reachability. It is SSRF-safe (a public host).
const probeTarget = "https://www.google.com"

// Prober probes reachability of target through proxyURL, returning a latency in
// milliseconds. It is an injectable seam so tests run without network access.
type Prober func(proxyURL, target string) (latencyMs int, err error)

// ProxyPoolService is the domain service for proxy pools (transport→domain→
// repository). It owns CRUD wrappers over the store, the connectivity test, and
// per-connection proxy resolution.
type ProxyPoolService struct {
	st       *store.Store
	prober   Prober
	resolver IPResolver
}

// NewProxyPoolService constructs the service over a store.
func NewProxyPoolService(st *store.Store) *ProxyPoolService {
	return &ProxyPoolService{st: st}
}

// SetProber injects the reachability prober (production wires the real proxied
// dial; tests inject a deterministic fake).
func (s *ProxyPoolService) SetProber(p Prober) {
	s.prober = p
}

// SetResolver injects the DNS resolver used by the SSRF guard. When unset, the
// system resolver is used; tests inject a deterministic fake.
func (s *ProxyPoolService) SetResolver(r IPResolver) {
	s.resolver = r
}

// ResolveProxyForConnection returns the outbound proxy URL for a connection
// bound to an active proxy pool. Stub: real wiring lands in T-proxywire STEP(b).
func (s *ProxyPoolService) ResolveProxyForConnection(conn *store.Connection) (string, bool) {
	return "", false
}

// ProxyTestResult is the outcome of a connectivity probe through a proxy pool.
type ProxyTestResult struct {
	OK        bool
	LatencyMs int
	Status    string // "ok" | "error" | "blocked"
}

// TestConnectivity probes the pool's proxy reachability and persists the result.
// The proxy host is SSRF-guarded BEFORE dialing: a proxy that points at a
// private/loopback/link-local address is refused (status "blocked") without
// invoking the prober. Reachable → status "ok"; prober error → status "error".
func (s *ProxyPoolService) TestConnectivity(id string) (ProxyTestResult, error) {
	pool, err := s.st.GetProxyPoolByID(id)
	if err != nil {
		return ProxyTestResult{}, err
	}

	// SSRF guard on the user-configured proxy host (PAR-AUTH-020 vector). A
	// resolution failure means the proxy host is unreachable, not a server
	// error: report it as a failed check rather than propagating a 500.
	blocked, _, berr := IsBlockedTarget(pool.Host, s.resolver)
	if berr != nil {
		res := ProxyTestResult{OK: false, Status: "error"}
		if perr := s.persistCheck(id, res.Status); perr != nil {
			return res, perr
		}
		return res, nil
	}
	if blocked {
		res := ProxyTestResult{OK: false, Status: "blocked"}
		if perr := s.persistCheck(id, res.Status); perr != nil {
			return res, perr
		}
		return res, nil
	}

	prober := s.prober
	if prober == nil {
		prober = defaultProber
	}
	proxyURL := proxyURLForPool(pool)
	latency, probeErr := prober(proxyURL, probeTarget)
	res := ProxyTestResult{}
	if probeErr != nil {
		res.OK = false
		res.Status = "error"
	} else {
		res.OK = true
		res.LatencyMs = latency
		res.Status = "ok"
	}
	if perr := s.persistCheck(id, res.Status); perr != nil {
		return res, perr
	}
	return res, nil
}

func (s *ProxyPoolService) persistCheck(id, status string) error {
	return s.st.SetProxyPoolCheck(id, status, time.Now().UTC().Format(time.RFC3339))
}

// proxyURLForPool builds the proxy URL (protocol://[user:pass@]host:port) for a
// pool. The credentials are taken from the pool's plaintext (decrypted) fields.
func proxyURLForPool(p *store.ProxyPool) string {
	protocol := p.Protocol
	if protocol == "" {
		protocol = "http"
	}
	u := &url.URL{Scheme: protocol, Host: p.Host}
	if p.Port > 0 {
		u.Host = fmt.Sprintf("%s:%d", p.Host, p.Port)
	}
	if p.Username != "" {
		if p.Password != "" {
			u.User = url.UserPassword(p.Username, p.Password)
		} else {
			u.User = url.User(p.Username)
		}
	}
	return u.String()
}

// defaultProber performs a real HEAD request to target THROUGH the configured
// proxy and returns the elapsed time in milliseconds. It is the production prober;
// unit tests inject a deterministic fake via SetProber.
func defaultProber(proxyURL, target string) (int, error) {
	pu, err := url.Parse(proxyURL)
	if err != nil {
		return 0, fmt.Errorf("parse proxy url: %w", err)
	}
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: &http.Transport{Proxy: http.ProxyURL(pu)},
	}
	start := time.Now()
	req, err := http.NewRequest(http.MethodHead, target, nil)
	if err != nil {
		return 0, fmt.Errorf("build probe request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("probe via proxy: %w", err)
	}
	resp.Body.Close()
	return int(time.Since(start).Milliseconds()), nil
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
