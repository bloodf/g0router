package inference

import (
	"encoding/json"
	"fmt"

	"github.com/bloodf/g0router/internal/store"
)

// projectIDProviders is the set of providers that require a project ID resolved
// on the first (cold) request (PAR-ROUTE-053). Mirrors 9router chat.js:191-198
// (antigravity / gemini-cli).
var projectIDProviders = map[string]bool{
	"antigravity": true,
	"gemini-cli":  true,
}

// ProjectIDResolver fetches the project ID for a connection that lacks one
// (PAR-ROUTE-053). It is the injectable network seam: production issues an HTTP
// request; tests inject a deterministic fake (no network).
type ProjectIDResolver interface {
	ResolveProjectID(conn *store.Connection) (string, error)
}

// ProjectIDPersister persists a resolved project ID for a connection (best-effort
// write-back). Production wires the store; tests inject a fake.
type ProjectIDPersister interface {
	PersistProjectID(connID, projectID string) error
}

// ProjectIDManager resolves and persists a connection's project ID on a cold
// miss for providers that require it (PAR-ROUTE-053). It is additive and
// optional: a nil manager is a safe no-op, so the existing request path is
// unchanged when no resolver is wired.
type ProjectIDManager struct {
	resolver  ProjectIDResolver
	persister ProjectIDPersister
}

// NewProjectIDManager creates a manager over the given resolve+persist seams.
func NewProjectIDManager(resolver ProjectIDResolver, persister ProjectIDPersister) *ProjectIDManager {
	return &ProjectIDManager{resolver: resolver, persister: persister}
}

// EnsureProjectID returns the connection's project ID, resolving and persisting
// it on a cold miss for an applicable provider (PAR-ROUTE-053). For a warm
// connection (project ID already in metadata) it returns the cached value
// without calling the resolver. For a non-applicable provider, or a nil manager,
// it returns ("", nil). The resolved ID is written back into the connection's
// in-memory Metadata so the caller sees it immediately.
func (m *ProjectIDManager) EnsureProjectID(conn *store.Connection) (string, error) {
	if m == nil || m.resolver == nil || conn == nil {
		return "", nil
	}
	if !projectIDProviders[conn.ProviderID] {
		return "", nil
	}
	// Warm path: already cached.
	if pid := projectIDFromMetadata(conn.Metadata); pid != "" {
		return pid, nil
	}
	// Cold miss: resolve.
	pid, err := m.resolver.ResolveProjectID(conn)
	if err != nil {
		return "", fmt.Errorf("resolve project id for %s: %w", conn.ID, err)
	}
	if pid == "" {
		return "", nil
	}
	// Write back into the connection's in-memory metadata.
	conn.Metadata = setProjectIDInMetadata(conn.Metadata, pid)
	// Best-effort persist.
	if m.persister != nil {
		if err := m.persister.PersistProjectID(conn.ID, pid); err != nil {
			return "", fmt.Errorf("persist project id for %s: %w", conn.ID, err)
		}
	}
	return pid, nil
}

// projectIDMetadata is the metadata shape carrying the cached project ID. It uses
// the existing free-form Connection.Metadata seam — no schema change
// (ESC-PROJ-PERSIST).
type projectIDMetadata struct {
	ProjectID string `json:"projectId"`
}

// projectIDFromMetadata extracts the cached project ID from a connection's
// free-form metadata JSON; returns "" when absent or unparseable.
func projectIDFromMetadata(metadata string) string {
	if metadata == "" {
		return ""
	}
	var m projectIDMetadata
	if json.Unmarshal([]byte(metadata), &m) != nil {
		return ""
	}
	return m.ProjectID
}

// setProjectIDInMetadata merges the project ID into the existing metadata JSON,
// preserving any other keys. On a parse failure it writes a fresh object.
func setProjectIDInMetadata(metadata, projectID string) string {
	obj := map[string]json.RawMessage{}
	if metadata != "" {
		if err := json.Unmarshal([]byte(metadata), &obj); err != nil {
			obj = map[string]json.RawMessage{}
		}
	}
	raw, _ := json.Marshal(projectID)
	obj["projectId"] = raw
	out, err := json.Marshal(obj)
	if err != nil {
		return metadata
	}
	return string(out)
}
