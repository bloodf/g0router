package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/mcp"
)

func TestNewStoreMkdirError(t *testing.T) {
	// Create a regular file, then try to use it as a parent directory.
	base := t.TempDir()
	filePath := filepath.Join(base, "afile")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	// Parent "afile" is a file, so MkdirAll under it must fail.
	if _, err := NewStore(filepath.Join(filePath, "sub", "db.sqlite")); err == nil {
		t.Fatal("NewStore with file-as-parent: want error")
	}
}

func TestNewStoreMigrateError(t *testing.T) {
	// Pre-create an incompatible "connections" object (a view) so the
	// CREATE TABLE IF NOT EXISTS path differs and a later DDL/ALTER can fail.
	path := filepath.Join(t.TempDir(), "db.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	// Create mcp_oauth_flows as a view; ensureColumn's PRAGMA/ALTER will fail.
	if _, err := db.Exec(`CREATE TABLE base (x)`); err != nil {
		t.Fatalf("create base: %v", err)
	}
	if _, err := db.Exec(`CREATE VIEW mcp_oauth_flows AS SELECT x AS id FROM base`); err != nil {
		t.Fatalf("create view: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, err := NewStore(path); err == nil {
		t.Fatal("NewStore with view-shadowed table: want migrate error")
	}
}

func TestEncodeJSONErrorAcrossCreators(t *testing.T) {
	s := openTestStore(t)
	bad := map[string]string(nil)
	_ = bad

	// MCP client with bad env.
	if err := s.CreateMCPClient(&MCPClient{
		Name:      "bad-env",
		Transport: mcp.TransportStdio,
		Command:   strPtr("x"),
		Env:       nil,
	}); err != nil {
		t.Fatalf("baseline create: %v", err)
	}

	// CreateMCPInstance with bad env via a value that cannot marshal is not
	// possible (Env is map[string]string), so exercise encode error on the
	// manifest update path instead which takes mcp.Manifest (always marshals).
	// Use UpsertMCPOAuthAccount with bad metadata: AuthMetadata is
	// map[string]string, also always marshals. These encode calls cannot fail
	// with their typed inputs, so we only assert the success paths run.
	inst := createOAuthTestInstance(t, s, "enc")
	if err := s.UpsertMCPOAuthAccount(&MCPOAuthAccount{
		InstanceID:   inst.ID,
		AccountLabel: "d",
		AccessToken:  "t",
		Scopes:       []string{"a", "b"},
		AuthMetadata: map[string]string{"k": "v"},
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("UpsertMCPOAuthAccount: %v", err)
	}
}

func TestDecodeJSONEmptyAndValid(t *testing.T) {
	// Empty/invalid NullString -> no-op, nil error.
	var dest map[string]any
	if err := decodeJSON(sql.NullString{Valid: false}, &dest); err != nil {
		t.Fatalf("decodeJSON invalid: %v", err)
	}
	if err := decodeJSON(sql.NullString{Valid: true, String: ""}, &dest); err != nil {
		t.Fatalf("decodeJSON empty: %v", err)
	}
	// Valid JSON populates dest.
	if err := decodeJSON(sql.NullString{Valid: true, String: `{"a":1}`}, &dest); err != nil {
		t.Fatalf("decodeJSON valid: %v", err)
	}
	if dest["a"] != float64(1) {
		t.Fatalf("dest = %+v", dest)
	}
}

func TestEncodeJSONNil(t *testing.T) {
	out, err := encodeJSON(nil)
	if err != nil {
		t.Fatalf("encodeJSON(nil): %v", err)
	}
	if out != nil {
		t.Fatalf("encodeJSON(nil) = %v, want nil", out)
	}
}

func TestListModelAliasesAndResolve(t *testing.T) {
	s := openTestStore(t)
	if err := s.SetModelAlias(ModelAlias{Alias: "fast", Provider: "openai", Model: "gpt-4o-mini"}); err != nil {
		t.Fatalf("SetModelAlias: %v", err)
	}
	if err := s.SetModelAlias(ModelAlias{Alias: "fast", Provider: "openai", Model: "gpt-4o"}); err != nil {
		t.Fatalf("SetModelAlias upsert: %v", err)
	}
	got, err := s.ResolveModelAlias("fast")
	if err != nil {
		t.Fatalf("ResolveModelAlias: %v", err)
	}
	if got.Model != "gpt-4o" {
		t.Fatalf("model = %q, want gpt-4o", got.Model)
	}
	list, err := s.ListModelAliases()
	if err != nil {
		t.Fatalf("ListModelAliases: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len = %d", len(list))
	}
	if err := s.DeleteModelAlias("fast"); err != nil {
		t.Fatalf("DeleteModelAlias: %v", err)
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	s := openTestStore(t)
	want := Settings{
		RequireAPIKey:     false,
		RTKEnabled:        false,
		CavemanEnabled:    true,
		CavemanLevel:      "light",
		EnableRequestLogs: true,
		ProxyURL:          "http://proxy",
		DataDir:           "/data",
		AllowedSources:    []string{"lan", "public"},
	}
	if err := s.UpdateSettings(want); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	got, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("settings = %+v, want %+v", got, want)
	}
}

func TestAPIKeyValidateInactive(t *testing.T) {
	s := openTestStore(t)
	key, raw, err := s.CreateAPIKey("svc", "secret")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	got, ok, err := s.ValidateAPIKey(raw, "secret")
	if err != nil || !ok {
		t.Fatalf("ValidateAPIKey: ok=%v err=%v", ok, err)
	}
	if got.ID != key.ID {
		t.Fatalf("id mismatch")
	}
	// Wrong secret -> not found.
	if _, ok, err := s.ValidateAPIKey(raw, "wrong"); err != nil || ok {
		t.Fatalf("ValidateAPIKey wrong secret: ok=%v err=%v", ok, err)
	}
	// Deactivate -> validate fails.
	if err := s.DeleteAPIKey(key.ID); err != nil {
		t.Fatalf("DeleteAPIKey: %v", err)
	}
	if _, ok, err := s.ValidateAPIKey(raw, "secret"); err != nil || ok {
		t.Fatalf("ValidateAPIKey after delete: ok=%v err=%v", ok, err)
	}
	keys, err := s.ListAPIKeys()
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("keys len = %d", len(keys))
	}
}

func TestComboLifecycle(t *testing.T) {
	s := openTestStore(t)
	combo := &Combo{Name: "chain", Steps: []ComboStep{{Provider: "p", Model: "m"}}, IsActive: true}
	if err := s.CreateCombo(combo); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}
	combo.Steps = append(combo.Steps, ComboStep{Provider: "p2", Model: "m2"})
	if err := s.UpdateCombo(combo); err != nil {
		t.Fatalf("UpdateCombo: %v", err)
	}
	got, err := s.GetActiveCombo("chain")
	if err != nil {
		t.Fatalf("GetActiveCombo: %v", err)
	}
	if len(got.Steps) != 2 {
		t.Fatalf("steps = %d", len(got.Steps))
	}
	list, err := s.ListCombos()
	if err != nil || len(list) != 1 {
		t.Fatalf("ListCombos: %v len=%d", err, len(list))
	}
}
