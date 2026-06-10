package admin

import (
	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/store"
)

// Handlers bundles the management API endpoints and their dependencies.
type Handlers struct {
	store    *store.Store
	sessions *auth.Sessions
	flows    map[string]*auth.OAuthFlow
}

// New creates the admin handler set. flows maps provider type → OAuth flow
// and may be nil when no OAuth providers are configured.
func New(st *store.Store, sessions *auth.Sessions, flows map[string]*auth.OAuthFlow) *Handlers {
	if flows == nil {
		flows = map[string]*auth.OAuthFlow{}
	}
	return &Handlers{store: st, sessions: sessions, flows: flows}
}

// pathID returns the {id} route parameter.
func pathID(v any) (string, bool) {
	s, ok := v.(string)
	return s, ok
}
