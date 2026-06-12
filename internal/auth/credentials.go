package auth

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/store"
)

// CredentialResolver finds a provider's OAuth connection, refreshes it when
// near expiry, and returns the resolved key plus provider-specific data.
type CredentialResolver struct {
	store *store.Store
	flows map[string]*OAuthFlow
	mu    sync.Mutex
	calls map[string]*refreshCall
}

type refreshCall struct {
	wg  sync.WaitGroup
	val *store.Connection
	err error
}

// NewCredentialResolver creates a resolver backed by the given store and flows.
func NewCredentialResolver(st *store.Store, flows map[string]*OAuthFlow) *CredentialResolver {
	if flows == nil {
		flows = map[string]*OAuthFlow{}
	}
	return &CredentialResolver{store: st, flows: flows, calls: map[string]*refreshCall{}}
}

// providerRecordByID finds the provider record for the given provider ID.
func (r *CredentialResolver) providerRecordByID(providerID string) (*store.ProviderRecord, error) {
	providers, err := r.store.ListProviders()
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}
	for _, p := range providers {
		if p.ID == providerID {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no provider record for id %s", providerID)
}

// ResolveKey finds the connection for the given provider ID, decrypts it,
// refreshes if needed (with single-flight dedup), persists the result, and
// returns the key plus provider-specific data.
func (r *CredentialResolver) ResolveKey(providerID string) (schemas.Key, map[string]string, error) {
	provider, err := r.providerRecordByID(providerID)
	if err != nil {
		return schemas.Key{}, nil, err
	}

	conns, err := r.store.ListConnections()
	if err != nil {
		return schemas.Key{}, nil, fmt.Errorf("list connections: %w", err)
	}
	var conn *store.Connection
	for _, c := range conns {
		if c.ProviderID == providerID {
			conn = c
			break
		}
	}
	if conn == nil {
		return schemas.Key{}, nil, fmt.Errorf("no connection for provider %s", providerID)
	}

	// Parse provider-specific data from metadata.
	psd := map[string]string{}
	if conn.Metadata != "" {
		if err := json.Unmarshal([]byte(conn.Metadata), &psd); err != nil {
			return schemas.Key{}, nil, fmt.Errorf("parse provider metadata: %w", err)
		}
	}

	// Refresh if needed.
	if shouldRefresh(provider.Type, conn) {
		refreshed, err := r.doRefresh(provider.Type, conn)
		if err != nil {
			return schemas.Key{}, nil, fmt.Errorf("refresh credentials: %w", err)
		}
		conn = refreshed
		// Re-parse psd after refresh.
		psd = map[string]string{}
		if conn.Metadata != "" {
			if err := json.Unmarshal([]byte(conn.Metadata), &psd); err != nil {
				return schemas.Key{}, nil, fmt.Errorf("parse provider metadata: %w", err)
			}
		}
	}

	key := schemas.Key{
		ID:       conn.ID,
		Provider: providerID,
		Value:    conn.AccessToken,
	}
	if key.Value == "" {
		key.Value = conn.Secret
	}
	return key, psd, nil
}

// RefreshCredentials refreshes the OAuth tokens for the connection identified
// by connectionID and persists the result. It is called by the chat handler
// after a 401/403, so it always attempts a refresh (not gated on expiry).
func (r *CredentialResolver) RefreshCredentials(connectionID string) (string, error) {
	conn, err := r.store.GetConnection(connectionID)
	if err != nil {
		return "", fmt.Errorf("get connection %s: %w", connectionID, err)
	}

	provider, err := r.providerRecordByID(conn.ProviderID)
	if err != nil {
		return "", fmt.Errorf("resolve provider for connection %s: %w", connectionID, err)
	}

	refreshed, err := r.doRefresh(provider.Type, conn)
	if err != nil {
		return "", fmt.Errorf("refresh credentials for connection %s: %w", connectionID, err)
	}
	return refreshed.AccessToken, nil
}

// shouldRefresh returns true when the connection's expiry is within the
// provider-specific lead window.
func shouldRefresh(providerType string, conn *store.Connection) bool {
	if conn.ExpiresAt == 0 {
		return false
	}
	lead := refreshLead(providerType)
	return time.Until(time.Unix(conn.ExpiresAt, 0)) < lead
}

// doRefresh performs an in-flight-deduplicated refresh for the connection.
func (r *CredentialResolver) doRefresh(providerType string, conn *store.Connection) (*store.Connection, error) {
	r.mu.Lock()
	if c, ok := r.calls[conn.ID]; ok {
		r.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := &refreshCall{}
	c.wg.Add(1)
	r.calls[conn.ID] = c
	r.mu.Unlock()

	c.val, c.err = r.refreshAndPersist(providerType, conn)

	r.mu.Lock()
	c.wg.Done()
	delete(r.calls, conn.ID)
	r.mu.Unlock()
	return c.val, c.err
}

func (r *CredentialResolver) refreshAndPersist(providerType string, conn *store.Connection) (*store.Connection, error) {
	flow, ok := r.flows[providerType]
	if !ok {
		return nil, fmt.Errorf("no oauth flow for provider %s", providerType)
	}

	token, err := flow.Refresh(conn.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}

	refreshed := &store.Connection{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.ExpiresAt,
	}
	merged, err := mergeRefreshedCredentials(conn, refreshed)
	if err != nil {
		return nil, fmt.Errorf("merge refreshed credentials: %w", err)
	}

	if err := r.store.UpdateConnection(merged); err != nil {
		return nil, fmt.Errorf("persist refreshed connection: %w", err)
	}
	return merged, nil
}

// mergeRefreshedCredentials overlays refreshed token fields onto current.
// Rules: access_token overwritten; empty new refresh_token preserves old;
// expires_at overwritten when non-zero; providerSpecificData shallow-merged.
func mergeRefreshedCredentials(current, refreshed *store.Connection) (*store.Connection, error) {
	out := *current
	out.AccessToken = refreshed.AccessToken
	if refreshed.RefreshToken != "" {
		out.RefreshToken = refreshed.RefreshToken
	}
	if refreshed.ExpiresAt != 0 {
		out.ExpiresAt = refreshed.ExpiresAt
	}

	// Shallow-merge provider-specific data from metadata.
	psd := map[string]string{}
	if current.Metadata != "" {
		if err := json.Unmarshal([]byte(current.Metadata), &psd); err != nil {
			return nil, fmt.Errorf("parse provider metadata: %w", err)
		}
	}
	newPSD := map[string]string{}
	if refreshed.Metadata != "" {
		if err := json.Unmarshal([]byte(refreshed.Metadata), &newPSD); err != nil {
			return nil, fmt.Errorf("parse provider metadata: %w", err)
		}
	}
	for k, v := range newPSD {
		psd[k] = v
	}
	if len(psd) > 0 {
		b, err := json.Marshal(psd)
		if err != nil {
			return nil, fmt.Errorf("marshal provider metadata: %w", err)
		}
		out.Metadata = string(b)
	}

	return &out, nil
}
