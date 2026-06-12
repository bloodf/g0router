package inference

import (
	"fmt"
	"time"
)

// Verdict is the outcome category produced by the error classifier for a
// failed request. w4-c owns this type; the w4-b classifier maps to it.
type Verdict int

const (
	VerdictUnknown   Verdict = iota
	VerdictRateLimit         // 429 / rate-limit text — temporary, backoff applies
	VerdictAuth              // 401/403 / auth text — connection broken
	VerdictTransient         // 5xx unclassified — temporary, fixed cooldown
	VerdictPermanent         // 400/404 / permanent text — request invalid
)

// LockStore abstracts the connection lock and backoff state operations.
type LockStore interface {
	LockModel(connID, model string, expiresAt int64) error
	LockAccount(connID string, expiresAt int64) error
	ClearLocks(connID string) error
	EarliestExpiry(model string, now int64) (int64, bool, error)
	SetBackoffLevel(connID string, level int) error
	GetBackoffLevel(connID string) (int, error)
}

const (
	// Ported from open-sse/config/errorConfig.js BACKOFF_CONFIG.
	backoffBase     int64 = 2000           // 2s in milliseconds
	backoffMax      int64 = 5 * 60 * 1000 // 5 minutes in milliseconds
	backoffMaxLevel int   = 15

	// Ported from TRANSIENT_COOLDOWN_MS = 30 * 1000.
	transientCooldownSec int64 = 30
)

// quotaCooldown returns the cooldown in milliseconds for the given backoff level.
// Mirrors getQuotaCooldown in open-sse/services/accountFallback.js.
func quotaCooldown(backoffLevel int) int64 {
	if backoffLevel > backoffMaxLevel {
		return backoffMax
	}
	level := backoffLevel - 1
	if level < 0 {
		level = 0
	}
	cooldown := backoffBase * (1 << level)
	if cooldown > backoffMax {
		return backoffMax
	}
	return cooldown
}

// CooldownEngine manages per-connection lock and backoff state.
// It mirrors the account-fallback logic in open-sse/services/accountFallback.js.
type CooldownEngine struct {
	st    LockStore
	clock func() time.Time
}

// NewCooldownEngine creates a CooldownEngine with an injected clock.
func NewCooldownEngine(st LockStore, clock func() time.Time) *CooldownEngine {
	return &CooldownEngine{st: st, clock: clock}
}

// MarkUnavailable records that the connection failed with the given verdict.
// RateLimit: increments backoff level and creates a timed model lock.
// Transient: creates a fixed 30s model lock.
// Auth/Permanent: no lock created (caller handles connection exclusion).
func (e *CooldownEngine) MarkUnavailable(connID, model string, verdict Verdict) error {
	switch verdict {
	case VerdictRateLimit:
		level, err := e.st.GetBackoffLevel(connID)
		if err != nil {
			return fmt.Errorf("get backoff level: %w", err)
		}
		newLevel := level + 1
		if newLevel > backoffMaxLevel {
			newLevel = backoffMaxLevel
		}
		if err := e.st.SetBackoffLevel(connID, newLevel); err != nil {
			return fmt.Errorf("set backoff level: %w", err)
		}
		cooldownSec := quotaCooldown(newLevel) / 1000
		expiresAt := e.clock().Unix() + cooldownSec
		return e.st.LockModel(connID, model, expiresAt)

	case VerdictTransient:
		expiresAt := e.clock().Unix() + transientCooldownSec
		return e.st.LockModel(connID, model, expiresAt)

	case VerdictAuth, VerdictPermanent:
		// Connection is broken, not temporarily overloaded; no timed lock.
		return nil
	}
	return nil
}

// MarkSuccess clears all locks and resets the backoff level for the connection.
// Mirrors the success-path in open-sse/services/auth.js (PAR-ROUTE-015).
func (e *CooldownEngine) MarkSuccess(connID string) error {
	if err := e.st.SetBackoffLevel(connID, 0); err != nil {
		return fmt.Errorf("reset backoff level: %w", err)
	}
	return e.st.ClearLocks(connID)
}

// GroupRetryAfter returns the earliest lock expiry across all connections for
// the given model. Returns (zero, false) when no active locks exist.
// Mirrors the retryAfter aggregation in open-sse/services/combo.js (PAR-ROUTE-049).
func (e *CooldownEngine) GroupRetryAfter(model string, now time.Time) (time.Time, bool) {
	earliest, ok, err := e.st.EarliestExpiry(model, now.Unix())
	if err != nil || !ok {
		return time.Time{}, false
	}
	return time.Unix(earliest, 0), true
}
