package logging

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
)

type fakeRequestLogStore struct {
	entries []store.RequestLogEntry
	err     error
}

func (f *fakeRequestLogStore) LogRequest(entry *store.RequestLogEntry) error {
	if f.err != nil {
		return f.err
	}
	f.entries = append(f.entries, *entry)
	return nil
}

func TestLoggerStoresRequestResponseMetadata(t *testing.T) {
	sink := &fakeRequestLogStore{}
	logger := NewLogger(sink)
	now := time.Date(2026, 6, 2, 18, 0, 0, 0, time.UTC)
	cost := 0.00725

	err := logger.Log(RequestLog{
		RequestID:    "req-1",
		Timestamp:    now,
		Provider:     "openai",
		Model:        "gpt-4o",
		ConnectionID: ptr("conn-1"),
		AuthType:     "api_key",
		Usage: &usage.Usage{
			InputTokens:     1000,
			OutputTokens:    500,
			TotalTokens:     1500,
			CacheReadTokens: 200,
		},
		CostUSD:        &cost,
		Latency:        250 * time.Millisecond,
		StatusCode:     200,
		SourceFormat:   ptr("openai"),
		TargetFormat:   ptr("anthropic"),
		RTKEnabled:     boolPtr(true),
		RTKBytesSaved:  intPtr(128),
		CavemanEnabled: boolPtr(false),
		ComboName:      ptr("primary"),
		APIKeyID:       ptr("key-1"),
		ClientTool:     ptr("codex"),
	})
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(sink.entries) != 1 {
		t.Fatalf("entries len = %d, want 1", len(sink.entries))
	}

	got := sink.entries[0]
	if got.RequestID != "req-1" || !got.Timestamp.Equal(now) || got.Provider != "openai" || got.Model != "gpt-4o" {
		t.Fatalf("entry identity mismatch: %+v", got)
	}
	if got.ConnectionID == nil || *got.ConnectionID != "conn-1" {
		t.Fatalf("connection id = %v, want conn-1", got.ConnectionID)
	}
	if got.AuthType != "api_key" {
		t.Fatalf("auth type = %q, want api_key", got.AuthType)
	}
	if got.InputTokens == nil || *got.InputTokens != 1000 {
		t.Fatalf("input tokens = %v, want 1000", got.InputTokens)
	}
	if got.OutputTokens == nil || *got.OutputTokens != 500 {
		t.Fatalf("output tokens = %v, want 500", got.OutputTokens)
	}
	if got.CacheReadTokens == nil || *got.CacheReadTokens != 200 {
		t.Fatalf("cache read tokens = %v, want 200", got.CacheReadTokens)
	}
	if got.TotalTokens == nil || *got.TotalTokens != 1500 {
		t.Fatalf("total tokens = %v, want 1500", got.TotalTokens)
	}
	if got.CostUSD == nil || *got.CostUSD != cost {
		t.Fatalf("cost = %v, want %f", got.CostUSD, cost)
	}
	if got.LatencyMS == nil || *got.LatencyMS != 250 {
		t.Fatalf("latency ms = %v, want 250", got.LatencyMS)
	}
	if got.StatusCode == nil || *got.StatusCode != 200 {
		t.Fatalf("status code = %v, want 200", got.StatusCode)
	}
	if got.RTKEnabled == nil || *got.RTKEnabled != true {
		t.Fatalf("rtk enabled = %v, want true", got.RTKEnabled)
	}
	if got.RTKBytesSaved == nil || *got.RTKBytesSaved != 128 {
		t.Fatalf("rtk bytes saved = %v, want 128", got.RTKBytesSaved)
	}
	if got.CavemanEnabled == nil || *got.CavemanEnabled != false {
		t.Fatalf("caveman enabled = %v, want false", got.CavemanEnabled)
	}
	if got.ClientTool == nil || *got.ClientTool != "codex" {
		t.Fatalf("client tool = %v, want codex", got.ClientTool)
	}
}

func TestLoggerStoresNilOptionalMetadata(t *testing.T) {
	sink := &fakeRequestLogStore{}
	logger := NewLogger(sink)

	err := logger.Log(RequestLog{
		RequestID: "req-2",
		Provider:  "anthropic",
		Model:     "claude-sonnet-4",
		AuthType:  "oauth",
	})
	if err != nil {
		t.Fatalf("Log: %v", err)
	}

	got := sink.entries[0]
	if got.InputTokens != nil || got.CostUSD != nil || got.LatencyMS != nil || got.StatusCode != nil {
		t.Fatalf("optional fields should stay nil: %+v", got)
	}
}

func TestLoggerWrapsStoreError(t *testing.T) {
	storeErr := errors.New("database unavailable")
	logger := NewLogger(&fakeRequestLogStore{err: storeErr})

	err := logger.Log(RequestLog{RequestID: "req-3", Provider: "openai", Model: "gpt-4o", AuthType: "api_key"})
	if !errors.Is(err, storeErr) {
		t.Fatalf("expected wrapped store error, got %v", err)
	}
	if !strings.Contains(err.Error(), "log request") {
		t.Fatalf("error = %q, want context", err)
	}
}

func ptr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}
