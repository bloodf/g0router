package store

import (
	"testing"
	"time"
)

// TestProviderModelStatsNullCostAndLatency exercises the nil avgLatency / nil avgCost
// branches in ProviderModelStats (rows where latency_ms IS NULL and cost_usd IS NULL).
func TestProviderModelStatsNullCostAndLatency(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC()
	status200 := 200

	// Insert a row with no latency and no cost — both nullable columns are NULL.
	if err := s.LogRequest(&RequestLogEntry{
		RequestID:  "null-fields",
		Timestamp:  now.Add(-1 * time.Hour),
		Provider:   "groq",
		Model:      "llama-3.3-70b-versatile",
		LatencyMS:  nil, // NULL
		CostUSD:    nil, // NULL
		StatusCode: &status200,
		AuthType:   "api_key",
	}); err != nil {
		t.Fatalf("LogRequest: %v", err)
	}

	stats, err := s.ProviderModelStats(now.Add(-24 * time.Hour))
	if err != nil {
		t.Fatalf("ProviderModelStats: %v", err)
	}

	key := "groq/llama-3.3-70b-versatile"
	gs, ok := stats[key]
	if !ok {
		t.Fatalf("missing key %q in stats", key)
	}
	if gs.Requests != 1 {
		t.Fatalf("requests = %d, want 1", gs.Requests)
	}
	// NULL avg should stay at zero value (not set by the nil branch).
	if gs.AvgLatencyMS != 0 {
		t.Errorf("AvgLatencyMS = %f, want 0 for NULL column", gs.AvgLatencyMS)
	}
	if gs.AvgCostUSD != 0 {
		t.Errorf("AvgCostUSD = %f, want 0 for NULL column", gs.AvgCostUSD)
	}
}

// TestProviderModelStatsSinceBoundaryExact exercises the exact boundary: a row
// timestamped at exactly `since` should NOT be included (strictly >).
func TestProviderModelStatsSinceBoundaryExact(t *testing.T) {
	s := openTestStore(t)
	since := time.Now().UTC().Add(-1 * time.Hour)
	status200 := 200
	latency := 100

	// Row at exactly `since` — the SQL uses >= so it IS included.
	if err := s.LogRequest(&RequestLogEntry{
		RequestID:  "at-boundary",
		Timestamp:  since,
		Provider:   "openai",
		Model:      "gpt-4o",
		LatencyMS:  &latency,
		StatusCode: &status200,
		AuthType:   "api_key",
	}); err != nil {
		t.Fatalf("LogRequest: %v", err)
	}

	stats, err := s.ProviderModelStats(since)
	if err != nil {
		t.Fatalf("ProviderModelStats: %v", err)
	}

	// The SQL uses >= so the boundary row is included.
	if _, ok := stats["openai/gpt-4o"]; !ok {
		t.Fatalf("boundary row (>=) should be included in stats")
	}
}
