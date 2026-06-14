package platform

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/bloodf/g0router/internal/store"
)

// ProviderNodeService is the domain service for provider nodes — the dynamic
// prefix-routing endpoints (w7-platnodes, transport→domain→repository). It owns
// node CRUD over the providers table, baseUrl sanitization, the SSRF-guarded
// validation probe, the update→connections cascade, and prefix resolution. It
// mirrors ProxyPoolService (constructor over *store.Store, injectable seams,
// errors-as-values, no init()).
type ProviderNodeService struct {
	st       *store.Store
	prober   NodeProber
	resolver IPResolver
}

// NewProviderNodeService constructs the service over a store.
func NewProviderNodeService(st *store.Store) *ProviderNodeService {
	return &ProviderNodeService{st: st}
}

// SetProber injects the validation probe (production = real /models→
// /chat/completions HTTP; tests inject a deterministic fake).
func (s *ProviderNodeService) SetProber(p NodeProber) {
	s.prober = p
}

// SetResolver injects the DNS resolver used by the SSRF guard. When unset, the
// system resolver is used; tests inject a deterministic fake.
func (s *ProviderNodeService) SetResolver(r IPResolver) {
	s.resolver = r
}

// NodeProbeRequest carries the transient inputs for a node reachability probe.
// The APIKey is used only for the probe and is never persisted by Validate.
type NodeProbeRequest struct {
	APIType string
	BaseURL string
	APIKey  string
	ModelID string
}

// NodeProbeResult is the outcome of a node validation probe.
type NodeProbeResult struct {
	Valid  bool
	Error  string
	Models []string
}

// NodeProber probes a node's reachability. It is an injectable seam so tests run
// without network access. Production performs the real /models→/chat/completions
// (or /embeddings) probe; tests inject a deterministic fake.
type NodeProber func(req NodeProbeRequest) (NodeProbeResult, error)

// NodeCreate is the input to Create. APIKey, when present, provisions a bound
// api_key connection (encrypted at rest, never echoed).
type NodeCreate struct {
	Name    string
	Type    string
	Prefix  string
	APIType string
	BaseURL string
	APIKey  string
}

// NodeUpdate is the input to Update.
type NodeUpdate struct {
	ID      string
	Name    string
	Type    string
	Prefix  string
	APIType string
	BaseURL string
	Enabled *bool
}

// SanitizeNodeBaseURL normalizes a node base URL by stripping the endpoint
// segment the node type appends at request time (PAR-PLAT-011): an
// anthropic-compatible node strips a trailing /messages, a custom-embedding node
// strips a trailing /embeddings, an openai-compatible node is left as the API
// root. The discriminator is the node TYPE. A single trailing slash is trimmed.
// Idempotent.
func SanitizeNodeBaseURL(nodeType, raw string) string {
	out := strings.TrimRight(strings.TrimSpace(raw), "/")
	switch nodeType {
	case "anthropic-compatible":
		out = trimTrailingSegment(out, "messages")
	case "custom-embedding":
		out = trimTrailingSegment(out, "embeddings")
	}
	return out
}

// trimTrailingSegment removes a single trailing /<segment> from a URL path.
func trimTrailingSegment(s, segment string) string {
	if strings.HasSuffix(s, "/"+segment) {
		return strings.TrimRight(strings.TrimSuffix(s, "/"+segment), "/")
	}
	return s
}

// List returns all provider nodes.
func (s *ProviderNodeService) List() ([]*store.ProviderRecord, error) {
	return s.st.ListProviderNodes()
}

// Get returns the node with the given id (ErrNotFound on miss).
func (s *ProviderNodeService) Get(id string) (*store.ProviderRecord, error) {
	return s.st.GetProvider(id)
}

// Create persists a new provider node with a sanitized base URL. When the input
// carries an APIKey, a bound api_key connection is provisioned so the node is
// usable immediately; the key is encrypted at rest and never echoed (ESC-PROVISION).
func (s *ProviderNodeService) Create(in NodeCreate) (*store.ProviderRecord, error) {
	rec := &store.ProviderRecord{
		Name:    in.Name,
		Type:    in.Type,
		BaseURL: SanitizeNodeBaseURL(in.Type, in.BaseURL),
		Enabled: true,
		Prefix:  in.Prefix,
		APIType: in.APIType,
	}
	if err := s.st.CreateProvider(rec); err != nil {
		return nil, fmt.Errorf("create provider node: %w", err)
	}
	if in.APIKey != "" {
		conn := &store.Connection{
			ProviderID: rec.ID,
			Name:       in.Name,
			Kind:       "api_key",
			Secret:     in.APIKey,
		}
		if err := s.st.CreateConnection(conn); err != nil {
			return nil, fmt.Errorf("provision node connection: %w", err)
		}
	}
	return rec, nil
}

