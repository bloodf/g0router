package store

import "fmt"

// nodeTypes is the set of provider record types that represent provider nodes
// (the dynamic prefix-routing endpoints, w7-platnodes / PAR-PLAT-010). A node is
// a providers row of one of these types carrying a routing prefix and api_type.
var nodeTypes = map[string]bool{
	"openai-compatible":    true,
	"anthropic-compatible": true,
	"custom-embedding":     true,
}

// IsNodeType reports whether typ is one of the provider-node types.
func IsNodeType(typ string) bool {
	return nodeTypes[typ]
}

// ListProviderNodes returns all provider rows whose type is a node type, ordered
// by creation time. Plain providers (openai/anthropic/...) are excluded.
func (s *Store) ListProviderNodes() ([]*ProviderRecord, error) {
	all, err := s.ListProviders()
	if err != nil {
		return nil, err
	}
	out := make([]*ProviderRecord, 0, len(all))
	for _, p := range all {
		if nodeTypes[p.Type] {
			out = append(out, p)
		}
	}
	return out, nil
}

// GetProviderNodeByPrefix returns the provider node registered under prefix. An
// empty prefix never matches (plain providers carry ''); a miss returns
// ErrNotFound (w7-platnodes / PAR-ROUTE-009).
func (s *Store) GetProviderNodeByPrefix(prefix string) (*ProviderRecord, error) {
	if prefix == "" {
		return nil, ErrNotFound
	}
	row := s.db.QueryRow(
		"SELECT id, name, type, base_url, enabled, prefix, api_type, created_at, updated_at FROM providers WHERE prefix = ? ORDER BY created_at, id LIMIT 1",
		prefix,
	)
	p, err := scanProvider(row)
	if err != nil {
		return nil, err
	}
	if !nodeTypes[p.Type] {
		return nil, ErrNotFound
	}
	return p, nil
}

// ProviderNodePrefix is a lightweight projection of a node's routing fields used
// by the inference prefix-override resolver.
type ProviderNodePrefix struct {
	Prefix  string
	ID      string
	APIType string
	BaseURL string
}

// ListProviderNodePrefixes returns the routing projection for every active
// provider node carrying a non-empty prefix.
func (s *Store) ListProviderNodePrefixes() ([]ProviderNodePrefix, error) {
	nodes, err := s.ListProviderNodes()
	if err != nil {
		return nil, fmt.Errorf("list provider node prefixes: %w", err)
	}
	out := make([]ProviderNodePrefix, 0, len(nodes))
	for _, n := range nodes {
		if n.Prefix == "" {
			continue
		}
		out = append(out, ProviderNodePrefix{
			Prefix:  n.Prefix,
			ID:      n.ID,
			APIType: n.APIType,
			BaseURL: n.BaseURL,
		})
	}
	return out, nil
}
