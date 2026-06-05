package store

import (
	"errors"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/mcp"
)

// closedStore returns a store whose underlying DB is closed, forcing every
// query/exec to error. Used to exercise the error branches of every method.
func closedStore(t *testing.T) *Store {
	t.Helper()
	s := openTestStore(t)
	if err := s.db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}
	return s
}

func badJSONValue() map[string]any {
	return map[string]any{"bad": make(chan int)}
}

func TestEncodeJSONErrorPaths(t *testing.T) {
	s := openTestStore(t)

	if err := s.CreateConnection(&Connection{
		Provider:             "x",
		AuthType:             AuthTypeAPIKey,
		ProviderSpecificData: badJSONValue(),
	}); err == nil {
		t.Fatal("CreateConnection with bad provider data: want error")
	}
	if err := s.UpdateConnection(&Connection{
		ID:                   "id",
		Provider:             "x",
		AuthType:             AuthTypeAPIKey,
		ProviderSpecificData: badJSONValue(),
	}); err == nil {
		t.Fatal("UpdateConnection with bad provider data: want error")
	}
	if err := s.CreateMCPClient(&MCPClient{
		Name:      "c",
		Transport: mcp.TransportStdio,
		Env:       map[string]string{"a": "b"},
		Args:      []string{"x"},
	}); err != nil {
		// sanity: well-formed should succeed
		t.Fatalf("CreateMCPClient sanity: %v", err)
	}
}

