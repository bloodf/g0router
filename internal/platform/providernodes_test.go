package platform

import (
	"errors"
	"net"
	"path/filepath"
	"testing"

	"github.com/bloodf/g0router/internal/store"
)

func openNodeStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	secret, err := store.LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	st, err := store.Open(filepath.Join(dir, "test.db"), secret)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// nodePublicResolver maps every host to a public IP so the SSRF guard allows it;
// nodeBlockedResolver maps to a private IP so the guard blocks it.
func nodePublicResolver(string) ([]net.IP, error)  { return []net.IP{net.ParseIP("93.184.216.34")}, nil }
func nodeBlockedResolver(string) ([]net.IP, error) { return []net.IP{net.ParseIP("127.0.0.1")}, nil }

func TestSanitizeNodeBaseURL(t *testing.T) {
	cases := []struct {
		name, apiType, raw, want string
	}{
		{"anthropic strips /messages", "anthropic-compatible", "https://x.example.com/v1/messages", "https://x.example.com/v1"},
		{"anthropic strips /messages/", "anthropic-compatible", "https://x.example.com/v1/messages/", "https://x.example.com/v1"},
		{"embedding strips /embeddings", "custom-embedding", "https://x.example.com/embeddings", "https://x.example.com"},
		{"openai untouched", "openai-compatible", "https://x.example.com/v1", "https://x.example.com/v1"},
		{"anthropic no messages segment untouched", "anthropic-compatible", "https://x.example.com/v1", "https://x.example.com/v1"},
		{"idempotent anthropic", "anthropic-compatible", "https://x.example.com/v1/messages/messages", "https://x.example.com/v1/messages"},
		{"trailing slash trimmed openai", "openai-compatible", "https://x.example.com/v1/", "https://x.example.com/v1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SanitizeNodeBaseURL(c.apiType, c.raw); got != c.want {
				t.Fatalf("SanitizeNodeBaseURL(%q,%q) = %q, want %q", c.apiType, c.raw, got, c.want)
			}
		})
	}
}

func TestValidateReachable(t *testing.T) {
	svc := NewProviderNodeService(openNodeStore(t))
	svc.SetResolver(nodePublicResolver)
	svc.SetProber(func(req NodeProbeRequest) (NodeProbeResult, error) {
		return NodeProbeResult{Valid: true}, nil
	})
	res, err := svc.Validate(NodeProbeRequest{APIType: "openai-compatible", BaseURL: "https://reachable.example.com/v1"})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !res.Valid {
		t.Fatalf("reachable node valid = %v, want true", res.Valid)
	}
}

func TestValidateFallbackToChatCompletions(t *testing.T) {
	svc := NewProviderNodeService(openNodeStore(t))
	svc.SetResolver(nodePublicResolver)
	// The fake prober is responsible for the /models→/chat/completions fallback;
	// here it succeeds when a modelID is provided.
	svc.SetProber(func(req NodeProbeRequest) (NodeProbeResult, error) {
		if req.ModelID == "" {
			return NodeProbeResult{Valid: false, Error: "no models endpoint"}, nil
		}
		return NodeProbeResult{Valid: true}, nil
	})
	res, err := svc.Validate(NodeProbeRequest{APIType: "openai-compatible", BaseURL: "https://x.example.com/v1", ModelID: "gpt-4"})
	if err != nil {
		t.Fatalf("Validate fallback: %v", err)
	}
	if !res.Valid {
		t.Fatalf("fallback valid = %v, want true", res.Valid)
	}
}

func TestValidateUnreachable(t *testing.T) {
	svc := NewProviderNodeService(openNodeStore(t))
	svc.SetResolver(nodePublicResolver)
	svc.SetProber(func(req NodeProbeRequest) (NodeProbeResult, error) {
		return NodeProbeResult{Valid: false, Error: "connection refused"}, nil
	})
	res, err := svc.Validate(NodeProbeRequest{APIType: "openai-compatible", BaseURL: "https://x.example.com/v1"})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if res.Valid {
		t.Fatalf("unreachable valid = %v, want false", res.Valid)
	}
	if res.Error == "" {
		t.Fatalf("unreachable should carry an error")
	}
}

func TestValidateSSRFBlockedNeverProbes(t *testing.T) {
	svc := NewProviderNodeService(openNodeStore(t))
	svc.SetResolver(nodeBlockedResolver)
	probed := false
	svc.SetProber(func(req NodeProbeRequest) (NodeProbeResult, error) {
		probed = true
		return NodeProbeResult{Valid: true}, nil
	})
	res, err := svc.Validate(NodeProbeRequest{APIType: "openai-compatible", BaseURL: "https://internal.example.com/v1"})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if res.Valid {
		t.Fatalf("SSRF-blocked valid = %v, want false", res.Valid)
	}
	if probed {
		t.Fatalf("prober was called for an SSRF-blocked target")
	}
}

func TestValidateMalformedURL(t *testing.T) {
	svc := NewProviderNodeService(openNodeStore(t))
	svc.SetResolver(nodePublicResolver)
	svc.SetProber(func(req NodeProbeRequest) (NodeProbeResult, error) {
		return NodeProbeResult{Valid: true}, nil
	})
	res, err := svc.Validate(NodeProbeRequest{APIType: "openai-compatible", BaseURL: "not a url"})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if res.Valid {
		t.Fatalf("malformed url valid = %v, want false", res.Valid)
	}
}

