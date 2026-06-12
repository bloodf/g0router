package usage

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

var errFake = errors.New("fake settings error")

type fakeSettingsReader struct {
	values map[string]string
	err    error
}

func (f *fakeSettingsReader) GetSettings() (map[string]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.values, nil
}

func TestObservabilityConfig(t *testing.T) {
	now := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }

	tests := []struct {
		name     string
		env      map[string]string
		settings map[string]string
		want     ObsConfig
	}{
		{
			name: "defaults",
			want: ObsConfig{
				Enabled:         true,
				MaxRecords:      200,
				BatchSize:       20,
				FlushIntervalMs: 5000,
				MaxJSONSize:     5 * 1024,
			},
		},
		{
			name: "settings override",
			settings: map[string]string{
				"enableObservability":          "false",
				"observabilityMaxRecords":      "50",
				"observabilityBatchSize":       "10",
				"observabilityFlushIntervalMs": "1000",
				"observabilityMaxJsonSize":     "10",
			},
			want: ObsConfig{
				Enabled:         false,
				MaxRecords:      50,
				BatchSize:       10,
				FlushIntervalMs: 1000,
				MaxJSONSize:     10 * 1024,
			},
		},
		{
			name: "env override",
			env: map[string]string{
				"OBSERVABILITY_ENABLED":           "false",
				"OBSERVABILITY_MAX_RECORDS":       "300",
				"OBSERVABILITY_BATCH_SIZE":        "30",
				"OBSERVABILITY_FLUSH_INTERVAL_MS": "2000",
				"OBSERVABILITY_MAX_JSON_SIZE":     "10",
			},
			want: ObsConfig{
				Enabled:         false,
				MaxRecords:      300,
				BatchSize:       30,
				FlushIntervalMs: 2000,
				MaxJSONSize:     10 * 1024,
			},
		},
		{
			name: "settings precedence over env",
			settings: map[string]string{
				"enableObservability":     "true",
				"observabilityMaxRecords": "100",
			},
			env: map[string]string{
				"OBSERVABILITY_ENABLED":     "false",
				"OBSERVABILITY_MAX_RECORDS": "400",
			},
			want: ObsConfig{
				Enabled:         true,
				MaxRecords:      100,
				BatchSize:       20,
				FlushIntervalMs: 5000,
				MaxJSONSize:     5 * 1024,
			},
		},
		{
			name: "enabled falls back to env when settings missing",
			env: map[string]string{
				"OBSERVABILITY_ENABLED": "false",
			},
			want: ObsConfig{
				Enabled:         false,
				MaxRecords:      200,
				BatchSize:       20,
				FlushIntervalMs: 5000,
				MaxJSONSize:     5 * 1024,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(key string) string { return tt.env[key] }
			loader := NewObsConfigLoader(&fakeSettingsReader{values: tt.settings}, getenv, clock)
			got := loader.Load()
			if got != tt.want {
				t.Errorf("Load() = %+v, want %+v", got, tt.want)
			}
		})
	}

	t.Run("cache 5s", func(t *testing.T) {
		settings := &fakeSettingsReader{values: map[string]string{
			"observabilityMaxRecords": "42",
		}}
		getenv := func(string) string { return "" }
		loader := NewObsConfigLoader(settings, getenv, clock)

		if got := loader.Load(); got.MaxRecords != 42 {
			t.Fatalf("first load MaxRecords = %d, want 42", got.MaxRecords)
		}

		settings.values = map[string]string{
			"observabilityMaxRecords": "99",
		}
		if got := loader.Load(); got.MaxRecords != 42 {
			t.Errorf("cached load MaxRecords = %d, want 42", got.MaxRecords)
		}

		now = now.Add(5 * time.Second)
		if got := loader.Load(); got.MaxRecords != 99 {
			t.Errorf("expired cache MaxRecords = %d, want 99", got.MaxRecords)
		}
	})

	t.Run("load error returns disabled defaults", func(t *testing.T) {
		settings := &fakeSettingsReader{err: errFake}
		getenv := func(string) string { return "" }
		loader := NewObsConfigLoader(settings, getenv, clock)
		got := loader.Load()
		want := ObsConfig{
			Enabled:         false,
			MaxRecords:      200,
			BatchSize:       20,
			FlushIntervalMs: 5000,
			MaxJSONSize:     5 * 1024,
		}
		if got != want {
			t.Errorf("Load() on error = %+v, want %+v", got, want)
		}
	})
}

