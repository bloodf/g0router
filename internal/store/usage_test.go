package store

import (
	"testing"
	"time"
)

func TestLogRequestAndGetUsage(t *testing.T) {
	s := openTestStore(t)
	inputTokens := 10
	outputTokens := 5
	totalTokens := 15
	cost := 0.0012
	latency := 250
	status := 200
	rtkEnabled := true
	cavemanEnabled := false
	ts := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)

	entry := &RequestLogEntry{
		RequestID:      "req-1",
		Timestamp:      ts,
		Provider:       "openai",
		Model:          "gpt-4o",
		AuthType:       "api_key",
		InputTokens:    &inputTokens,
		OutputTokens:   &outputTokens,
		TotalTokens:    &totalTokens,
		CostUSD:        &cost,
		LatencyMS:      &latency,
		StatusCode:     &status,
		SourceFormat:   stringPtr("openai"),
		TargetFormat:   stringPtr("openai"),
		RTKEnabled:     &rtkEnabled,
		CavemanEnabled: &cavemanEnabled,
		ComboName:      stringPtr("combo-a"),
		ClientTool:     stringPtr("codex"),
	}

	if err := s.LogRequest(entry); err != nil {
		t.Fatalf("LogRequest: %v", err)
	}

	entries, err := s.GetUsage(UsageFilter{})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries len = %d, want 1", len(entries))
	}

	got := entries[0]
	if got.ID == 0 {
		t.Error("ID should be populated")
	}
	if got.RequestID != entry.RequestID || got.Provider != entry.Provider || got.Model != entry.Model {
		t.Fatalf("entry mismatch: %+v", got)
	}
	if !got.Timestamp.Equal(ts) {
		t.Fatalf("timestamp = %s, want %s", got.Timestamp, ts)
	}
	if got.InputTokens == nil || *got.InputTokens != inputTokens {
		t.Fatalf("input tokens = %v, want %d", got.InputTokens, inputTokens)
	}
	if got.CostUSD == nil || *got.CostUSD != cost {
		t.Fatalf("cost = %v, want %f", got.CostUSD, cost)
	}
	if got.RTKEnabled == nil || *got.RTKEnabled != rtkEnabled {
		t.Fatalf("rtk enabled = %v, want %t", got.RTKEnabled, rtkEnabled)
	}
	if got.CavemanEnabled == nil || *got.CavemanEnabled != cavemanEnabled {
		t.Fatalf("caveman enabled = %v, want %t", got.CavemanEnabled, cavemanEnabled)
	}
	if got.ClientTool == nil || *got.ClientTool != "codex" {
		t.Fatalf("client tool = %v, want codex", got.ClientTool)
	}
}

func TestLogRequestNullableFields(t *testing.T) {
	s := openTestStore(t)

	err := s.LogRequest(&RequestLogEntry{
		RequestID: "req-null",
		Timestamp: time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC),
		Provider:  "anthropic",
		Model:     "claude-sonnet-4",
		AuthType:  "oauth",
	})
	if err != nil {
		t.Fatalf("LogRequest: %v", err)
	}

	entries, err := s.GetUsage(UsageFilter{})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries len = %d, want 1", len(entries))
	}
	if entries[0].InputTokens != nil || entries[0].CostUSD != nil || entries[0].RTKEnabled != nil {
		t.Fatalf("nullable fields should stay nil: %+v", entries[0])
	}
}

func TestGetUsageFilterByProvider(t *testing.T) {
	s := openTestStore(t)
	logUsageEntries(t, s, []RequestLogEntry{
		minimalUsageEntry("req-1", "openai", "gpt-4o", time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)),
		minimalUsageEntry("req-2", "anthropic", "claude-sonnet-4", time.Date(2026, 6, 2, 10, 1, 0, 0, time.UTC)),
		minimalUsageEntry("req-3", "openai", "gpt-4o-mini", time.Date(2026, 6, 2, 10, 2, 0, 0, time.UTC)),
	})

	entries, err := s.GetUsage(UsageFilter{Provider: stringPtr("openai")})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries len = %d, want 2", len(entries))
	}
	for _, entry := range entries {
		if entry.Provider != "openai" {
			t.Fatalf("provider = %q, want openai", entry.Provider)
		}
	}
}