func TestClosedStoreErrorPaths(t *testing.T) {
	s := closedStore(t)

	checks := []struct {
		name string
		fn   func() error
	}{
		{"CreateConnection", func() error { return s.CreateConnection(&Connection{Provider: "p", AuthType: AuthTypeAPIKey}) }},
		{"GetConnection", func() error { _, err := s.GetConnection("x"); return err }},
		{"GetConnections", func() error { _, err := s.GetConnections("p"); return err }},
		{"ListConnections", func() error { _, err := s.ListConnections(); return err }},
		{"GetActiveConnections", func() error { _, err := s.GetActiveConnections("p"); return err }},
		{"UpdateConnection", func() error { return s.UpdateConnection(&Connection{ID: "x", Provider: "p", AuthType: AuthTypeAPIKey}) }},
		{"UpdateConnectionCredentials", func() error { return s.UpdateConnectionCredentials("x", nil, nil, nil) }},
		{"DeleteConnection", func() error { return s.DeleteConnection("x") }},
		{"SetModelAlias", func() error { return s.SetModelAlias(ModelAlias{Alias: "a", Provider: "p", Model: "m"}) }},
		{"ResolveModelAlias", func() error { _, err := s.ResolveModelAlias("a"); return err }},
		{"ListModelAliases", func() error { _, err := s.ListModelAliases(); return err }},
		{"DeleteModelAlias", func() error { return s.DeleteModelAlias("a") }},
		{"CreateAPIKey", func() error { _, _, err := s.CreateAPIKey("n", "s"); return err }},
		{"ValidateAPIKey", func() error { _, _, err := s.ValidateAPIKey("r", "s"); return err }},
		{"ListAPIKeys", func() error { _, err := s.ListAPIKeys(); return err }},
		{"DeleteAPIKey", func() error { return s.DeleteAPIKey("x") }},
		{"CreateCombo", func() error { return s.CreateCombo(&Combo{Name: "c"}) }},
		{"GetCombo", func() error { _, err := s.GetCombo("x"); return err }},
		{"GetActiveCombo", func() error { _, err := s.GetActiveCombo("c"); return err }},
		{"ListCombos", func() error { _, err := s.ListCombos(); return err }},
		{"UpdateCombo", func() error { return s.UpdateCombo(&Combo{ID: "x", Name: "c"}) }},
		{"DeleteCombo", func() error { return s.DeleteCombo("x") }},
		{"CreateMCPClient", func() error { return s.CreateMCPClient(&MCPClient{Name: "c", Transport: mcp.TransportStdio}) }},
		{"GetMCPClient", func() error { _, err := s.GetMCPClient("x"); return err }},
		{"ListMCPClients", func() error { _, err := s.ListMCPClients(); return err }},
		{"UpdateMCPClientManifest", func() error { return s.UpdateMCPClientManifest("x", mcp.Manifest{}) }},
		{"UpdateMCPClientHealth", func() error { return s.UpdateMCPClientHealth("x", "ok") }},
		{"DeleteMCPClient", func() error { return s.DeleteMCPClient("x") }},
		{"GetMCPInstance", func() error { _, err := s.GetMCPInstance("x"); return err }},
		{"ListMCPInstances", func() error { _, err := s.ListMCPInstances(); return err }},
		{"ListActiveMCPInstances", func() error { _, err := s.ListActiveMCPInstances(); return err }},
		{"UpdateMCPInstanceManifest", func() error { return s.UpdateMCPInstanceManifest("x", mcp.Manifest{}) }},
		{"UpdateMCPInstanceHealth", func() error { return s.UpdateMCPInstanceHealth("x", "ok") }},
		{"DeleteMCPInstance", func() error { return s.DeleteMCPInstance("x") }},
		{"CreateMCPOAuthFlow", func() error { return s.CreateMCPOAuthFlow(&MCPOAuthFlow{InstanceID: "i", ExpiresAt: time.Now()}) }},
		{"ConsumeMCPOAuthFlow", func() error { _, err := s.ConsumeMCPOAuthFlow("i", "s"); return err }},
		{"UpsertMCPOAuthAccount", func() error { return s.UpsertMCPOAuthAccount(&MCPOAuthAccount{InstanceID: "i", AccountLabel: "d", AccessToken: "t"}) }},
		{"ListMCPOAuthAccounts", func() error { _, err := s.ListMCPOAuthAccounts("i"); return err }},
		{"GetValidMCPOAuthAccount", func() error { _, err := s.GetValidMCPOAuthAccount("i", "r"); return err }},
		{"ConsumeFlow", func() error { _, err := s.ConsumeFlow("i", "s"); return err }},
		{"SaveAccount", func() error { return s.SaveAccount(mcp.OAuthAccount{InstanceID: "i", AccessToken: "t"}) }},
		{"AccountLabelForInstance", func() error { _, err := s.AccountLabelForInstance("i"); return err }},
		{"CreateOAuthSession", func() error { return s.CreateOAuthSession(&OAuthSession{Provider: "p", ExpiresAt: time.Now()}) }},
		{"ConsumeOAuthSession", func() error { _, err := s.ConsumeOAuthSession("s"); return err }},
		{"GetOAuthSession", func() error { _, err := s.GetOAuthSession("s"); return err }},
		{"SetPricingOverride", func() error { return s.SetPricingOverride(PricingOverride{Provider: "p", Model: "m"}) }},
		{"GetPricingOverride", func() error { _, err := s.GetPricingOverride("p", "m"); return err }},
		{"PricingOverride", func() error { _, _, err := s.PricingOverride("p", "m"); return err }},
		{"ListPricingOverrides", func() error { _, err := s.ListPricingOverrides(); return err }},
		{"DeletePricingOverride", func() error { return s.DeletePricingOverride("p", "m") }},
		{"GetSettings", func() error { _, err := s.GetSettings(); return err }},
		{"UpdateSettings", func() error { return s.UpdateSettings(Settings{}) }},
		{"LogRequest", func() error { return s.LogRequest(&RequestLogEntry{RequestID: "r", Provider: "p", Model: "m", AuthType: "api_key"}) }},
		{"GetUsage", func() error { _, err := s.GetUsage(UsageFilter{}); return err }},
		{"GetUsageSummary", func() error { _, err := s.GetUsageSummary(UsageFilter{}); return err }},
	}

	for _, c := range checks {
		if err := c.fn(); err == nil {
			t.Errorf("%s on closed DB: want error, got nil", c.name)
		}
	}
}

