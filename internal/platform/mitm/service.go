package mitm

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

const (
	// defaultMitmAddr is the listen address the MITM reverse proxy binds when
	// enabled. The actual bind is integration-only (§1.9).
	defaultMitmAddr = "127.0.0.1:8443"

	backoffBase   = 1 * time.Second
	backoffCap    = 30 * time.Second
	backoffMaxTry = 5
)

// Service is the MITM domain service (transport→domain→repository). It owns the
// global enable/disable + per-tool toggle state machine, lazily loads the root CA
// (for the raw-PEM ca-cert endpoint), and drives the injectable proxy listener.
// The CA is real in tests (pure crypto, cheap+deterministic); only the listener
// is injected via SetProxy so the admin/service tests never bind a port.
type Service struct {
	st *store.Store

	mu    sync.Mutex
	ca    *CA
	proxy MitmProxy
}

// NewService constructs the service over a store WITHOUT binding any listener or
// generating the CA (both are lazy / on-enable), mirroring tunnel.NewService.
func NewService(st *store.Store) *Service {
	return &Service{st: st}
}

// SetProxy overrides the proxy listener (tests inject a deterministic fake that
// records Start/Stop without binding a port). Mirrors tunnel.Service.SetRunner.
func (s *Service) SetProxy(p MitmProxy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.proxy = p
}

// ensureCA lazily loads-or-creates the root CA from the data dir. The key is
// persisted 0o600; only the public cert is ever served.
func (s *Service) ensureCA() (*CA, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ca != nil {
		return s.ca, nil
	}
	ca, err := LoadOrCreateCA(s.st.DataDir())
	if err != nil {
		return nil, err
	}
	s.ca = ca
	return ca, nil
}

// ensureProxy returns the configured proxy, lazily building the real
// listenerProxy over the CA when none was injected.
func (s *Service) ensureProxy() (MitmProxy, error) {
	s.mu.Lock()
	if s.proxy != nil {
		p := s.proxy
		s.mu.Unlock()
		return p, nil
	}
	s.mu.Unlock()

	ca, err := s.ensureCA()
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.proxy == nil {
		s.proxy = newListenerProxy(ca)
	}
	return s.proxy, nil
}

// Status overlays the stored global flag + tool rows. It lazily seeds the two
// named tools so the list always surfaces >=2 entries (ESC-SEED-ROWS).
func (s *Service) Status() (bool, []store.MitmTool, error) {
	if err := s.st.EnsureMitmTools(); err != nil {
		return false, nil, err
	}
	enabled, err := s.st.GetMitmEnabled()
	if err != nil {
		return false, nil, err
	}
	tools, err := s.st.ListMitmTools()
	if err != nil {
		return false, nil, err
	}
	return enabled, tools, nil
}

// Toggle flips the global flag, persists it, and best-effort starts/stops the
// MITM proxy listener. The flag flip + persist is the unit-tested core; the real
// listener bind is integration-only (§1.9) and wrapped in restart backoff.
func (s *Service) Toggle() (bool, error) {
	cur, err := s.st.GetMitmEnabled()
	if err != nil {
		return false, err
	}
	next := !cur
	if err := s.st.SetMitmEnabled(next); err != nil {
		return false, fmt.Errorf("persist mitm enabled: %w", err)
	}

	proxy, err := s.ensureProxy()
	if err != nil {
		return next, nil // flag persisted; listener wiring is best-effort
	}
	if next {
		s.startWithBackoff(proxy, defaultMitmAddr)
	} else {
		if err := proxy.Stop(); err != nil {
			log.Printf("mitm: stop proxy: %v", err)
		}
	}
	return next, nil
}

// startWithBackoff attempts to start the proxy listener, retrying transient bind
// failures with a bounded exponential backoff. The retry sleeps are
// integration-only; the nextBackoff POLICY is the pure unit-tested factor.
func (s *Service) startWithBackoff(proxy MitmProxy, addr string) {
	for attempt := 0; attempt < backoffMaxTry; attempt++ {
		if err := proxy.Start(addr); err == nil {
			return
		} else {
			log.Printf("mitm: start proxy attempt %d: %v", attempt+1, err)
		}
		if attempt+1 < backoffMaxTry {
			time.Sleep(nextBackoff(attempt))
		}
	}
}

// ToggleTool flips a tool's enabled flag (deriving status) and persists it. It
// lazily seeds the named tools so a fresh store can toggle them; an unknown id
// still returns store.ErrNotFound (→ 404).
func (s *Service) ToggleTool(id string) (store.MitmTool, error) {
	if err := s.st.EnsureMitmTools(); err != nil {
		return store.MitmTool{}, err
	}
	cur, err := s.st.GetMitmTool(id)
	if err != nil {
		return store.MitmTool{}, err
	}
	return s.st.SetMitmToolEnabled(id, !cur.Enabled)
}

// CACertPEM returns the PUBLIC root CA cert PEM (the application/x-pem-file body),
// lazily generating + persisting the CA on first call. The private key is never
// returned.
func (s *Service) CACertPEM() ([]byte, error) {
	ca, err := s.ensureCA()
	if err != nil {
		return nil, err
	}
	return ca.CertPEM(), nil
}

// nextBackoff is the PURE restart-backoff policy: 1s doubling, capped at 30s.
func nextBackoff(attempt int) time.Duration {
	d := backoffBase << attempt
	if d <= 0 || d > backoffCap {
		return backoffCap
	}
	return d
}
