package usage

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultMaxRecords      = 200
	defaultBatchSize       = 20
	defaultFlushIntervalMs = 5000
	defaultMaxJSONSizeKB   = 5
	configCacheTTL         = 5 * time.Second
)

// SettingsReader reads the full settings map. The store satisfies this.
type SettingsReader interface {
	GetSettings() (map[string]string, error)
}

// ObsConfig holds the observability writer configuration.
type ObsConfig struct {
	Enabled         bool
	MaxRecords      int
	BatchSize       int
	FlushIntervalMs int
	MaxJSONSize     int
}

// ObsConfigLoader loads and caches observability config from settings and env.
type ObsConfigLoader struct {
	settings SettingsReader
	getenv   func(string) string
	clock    func() time.Time

	mu        sync.Mutex
	cached    ObsConfig
	cachedAt  time.Time
	cacheHits bool
}

// NewObsConfigLoader creates a loader with injected dependencies.
func NewObsConfigLoader(settings SettingsReader, getenv func(string) string, clock func() time.Time) *ObsConfigLoader {
	return &ObsConfigLoader{
		settings: settings,
		getenv:   getenv,
		clock:    clock,
	}
}

// Load returns the current observability config, caching it for 5 seconds.
func (l *ObsConfigLoader) Load() ObsConfig {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.clock()
	if l.cacheHits && now.Sub(l.cachedAt) < configCacheTTL {
		return l.cached
	}

	cfg, err := l.loadUnlocked()
	if err != nil {
		cfg = ObsConfig{
			Enabled:         false,
			MaxRecords:      defaultMaxRecords,
			BatchSize:       defaultBatchSize,
			FlushIntervalMs: defaultFlushIntervalMs,
			MaxJSONSize:     defaultMaxJSONSizeKB * 1024,
		}
	}

	l.cached = cfg
	l.cachedAt = now
	l.cacheHits = true
	return cfg
}

func (l *ObsConfigLoader) loadUnlocked() (ObsConfig, error) {
	settings, err := l.settings.GetSettings()
	if err != nil {
		return ObsConfig{}, fmt.Errorf("read settings: %w", err)
	}

	enabled := l.getenv("OBSERVABILITY_ENABLED") != "false"
	if v, ok := settings["enableObservability"]; ok {
		b, err := strconv.ParseBool(v)
		if err == nil {
			enabled = b
		}
	}

	maxRecords := intOrDefault(settings["observabilityMaxRecords"], l.getenv("OBSERVABILITY_MAX_RECORDS"), defaultMaxRecords)
	batchSize := intOrDefault(settings["observabilityBatchSize"], l.getenv("OBSERVABILITY_BATCH_SIZE"), defaultBatchSize)
	flushIntervalMs := intOrDefault(settings["observabilityFlushIntervalMs"], l.getenv("OBSERVABILITY_FLUSH_INTERVAL_MS"), defaultFlushIntervalMs)
	maxJSONSizeKB := intOrDefault(settings["observabilityMaxJsonSize"], l.getenv("OBSERVABILITY_MAX_JSON_SIZE"), defaultMaxJSONSizeKB)

	return ObsConfig{
		Enabled:         enabled,
		MaxRecords:      maxRecords,
		BatchSize:       batchSize,
		FlushIntervalMs: flushIntervalMs,
		MaxJSONSize:     maxJSONSizeKB * 1024,
	}, nil
}

func intOrDefault(settingsValue, envValue string, defaultValue int) int {
	if settingsValue != "" {
		if n, err := strconv.Atoi(settingsValue); err == nil {
			return n
		}
	}
	if envValue != "" {
		if n, err := strconv.Atoi(envValue); err == nil {
			return n
		}
	}
	return defaultValue
}

var sensitiveHeaderKeys = []string{"authorization", "x-api-key", "cookie", "token", "api-key"}

// SanitizeHeaders returns a copy of headers with sensitive keys removed.
// Matching is case-insensitive and by substring, matching the reference.
func SanitizeHeaders(headers map[string]string) map[string]string {
	out := make(map[string]string)
	if headers == nil {
		return out
	}
	for k, v := range headers {
		lower := strings.ToLower(k)
		sensitive := false
		for _, s := range sensitiveHeaderKeys {
			if strings.Contains(lower, s) {
				sensitive = true
				break
			}
		}
		if !sensitive {
			out[k] = v
		}
	}
	return out
}

// TruncateField replaces oversized JSON values with a truncation marker.
// Nil input is treated as an empty object, matching the reference.
func TruncateField(v any, maxSize int) any {
	if v == nil {
		v = map[string]any{}
	}
	str, err := json.Marshal(v)
	if err != nil {
		str = []byte("null")
	}
	if len(str) <= maxSize {
		return v
	}
	return map[string]any{
		"_truncated":    true,
		"_originalSize": len(str),
		"_preview":      string(str[:200]),
	}
}