func TestListConnectionsReturnsAll(t *testing.T) {
	s := openTestStore(t)
	for _, p := range []string{"anthropic", "openai"} {
		if err := s.CreateConnection(&Connection{Provider: p, AuthType: AuthTypeAPIKey, IsActive: true}); err != nil {
			t.Fatalf("CreateConnection: %v", err)
		}
	}
	conns, err := s.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections: %v", err)
	}
	if len(conns) != 2 {
		t.Fatalf("len = %d, want 2", len(conns))
	}
}

func TestScanConnectionDecodeErrors(t *testing.T) {
	s := openTestStore(t)
	conn := &Connection{Provider: "p", AuthType: AuthTypeAPIKey, IsActive: true}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if _, err := s.db.Exec("UPDATE connections SET provider_specific_data = ? WHERE id = ?", "{not json", conn.ID); err != nil {
		t.Fatalf("corrupt provider data: %v", err)
	}
	if _, err := s.GetConnection(conn.ID); err == nil {
		t.Fatal("GetConnection with corrupt provider data: want decode error")
	}

	conn2 := &Connection{Provider: "p2", AuthType: AuthTypeAPIKey, IsActive: true}
	if err := s.CreateConnection(conn2); err != nil {
		t.Fatalf("CreateConnection2: %v", err)
	}
	if _, err := s.db.Exec("UPDATE connections SET model_locks = ? WHERE id = ?", "{not json", conn2.ID); err != nil {
		t.Fatalf("corrupt model locks: %v", err)
	}
	if _, err := s.GetConnection(conn2.ID); err == nil {
		t.Fatal("GetConnection with corrupt model locks: want decode error")
	}
}

func TestScanComboDecodeError(t *testing.T) {
	s := openTestStore(t)
	combo := &Combo{Name: "c", Steps: []ComboStep{{Provider: "p", Model: "m"}}, IsActive: true}
	if err := s.CreateCombo(combo); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}
	if _, err := s.db.Exec("UPDATE combos SET steps = ? WHERE id = ?", "{not json", combo.ID); err != nil {
		t.Fatalf("corrupt steps: %v", err)
	}
	if _, err := s.GetCombo(combo.ID); err == nil {
		t.Fatal("GetCombo with corrupt steps: want decode error")
	}
}

func TestScanMCPClientDecodeError(t *testing.T) {
	s := openTestStore(t)
	client := &MCPClient{Name: "c", Transport: mcp.TransportStdio, Command: strPtr("x")}
	if err := s.CreateMCPClient(client); err != nil {
		t.Fatalf("CreateMCPClient: %v", err)
	}
	if _, err := s.db.Exec("UPDATE mcp_clients SET tool_manifest = ? WHERE id = ?", "{not json", client.ID); err != nil {
		t.Fatalf("corrupt manifest: %v", err)
	}
	if _, err := s.GetMCPClient(client.ID); err == nil {
		t.Fatal("GetMCPClient with corrupt manifest: want decode error")
	}
}

func TestMCPClientNotFoundAndConfig(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.GetMCPClient("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetMCPClient missing: %v", err)
	}
	if err := s.UpdateMCPClientManifest("missing", mcp.Manifest{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateMCPClientManifest missing: %v", err)
	}
	if err := s.UpdateMCPClientHealth("missing", "ok"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateMCPClientHealth missing: %v", err)
	}
	if err := s.DeleteMCPClient("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeleteMCPClient missing: %v", err)
	}

	client := &MCPClient{
		Name:      "cfg",
		Transport: mcp.TransportStdio,
		Command:   strPtr("/bin/echo"),
		Args:      []string{"hi"},
		Env:       map[string]string{"K": "V"},
		URL:       strPtr("http://x"),
	}
	if err := s.CreateMCPClient(client); err != nil {
		t.Fatalf("CreateMCPClient: %v", err)
	}
	got, err := s.GetMCPClient(client.ID)
	if err != nil {
		t.Fatalf("GetMCPClient: %v", err)
	}
	cfg := got.ClientConfig()
	if cfg.Command != "/bin/echo" || cfg.URL != "http://x" || cfg.Env["K"] != "V" || len(cfg.Args) != 1 {
		t.Fatalf("ClientConfig mismatch: %+v", cfg)
	}
}

