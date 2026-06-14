package mcp

import (
	"context"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

// accountHealth derives an MCP OAuth account's health status from its expiry at
// now: "expired" once past expiresAt, otherwise "connected". A zero expiresAt
// (unknown) is treated as connected. PURE (PAR-MCP-038).
func accountHealth(account *store.MCPOAuthAccount, now time.Time) string {
	if account.ExpiresAt == 0 {
		return "connected"
	}
	if time.Unix(account.ExpiresAt, 0).Before(now) {
		return "expired"
	}
	return "connected"
}

// accountsNeedingRefresh returns the accounts whose tokens are within lead of
// expiry (or already expired) at now, skipping unknown-expiry accounts. PURE.
func accountsNeedingRefresh(accounts []*store.MCPOAuthAccount, now time.Time, lead time.Duration) []*store.MCPOAuthAccount {
	var out []*store.MCPOAuthAccount
	for _, a := range accounts {
		if needsRefresh(a.ExpiresAt, now, lead) {
			out = append(out, a)
		}
	}
	return out
}

// HealthMonitor periodically refreshes near-expiry MCP OAuth accounts via the
// engine. The PURE derivations above are unit-tested; the ticker loop here is
// INTEGRATION-ONLY (never exercised by a unit test — no real ticker in the suite).
type HealthMonitor struct {
	store    *store.Store
	interval time.Duration
	lead     time.Duration
}

// NewHealthMonitor builds a monitor over the store.
func NewHealthMonitor(st *store.Store, interval, lead time.Duration) *HealthMonitor {
	return &HealthMonitor{store: st, interval: interval, lead: lead}
}

// Run ticks until ctx is canceled, marking expired accounts. INTEGRATION-ONLY.
func (m *HealthMonitor) Run(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.sweep(time.Now())
		}
	}
}

// sweep marks every expired account's status, using the PURE derivation.
func (m *HealthMonitor) sweep(now time.Time) {
	accounts, err := m.store.ListMCPOAuthAccounts()
	if err != nil {
		return
	}
	for _, a := range accounts {
		status := accountHealth(a, now)
		if status != a.Status {
			a.Status = status
			_, _ = m.store.UpsertMCPOAuthAccount(a)
		}
	}
}
