package inference

import "testing"

// TestIsFreeProviderFromCatalog verifies PAR-ROUTE-039: a provider flagged noAuth
// in the catalog (e.g. opencode) is a free provider; a normal provider is not.
func TestIsFreeProviderFromCatalog(t *testing.T) {
	if !isFreeProvider("opencode") {
		t.Error("opencode should be a free (noAuth) provider")
	}
	if isFreeProvider("deepseek") {
		t.Error("deepseek should NOT be a free provider")
	}
	if isFreeProvider("nonexistent-provider") {
		t.Error("unknown provider should NOT be a free provider")
	}
}

// TestSyntheticFreeConnection verifies PAR-ROUTE-039: the synthetic connection
// builder produces a no-auth virtual connection bound to the free provider, with
// a sentinel name and no secret (auth.js:36-53, providers.js:14).
func TestSyntheticFreeConnection(t *testing.T) {
	conn := syntheticFreeConnection("opencode")
	if conn == nil {
		t.Fatal("syntheticFreeConnection returned nil")
	}
	if conn.ProviderID != "opencode" {
		t.Errorf("ProviderID = %q, want opencode", conn.ProviderID)
	}
	if conn.Kind != "api_key" {
		t.Errorf("Kind = %q, want api_key", conn.Kind)
	}
	if conn.Secret != "" {
		t.Errorf("Secret = %q, want empty (no auth)", conn.Secret)
	}
	if conn.ID == "" || conn.Name == "" {
		t.Errorf("synthetic conn must carry a sentinel ID+Name, got ID=%q Name=%q", conn.ID, conn.Name)
	}
}
