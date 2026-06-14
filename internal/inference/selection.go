package inference

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

// selectionMu is the single global selection mutex (PAR-ROUTE-017).
// Mirrors the global promise-chain mutex in open-sse/services/auth.js:9,24-30.
var selectionMu sync.Mutex

// ConnStore provides connection listing and active-lock querying for selection.
type ConnStore interface {
	ListConnections() ([]*store.Connection, error)
	ActiveLocks(connID string, now int64) ([]*store.ModelLock, error)
}

// SettingStore provides access to strategy configuration settings.
type SettingStore interface {
	GetSetting(key string) (string, error)
}

// Cooldown manages per-connection availability state for the selection engine.
type Cooldown interface {
	MarkUnavailable(connID, providerID, model string, verdict Verdict) error
	MarkSuccess(connID string) error
	GroupRetryAfter(providerID, model string, now time.Time) (time.Time, bool, error)
}

// ProxyResolver resolves the outbound proxy URL for a selected connection
// (PAR-PLAT-009). It is an optional, additive dependency of the SelectionEngine:
// when unset, no proxy is resolved and behavior is unchanged. Implemented by
// platform.ProxyPoolService.
type ProxyResolver interface {
	ResolveProxyForConnection(conn *store.Connection) (proxyURL string, ok bool)
}

type rrState struct {
	currentConnID string
	count         int // consecutive use count for currentConnID
}

// SelectionEngine selects eligible connections and orchestrates account-level fallback.
// Mirrors open-sse/services/auth.js getProviderCredentials + the fallback loop in chat.js.
type SelectionEngine struct {
	cs       ConnStore
	ss       SettingStore
	cd       Cooldown
	clock    func() time.Time
	rrStates map[string]*rrState // "providerID:model" → round-robin state
	pr       ProxyResolver       // optional, additive (PAR-PLAT-009); nil = no proxy
}

// NewSelectionEngine creates a SelectionEngine with injected dependencies.
func NewSelectionEngine(cs ConnStore, ss SettingStore, cd Cooldown, clock func() time.Time) *SelectionEngine {
	return &SelectionEngine{cs: cs, ss: ss, cd: cd, clock: clock, rrStates: make(map[string]*rrState)}
}

// SetProxyResolver wires the optional per-connection proxy resolver (PAR-PLAT-009).
// Stub: real wiring lands in T-proxywire STEP(b).
func (e *SelectionEngine) SetProxyResolver(pr ProxyResolver) {}

// ResolveProxy returns the outbound proxy URL for the selected connection.
// Stub: real wiring lands in T-proxywire STEP(b).
func (e *SelectionEngine) ResolveProxy(conn *store.Connection) (proxyURL string, ok bool) {
	return "", false
}

type providerStrategyConfig struct {
	FallbackStrategy      string `json:"fallbackStrategy"`
	StickyRoundRobinLimit int    `json:"stickyRoundRobinLimit"`
}

// resolveStrategy returns the effective fallback strategy and sticky limit
// for the given provider. Mirrors auth.js:102-116.
func (e *SelectionEngine) resolveStrategy(providerID string) (strategy string, stickyLimit int, err error) {
	// Per-provider override from providerStrategies JSON setting.
	raw, err := e.ss.GetSetting("providerStrategies")
	if err != nil {
		return "", 0, fmt.Errorf("get providerStrategies setting: %w", err)
	}
	if raw != "" {
		var pmap map[string]providerStrategyConfig
		if json.Unmarshal([]byte(raw), &pmap) == nil {
			if ps, ok := pmap[providerID]; ok {
				if ps.FallbackStrategy != "" {
					strategy = ps.FallbackStrategy
				}
				if ps.StickyRoundRobinLimit > 0 {
					stickyLimit = ps.StickyRoundRobinLimit
				}
			}
		}
	}
	// Global fallbackStrategy setting.
	if strategy == "" {
		s, err := e.ss.GetSetting("fallbackStrategy")
		if err != nil {
			return "", 0, fmt.Errorf("get fallbackStrategy setting: %w", err)
		}
		if s != "" {
			strategy = s
		}
	}
	if strategy == "" {
		strategy = "fill-first" // ref default per auth.js:103
	}
	// Resolve stickyRoundRobinLimit from global setting if not already set.
	if stickyLimit == 0 && strategy == "round-robin" {
		s, err := e.ss.GetSetting("stickyRoundRobinLimit")
		if err != nil {
			return "", 0, fmt.Errorf("get stickyRoundRobinLimit setting: %w", err)
		}
		if s != "" {
			if n, convErr := strconv.Atoi(s); convErr == nil && n > 0 {
				stickyLimit = n
			}
		}
		if stickyLimit == 0 {
			stickyLimit = 3 // ref default per auth.js:116
		}
	}
	return
}

