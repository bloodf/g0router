package usage

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

func newTestChartService(now time.Time) (*StatsService, *fakeUsageReader) {
	reader := &fakeUsageReader{}
	tracker := NewTracker(func() time.Time { return now }, nil, NewEvents())
	ring := NewRing(100)
	svc := NewStatsService(reader, &fakeNameSource{provider: map[string]string{"openai": "OpenAI"}}, tracker, ring, func() time.Time { return now })
	return svc, reader
}

func TestChartToday(t *testing.T) {
	now := time.Date(2026, 6, 12, 15, 30, 0, 0, time.UTC)
	svc, reader := newTestChartService(now)

	reader.logs = []*store.RequestLogEntry{
		{Timestamp: "2026-06-12T14:05:00Z", Provider: "openai", Model: "gpt-4o", PromptTokens: 10, CompletionTokens: 5, Cost: 0.1},
		{Timestamp: "2026-06-12T15:05:00Z", Provider: "openai", Model: "gpt-4o", PromptTokens: 20, CompletionTokens: 10, Cost: 0.2},
		{Timestamp: "2026-06-11T23:59:59Z", Provider: "openai", Model: "gpt-4o", PromptTokens: 99, CompletionTokens: 99, Cost: 0.99},
	}

	buckets, err := svc.Chart("today")
	if err != nil {
		t.Fatalf("Chart(today): %v", err)
	}
	if len(buckets) != 24 {
		t.Fatalf("len = %d, want 24", len(buckets))
	}

	if buckets[14].Tokens != 15 || buckets[14].Cost != 0.1 {
		t.Errorf("14:00 bucket = %+v, want tokens=15 cost=0.1", buckets[14])
	}
	if buckets[15].Tokens != 30 || buckets[15].Cost != 0.2 {
		t.Errorf("15:00 bucket = %+v, want tokens=30 cost=0.2", buckets[15])
	}
	if buckets[14].Label != "14:00" {
		t.Errorf("14:00 label = %q, want 14:00", buckets[14].Label)
	}
}

func TestChart24hClamp(t *testing.T) {
	now := time.Date(2026, 6, 12, 15, 30, 0, 0, time.UTC)
	svc, reader := newTestChartService(now)

	reader.logs = []*store.RequestLogEntry{
		{Timestamp: "2026-06-11T15:31:00Z", Provider: "openai", Model: "gpt-4o", PromptTokens: 10, CompletionTokens: 5, Cost: 0.1},
		{Timestamp: "2026-06-11T15:30:00Z", Provider: "openai", Model: "gpt-4o", PromptTokens: 20, CompletionTokens: 10, Cost: 0.2},
		{Timestamp: "2026-06-12T15:30:00Z", Provider: "openai", Model: "gpt-4o", PromptTokens: 30, CompletionTokens: 15, Cost: 0.3},
	}

	buckets, err := svc.Chart("24h")
	if err != nil {
		t.Fatalf("Chart(24h): %v", err)
	}
	if len(buckets) != 24 {
		t.Fatalf("len = %d, want 24", len(buckets))
	}

	// 15:30 now is exactly the last bucket boundary; reference clamps to last index.
	if buckets[23].Tokens != 30+15 || abs(buckets[23].Cost-0.3) > 1e-9 {
		t.Errorf("last bucket = %+v, want tokens=45 cost=0.3", buckets[23])
	}
	// 15:30 and 15:31 previous day both fall into bucket 0.
	if buckets[0].Tokens != 20+10+10+5 || abs(buckets[0].Cost-0.3) > 1e-9 {
		t.Errorf("first bucket = %+v, want tokens=45 cost=0.3", buckets[0])
	}
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func TestChartDailyZeroFill(t *testing.T) {
	now := time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC)
	svc, reader := newTestChartService(now)

	day := map[string]any{
		"requests":         1,
		"promptTokens":     10,
		"completionTokens": 5,
		"cost":             0.1,
		"byProvider":       map[string]any{},
		"byModel":          map[string]any{},
		"byAccount":        map[string]any{},
		"byApiKey":         map[string]any{},
		"byEndpoint":       map[string]any{},
	}
	b, _ := json.Marshal(day)
	reader.daily = []*store.UsageDailyRow{
		{DateKey: "2026-06-10", Data: string(b)},
	}

	buckets, err := svc.Chart("7d")
	if err != nil {
		t.Fatalf("Chart(7d): %v", err)
	}
	if len(buckets) != 7 {
		t.Fatalf("len = %d, want 7", len(buckets))
	}

	// Buckets run from 2026-06-06 to 2026-06-12; only 2026-06-10 has data.
	if buckets[4].Label != "Jun 10" {
		t.Errorf("label = %q, want Jun 10", buckets[4].Label)
	}
	if buckets[4].Tokens != 15 || buckets[4].Cost != 0.1 {
		t.Errorf("Jun 10 bucket = %+v, want tokens=15 cost=0.1", buckets[4])
	}
	if buckets[0].Tokens != 0 || buckets[6].Tokens != 0 {
		t.Errorf("zero-fill failed: bucket0=%+v bucket6=%+v", buckets[0], buckets[6])
	}
}