func TestResolveByPrefixHitMissInactive(t *testing.T) {
	st := openNodeStore(t)
	svc := NewProviderNodeService(st)

	rec := &store.ProviderRecord{
		Name: "Compat", Type: "openai-compatible", BaseURL: "https://compat.example.com/v1",
		Enabled: true, Prefix: "co", APIType: "openai",
	}
	if err := st.CreateProvider(rec); err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	id, baseURL, apiType, ok := svc.ResolveByPrefix("co")
	if !ok {
		t.Fatalf("ResolveByPrefix hit ok = false")
	}
	if id != rec.ID || baseURL != "https://compat.example.com/v1" || apiType != "openai" {
		t.Fatalf("ResolveByPrefix = (%q,%q,%q), want (%s,...,openai)", id, baseURL, apiType, rec.ID)
	}

	if _, _, _, ok := svc.ResolveByPrefix("nope"); ok {
		t.Fatalf("ResolveByPrefix miss ok = true, want false")
	}

	// A disabled node does not resolve.
	dis := &store.ProviderRecord{
		Name: "Off", Type: "openai-compatible", BaseURL: "https://off.example.com",
		Enabled: false, Prefix: "off", APIType: "openai",
	}
	if err := st.CreateProvider(dis); err != nil {
		t.Fatalf("CreateProvider disabled: %v", err)
	}
	if _, _, _, ok := svc.ResolveByPrefix("off"); ok {
		t.Fatalf("ResolveByPrefix inactive ok = true, want false")
	}
}

func TestCreateSanitizesAndProvisionsConnection(t *testing.T) {
	st := openNodeStore(t)
	svc := NewProviderNodeService(st)

	// Create WITH an api_key → sanitized base URL + a bound api_key connection.
	node, err := svc.Create(NodeCreate{
		Name: "Anthropic Node", Type: "anthropic-compatible", Prefix: "an", APIType: "anthropic",
		BaseURL: "https://an.example.com/v1/messages", APIKey: "sk-node-secret",
	})
	if err != nil {
		t.Fatalf("Create with key: %v", err)
	}
	if node.BaseURL != "https://an.example.com/v1" {
		t.Fatalf("create did not sanitize base_url: %q", node.BaseURL)
	}
	conns, err := st.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections: %v", err)
	}
	var found *store.Connection
	for _, c := range conns {
		if c.ProviderID == node.ID {
			found = c
		}
	}
	if found == nil {
		t.Fatalf("create with key did not provision a bound connection")
	}
	if found.Secret != "sk-node-secret" || found.Kind != "api_key" {
		t.Fatalf("provisioned connection = %+v", found)
	}

	// Create WITHOUT a key → no connection provisioned.
	node2, err := svc.Create(NodeCreate{
		Name: "Compat", Type: "openai-compatible", Prefix: "co", APIType: "openai",
		BaseURL: "https://co.example.com/v1",
	})
	if err != nil {
		t.Fatalf("Create without key: %v", err)
	}
	conns, _ = st.ListConnections()
	for _, c := range conns {
		if c.ProviderID == node2.ID {
			t.Fatalf("create without key provisioned a connection: %+v", c)
		}
	}
}

func TestUpdateCascadesBaseURLToProvidersRow(t *testing.T) {
	st := openNodeStore(t)
	svc := NewProviderNodeService(st)

	node, err := svc.Create(NodeCreate{
		Name: "Compat", Type: "openai-compatible", Prefix: "co", APIType: "openai",
		BaseURL: "https://old.example.com/v1",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// A connection bound to the node resolves its base URL transitively via the
	// providers row (connections store no base URL of their own).
	if err := st.CreateConnection(&store.Connection{ProviderID: node.ID, Name: "k", Kind: "api_key", Secret: "sk-1"}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	node.BaseURL = "https://new.example.com/v2/messages"
	node.APIType = "anthropic"
	updated, err := svc.Update(NodeUpdate{
		ID: node.ID, Name: node.Name, Type: "anthropic-compatible",
		Prefix: "co", APIType: "anthropic", BaseURL: node.BaseURL,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	// Anthropic sanitization stripped /messages, cascade persisted on the row.
	if updated.BaseURL != "https://new.example.com/v2" {
		t.Fatalf("cascade base_url = %q, want sanitized https://new.example.com/v2", updated.BaseURL)
	}
	got, err := st.GetProvider(node.ID)
	if err != nil {
		t.Fatalf("GetProvider: %v", err)
	}
	if got.BaseURL != "https://new.example.com/v2" || got.APIType != "anthropic" {
		t.Fatalf("providers row not cascaded: %+v", got)
	}
}

func TestGetAndDelete(t *testing.T) {
	st := openNodeStore(t)
	svc := NewProviderNodeService(st)

	node, err := svc.Create(NodeCreate{Name: "Compat", Type: "openai-compatible", Prefix: "co", APIType: "openai", BaseURL: "https://co.example.com/v1"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := svc.Get(node.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != node.ID {
		t.Fatalf("Get id = %q, want %q", got.ID, node.ID)
	}
	if err := svc.Delete(node.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Get(node.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Get after delete err = %v, want ErrNotFound", err)
	}
}