func TestMCPInstanceLifecycleAndErrors(t *testing.T) {
	s := openTestStore(t)
	inst := createOAuthTestInstance(t, s, "lifecycle")

	insts, err := s.ListMCPInstances()
	if err != nil {
		t.Fatalf("ListMCPInstances: %v", err)
	}
	if len(insts) != 1 {
		t.Fatalf("ListMCPInstances len = %d", len(insts))
	}
	active, err := s.ListActiveMCPInstances()
	if err != nil {
		t.Fatalf("ListActiveMCPInstances: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("ListActiveMCPInstances len = %d", len(active))
	}

	if err := s.UpdateMCPInstanceManifest(inst.ID, mcp.Manifest{}); err != nil {
		t.Fatalf("UpdateMCPInstanceManifest: %v", err)
	}
	if err := s.UpdateMCPInstanceHealth(inst.ID, "healthy"); err != nil {
		t.Fatalf("UpdateMCPInstanceHealth: %v", err)
	}
	if err := s.UpdateMCPInstanceManifest("missing", mcp.Manifest{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateMCPInstanceManifest missing: %v", err)
	}
	if err := s.UpdateMCPInstanceHealth("missing", "ok"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateMCPInstanceHealth missing: %v", err)
	}
	if err := s.DeleteMCPInstance("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeleteMCPInstance missing: %v", err)
	}
	if err := s.DeleteMCPInstance(inst.ID); err != nil {
		t.Fatalf("DeleteMCPInstance: %v", err)
	}
	if _, err := s.GetMCPInstance(inst.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetMCPInstance after delete: %v", err)
	}
}

func TestMCPInstanceCreateInvalidConfig(t *testing.T) {
	s := openTestStore(t)
	// HTTP launch type with stdio transport is an invalid combination.
	err := s.CreateMCPInstance(&MCPInstance{
		Name:       "invalid",
		ServerKey:  "k",
		LaunchType: mcp.LaunchHTTP,
		Transport:  mcp.TransportStdio,
	})
	if err == nil {
		t.Fatal("CreateMCPInstance with invalid config: want error")
	}
}

func TestSaveAccountAndConsumeFlowMapping(t *testing.T) {
	s := openTestStore(t)
	inst := createOAuthTestInstance(t, s, "save-acct")

	// SaveAccount with empty label defaults to "default".
	if err := s.SaveAccount(mcp.OAuthAccount{
		InstanceID:  inst.ID,
		AccessToken: "tok",
		ResourceURI: "https://mcp.example",
		ExpiresAt:   time.Now().Add(time.Hour),
		Scopes:      []string{"read"},
	}); err != nil {
		t.Fatalf("SaveAccount: %v", err)
	}
	accounts, err := s.ListMCPOAuthAccounts(inst.ID)
	if err != nil {
		t.Fatalf("ListMCPOAuthAccounts: %v", err)
	}
	if len(accounts) != 1 || accounts[0].AccountLabel != "default" {
		t.Fatalf("accounts = %+v", accounts)
	}

	// ConsumeFlow on missing flow returns ErrOAuthFlowNotFound.
	if _, err := s.ConsumeFlow(inst.ID, "nope"); !errors.Is(err, mcp.ErrOAuthFlowNotFound) {
		t.Fatalf("ConsumeFlow missing: %v", err)
	}

	// Create a real flow, then ConsumeFlow maps it through.
	flow := &MCPOAuthFlow{
		InstanceID:         inst.ID,
		State:              "st",
		CodeVerifierSecret: "ver",
		ClientID:           "cid",
		ExpiresAt:          time.Now().Add(time.Hour),
	}
	if err := s.CreateMCPOAuthFlow(flow); err != nil {
		t.Fatalf("CreateMCPOAuthFlow: %v", err)
	}
	got, err := s.ConsumeFlow(inst.ID, "st")
	if err != nil {
		t.Fatalf("ConsumeFlow: %v", err)
	}
	if got.ClientID != "cid" || got.State != "st" {
		t.Fatalf("ConsumeFlow mapping: %+v", got)
	}
}

func TestConsumeMCPOAuthFlowExpired(t *testing.T) {
	s := openTestStore(t)
	inst := createOAuthTestInstance(t, s, "expired-flow")
	flow := &MCPOAuthFlow{
		InstanceID:         inst.ID,
		State:              "old",
		CodeVerifierSecret: "ver",
		ExpiresAt:          time.Now().Add(-time.Hour),
	}
	if err := s.CreateMCPOAuthFlow(flow); err != nil {
		t.Fatalf("CreateMCPOAuthFlow: %v", err)
	}
	if _, err := s.ConsumeMCPOAuthFlow(inst.ID, "old"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ConsumeMCPOAuthFlow expired: %v", err)
	}
	// Flow should have been deleted.
	if _, err := s.ConsumeMCPOAuthFlow(inst.ID, "old"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ConsumeMCPOAuthFlow second: %v", err)
	}
}

func TestGetValidMCPOAuthAccountSkipsExpired(t *testing.T) {
	s := openTestStore(t)
	inst := createOAuthTestInstance(t, s, "valid-acct")
	if err := s.UpsertMCPOAuthAccount(&MCPOAuthAccount{
		InstanceID:   inst.ID,
		AccountLabel: "expired",
		ResourceURI:  "https://res",
		AccessToken:  "t1",
		ExpiresAt:    time.Now().Add(-time.Hour),
	}); err != nil {
		t.Fatalf("upsert expired: %v", err)
	}
	if _, err := s.GetValidMCPOAuthAccount(inst.ID, "https://res"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetValidMCPOAuthAccount only-expired: %v", err)
	}
	if err := s.UpsertMCPOAuthAccount(&MCPOAuthAccount{
		InstanceID:   inst.ID,
		AccountLabel: "valid",
		ResourceURI:  "https://res",
		AccessToken:  "t2",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("upsert valid: %v", err)
	}
	got, err := s.GetValidMCPOAuthAccount(inst.ID, "https://res")
	if err != nil {
		t.Fatalf("GetValidMCPOAuthAccount: %v", err)
	}
	if got.AccessToken != "t2" {
		t.Fatalf("got token %q, want t2", got.AccessToken)
	}
}

func TestAccountLabelForInstance(t *testing.T) {
	s := openTestStore(t)
	inst := createOAuthTestInstance(t, s, "label-none")
	label, err := s.AccountLabelForInstance(inst.ID)
	if err != nil {
		t.Fatalf("AccountLabelForInstance: %v", err)
	}
	if label != "" {
		t.Fatalf("label = %q, want empty", label)
	}
	if _, err := s.AccountLabelForInstance("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("AccountLabelForInstance missing: %v", err)
	}

	withLabel := &MCPInstance{
		Name:         "label-set",
		ServerKey:    "linear",
		LaunchType:   "http",
		Transport:    "streamable-http",
		URL:          strPtr("https://mcp.example/mcp"),
		AccountLabel: strPtr("team"),
		IsActive:     true,
	}
	if err := s.CreateMCPInstance(withLabel); err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}
	label, err = s.AccountLabelForInstance(withLabel.ID)
	if err != nil {
		t.Fatalf("AccountLabelForInstance set: %v", err)
	}
	if label != "team" {
		t.Fatalf("label = %q, want team", label)
	}
}

func TestPricingOverrideWrapper(t *testing.T) {
	s := openTestStore(t)
	// Not found -> ok=false, no error.
	_, ok, err := s.PricingOverride("p", "m")
	if err != nil || ok {
		t.Fatalf("PricingOverride missing: ok=%v err=%v", ok, err)
	}
	if err := s.SetPricingOverride(PricingOverride{Provider: "p", Model: "m", InputCostPerToken: 1.5, OutputCostPerToken: 2.5}); err != nil {
		t.Fatalf("SetPricingOverride: %v", err)
	}
	po, ok, err := s.PricingOverride("p", "m")
	if err != nil || !ok {
		t.Fatalf("PricingOverride: ok=%v err=%v", ok, err)
	}
	if po.InputCostPerToken != 1.5 || po.OutputCostPerToken != 2.5 {
		t.Fatalf("override = %+v", po)
	}
	overrides, err := s.ListPricingOverrides()
	if err != nil || len(overrides) != 1 {
		t.Fatalf("ListPricingOverrides: %v len=%d", err, len(overrides))
	}
	if err := s.DeletePricingOverride("p", "m"); err != nil {
		t.Fatalf("DeletePricingOverride: %v", err)
	}
	if err := s.DeletePricingOverride("p", "m"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeletePricingOverride missing: %v", err)
	}
}

func TestOAuthSessionExpiry(t *testing.T) {
	s := openTestStore(t)
	session := &OAuthSession{
		Provider:     "anthropic",
		State:        "st-exp",
		CodeVerifier: "v",
		RedirectURI:  "http://cb",
		AccountLabel: "lbl",
		ExpiresAt:    time.Now().Add(-time.Hour),
	}
	if err := s.CreateOAuthSession(session); err != nil {
		t.Fatalf("CreateOAuthSession: %v", err)
	}
	if _, err := s.GetOAuthSession("st-exp"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetOAuthSession expired: %v", err)
	}
	if _, err := s.ConsumeOAuthSession("st-exp"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ConsumeOAuthSession expired: %v", err)
	}

	fresh := &OAuthSession{Provider: "anthropic", State: "st-fresh", ExpiresAt: time.Now().Add(time.Hour)}
	if err := s.CreateOAuthSession(fresh); err != nil {
		t.Fatalf("CreateOAuthSession fresh: %v", err)
	}
	got, err := s.GetOAuthSession("st-fresh")
	if err != nil {
		t.Fatalf("GetOAuthSession fresh: %v", err)
	}
	if got.State != "st-fresh" {
		t.Fatalf("state = %q", got.State)
	}
	consumed, err := s.ConsumeOAuthSession("st-fresh")
	if err != nil {
		t.Fatalf("ConsumeOAuthSession fresh: %v", err)
	}
	if consumed.State != "st-fresh" {
		t.Fatalf("consumed state = %q", consumed.State)
	}
	if _, err := s.GetOAuthSession("st-fresh"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetOAuthSession after consume: %v", err)
	}
}

func TestUsageLogAndQuery(t *testing.T) {
	s := openTestStore(t)
	in, out, total := 10, 20, 30
	cost := 0.5
	ts := time.Now().UTC()
	if err := s.LogRequest(&RequestLogEntry{
		Timestamp:    ts,
		RequestID:    "r1",
		Provider:     "anthropic",
		Model:        "claude",
		AuthType:     "oauth",
		InputTokens:  &in,
		OutputTokens: &out,
		TotalTokens:  &total,
		CostUSD:      &cost,
		RTKEnabled:   boolPtr(true),
		CavemanEnabled: boolPtr(false),
	}); err != nil {
		t.Fatalf("LogRequest: %v", err)
	}
	prov := "anthropic"
	model := "claude"
	auth := "oauth"
	from := ts.Add(-time.Hour)
	to := ts.Add(time.Hour)
	entries, err := s.GetUsage(UsageFilter{
		Provider: &prov, Model: &model, AuthType: &auth,
		From: &from, To: &to, Limit: 10, Offset: 0,
	})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d", len(entries))
	}
	if entries[0].RTKEnabled == nil || !*entries[0].RTKEnabled {
		t.Fatal("RTKEnabled mismatch")
	}
	summary, err := s.GetUsageSummary(UsageFilter{Provider: &prov})
	if err != nil {
		t.Fatalf("GetUsageSummary: %v", err)
	}
	if summary.RequestCount != 1 || summary.TotalTokens != 30 {
		t.Fatalf("summary = %+v", summary)
	}
}

func boolPtr(b bool) *bool { return &b }
