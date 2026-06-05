package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadInvalidPortInt(t *testing.T) {
	clearEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PORT", "not-a-number")
	t.Setenv("API_KEY_SECRET", "secret")

	_, err := Load()
	if err == nil {
		t.Fatal("Load should fail with non-integer PORT")
	}
	if !strings.Contains(err.Error(), "PORT must be an integer") {
		t.Fatalf("error = %q", err)
	}
}

func TestLoadInvalidRequireAPIKey(t *testing.T) {
	clearEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("REQUIRE_API_KEY", "maybe")
	t.Setenv("API_KEY_SECRET", "secret")

	_, err := Load()
	if err == nil {
		t.Fatal("Load should fail with invalid REQUIRE_API_KEY")
	}
	if !strings.Contains(err.Error(), "REQUIRE_API_KEY") {
		t.Fatalf("error = %q", err)
	}
}

func TestLoadInvalidEnableRequestLogs(t *testing.T) {
	clearEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("API_KEY_SECRET", "secret")
	t.Setenv("REQUIRE_API_KEY", "false")
	t.Setenv("ENABLE_REQUEST_LOGS", "maybe")

	_, err := Load()
	if err == nil {
		t.Fatal("Load should fail with invalid ENABLE_REQUEST_LOGS")
	}
	if !strings.Contains(err.Error(), "ENABLE_REQUEST_LOGS") {
		t.Fatalf("error = %q", err)
	}
}

func TestLoadInvalidCavemanEnabled(t *testing.T) {
	clearEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("API_KEY_SECRET", "secret")
	t.Setenv("REQUIRE_API_KEY", "false")
	t.Setenv("CAVEMAN_ENABLED", "maybe")

	_, err := Load()
	if err == nil {
		t.Fatal("Load should fail with invalid CAVEMAN_ENABLED")
	}
	if !strings.Contains(err.Error(), "CAVEMAN_ENABLED") {
		t.Fatalf("error = %q", err)
	}
}

func TestExpandDataDirTildeOnly(t *testing.T) {
	got, err := expandDataDir("~")
	if err != nil {
		t.Fatalf("expandDataDir ~: %v", err)
	}
	home, _ := os.UserHomeDir()
	if got != home {
		t.Fatalf("expandDataDir ~ = %q, want %q", got, home)
	}
}

func TestExpandDataDirAbsolutePath(t *testing.T) {
	got, err := expandDataDir("/tmp/mydir")
	if err != nil {
		t.Fatalf("expandDataDir absolute: %v", err)
	}
	if got != "/tmp/mydir" {
		t.Fatalf("expandDataDir absolute = %q, want /tmp/mydir", got)
	}
}

func TestLoadDataDirExpandError(t *testing.T) {
	// Can't easily trigger os.UserHomeDir error in tests, but we can cover
	// the ensureWritableDir error path by using a read-only parent dir.
	clearEnv(t)
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })

	// Use a subdir of the read-only parent as DATA_DIR
	t.Setenv("DATA_DIR", parent+"/subdir")
	t.Setenv("API_KEY_SECRET", "secret")
	t.Setenv("REQUIRE_API_KEY", "false")

	_, err := Load()
	if err == nil {
		t.Skip("running as root or filesystem allows writes; skipping read-only dir test")
	}
	if !strings.Contains(err.Error(), "data dir not writable") {
		t.Fatalf("error = %q", err)
	}
}

func TestEnvBoolFalseVariants(t *testing.T) {
	for _, val := range []string{"false", "0", "no", "FALSE", "No"} {
		t.Run(val, func(t *testing.T) {
			t.Setenv("TEST_BOOL_VAR", val)
			got, err := envBool("TEST_BOOL_VAR", true)
			if err != nil {
				t.Fatalf("envBool %q: %v", val, err)
			}
			if got {
				t.Fatalf("envBool %q = true, want false", val)
			}
		})
	}
}

func TestEnvStringReturnsDefault(t *testing.T) {
	t.Setenv("MY_ABSENT_KEY", "")
	got := envString("MY_ABSENT_KEY", "default-value")
	if got != "default-value" {
		t.Fatalf("envString default = %q, want default-value", got)
	}
}

func TestEnvStringReturnsValue(t *testing.T) {
	t.Setenv("MY_PRESENT_KEY", "set-value")
	got := envString("MY_PRESENT_KEY", "default-value")
	if got != "set-value" {
		t.Fatalf("envString = %q, want set-value", got)
	}
}
