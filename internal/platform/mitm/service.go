package mitm

import (
	"errors"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

var errNotImplemented = errors.New("mitm: not implemented")

// Service is the MITM domain service (transport→domain→repository). It owns the
// global enable/disable + per-tool toggle state machine, lazily loads the root CA
// (for the raw-PEM ca-cert endpoint), and drives the injectable proxy listener.
type Service struct {
	st *store.Store
}

// NewService constructs the service over a store WITHOUT binding any listener or
// generating the CA (both are lazy / on-enable), mirroring tunnel.NewService.
func NewService(st *store.Store) *Service { return &Service{st: st} }

// SetProxy overrides the proxy listener (tests inject a deterministic fake).
func (s *Service) SetProxy(p MitmProxy) {}

// Status overlays the stored global flag + tool rows.
func (s *Service) Status() (bool, []store.MitmTool, error) { return false, nil, errNotImplemented }

// Toggle flips the global flag and best-effort starts/stops the listener.
func (s *Service) Toggle() (bool, error) { return false, errNotImplemented }

// ToggleTool flips a tool's enabled flag and persists.
func (s *Service) ToggleTool(id string) (store.MitmTool, error) {
	return store.MitmTool{}, errNotImplemented
}

// CACertPEM returns the PUBLIC root CA cert PEM, lazily generating the CA.
func (s *Service) CACertPEM() ([]byte, error) { return nil, errNotImplemented }

// nextBackoff is the PURE restart-backoff policy: doubling from 1s, capped at 30s.
func nextBackoff(attempt int) time.Duration { return 0 }