func TestGetUsageFilterByDateRange(t *testing.T) {
	s := openTestStore(t)
	start := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	logUsageEntries(t, s, []RequestLogEntry{
		minimalUsageEntry("req-1", "openai", "gpt-4o", start.Add(-time.Hour)),
		minimalUsageEntry("req-2", "openai", "gpt-4o", start),
		minimalUsageEntry("req-3", "openai", "gpt-4o", start.Add(time.Hour)),
	})

	entries, err := s.GetUsage(UsageFilter{From: &start, To: timePtr(start.Add(30 * time.Minute))})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries len = %d, want 1", len(entries))
	}
	if entries[0].RequestID != "req-2" {
		t.Fatalf("request id = %q, want req-2", entries[0].RequestID)
	}
}

func TestGetUsagePagination(t *testing.T) {
	s := openTestStore(t)
	base := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	var entries []RequestLogEntry
	for i := 0; i < 10; i++ {
		entries = append(entries, minimalUsageEntry("req-"+string(rune('0'+i)), "openai", "gpt-4o", base.Add(time.Duration(i)*time.Minute)))
	}
	logUsageEntries(t, s, entries)

	got, err := s.GetUsage(UsageFilter{Limit: 3, Offset: 2})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("entries len = %d, want 3", len(got))
	}
	if got[0].RequestID != "req-7" || got[2].RequestID != "req-5" {
		t.Fatalf("pagination order mismatch: %+v", got)
	}
}

func TestGetUsageSummary(t *testing.T) {
	s := openTestStore(t)
	tokensA := 10
	tokensB := 20
	tokensC := 5
	costA := 0.25
	costB := 0.75

	entryA := minimalUsageEntry("req-1", "openai", "gpt-4o", time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC))
	entryA.TotalTokens = &tokensA
	entryA.CostUSD = &costA
	entryB := minimalUsageEntry("req-2", "openai", "gpt-4o", time.Date(2026, 6, 2, 10, 1, 0, 0, time.UTC))
	entryB.TotalTokens = &tokensB
	entryB.CostUSD = &costB
	entryC := minimalUsageEntry("req-3", "anthropic", "claude-sonnet-4", time.Date(2026, 6, 2, 10, 2, 0, 0, time.UTC))
	entryC.TotalTokens = &tokensC
	logUsageEntries(t, s, []RequestLogEntry{entryA, entryB, entryC})

	summary, err := s.GetUsageSummary(UsageFilter{Provider: stringPtr("openai")})
	if err != nil {
		t.Fatalf("GetUsageSummary: %v", err)
	}
	if summary.RequestCount != 2 {
		t.Fatalf("request count = %d, want 2", summary.RequestCount)
	}
	if summary.TotalTokens != 30 {
		t.Fatalf("total tokens = %d, want 30", summary.TotalTokens)
	}
	if summary.TotalCostUSD != 1.0 {
		t.Fatalf("total cost = %f, want 1.0", summary.TotalCostUSD)
	}
}

func TestGetUsageSummaryEmpty(t *testing.T) {
	s := openTestStore(t)

	summary, err := s.GetUsageSummary(UsageFilter{})
	if err != nil {
		t.Fatalf("GetUsageSummary: %v", err)
	}
	if summary.RequestCount != 0 || summary.TotalTokens != 0 || summary.TotalCostUSD != 0 {
		t.Fatalf("summary = %+v, want zeros", summary)
	}
}

func logUsageEntries(t *testing.T, s *Store, entries []RequestLogEntry) {
	t.Helper()

	for i := range entries {
		if err := s.LogRequest(&entries[i]); err != nil {
			t.Fatalf("LogRequest %q: %v", entries[i].RequestID, err)
		}
	}
}

func minimalUsageEntry(requestID, provider, model string, timestamp time.Time) RequestLogEntry {
	return RequestLogEntry{
		RequestID: requestID,
		Timestamp: timestamp,
		Provider:  provider,
		Model:     model,
		AuthType:  "api_key",
	}
}

func stringPtr(value string) *string {
	return &value
}

func timePtr(value time.Time) *time.Time {
	return &value
}
