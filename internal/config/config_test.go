package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"PORT",
		"DATA_DIR",
		"JWT_SECRET",
		"API_KEY_SECRET",
		"REQUIRE_API_KEY",
		"BIND_ADDRESS",
		"ENABLE_REQUEST_LOGS",
		"RTK_ENABLED",
		"CAVEMAN_ENABLED",
		"CAVEMAN_LEVEL",
	} {
		t.Setenv(key, "")
	}
}

func TestLoadDefaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("API_KEY_SECRET", "test-api-key-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Port != 20128 {
		t.Errorf("Port = %d, want 20128", cfg.Port)
	}
	if cfg.DataDir != filepath.Join(os.Getenv("HOME"), ".g0router") {
		t.Errorf("DataDir = %q, want home .g0router", cfg.DataDir)
	}
	if cfg.BindAddress != "127.0.0.1" {
		t.Errorf("BindAddress = %q, want 127.0.0.1", cfg.BindAddress)
	}
	if cfg.RequireAPIKey != true {
		t.Error("RequireAPIKey should default true")
	}
	if cfg.EnableRequestLogs != false {
		t.Error("EnableRequestLogs should default false")
	}
	if cfg.RTKEnabled != true {
		t.Error("RTKEnabled should default true")
	}
	if cfg.CavemanEnabled != false {
		t.Error("CavemanEnabled should default false")
	}
	if cfg.CavemanLevel != "full" {
		t.Errorf("CavemanLevel = %q, want full", cfg.CavemanLevel)
	}
}

func TestLoadFromEnv(t *testing.T) {
	clearEnv(t)
	dir := filepath.Join(t.TempDir(), "data")
	t.Setenv("PORT", "8080")
	t.Setenv("DATA_DIR", dir)
	t.Setenv("BIND_ADDRESS", "0.0.0.0")
	t.Setenv("JWT_SECRET", "jwt-secret")
	t.Setenv("API_KEY_SECRET", "api-key-secret")
	t.Setenv("REQUIRE_API_KEY", "false")
	t.Setenv("ENABLE_REQUEST_LOGS", "true")
	t.Setenv("RTK_ENABLED", "false")
	t.Setenv("CAVEMAN_ENABLED", "true")
	t.Setenv("CAVEMAN_LEVEL", "ultra")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.DataDir != dir {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, dir)
	}
	if cfg.BindAddress != "0.0.0.0" {
		t.Errorf("BindAddress = %q, want 0.0.0.0", cfg.BindAddress)
	}
	if cfg.JWTSecret != "jwt-secret" {
		t.Errorf("JWTSecret = %q", cfg.JWTSecret)
	}
	if cfg.APIKeySecret != "api-key-secret" {
		t.Errorf("APIKeySecret = %q", cfg.APIKeySecret)
	}
	if cfg.RequireAPIKey {
		t.Error("RequireAPIKey should be false")
	}
	if !cfg.EnableRequestLogs {
		t.Error("EnableRequestLogs should be true")
	}
	if cfg.RTKEnabled {
		t.Error("RTKEnabled should be false")
	}
	if !cfg.CavemanEnabled {
		t.Error("CavemanEnabled should be true")
	}
	if cfg.CavemanLevel != "ultra" {
		t.Errorf("CavemanLevel = %q, want ultra", cfg.CavemanLevel)
	}
}

func TestLoadValidatesAPIKeySecret(t *testing.T) {
	clearEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("REQUIRE_API_KEY", "true")

	_, err := Load()
	if err == nil {
		t.Fatal("Load should fail")
	}
	if !strings.Contains(err.Error(), "API_KEY_SECRET required when REQUIRE_API_KEY=true") {
		t.Fatalf("error = %q", err)
	}
}

func TestLoadRequireAPIKeyFalse(t *testing.T) {
	clearEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("REQUIRE_API_KEY", "false")

	if _, err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
}

func TestLoadInvalidPort(t *testing.T) {
	clearEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PORT", "99999")
	t.Setenv("API_KEY_SECRET", "test-api-key-secret")

	_, err := Load()
	if err == nil {
		t.Fatal("Load should fail")
	}
	if !strings.Contains(err.Error(), "port must be 1-65535") {
		t.Fatalf("error = %q", err)
	}
}

func TestLoadInvalidBindAddress(t *testing.T) {
	clearEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("API_KEY_SECRET", "test-api-key-secret")
	t.Setenv("BIND_ADDRESS", "not an ip")

	_, err := Load()
	if err == nil {
		t.Fatal("Load should fail")
	}
	if !strings.Contains(err.Error(), "BIND_ADDRESS must be an IP address") {
		t.Fatalf("error = %q", err)
	}
}

func TestLoadInvalidCavemanLevel(t *testing.T) {
	clearEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("API_KEY_SECRET", "test-api-key-secret")
	t.Setenv("CAVEMAN_LEVEL", "mega")

	_, err := Load()
	if err == nil {
		t.Fatal("Load should fail")
	}
	if !strings.Contains(err.Error(), "caveman level must be lite, full, or ultra") {
		t.Fatalf("error = %q", err)
	}
}

func TestLoadBooleanParsing(t *testing.T) {
	clearEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("API_KEY_SECRET", "test-api-key-secret")
	t.Setenv("REQUIRE_API_KEY", "yes")
	t.Setenv("ENABLE_REQUEST_LOGS", "1")
	t.Setenv("RTK_ENABLED", "TRUE")
	t.Setenv("CAVEMAN_ENABLED", "yes")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !cfg.RequireAPIKey {
		t.Error("RequireAPIKey should parse yes")
	}
	if !cfg.EnableRequestLogs {
		t.Error("EnableRequestLogs should parse 1")
	}
	if !cfg.RTKEnabled {
		t.Error("RTKEnabled should parse TRUE")
	}
	if !cfg.CavemanEnabled {
		t.Error("CavemanEnabled should parse yes")
	}
}

func TestLoadInvalidBoolean(t *testing.T) {
	clearEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("API_KEY_SECRET", "test-api-key-secret")
	t.Setenv("RTK_ENABLED", "maybe")

	_, err := Load()
	if err == nil {
		t.Fatal("Load should fail")
	}
	if !strings.Contains(err.Error(), "RTK_ENABLED") {
		t.Fatalf("error = %q", err)
	}
}

func TestLoadHomeDirExpansion(t *testing.T) {
	clearEnv(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("API_KEY_SECRET", "test-api-key-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !strings.HasPrefix(cfg.DataDir, home) {
		t.Errorf("DataDir = %q, want prefix %q", cfg.DataDir, home)
	}
}

func TestLoadCreatesDataDir(t *testing.T) {
	clearEnv(t)
	dir := filepath.Join(t.TempDir(), "missing")
	t.Setenv("DATA_DIR", dir)
	t.Setenv("API_KEY_SECRET", "test-api-key-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	info, err := os.Stat(cfg.DataDir)
	if err != nil {
		t.Fatalf("stat data dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("data dir is not a directory")
	}
}

func TestLoadValidatesDataDirWritable(t *testing.T) {
	clearEnv(t)
	path := filepath.Join(t.TempDir(), "data-file")
	if err := os.WriteFile(path, []byte("not a dir"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	t.Setenv("DATA_DIR", path)
	t.Setenv("API_KEY_SECRET", "test-api-key-secret")

	_, err := Load()
	if err == nil {
		t.Fatal("Load should fail")
	}
	if !strings.Contains(err.Error(), "data dir not writable") {
		t.Fatalf("error = %q", err)
	}
}
