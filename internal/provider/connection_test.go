package provider

import (
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/store"
)

type fakeConnectionStore struct {
	connections map[string][]*store.Connection
	err         error
}

func (f fakeConnectionStore) GetActiveConnections(provider string) ([]*store.Connection, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.connections[provider], nil
}

func TestConnectionManagerSelectsActiveConnectionsRoundRobin(t *testing.T) {
	manager := NewConnectionManager(fakeConnectionStore{
		connections: map[string][]*store.Connection{
			"openai": {
				{ID: "conn-1", Provider: "openai", IsActive: true},
				{ID: "conn-2", Provider: "openai", IsActive: true},
				{ID: "conn-3", Provider: "openai", IsActive: true},
			},
		},
	})

	wantIDs := []string{"conn-1", "conn-2", "conn-3", "conn-1", "conn-2"}
	for _, wantID := range wantIDs {
		conn, err := manager.Next("openai")
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if conn.ID != wantID {
			t.Fatalf("connection ID = %q, want %q", conn.ID, wantID)
		}
	}
}

func TestConnectionManagerTracksProvidersIndependently(t *testing.T) {
	manager := NewConnectionManager(fakeConnectionStore{
		connections: map[string][]*store.Connection{
			"openai": {
				{ID: "openai-1", Provider: "openai", IsActive: true},
				{ID: "openai-2", Provider: "openai", IsActive: true},
			},
			"anthropic": {
				{ID: "anthropic-1", Provider: "anthropic", IsActive: true},
				{ID: "anthropic-2", Provider: "anthropic", IsActive: true},
			},
		},
	})

	if conn, err := manager.Next("openai"); err != nil || conn.ID != "openai-1" {
		t.Fatalf("first openai = %+v, %v", conn, err)
	}
	if conn, err := manager.Next("anthropic"); err != nil || conn.ID != "anthropic-1" {
		t.Fatalf("first anthropic = %+v, %v", conn, err)
	}
	if conn, err := manager.Next("openai"); err != nil || conn.ID != "openai-2" {
		t.Fatalf("second openai = %+v, %v", conn, err)
	}
	if conn, err := manager.Next("anthropic"); err != nil || conn.ID != "anthropic-2" {
		t.Fatalf("second anthropic = %+v, %v", conn, err)
	}
}

func TestConnectionManagerNoActiveConnections(t *testing.T) {
	manager := NewConnectionManager(fakeConnectionStore{
		connections: map[string][]*store.Connection{
			"openai": nil,
		},
	})

	_, err := manager.Next("openai")
	if !errors.Is(err, ErrNoActiveConnections) {
		t.Fatalf("expected ErrNoActiveConnections, got %v", err)
	}
}

func TestConnectionManagerWrapsStoreError(t *testing.T) {
	storeErr := errors.New("database unavailable")
	manager := NewConnectionManager(fakeConnectionStore{err: storeErr})

	_, err := manager.Next("openai")
	if !errors.Is(err, storeErr) {
		t.Fatalf("expected wrapped store error, got %v", err)
	}
}