func TestSanitizeHeaders(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]string
		want map[string]string
	}{
		{
			name: "sensitive keys removed",
			in: map[string]string{
				"Authorization": "Bearer secret",
				"X-Api-Key":     "key",
				"Cookie":        "session=abc",
				"X-Auth-Token":  "token",
				"api-key":       "key2",
				"Content-Type":  "application/json",
			},
			want: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name: "substring match removes x-csrf-token",
			in: map[string]string{
				"X-CSRF-Token": "csrf",
				"Accept":       "*/*",
			},
			want: map[string]string{
				"Accept": "*/*",
			},
		},
		{
			name: "nil returns empty non-nil map",
			in:   nil,
			want: map[string]string{},
		},
		{
			name: "empty map stays empty",
			in:   map[string]string{},
			want: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeHeaders(tt.in)
			if len(got) != len(tt.want) {
				t.Errorf("len = %d, want %d; got=%v", len(got), len(tt.want), got)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("%q = %q, want %q", k, got[k], v)
				}
			}
			if tt.in != nil && reflect.ValueOf(got).Pointer() == reflect.ValueOf(tt.in).Pointer() {
				t.Error("SanitizeHeaders returned the same map, want a copy")
			}
		})
	}
}

func TestTruncateField(t *testing.T) {
	small := map[string]any{"key": "value"}
	got := TruncateField(small, 1024)
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("small field type = %T, want map[string]any", got)
	}
	if m["key"] != "value" {
		t.Errorf("small field = %v, want unchanged", got)
	}

	large := map[string]any{"data": strings.Repeat("a", 6000)}
	got = TruncateField(large, 1024)
	marker, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("large field type = %T, want map[string]any", got)
	}
	if marker["_truncated"] != true {
		t.Errorf("_truncated = %v, want true", marker["_truncated"])
	}
	if marker["_originalSize"] == nil {
		t.Error("_originalSize missing")
	}
	preview, ok := marker["_preview"].(string)
	if !ok {
		t.Fatalf("_preview type = %T, want string", marker["_preview"])
	}
	if len(preview) != 200 {
		t.Errorf("preview len = %d, want 200", len(preview))
	}
}

func TestTruncateFieldShortOversize(t *testing.T) {
	value := map[string]any{"data": strings.Repeat("a", 30)}
	got := TruncateField(value, 10)

	marker, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("type = %T, want map[string]any", got)
	}
	if marker["_truncated"] != true {
		t.Errorf("_truncated = %v, want true", marker["_truncated"])
	}
	originalJSON, _ := json.Marshal(value)
	if marker["_originalSize"] != len(originalJSON) {
		t.Errorf("_originalSize = %v, want %d", marker["_originalSize"], len(originalJSON))
	}
	preview, ok := marker["_preview"].(string)
	if !ok {
		t.Fatalf("_preview type = %T, want string", marker["_preview"])
	}
	if preview != string(originalJSON) {
		t.Errorf("preview = %q, want full JSON %q", preview, originalJSON)
	}
}

func TestTruncateFieldNoHTMLEscape(t *testing.T) {
	// value whose marshaled JSON contains <, >, &. The reference uses
	// JSON.stringify which does not escape these; Go's json.Marshal
	// defaults to HTML-escaping, expanding "<" to "\u003c", etc. The
	// blob-size threshold decision must agree with the reference, so
	// TruncateField must marshal blob fields without HTML-escaping.
	value := map[string]any{"text": "<b>&"}
	got := TruncateField(value, 5)

	marker, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("type = %T, want map[string]any", got)
	}
	if marker["_truncated"] != true {
		t.Errorf("_truncated = %v, want true", marker["_truncated"])
	}
	// JSON.stringify({"text":"<b>&"}) === '{"text":"<b>&"}' (15 bytes).
	wantSize := len(`{"text":"<b>&"}`)
	if marker["_originalSize"] != wantSize {
		t.Errorf("_originalSize = %v, want %d (JS byte length)", marker["_originalSize"], wantSize)
	}
	preview, ok := marker["_preview"].(string)
	if !ok {
		t.Fatalf("_preview type = %T, want string", marker["_preview"])
	}
	if !strings.Contains(preview, "<b>&") {
		t.Errorf("preview = %q, want to contain literal \"<b>&\"", preview)
	}
}

func TestTruncateFieldUTF8Preview(t *testing.T) {
	// Construct a value whose marshaled JSON contains multibyte runes
	// (e.g. "é") straddling the 200-byte preview boundary. The JSON
	// encoding of "é" is two raw UTF-8 bytes (0xc3 0xa9); with 200
	// copies the data section spans bytes 9..408, so a 200-byte slice
	// will cut "é" #96 in the middle. Byte-based truncation produces
	// invalid UTF-8; rune-based truncation must yield a valid, <=200
	// rune preview.
	value := map[string]any{"data": strings.Repeat("é", 200)}
	got := TruncateField(value, 250)

	marker, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("type = %T, want map[string]any", got)
	}
	if marker["_truncated"] != true {
		t.Errorf("_truncated = %v, want true", marker["_truncated"])
	}
	preview, ok := marker["_preview"].(string)
	if !ok {
		t.Fatalf("_preview type = %T, want string", marker["_preview"])
	}
	if !utf8.ValidString(preview) {
		t.Errorf("preview is not valid UTF-8: bytes=%x", []byte(preview))
	}
	if n := len([]rune(preview)); n > 200 {
		t.Errorf("preview rune count = %d, want <= 200", n)
	}
}
