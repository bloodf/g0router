package inference

import (
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/store"
)

// fakeProjectIDResolver implements ProjectIDResolver: a deterministic fetch seam.
type fakeProjectIDResolver struct {
	id       string
	err      error
	calls    int
	lastConn string
}

func (f *fakeProjectIDResolver) ResolveProjectID(conn *store.Connection) (string, error) {
	f.calls++
	f.lastConn = conn.ID
	return f.id, f.err
}

// fakeProjectIDPersister captures persisted (connID, projectID) pairs.
type fakeProjectIDPersister struct {
	saved map[string]string
	err   error
}

func (f *fakeProjectIDPersister) PersistProjectID(connID, projectID string) error {
	if f.err != nil {
		return f.err
	}
	if f.saved == nil {
		f.saved = map[string]string{}
	}
	f.saved[connID] = projectID
	return nil
}

// TestProjectIDColdMissResolvesAndPersists verifies PAR-ROUTE-053: an
// antigravity connection with no cached project ID triggers a single resolve and
// a persist (chat.js:191-198).
func TestProjectIDColdMissResolvesAndPersists(t *testing.T) {
	resolver := &fakeProjectIDResolver{id: "proj-123"}
	persister := &fakeProjectIDPersister{}
	m := NewProjectIDManager(resolver, persister)

	conn := &store.Connection{ID: "c1", ProviderID: "antigravity", Name: "ag-prod"}
	got, err := m.EnsureProjectID(conn)
	if err != nil {
		t.Fatalf("EnsureProjectID: %v", err)
	}
	if got != "proj-123" {
		t.Errorf("project id = %q, want proj-123", got)
	}
	if resolver.calls != 1 {
		t.Errorf("resolver calls = %d, want 1", resolver.calls)
	}
	if persister.saved["c1"] != "proj-123" {
		t.Errorf("persisted[c1] = %q, want proj-123", persister.saved["c1"])
	}
	// The connection's in-memory metadata now carries the project id.
	if pid := projectIDFromMetadata(conn.Metadata); pid != "proj-123" {
		t.Errorf("conn metadata project id = %q, want proj-123", pid)
	}
}

// TestProjectIDWarmNoResolve verifies a connection that already has a cached
// project ID does NOT call the resolver (warm path).
func TestProjectIDWarmNoResolve(t *testing.T) {
	resolver := &fakeProjectIDResolver{id: "should-not-be-used"}
	persister := &fakeProjectIDPersister{}
	m := NewProjectIDManager(resolver, persister)

	conn := &store.Connection{
		ID:         "c1",
		ProviderID: "gemini-cli",
		Name:       "g-prod",
		Metadata:   `{"projectId":"existing-proj"}`,
	}
	got, err := m.EnsureProjectID(conn)
	if err != nil {
		t.Fatalf("EnsureProjectID: %v", err)
	}
	if got != "existing-proj" {
		t.Errorf("project id = %q, want existing-proj", got)
	}
	if resolver.calls != 0 {
		t.Errorf("resolver calls = %d, want 0 (warm)", resolver.calls)
	}
	if len(persister.saved) != 0 {
		t.Errorf("persisted %v, want none (warm)", persister.saved)
	}
}

// TestProjectIDNonApplicableProvider verifies a provider that does not need a
// project ID is left untouched (no resolve, no persist).
func TestProjectIDNonApplicableProvider(t *testing.T) {
	resolver := &fakeProjectIDResolver{id: "x"}
	persister := &fakeProjectIDPersister{}
	m := NewProjectIDManager(resolver, persister)

	conn := &store.Connection{ID: "c1", ProviderID: "openai", Name: "o"}
	got, err := m.EnsureProjectID(conn)
	if err != nil {
		t.Fatalf("EnsureProjectID: %v", err)
	}
	if got != "" {
		t.Errorf("project id = %q, want empty (non-applicable)", got)
	}
	if resolver.calls != 0 {
		t.Errorf("resolver calls = %d, want 0", resolver.calls)
	}
}

// TestProjectIDNilManager verifies a nil manager (no resolver wired) is a safe
// no-op (additive, backward-compatible).
func TestProjectIDNilManager(t *testing.T) {
	var m *ProjectIDManager
	conn := &store.Connection{ID: "c1", ProviderID: "antigravity"}
	got, err := m.EnsureProjectID(conn)
	if err != nil || got != "" {
		t.Errorf("nil manager = (%q,%v), want empty no-op", got, err)
	}
}

// TestProjectIDResolverErrorPropagates verifies a fetch error is surfaced.
func TestProjectIDResolverErrorPropagates(t *testing.T) {
	resolver := &fakeProjectIDResolver{err: errors.New("fetch failed")}
	m := NewProjectIDManager(resolver, &fakeProjectIDPersister{})
	conn := &store.Connection{ID: "c1", ProviderID: "antigravity"}
	if _, err := m.EnsureProjectID(conn); err == nil {
		t.Fatal("expected resolver error, got nil")
	}
}
