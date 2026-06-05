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

func TestDeleteRequestLogsOlderThan(t *testing.T) {
	s := openTestStore(t)
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	logUsageEntries(t, s, []RequestLogEntry{
		minimalUsageEntry("old-1", "openai", "gpt-4o", now.Add(-10*24*time.Hour)),
		minimalUsageEntry("old-2", "openai", "gpt-4o", now.Add(-8*24*time.Hour)),
		minimalUsageEntry("new-1", "openai", "gpt-4o", now.Add(-1*24*time.Hour)),
	})

	cutoff := now.Add(-7 * 24 * time.Hour)
	deleted, err := s.DeleteRequestLogsOlderThan(cutoff)
	if err != nil {
		t.Fatalf("DeleteRequestLogsOlderThan: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("deleted = %d, want 2", deleted)
	}

	remaining, err := s.GetUsage(UsageFilter{})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(remaining) != 1 || remaining[0].RequestID != "new-1" {
		t.Fatalf("remaining = %+v, want only new-1", remaining)
	}
}

func TestGetUsageFilterBySourceFormat(t *testing.T) {
	s := openTestStore(t)
	base := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	a := minimalUsageEntry("req-a", "openai", "gpt-4o", base)
	a.SourceFormat = stringPtr("anthropic")
	b := minimalUsageEntry("req-b", "openai", "gpt-4o", base.Add(time.Minute))
	b.SourceFormat = stringPtr("openai")
	logUsageEntries(t, s, []RequestLogEntry{a, b})

	entries, err := s.GetUsage(UsageFilter{SourceFormat: stringPtr("anthropic")})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 1 || entries[0].RequestID != "req-a" {
		t.Fatalf("entries = %+v, want only req-a", entries)
	}
}

func TestGetUsageFilterByAuthType(t *testing.T) {
	s := openTestStore(t)
	base := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	a := minimalUsageEntry("req-a", "openai", "gpt-4o", base)
	a.AuthType = "oauth"
	b := minimalUsageEntry("req-b", "openai", "gpt-4o", base.Add(time.Minute))
	b.AuthType = "api_key"
	logUsageEntries(t, s, []RequestLogEntry{a, b})

	entries, err := s.GetUsage(UsageFilter{AuthType: stringPtr("oauth")})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 1 || entries[0].RequestID != "req-a" {
		t.Fatalf("entries = %+v, want only req-a", entries)
	}
}

func TestGetUsageFilterByStatusClass(t *testing.T) {
	s := openTestStore(t)
	base := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	ok := minimalUsageEntry("ok", "openai", "gpt-4o", base)
	ok.StatusCode = intPtr(200)
	client := minimalUsageEntry("client", "openai", "gpt-4o", base.Add(time.Minute))
	client.StatusCode = intPtr(404)
	server := minimalUsageEntry("server", "openai", "gpt-4o", base.Add(2*time.Minute))
	server.StatusCode = intPtr(503)
	logUsageEntries(t, s, []RequestLogEntry{ok, client, server})

	cases := map[string]string{"success": "ok", "client_error": "client", "server_error": "server"}
	for class, wantID := range cases {
		entries, err := s.GetUsage(UsageFilter{StatusClass: class})
		if err != nil {
			t.Fatalf("GetUsage %s: %v", class, err)
		}
		if len(entries) != 1 || entries[0].RequestID != wantID {
			t.Fatalf("status class %s = %+v, want %s", class, entries, wantID)
		}
	}
}

func TestGetUsageFilterBySearch(t *testing.T) {
	s := openTestStore(t)
	base := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	a := minimalUsageEntry("alpha-request", "openai", "gpt-4o", base)
	b := minimalUsageEntry("beta-request", "openai", "claude-sonnet", base.Add(time.Minute))
	c := minimalUsageEntry("gamma-request", "openai", "gpt-4o", base.Add(2*time.Minute))
	c.Error = stringPtr("rate_limit: too many requests")
	logUsageEntries(t, s, []RequestLogEntry{a, b, c})

	byID, err := s.GetUsage(UsageFilter{Search: "ALPHA"})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(byID) != 1 || byID[0].RequestID != "alpha-request" {
		t.Fatalf("search by id = %+v, want alpha-request", byID)
	}

	byModel, err := s.GetUsage(UsageFilter{Search: "claude"})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(byModel) != 1 || byModel[0].RequestID != "beta-request" {
		t.Fatalf("search by model = %+v, want beta-request", byModel)
	}

	byErr, err := s.GetUsage(UsageFilter{Search: "rate_limit"})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(byErr) != 1 || byErr[0].RequestID != "gamma-request" {
		t.Fatalf("search by error = %+v, want gamma-request", byErr)
	}
}

func TestGetUsageFilterByStartEnd(t *testing.T) {
	s := openTestStore(t)
	base := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	logUsageEntries(t, s, []RequestLogEntry{
		minimalUsageEntry("before", "openai", "gpt-4o", base.Add(-time.Hour)),
		minimalUsageEntry("inside", "openai", "gpt-4o", base),
		minimalUsageEntry("after", "openai", "gpt-4o", base.Add(time.Hour)),
	})

	entries, err := s.GetUsage(UsageFilter{Start: timePtr(base.Add(-time.Minute)), End: timePtr(base.Add(time.Minute))})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 1 || entries[0].RequestID != "inside" {
		t.Fatalf("entries = %+v, want only inside", entries)
	}
}

func TestCountUsageIgnoresLimitOffset(t *testing.T) {
	s := openTestStore(t)
	base := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	var entries []RequestLogEntry
	for i := 0; i < 5; i++ {
		entries = append(entries, minimalUsageEntry("req-"+string(rune('0'+i)), "openai", "gpt-4o", base.Add(time.Duration(i)*time.Minute)))
	}
	entries = append(entries, minimalUsageEntry("other", "anthropic", "claude", base.Add(10*time.Minute)))
	logUsageEntries(t, s, entries)

	total, err := s.CountUsage(UsageFilter{Provider: stringPtr("openai"), Limit: 2, Offset: 1})
	if err != nil {
		t.Fatalf("CountUsage: %v", err)
	}
	if total != 5 {
		t.Fatalf("count = %d, want 5 (ignoring limit/offset)", total)
	}
}

func intPtr(value int) *int {
	return &value
}

func TestGetUsageSearchLiteralPercent(t *testing.T) {
	s := openTestStore(t)
	base := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	a := minimalUsageEntry("req-percent", "openai", "gpt-4o", base)
	a.Error = stringPtr("rate%limit exceeded")
	b := minimalUsageEntry("req-other", "openai", "gpt-4o", base.Add(time.Minute))
	b.Error = stringPtr("connection refused")
	logUsageEntries(t, s, []RequestLogEntry{a, b})

	// Searching for literal "%" must not act as a wildcard — only req-percent matches.
	entries, err := s.GetUsage(UsageFilter{Search: "%"})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 1 || entries[0].RequestID != "req-percent" {
		t.Fatalf("entries = %+v, want only req-percent", entries)
	}
}

func TestGetUsageFromToAliasesStartEnd(t *testing.T) {
	s := openTestStore(t)
	base := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	logUsageEntries(t, s, []RequestLogEntry{
		minimalUsageEntry("before", "openai", "gpt-4o", base.Add(-time.Hour)),
		minimalUsageEntry("inside", "openai", "gpt-4o", base),
		minimalUsageEntry("after", "openai", "gpt-4o", base.Add(time.Hour)),
	})

	window := base.Add(-time.Minute)
	windowEnd := base.Add(time.Minute)

	// from/to should behave identically to start/end.
	byFromTo, err := s.GetUsage(UsageFilter{From: &window, To: &windowEnd})
	if err != nil {
		t.Fatalf("GetUsage from/to: %v", err)
	}
	byStartEnd, err := s.GetUsage(UsageFilter{Start: &window, End: &windowEnd})
	if err != nil {
		t.Fatalf("GetUsage start/end: %v", err)
	}
	if len(byFromTo) != 1 || byFromTo[0].RequestID != "inside" {
		t.Fatalf("from/to entries = %+v, want only inside", byFromTo)
	}
	if len(byStartEnd) != len(byFromTo) || byStartEnd[0].RequestID != byFromTo[0].RequestID {
		t.Fatalf("start/end = %+v, from/to = %+v, want identical", byStartEnd, byFromTo)
	}
}

func TestMigrateCreatesRequestLogIndexes(t *testing.T) {
	s := openTestStore(t)

	wantIndexes := []string{
		"idx_request_log_timestamp",
		"idx_request_log_status_code",
		"idx_request_log_source_format",
		"idx_request_log_provider_model_ts",
	}
	for _, idx := range wantIndexes {
		var name string
		err := s.db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='index' AND name=?", idx,
		).Scan(&name)
		if err != nil {
			t.Errorf("index %q missing: %v", idx, err)
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