// isConnLocked reports whether the connection has an active lock for the given model
// or the account-level "__all" sentinel.
func isConnLocked(cs ConnStore, connID, model string, now int64) (bool, error) {
	locks, err := cs.ActiveLocks(connID, now)
	if err != nil {
		return false, fmt.Errorf("active locks %s: %w", connID, err)
	}
	for _, l := range locks {
		if l.Model == model || l.Model == "__all" {
			return true, nil
		}
	}
	return false, nil
}

// SelectConnection picks an eligible connection for the given provider and model.
// All callers are serialized through selectionMu (PAR-ROUTE-017).
// exclude lists connection IDs to skip (grown by the fallback loop).
// preferredConnID, when set and eligible, is returned immediately (PAR-ROUTE-051).
func (e *SelectionEngine) SelectConnection(providerID, model string, exclude []string, preferredConnID string) (*store.Connection, error) {
	selectionMu.Lock()
	defer selectionMu.Unlock()

	all, err := e.cs.ListConnections()
	if err != nil {
		return nil, fmt.Errorf("list connections: %w", err)
	}

	now := e.clock().Unix()
	excludeSet := make(map[string]bool, len(exclude))
	for _, id := range exclude {
		excludeSet[id] = true
	}

	// Build eligible slice: matching provider, not excluded, not locked for model.
	var eligible []*store.Connection
	for _, c := range all {
		if c.ProviderID != providerID || excludeSet[c.ID] {
			continue
		}
		locked, err := isConnLocked(e.cs, c.ID, model, now)
		if err != nil {
			return nil, err
		}
		if !locked {
			eligible = append(eligible, c)
		}
	}

	if len(eligible) == 0 {
		return nil, errors.New("no eligible connections")
	}

	// Pinned preference (PAR-ROUTE-051): if preferred is eligible, return it immediately.
	if preferredConnID != "" {
		for _, c := range eligible {
			if c.ID == preferredConnID {
				return c, nil
			}
		}
		// Preferred not eligible — fall through to strategy.
	}

	strategy, stickyLimit, err := e.resolveStrategy(providerID)
	if err != nil {
		return nil, err
	}
	key := providerID + ":" + model

	switch strategy {
	case "round-robin":
		state, ok := e.rrStates[key]
		if !ok {
			state = &rrState{}
			e.rrStates[key] = state
		}

		// Find current connection in eligible.
		currentIdx := -1
		for i, c := range eligible {
			if c.ID == state.currentConnID {
				currentIdx = i
				break
			}
		}

		if currentIdx >= 0 && state.count < stickyLimit {
			// Stick with current connection (PAR-ROUTE-019).
			state.count++
			return eligible[currentIdx], nil
		}

		// Rotate to next eligible (wrapping), reset count.
		nextIdx := 0
		if currentIdx >= 0 {
			nextIdx = (currentIdx + 1) % len(eligible)
		}
		state.currentConnID = eligible[nextIdx].ID
		state.count = 1
		return eligible[nextIdx], nil

	default: // "fill-first": return first eligible (already stable DB order).
		return eligible[0], nil
	}
}

// ErrAllUnavailable is returned by WithAccountFallback when all connections are exhausted.
var ErrAllUnavailable = errors.New("all accounts unavailable")

// WithAccountFallback executes fn against successive connections, falling back on failure.
// On rate-limit, transient, or auth failure, marks the connection unavailable and retries
// with the next eligible connection. Terminates when all are excluded (PR-640).
// Mirrors the fallback loop in open-sse/handlers/chat.js:162-245.
func (e *SelectionEngine) WithAccountFallback(providerID, model string, fn func(*store.Connection) (Verdict, error)) error {
	var exclude []string
	for {
		conn, err := e.SelectConnection(providerID, model, exclude, "")
		if err != nil {
			// All connections exhausted — attach retry-after info if available.
			now := e.clock()
			retryAt, ok, grErr := e.cd.GroupRetryAfter(providerID, model, now)
			if grErr != nil {
				return fmt.Errorf("%w: %w", ErrAllUnavailable, grErr)
			}
			if ok {
				return fmt.Errorf("%w: retry after %v", ErrAllUnavailable, retryAt)
			}
			return ErrAllUnavailable
		}

		verdict, fnErr := fn(conn)

		// Success path: no verdict + no error.
		if verdict == VerdictUnknown && fnErr == nil {
			if markErr := e.cd.MarkSuccess(conn.ID); markErr != nil {
				return fmt.Errorf("mark success: %w", markErr)
			}
			return nil
		}

		// Permanent failure: request is invalid, fallback won't help.
		if verdict == VerdictPermanent {
			if fnErr != nil {
				return fnErr
			}
			return fmt.Errorf("permanent failure for %s/%s", providerID, model)
		}

		// Temporary failure (rate-limit / transient / auth): mark unavailable, exclude, retry.
		if markErr := e.cd.MarkUnavailable(conn.ID, providerID, model, verdict); markErr != nil {
			return fmt.Errorf("mark unavailable: %w", markErr)
		}
		exclude = append(exclude, conn.ID)
	}
}