// Update persists mutable node fields with a sanitized base URL and cascades the
// change to bound connections (PAR-PLAT-012). A node IS a providers row and
// connections bind by provider_id and store no base URL of their own, so
// persisting the sanitized base_url/api_type/prefix on the row propagates
// transitively to every bound connection at resolve time.
func (s *ProviderNodeService) Update(in NodeUpdate) (*store.ProviderRecord, error) {
	rec, err := s.st.GetProvider(in.ID)
	if err != nil {
		return nil, err
	}
	rec.Name = in.Name
	if in.Type != "" {
		rec.Type = in.Type
	}
	rec.Prefix = in.Prefix
	rec.APIType = in.APIType
	rec.BaseURL = SanitizeNodeBaseURL(rec.Type, in.BaseURL)
	if in.Enabled != nil {
		rec.Enabled = *in.Enabled
	}
	if err := s.st.UpdateProvider(rec); err != nil {
		return nil, fmt.Errorf("update provider node: %w", err)
	}
	return rec, nil
}

// Delete removes the node (ErrNotFound on miss). Bound connections cascade-delete
// via the providers→connections foreign key.
func (s *ProviderNodeService) Delete(id string) error {
	return s.st.DeleteProvider(id)
}

// Validate runs the node reachability probe, hermetic in tests via the injectable
// prober (PAR-PLAT-013). The base URL host is SSRF-guarded BEFORE dialing: a host
// resolving to a private/loopback/link-local address is refused (valid:false)
// without invoking the prober. A malformed URL is rejected. The APIKey is used
// transiently and never persisted.
func (s *ProviderNodeService) Validate(req NodeProbeRequest) (NodeProbeResult, error) {
	host, ok := hostFromURL(req.BaseURL)
	if !ok {
		return NodeProbeResult{Valid: false, Error: "invalid url"}, nil
	}
	// SSRF guard on the user-controllable base URL host. A definitively-blocked
	// target (private/loopback/link-local/etc.) is refused without probing. A
	// resolution error is NOT treated as blocked here: it falls through to the
	// prober, which decides reachability (preserving the w6-f URL-shape success
	// path when no prober is injected).
	if blocked, _, berr := IsBlockedTarget(host, s.resolver); berr == nil && blocked {
		return NodeProbeResult{Valid: false, Error: "target blocked"}, nil
	}
	prober := s.prober
	if prober == nil {
		prober = defaultNodeProber
	}
	res, err := prober(req)
	if err != nil {
		return NodeProbeResult{Valid: false, Error: err.Error()}, nil
	}
	return res, nil
}

// ResolveByPrefix resolves a routing prefix to the node's provider id, base URL,
// and api type. It returns ok=false when no active node carries the prefix
// (PAR-ROUTE-009/040). A disabled node does not resolve.
func (s *ProviderNodeService) ResolveByPrefix(prefix string) (providerID, baseURL, apiType string, ok bool) {
	node, err := s.st.GetProviderNodeByPrefix(prefix)
	if err != nil {
		return "", "", "", false
	}
	if !node.Enabled {
		return "", "", "", false
	}
	return node.ID, node.BaseURL, node.APIType, true
}

// hostFromURL extracts the host from an absolute http(s) URL. It returns ok=false
// for a malformed or non-http(s) URL.
func hostFromURL(raw string) (string, bool) {
	if raw == "" {
		return "", false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", false
	}
	if u.Host == "" {
		return "", false
	}
	return u.Host, true
}

// defaultNodeProber is the fallback when no prober is injected. It preserves the
// w6-f best-effort behavior: a well-formed, SSRF-passed base URL validates on URL
// shape alone (no network). Production injects the real /models→/chat/completions
// prober via SetProber; this fallback keeps validation deterministic otherwise.
func defaultNodeProber(req NodeProbeRequest) (NodeProbeResult, error) {
	return NodeProbeResult{Valid: true}, nil
}
