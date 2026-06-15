package store

import "testing"

// TestMCPClientConfigFlags proves the additive per-client config-blob flags
// (allow_on_all_virtual_keys / disable_auto_tool_inject / is_code_mode_client)
// round-trip through mcp_clients.config_json, default false when absent, and that
// an old blob without the keys still unmarshals cleanly (bf-mcp-2 D6/D7/D8).
func TestMCPClientConfigFlags(t *testing.T) {
	st := newTestStore(t)

	created, err := st.CreateMCPClient(&MCPClient{
		Name: "exa",
		Type: "custom",
		Config: map[string]any{
			"allow_on_all_virtual_keys": true,
			"disable_auto_tool_inject":  true,
			"is_code_mode_client":       true,
		},
	})
	if err != nil {
		t.Fatalf("CreateMCPClient: %v", err)
	}
	got, err := st.GetMCPClient(created.ID)
	if err != nil {
		t.Fatalf("GetMCPClient: %v", err)
	}
	if !MCPClientAllowOnAllVKs(got) {
		t.Fatalf("allow_on_all_virtual_keys did not round-trip")
	}
	if !MCPClientDisableAutoToolInject(got) {
		t.Fatalf("disable_auto_tool_inject did not round-trip")
	}
	if !MCPClientIsCodeMode(got) {
		t.Fatalf("is_code_mode_client did not round-trip")
	}

	// Old blob without the keys -> all flags default false, no panic.
	plain, err := st.CreateMCPClient(&MCPClient{Name: "fs", Type: "custom"})
	if err != nil {
		t.Fatalf("CreateMCPClient plain: %v", err)
	}
	plainGot, _ := st.GetMCPClient(plain.ID)
	if MCPClientAllowOnAllVKs(plainGot) || MCPClientDisableAutoToolInject(plainGot) || MCPClientIsCodeMode(plainGot) {
		t.Fatalf("absent flags should default false: %+v", plainGot.Config)
	}
}

// TestCanonicalizeExtraHeaders proves the AllowedExtraHeaders whitelist
// canonicalization (PAR-BF-MCP-071, D8 config-only): lowercase, trimmed, no
// empties, no duplicates, order preserved.
func TestCanonicalizeExtraHeaders(t *testing.T) {
	got := CanonicalizeExtraHeaders([]string{" X-Trace ", "x-trace", "", "Authorization", "  "})
	want := []string{"x-trace", "authorization"}
	if len(got) != len(want) {
		t.Fatalf("canonicalized = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("canonicalized[%d] = %q, want %q (%v)", i, got[i], want[i], got)
		}
	}
}
