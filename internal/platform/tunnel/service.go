package tunnel

import (
	"errors"
	"fmt"

	"github.com/bloodf/g0router/internal/store"
)

// ErrUnknownType is returned for any tunnel type other than the two known kinds.
var ErrUnknownType = errors.New("tunnel: unknown type")

// knownTypes is the fixed, ordered set of tunnel types. List() overlays stored
// state onto these so it always returns exactly two entries (matching the UI's
// 2-card contract) WITHOUT a seed migration.
var knownTypes = []string{TypeCloudflare, TypeTailscale}

// Service is the tunnel domain service (transport→domain→repository). It owns the
// enable/disable/status/health state machine over an injectable Runner per type
// and persists state via the store. The Runner map holds real defaults
// (cloudflared/tailscale shell impls) constructed at NewService; SetRunner
// overrides a runner for tests — mirroring platform.ProxyPoolService's
// Prober/SetProber injection (proxypools.go:30,36).
type Service struct {
	st      *store.Store
	runners map[string]Runner
}

// NewService constructs the service over a store with the real default runners.
// The real runners are the ONLY place a process/binary is referenced; no unit
// test exercises them (tests inject fakes via SetRunner).
func NewService(st *store.Store) *Service {
	return &Service{
		st: st,
		runners: map[string]Runner{
			TypeCloudflare: newCloudflaredRunner(),
			TypeTailscale:  newTailscaleRunner(),
		},
	}
}

// SetRunner overrides the runner for a tunnel type (tests inject a deterministic
// fake). Mirrors ProxyPoolService.SetProber (proxypools.go:36).
func (s *Service) SetRunner(typ string, r Runner) {
	s.runners[typ] = r
}

// Enable starts the tunnel of the given type and persists the resulting state.
// On a runner failure it records status="error" with the error detail and leaves
// the tunnel enabled-but-failing (no hard error is returned to the caller; the
// failure is surfaced via the persisted/returned status).
func (s *Service) Enable(typ, token, mode string) (store.Tunnel, error) {
	runner, ok := s.runners[typ]
	if !ok {
		return store.Tunnel{}, ErrUnknownType
	}

	// Persist the requested config (token at rest) + starting state.
	if err := s.st.UpsertTunnel(store.Tunnel{
		Type:      typ,
		IsEnabled: true,
		Status:    StatusStarting,
		Token:     token,
		Mode:      mode,
	}); err != nil {
		return store.Tunnel{}, fmt.Errorf("persist starting state: %w", err)
	}

	url, startErr := runner.Start(StartOpts{Type: typ, Token: token, Mode: mode})
	if startErr != nil {
		if err := s.st.SetTunnelState(typ, StatusError, "", startErr.Error(), true); err != nil {
			return store.Tunnel{}, fmt.Errorf("persist error state: %w", err)
		}
		return s.st.GetTunnel(typ)
	}

	if err := s.st.SetTunnelState(typ, StatusActive, url, "", true); err != nil {
		return store.Tunnel{}, fmt.Errorf("persist active state: %w", err)
	}
	return s.st.GetTunnel(typ)
}

// Disable stops the tunnel and persists the inactive state. Idempotent: stopping
// a never-enabled tunnel returns an inactive entry without error.
func (s *Service) Disable(typ string) (store.Tunnel, error) {
	runner, ok := s.runners[typ]
	if !ok {
		return store.Tunnel{}, ErrUnknownType
	}
	if err := runner.Stop(); err != nil {
		return store.Tunnel{}, fmt.Errorf("stop tunnel %s: %w", typ, err)
	}

	// Ensure a row exists, then write the inactive state.
	if _, err := s.st.GetTunnel(typ); errors.Is(err, store.ErrNotFound) {
		if uerr := s.st.UpsertTunnel(store.Tunnel{Type: typ, Status: StatusInactive}); uerr != nil {
			return store.Tunnel{}, fmt.Errorf("persist inactive state: %w", uerr)
		}
		return s.st.GetTunnel(typ)
	} else if err != nil {
		return store.Tunnel{}, err
	}

	if err := s.st.SetTunnelState(typ, StatusInactive, "", "", false); err != nil {
		return store.Tunnel{}, fmt.Errorf("persist inactive state: %w", err)
	}
	return s.st.GetTunnel(typ)
}

// Status returns the stored state of a single tunnel type, or a synthesized
// inactive entry when no row exists yet.
func (s *Service) Status(typ string) (store.Tunnel, error) {
	if _, ok := s.runners[typ]; !ok {
		return store.Tunnel{}, ErrUnknownType
	}
	tn, err := s.st.GetTunnel(typ)
	if errors.Is(err, store.ErrNotFound) {
		return store.Tunnel{Type: typ, Status: StatusInactive}, nil
	}
	if err != nil {
		return store.Tunnel{}, err
	}
	return tn, nil
}

// List returns exactly the two known tunnel types, overlaying any stored state.
// A type with no stored row is reported as inactive. This guarantees the UI's
// 2-card contract without a seed migration.
func (s *Service) List() ([]store.Tunnel, error) {
	stored, err := s.st.ListTunnels()
	if err != nil {
		return nil, err
	}
	byType := make(map[string]store.Tunnel, len(stored))
	for _, tn := range stored {
		byType[tn.Type] = tn
	}
	out := make([]store.Tunnel, 0, len(knownTypes))
	for _, typ := range knownTypes {
		if tn, ok := byType[typ]; ok {
			out = append(out, tn)
		} else {
			out = append(out, store.Tunnel{Type: typ, Status: StatusInactive})
		}
	}
	return out, nil
}

// Health reports whether the gateway's tunnels are healthy: true iff every
// ENABLED tunnel reports a non-error status. An all-disabled gateway is healthy.
func (s *Service) Health() (bool, error) {
	list, err := s.List()
	if err != nil {
		return false, err
	}
	for _, tn := range list {
		if tn.IsEnabled && tn.Status == StatusError {
			return false, nil
		}
	}
	return true, nil
}
