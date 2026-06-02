package provider

import (
	"errors"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

type fakeFallbackStore struct {
	connections map[string][]*store.Connection
	updateErr   error
	updated     []*store.Connection
}

func (f *fakeFallbackStore) GetActiveConnections(provider string) ([]*store.Connection, error) {
	return f.connections[provider], nil
}

func (f *fakeFallbackStore) UpdateConnection(conn *store.Connection) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updated = append(f.updated, conn)
	return nil
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

func TestFallbackManagerSelectsRoundRobinSkippingUnavailableConnections(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	globalLockedUntil := now.Add(time.Minute).Unix()
	modelLockedUntil := now.Add(time.Minute).Unix()
	store := &fakeFallbackStore{
		connections: map[string][]*store.Connection{
			"openai": {
				{ID: "conn-1", Provider: "openai", IsActive: true, ModelLocks: map[string]int64{"gpt-4o": modelLockedUntil}},
				{ID: "conn-2", Provider: "openai", IsActive: true},
				{ID: "conn-3", Provider: "openai", IsActive: true, UnavailableUntil: &globalLockedUntil},
				{ID: "conn-4", Provider: "openai", IsActive: true},
			},
		},
	}
	manager := NewFallbackManagerWithClock(store, fixedClock{now: now})

	first, err := manager.Next("openai", "gpt-4o")
	if err != nil {
		t.Fatalf("Next first: %v", err)
	}
	second, err := manager.Next("openai", "gpt-4o")
	if err != nil {
		t.Fatalf("Next second: %v", err)
	}
	third, err := manager.Next("openai", "gpt-4o")
	if err != nil {
		t.Fatalf("Next third: %v", err)
	}

	gotIDs := []string{first.ID, second.ID, third.ID}
	wantIDs := []string{"conn-2", "conn-4", "conn-2"}
	for i, wantID := range wantIDs {
		if gotIDs[i] != wantID {
			t.Fatalf("selection %d = %q, want %q", i, gotIDs[i], wantID)
		}
	}
}

func TestFallbackManagerModelLocksDoNotBlockOtherModels(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	lockedUntil := now.Add(time.Minute).Unix()
	store := &fakeFallbackStore{
		connections: map[string][]*store.Connection{
			"openai": {
				{ID: "conn-1", Provider: "openai", IsActive: true, ModelLocks: map[string]int64{"gpt-4o": lockedUntil}},
				{ID: "conn-2", Provider: "openai", IsActive: true},
			},
		},
	}
	manager := NewFallbackManagerWithClock(store, fixedClock{now: now})

	conn, err := manager.Next("openai", "gpt-4o-mini")
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if conn.ID != "conn-1" {
		t.Fatalf("connection ID = %q, want conn-1", conn.ID)
	}
}

func TestFallbackManagerRecordsExponentialBackoffPerModel(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	conn := &store.Connection{ID: "conn-1", Provider: "openai", IsActive: true}
	store := &fakeFallbackStore{}
	manager := NewFallbackManagerWithClock(store, fixedClock{now: now})

	if err := manager.RecordFailure(conn, "gpt-4o"); err != nil {
		t.Fatalf("RecordFailure first: %v", err)
	}
	if conn.BackoffLevel != 1 {
		t.Fatalf("backoff level after first failure = %d, want 1", conn.BackoffLevel)
	}
	if got := conn.ModelLocks["gpt-4o"]; got != now.Add(time.Second).Unix() {
		t.Fatalf("first lock = %d, want %d", got, now.Add(time.Second).Unix())
	}

	if err := manager.RecordFailure(conn, "gpt-4o"); err != nil {
		t.Fatalf("RecordFailure second: %v", err)
	}
	if conn.BackoffLevel != 2 {
		t.Fatalf("backoff level after second failure = %d, want 2", conn.BackoffLevel)
	}
	if got := conn.ModelLocks["gpt-4o"]; got != now.Add(2*time.Second).Unix() {
		t.Fatalf("second lock = %d, want %d", got, now.Add(2*time.Second).Unix())
	}
	if len(store.updated) != 2 {
		t.Fatalf("updates = %d, want 2", len(store.updated))
	}
}

func TestFallbackManagerCapsBackoffAtFourMinutes(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	conn := &store.Connection{ID: "conn-1", Provider: "openai", IsActive: true, BackoffLevel: 30}
	manager := NewFallbackManagerWithClock(&fakeFallbackStore{}, fixedClock{now: now})

	if err := manager.RecordFailure(conn, "gpt-4o"); err != nil {
		t.Fatalf("RecordFailure: %v", err)
	}
	if got := conn.ModelLocks["gpt-4o"]; got != now.Add(4*time.Minute).Unix() {
		t.Fatalf("lock = %d, want %d", got, now.Add(4*time.Minute).Unix())
	}
}

func TestFallbackManagerRecordSuccessClearsModelBackoff(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	lockedUntil := now.Add(time.Minute).Unix()
	conn := &store.Connection{
		ID:           "conn-1",
		Provider:     "openai",
		IsActive:     true,
		BackoffLevel: 3,
		ModelLocks:   map[string]int64{"gpt-4o": lockedUntil, "gpt-4o-mini": lockedUntil},
	}
	manager := NewFallbackManagerWithClock(&fakeFallbackStore{}, fixedClock{now: now})

	if err := manager.RecordSuccess(conn, "gpt-4o"); err != nil {
		t.Fatalf("RecordSuccess: %v", err)
	}
	if conn.BackoffLevel != 0 {
		t.Fatalf("backoff level = %d, want 0", conn.BackoffLevel)
	}
	if _, ok := conn.ModelLocks["gpt-4o"]; ok {
		t.Fatal("gpt-4o lock was not cleared")
	}
	if _, ok := conn.ModelLocks["gpt-4o-mini"]; !ok {
		t.Fatal("unrelated model lock was cleared")
	}
}

func TestFallbackManagerReturnsNoActiveConnectionsWhenAllLocked(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	lockedUntil := now.Add(time.Minute).Unix()
	manager := NewFallbackManagerWithClock(&fakeFallbackStore{
		connections: map[string][]*store.Connection{
			"openai": {
				{ID: "conn-1", Provider: "openai", IsActive: true, ModelLocks: map[string]int64{"gpt-4o": lockedUntil}},
			},
		},
	}, fixedClock{now: now})

	_, err := manager.Next("openai", "gpt-4o")
	if !errors.Is(err, ErrNoActiveConnections) {
		t.Fatalf("expected ErrNoActiveConnections, got %v", err)
	}
}

func TestFallbackManagerWrapsUpdateErrors(t *testing.T) {
	updateErr := errors.New("database unavailable")
	manager := NewFallbackManagerWithClock(&fakeFallbackStore{updateErr: updateErr}, fixedClock{now: time.Unix(1_700_000_000, 0)})

	err := manager.RecordFailure(&store.Connection{ID: "conn-1"}, "gpt-4o")
	if !errors.Is(err, updateErr) {
		t.Fatalf("expected wrapped update error, got %v", err)
	}
}
