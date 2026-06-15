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
