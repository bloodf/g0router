package usage

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

type fakeNameSource struct {
	conn     map[string]string
	provider map[string]string
	apiKey   map[string]string
}

func (f *fakeNameSource) ConnectionName(id string) string {
	if f.conn == nil {
		return ""
	}
	return f.conn[id]
}

func (f *fakeNameSource) ProviderName(id string) string {
	if f.provider == nil {
		return id
	}
	if n, ok := f.provider[id]; ok {
		return n
	}
	return id
}

func (f *fakeNameSource) APIKeyName(key string) string {
	if f.apiKey == nil {
		return ""
	}
	return f.apiKey[key]
}

type fakeUsageReader struct {
	daily []*store.UsageDailyRow
	logs  []*store.RequestLogEntry
}

func (f *fakeUsageReader) LoadDailyRange(maxDays int) ([]*store.UsageDailyRow, error) {
	return f.daily, nil
}

func (f *fakeUsageReader) RangeRequestLogs(sinceISO, untilISO string) ([]*store.RequestLogEntry, error) {
	var out []*store.RequestLogEntry
	for _, e := range f.logs {
		if e.Timestamp >= sinceISO && e.Timestamp <= untilISO {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeUsageReader) ListRecentRequestLogs(limit int) ([]*store.RequestLogEntry, error) {
	return f.logs, nil
}

func newTestStatsService(now time.Time) (*StatsService, *fakeUsageReader, *fakeNameSource) {
	reader := &fakeUsageReader{}
	names := &fakeNameSource{
		conn:     map[string]string{"conn-1": "Main Account"},
		provider: map[string]string{"openai": "OpenAI", "anthropic": "Anthropic"},
	}
	tracker := NewTracker(func() time.Time { return now }, nil, NewEvents())
	ring := NewRing(100)
	return NewStatsService(reader, names, tracker, ring, func() time.Time { return now }), reader, names
}

func TestLastUsedOverlay(t *testing.T) {
	now := time.Date(2026, 6, 12, 15, 0, 0, 0, time.UTC)
	svc, reader, _ := newTestStatsService(now)

	day := map[string]any{
		"requests":         1,
		"promptTokens":     10,
		"completionTokens": 5,
		"cost":             0.1,
		"byProvider":       map[string]any{"openai": map[string]any{"requests": 1, "promptTokens": 10, "completionTokens": 5, "cost": 0.1}},
		"byModel": map[string]any{
			"gpt-4o|openai": map[string]any{"requests": 1, "promptTokens": 10, "completionTokens": 5, "cost": 0.1, "rawModel": "gpt-4o", "provider": "openai"},
		},
		"byAccount": map[string]any{
			"conn-1": map[string]any{"requests": 1, "promptTokens": 10, "completionTokens": 5, "cost": 0.1, "rawModel": "gpt-4o", "provider": "openai"},
		},
		"byApiKey": map[string]any{
			"key-1|gpt-4o|openai": map[string]any{"requests": 1, "promptTokens": 10, "completionTokens": 5, "cost": 0.1, "rawModel": "gpt-4o", "provider": "openai", "apiKey": "key-1"},
		},
		"byEndpoint": map[string]any{
			"/chat/completions|gpt-4o|openai": map[string]any{"requests": 1, "promptTokens": 10, "completionTokens": 5, "cost": 0.1, "endpoint": "/chat/completions", "rawModel": "gpt-4o", "provider": "openai"},
		},
	}
	b, _ := json.Marshal(day)
	reader.daily = []*store.UsageDailyRow{{DateKey: "2026-06-12", Data: string(b)}}
	reader.logs = []*store.RequestLogEntry{
		{Timestamp: "2026-06-12T14:33:00Z", Provider: "openai", Model: "gpt-4o", ConnectionID: "conn-1", APIKey: "key-1", Endpoint: "/chat/completions", PromptTokens: 1, CompletionTokens: 1},
	}

	stats, err := svc.Stats("7d")
	if err != nil {
		t.Fatalf("Stats(7d): %v", err)
	}
	want := "2026-06-12T14:33:00Z"
	if stats.ByModel["gpt-4o (openai)"].LastUsed != want {
		t.Errorf("ByModel lastUsed = %q, want %q", stats.ByModel["gpt-4o (openai)"].LastUsed, want)
	}
	if stats.ByAccount["gpt-4o (openai - Main Account)"].LastUsed != want {
		t.Errorf("ByAccount lastUsed = %q, want %q", stats.ByAccount["gpt-4o (openai - Main Account)"].LastUsed, want)
	}
	if stats.ByAPIKey["key-1|gpt-4o|openai"].LastUsed != want {
		t.Errorf("ByAPIKey lastUsed = %q, want %q", stats.ByAPIKey["key-1|gpt-4o|openai"].LastUsed, want)
	}
	if stats.ByEndpoint["/chat/completions|gpt-4o|openai"].LastUsed != want {
		t.Errorf("ByEndpoint lastUsed = %q, want %q", stats.ByEndpoint["/chat/completions|gpt-4o|openai"].LastUsed, want)
	}
}

func TestUsageStatsDailyPath(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	svc, reader, _ := newTestStatsService(now)

	day11 := map[string]any{
		"requests":         1,
		"promptTokens":     10,
		"completionTokens": 5,
		"cost":             0.1,
		"byProvider":       map[string]any{"openai": map[string]any{"requests": 1, "promptTokens": 10, "completionTokens": 5, "cost": 0.1}},
		"byModel": map[string]any{
			"gpt-4o|openai": map[string]any{"requests": 1, "promptTokens": 10, "completionTokens": 5, "cost": 0.1, "rawModel": "gpt-4o", "provider": "openai"},
		},
		"byAccount": map[string]any{
			"conn-1": map[string]any{"requests": 1, "promptTokens": 10, "completionTokens": 5, "cost": 0.1, "rawModel": "gpt-4o", "provider": "openai"},
		},
		"byApiKey": map[string]any{
			"key-1|gpt-4o|openai": map[string]any{"requests": 1, "promptTokens": 10, "completionTokens": 5, "cost": 0.1, "rawModel": "gpt-4o", "provider": "openai", "apiKey": "key-1"},
		},
		"byEndpoint": map[string]any{
			"/chat/completions|gpt-4o|openai": map[string]any{"requests": 1, "promptTokens": 10, "completionTokens": 5, "cost": 0.1, "endpoint": "/chat/completions", "rawModel": "gpt-4o", "provider": "openai"},
		},
	}
	day12 := map[string]any{
		"requests":         2,
		"promptTokens":     20,
		"completionTokens": 10,
		"cost":             0.2,
		"byProvider":       map[string]any{"openai": map[string]any{"requests": 2, "promptTokens": 20, "completionTokens": 10, "cost": 0.2}},
		"byModel": map[string]any{
			"gpt-4o|openai": map[string]any{"requests": 2, "promptTokens": 20, "completionTokens": 10, "cost": 0.2, "rawModel": "gpt-4o", "provider": "openai"},
		},
		"byAccount": map[string]any{
			"conn-1": map[string]any{"requests": 2, "promptTokens": 20, "completionTokens": 10, "cost": 0.2, "rawModel": "gpt-4o", "provider": "openai"},
		},
		"byApiKey": map[string]any{
			"key-1|gpt-4o|openai": map[string]any{"requests": 2, "promptTokens": 20, "completionTokens": 10, "cost": 0.2, "rawModel": "gpt-4o", "provider": "openai", "apiKey": "key-1"},
		},
		"byEndpoint": map[string]any{
			"/chat/completions|gpt-4o|openai": map[string]any{"requests": 2, "promptTokens": 20, "completionTokens": 10, "cost": 0.2, "endpoint": "/chat/completions", "rawModel": "gpt-4o", "provider": "openai"},
		},
	}
	b11, _ := json.Marshal(day11)
	b12, _ := json.Marshal(day12)
	reader.daily = []*store.UsageDailyRow{
		{DateKey: "2026-06-11", Data: string(b11)},
		{DateKey: "2026-06-12", Data: string(b12)},
	}

	stats, err := svc.Stats("7d")
	if err != nil {
		t.Fatalf("Stats(7d): %v", err)
	}

	if stats.TotalRequests != 3 {
		t.Errorf("TotalRequests = %v, want 3", stats.TotalRequests)
	}
	if stats.TotalPromptTokens != 30 {
		t.Errorf("TotalPromptTokens = %v, want 30", stats.TotalPromptTokens)
	}
	if stats.TotalCompletionTokens != 15 {
		t.Errorf("TotalCompletionTokens = %v, want 15", stats.TotalCompletionTokens)
	}
	if stats.TotalCost != 0.30000000000000004 && stats.TotalCost != 0.3 {
		t.Errorf("TotalCost = %v, want 0.3", stats.TotalCost)
	}

	byProvider := stats.ByProvider
	if len(byProvider) != 1 {
		t.Fatalf("ByProvider len = %d, want 1", len(byProvider))
	}
	if byProvider["openai"].Requests != 3 {
		t.Errorf("ByProvider[openai].Requests = %v, want 3", byProvider["openai"].Requests)
	}

	byModel := stats.ByModel
	if len(byModel) != 1 {
		t.Fatalf("ByModel len = %d, want 1", len(byModel))
	}
	m := byModel["gpt-4o (openai)"]
	if m == nil {
		t.Fatal("ByModel['gpt-4o (openai)'] missing")
	}
	if m.Requests != 3 || m.RawModel != "gpt-4o" || m.Provider != "OpenAI" {
		t.Errorf("ByModel entry = %+v, want requests=3 rawModel=gpt-4o provider=OpenAI", m)
	}

	byAccount := stats.ByAccount
	if len(byAccount) != 1 {
		t.Fatalf("ByAccount len = %d, want 1", len(byAccount))
	}
	a := byAccount["gpt-4o (openai - Main Account)"]
	if a == nil {
		t.Fatal("ByAccount['gpt-4o (openai - Main Account)'] missing")
	}
	if a.Requests != 3 || a.AccountName != "Main Account" {
		t.Errorf("ByAccount entry = %+v", a)
	}

	byApiKey := stats.ByAPIKey
	if len(byApiKey) != 1 {
		t.Fatalf("ByAPIKey len = %d, want 1", len(byApiKey))
	}
	k := byApiKey["key-1|gpt-4o|openai"]
	if k == nil {
		t.Fatal("ByAPIKey['key-1|gpt-4o|openai'] missing")
	}
	if k.Requests != 3 || k.KeyName != "key-1..." {
		t.Errorf("ByAPIKey entry = %+v", k)
	}

	byEndpoint := stats.ByEndpoint
	if len(byEndpoint) != 1 {
		t.Fatalf("ByEndpoint len = %d, want 1", len(byEndpoint))
	}
	e := byEndpoint["/chat/completions|gpt-4o|openai"]
	if e == nil {
		t.Fatal("ByEndpoint['/chat/completions|gpt-4o|openai'] missing")
	}
	if e.Requests != 3 || e.Endpoint != "/chat/completions" {
		t.Errorf("ByEndpoint entry = %+v", e)
	}
}

func TestLast10MinuteBuckets(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	svc, reader, _ := newTestStatsService(now)

	reader.logs = []*store.RequestLogEntry{
		{Timestamp: "2026-06-12T11:59:00Z", Provider: "openai", Model: "gpt-4o", PromptTokens: 10, CompletionTokens: 5, Cost: 0.1},
		{Timestamp: "2026-06-12T11:51:00Z", Provider: "openai", Model: "gpt-4o", PromptTokens: 20, CompletionTokens: 10, Cost: 0.2},
		{Timestamp: "2026-06-12T11:49:00Z", Provider: "openai", Model: "gpt-4o", PromptTokens: 30, CompletionTokens: 15, Cost: 0.3},
	}

	stats, err := svc.Stats("today")
	if err != nil {
		t.Fatalf("Stats(today): %v", err)
	}
	if len(stats.Last10Minutes) != 10 {
		t.Fatalf("Last10Minutes len = %d, want 10", len(stats.Last10Minutes))
	}

	// Bucket 0 covers 11:51:00; bucket 8 covers 11:59:00; 11:49:00 is outside.
	if stats.Last10Minutes[0].Requests != 1 || stats.Last10Minutes[0].PromptTokens != 20 {
		t.Errorf("bucket 0 = %+v, want requests=1 promptTokens=20", stats.Last10Minutes[0])
	}
	if stats.Last10Minutes[8].Requests != 1 || stats.Last10Minutes[8].PromptTokens != 10 {
		t.Errorf("bucket 8 = %+v, want requests=1 promptTokens=10", stats.Last10Minutes[8])
	}
	if stats.Last10Minutes[9].Requests != 0 {
		t.Errorf("bucket 9 requests = %d, want 0", stats.Last10Minutes[9].Requests)
	}
}

func TestUsageStatsLivePath(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	svc, reader, _ := newTestStatsService(now)

	reader.logs = []*store.RequestLogEntry{
		{Timestamp: "2026-06-11T23:59:59Z", Provider: "openai", Model: "gpt-4o", PromptTokens: 1, CompletionTokens: 1, Cost: 0.01},
		{Timestamp: "2026-06-12T00:00:00Z", Provider: "openai", Model: "gpt-4o", PromptTokens: 2, CompletionTokens: 2, Cost: 0.02},
		{Timestamp: "2026-06-12T11:59:59Z", Provider: "openai", Model: "gpt-4o", PromptTokens: 3, CompletionTokens: 3, Cost: 0.03},
		{Timestamp: "2026-06-12T12:00:00Z", Provider: "openai", Model: "gpt-4o", PromptTokens: 4, CompletionTokens: 4, Cost: 0.04},
	}

	// today should include rows from start of day (00:00:00) up to now (12:00:00).
	stats, err := svc.Stats("today")
	if err != nil {
		t.Fatalf("Stats(today): %v", err)
	}
	if stats.TotalRequests != 3 {
		t.Errorf("today TotalRequests = %v, want 3", stats.TotalRequests)
	}

	// 24h should include rows from now-24h (12:00:00 previous day) up to now.
	stats, err = svc.Stats("24h")
	if err != nil {
		t.Fatalf("Stats(24h): %v", err)
	}
	if stats.TotalRequests != 4 {
		t.Errorf("24h TotalRequests = %v, want 4", stats.TotalRequests)
	}
}
