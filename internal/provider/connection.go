package provider

import (
	"errors"
	"fmt"
	"sync"

	"github.com/bloodf/g0router/internal/store"
)

var ErrNoActiveConnections = errors.New("provider: no active connections")

type ActiveConnectionStore interface {
	GetActiveConnections(provider string) ([]*store.Connection, error)
}

type ConnectionManager struct {
	store   ActiveConnectionStore
	mu      sync.Mutex
	cursors map[string]int
}

func NewConnectionManager(store ActiveConnectionStore) *ConnectionManager {
	return &ConnectionManager{
		store:   store,
		cursors: make(map[string]int),
	}
}

func (m *ConnectionManager) Next(provider string) (*store.Connection, error) {
	connections, err := m.store.GetActiveConnections(provider)
	if err != nil {
		return nil, fmt.Errorf("get active connections: %w", err)
	}
	if len(connections) == 0 {
		return nil, ErrNoActiveConnections
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	index := m.cursors[provider] % len(connections)
	m.cursors[provider] = index + 1

	return connections[index], nil
}
