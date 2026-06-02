package provider

import (
	"fmt"
	"sync"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

const (
	initialBackoff = time.Second
	maxBackoff     = 4 * time.Minute
)

type FallbackStore interface {
	GetActiveConnections(provider string) ([]*store.Connection, error)
	UpdateConnection(conn *store.Connection) error
}

type fallbackClock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

type FallbackManager struct {
	store   FallbackStore
	clock   fallbackClock
	mu      sync.Mutex
	cursors map[string]int
}

func NewFallbackManager(store FallbackStore) *FallbackManager {
	return NewFallbackManagerWithClock(store, realClock{})
}

func NewFallbackManagerWithClock(store FallbackStore, clock fallbackClock) *FallbackManager {
	return &FallbackManager{
		store:   store,
		clock:   clock,
		cursors: make(map[string]int),
	}
}

func (m *FallbackManager) Next(provider string, model string) (*store.Connection, error) {
	connections, err := m.store.GetActiveConnections(provider)
	if err != nil {
		return nil, fmt.Errorf("get active connections: %w", err)
	}
	if len(connections) == 0 {
		return nil, ErrNoActiveConnections
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.clock.Now().Unix()
	start := m.cursors[provider] % len(connections)
	for offset := range connections {
		index := (start + offset) % len(connections)
		conn := connections[index]
		if connectionAvailable(conn, model, now) {
			m.cursors[provider] = index + 1
			return conn, nil
		}
	}

	return nil, ErrNoActiveConnections
}

func (m *FallbackManager) RecordFailure(conn *store.Connection, model string) error {
	conn.BackoffLevel++
	delay := backoffDelay(conn.BackoffLevel)
	if conn.ModelLocks == nil {
		conn.ModelLocks = make(map[string]int64)
	}
	conn.ModelLocks[model] = m.clock.Now().Add(delay).Unix()

	if err := m.store.UpdateConnection(conn); err != nil {
		return fmt.Errorf("update connection backoff: %w", err)
	}
	return nil
}

func (m *FallbackManager) RecordSuccess(conn *store.Connection, model string) error {
	conn.BackoffLevel = 0
	if conn.ModelLocks != nil {
		delete(conn.ModelLocks, model)
	}

	if err := m.store.UpdateConnection(conn); err != nil {
		return fmt.Errorf("update connection backoff: %w", err)
	}
	return nil
}

func connectionAvailable(conn *store.Connection, model string, now int64) bool {
	if conn.UnavailableUntil != nil && *conn.UnavailableUntil > now {
		return false
	}
	if conn.ModelLocks != nil && conn.ModelLocks[model] > now {
		return false
	}
	return true
}

func backoffDelay(level int) time.Duration {
	if level <= 0 {
		return initialBackoff
	}

	delay := initialBackoff
	for i := 1; i < level; i++ {
		delay *= 2
		if delay >= maxBackoff {
			return maxBackoff
		}
	}
	return delay
}
