package provider

import (
	"errors"
	"testing"
)

func TestNewFallbackManager(t *testing.T) {
	store := &fakeFallbackStore{}
	m := NewFallbackManager(store)
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
	if m.store != store {
		t.Fatal("expected store to be set")
	}
	if m.cursors == nil {
		t.Fatal("expected cursors to be initialized")
	}

	conn, err := m.Next("openai", "gpt-4")
	if !errors.Is(err, ErrNoActiveConnections) {
		t.Fatalf("expected ErrNoActiveConnections, got %v", err)
	}
	if conn != nil {
		t.Fatal("expected nil connection")
	}
}
