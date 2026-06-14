package store

import "errors"

// MitmTool is a configured MITM tool row (the 5-field UI MitmTool shape). The
// table holds NO secret: dns_override is config, not a credential. The only
// secret in the MITM domain is the root CA private key, handled out-of-band in
// internal/platform/mitm (a 0o600 file under the data dir), never in this table.
type MitmTool struct {
	ID          string
	Name        string
	Enabled     bool
	DNSOverride string
	Status      string // active|inactive
	UpdatedAt   int64
}

const mitmEnabledSettingKey = "mitmEnabled"

var errMitmNotImplemented = errors.New("store: mitm not implemented")

// ListMitmTools returns all MITM tool rows in deterministic order (by id).
func (s *Store) ListMitmTools() ([]MitmTool, error) { return nil, errMitmNotImplemented }

// GetMitmTool returns the MITM tool row for id, or ErrNotFound.
func (s *Store) GetMitmTool(id string) (MitmTool, error) {
	return MitmTool{}, errMitmNotImplemented
}

// UpsertMitmTool inserts or updates a MITM tool keyed by id.
func (s *Store) UpsertMitmTool(t MitmTool) error { return errMitmNotImplemented }

// SetMitmToolEnabled flips a tool's enabled flag, derives its status, persists,
// and returns the updated row. ErrNotFound on an unknown id.
func (s *Store) SetMitmToolEnabled(id string, enabled bool) (MitmTool, error) {
	return MitmTool{}, errMitmNotImplemented
}

// GetMitmEnabled reports the global MITM enable flag.
func (s *Store) GetMitmEnabled() (bool, error) { return false, errMitmNotImplemented }

// SetMitmEnabled persists the global MITM enable flag.
func (s *Store) SetMitmEnabled(enabled bool) error { return errMitmNotImplemented }

// EnsureMitmTools seeds the two named MITM tools on first migrate.
func (s *Store) EnsureMitmTools() error { return errMitmNotImplemented }
