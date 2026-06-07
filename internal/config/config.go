package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Port              int
	DataDir           string
	BindAddress       string
	JWTSecret         string
	APIKeySecret      string
	RequireAPIKey     bool
	EnableRequestLogs bool
	RTKEnabled        bool
	CavemanEnabled    bool
	CavemanLevel      string
}

func Load() (*Config, error) {
	port, err := envInt("PORT", 20128)
	if err != nil {
		return nil, err
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("port must be 1-65535")
	}

	dataDir, err := expandDataDir(envString("DATA_DIR", "~/.g0router"))
	if err != nil {
		return nil, err
	}
	if err := ensureWritableDir(dataDir); err != nil {
		return nil, err
	}

	bindAddress := envString("BIND_ADDRESS", "127.0.0.1")
	if net.ParseIP(bindAddress) == nil {
		return nil, fmt.Errorf("BIND_ADDRESS must be an IP address")
	}

	requireAPIKey, err := envBool("REQUIRE_API_KEY", true)
	if err != nil {
		return nil, err
	}
	enableRequestLogs, err := envBool("ENABLE_REQUEST_LOGS", false)
	if err != nil {
		return nil, err
	}
	rtkEnabled, err := envBool("RTK_ENABLED", true)
	if err != nil {
		return nil, err
	}
	cavemanEnabled, err := envBool("CAVEMAN_ENABLED", false)
	if err != nil {
		return nil, err
	}

	cavemanLevel := envString("CAVEMAN_LEVEL", "full")
	if cavemanLevel != "lite" && cavemanLevel != "full" && cavemanLevel != "ultra" {
		return nil, fmt.Errorf("caveman level must be lite, full, or ultra")
	}

	apiKeySecret := os.Getenv("API_KEY_SECRET")

	return &Config{
		Port:              port,
		DataDir:           dataDir,
		BindAddress:       bindAddress,
		JWTSecret:         os.Getenv("JWT_SECRET"),
		APIKeySecret:      apiKeySecret,
		RequireAPIKey:     requireAPIKey,
		EnableRequestLogs: enableRequestLogs,
		RTKEnabled:        rtkEnabled,
		CavemanEnabled:    cavemanEnabled,
		CavemanLevel:      cavemanLevel,
	}, nil
}

func envString(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func envInt(key string, defaultValue int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return parsed, nil
}

func envBool(key string, defaultValue bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}
	switch strings.ToLower(value) {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be a boolean", key)
	}
}

func expandDataDir(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("find home dir: %w", err)
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

func ensureWritableDir(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("data dir not writable: %s: %w", path, err)
	}

	file, err := os.CreateTemp(path, ".writable-")
	if err != nil {
		return fmt.Errorf("data dir not writable: %s: %w", path, err)
	}
	name := file.Name()
	if err := file.Close(); err != nil {
		return fmt.Errorf("data dir not writable: %s: %w", path, err)
	}
	if err := os.Remove(name); err != nil {
		return fmt.Errorf("data dir not writable: %s: %w", path, err)
	}
	return nil
}
