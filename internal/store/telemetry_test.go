package store

import (
	"testing"
	"time"
)

func TestProviderModelStatsAggregates(t *testing.T) {
	s := openTestStore(t)

	now := time.Now().UTC()
	latency100 := 100
	latency200 := 200
	cost10 := 0.001
	cost20 := 0.002
	status200 := 200
	status500 := 500

	entries := []*RequestLogEntry{
		{
			RequestID:  "r1",
			Timestamp:  now.Add(-1 * time.Hour),
			Provider:   "groq",
			Model:      "llama-3.3-70b-versatile",
			LatencyMS:  &latency100,
			CostUSD:    &cost10,
			StatusCode: &status200,
			AuthType:   "api_key",
		},
		{
			RequestID:  "r2",
			Timestamp:  now.Add(-30 * time.Minute),
			Provider:   "groq",
			Model:      "llama-3.3-70b-versatile",
			LatencyMS:  &latency200,
			CostUSD:    &cost20,
			StatusCode: &status200,
			AuthType:   "api_key",
		},
		// should be excluded: status >= 400
		{
			RequestID:  "r3",
			Timestamp:  now.Add(-20 * time.Minute),
			Provider:   "groq",
			Model:      "llama-3.3-70b-versatile",
			LatencyMS:  &latency100,
			CostUSD:    &cost10,
			StatusCode: &status500,
			AuthType:   "api_key",
		},
		// different provider/model
		{
			RequestID:  "r4",
			Timestamp:  now.Add(-10 * time.Minute),
			Provider:   "openai",
			Model:      "gpt-4o-mini",
			LatencyMS:  &latency100,
			CostUSD:    &cost10,
			StatusCode: &status200,
			AuthType:   "api_key",
		},
		// should be excluded: before since window
		{
			RequestID:  "r5",
			Timestamp:  now.Add(-48 * time.Hour),
			Provider:   "groq",
			Model:      "llama-3.3-70b-versatile",
			LatencyMS:  &latency100,
			CostUSD:    &cost10,
			StatusCode: &status200,
			AuthType:   "api_key",
		},
	}

	for _, e := range entries {
		if err := s.LogRequest(e); err != nil {
			t.Fatalf("LogRequest: %v", err)
		}
	}

	since := now.Add(-24 * time.Hour)
	stats, err := s.ProviderModelStats(since)
	if err != nil {
		t.Fatalf("ProviderModelStats: %v", err)
	}

	// groq/llama: r1 + r2 (r3 excluded 500, r5 excluded before window)
	groqKey := "groq/llama-3.3-70b-versatile"
	gs, ok := stats[groqKey]
	if !ok {
		t.Fatalf("missing key %q in stats", groqKey)
	}
	if gs.Requests != 2 {
		t.Fatalf("groq requests = %d, want 2", gs.Requests)
	}
	wantAvgLatency := float64(latency100+latency200) / 2
	if gs.AvgLatencyMS != wantAvgLatency {
		t.Fatalf("groq avg latency = %f, want %f", gs.AvgLatencyMS, wantAvgLatency)
	}
	wantAvgCost := (cost10 + cost20) / 2
	if gs.AvgCostUSD != wantAvgCost {
		t.Fatalf("groq avg cost = %f, want %f", gs.AvgCostUSD, wantAvgCost)
	}

	// openai/gpt-4o-mini: r4 only
	openaiKey := "openai/gpt-4o-mini"
	os, ok := stats[openaiKey]
	if !ok {
		t.Fatalf("missing key %q in stats", openaiKey)
	}
	if os.Requests != 1 {
		t.Fatalf("openai requests = %d, want 1", os.Requests)
	}

	// r3 (500) must not create a stat row that inflates groq count
	if gs.Requests != 2 {
		t.Fatal("status>=400 row should be excluded from stats")
	}
}

func TestProviderModelStatsEmptyWindow(t *testing.T) {
	s := openTestStore(t)

	stats, err := s.ProviderModelStats(time.Now())
	if err != nil {
		t.Fatalf("ProviderModelStats on empty db: %v", err)
	}
	if len(stats) != 0 {
		t.Fatalf("expected empty stats, got %v", stats)
	}
}

func TestProviderModelStatsIgnoresOldRows(t *testing.T) {
	s := openTestStore(t)

	now := time.Now().UTC()
	latency := 50
	status := 200
	if err := s.LogRequest(&RequestLogEntry{
		RequestID:  "old",
		Timestamp:  now.Add(-72 * time.Hour),
		Provider:   "groq",
		Model:      "llama-3.3-70b-versatile",
		LatencyMS:  &latency,
		StatusCode: &status,
		AuthType:   "api_key",
	}); err != nil {
		t.Fatalf("LogRequest: %v", err)
	}

	stats, err := s.ProviderModelStats(now.Add(-24 * time.Hour))
	if err != nil {
		t.Fatalf("ProviderModelStats: %v", err)
	}
	if len(stats) != 0 {
		t.Fatalf("old rows should be excluded, got %v", stats)
	}
}

// openTestStore is defined in settings_test.go (package store).
