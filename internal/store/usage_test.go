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

func TestGetUsageResolvesAPIKeyAndConnectionLabels(t *testing.T) {
	s := openTestStore(t)

	// Create an api key (name only; key hash/secret irrelevant for JOIN resolution).
	key, _, err := s.CreateAPIKey("my-key", "testsecret")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	// Create a connection with provider+email.
	email := "user@example.com"
	conn := &Connection{
		Provider: "anthropic",
		Name:     "work-account",
		AuthType: AuthTypeAPIKey,
		IsActive: true,
		Email:    &email,
	}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	conns, err := s.GetConnections("anthropic")
	if err != nil || len(conns) == 0 {
		t.Fatalf("GetConnections: %v (len=%d)", err, len(conns))
	}
	connID := conns[0].ID

	ts := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	entry := RequestLogEntry{
		RequestID:    "req-labeled",
		Timestamp:    ts,
		Provider:     "anthropic",
		Model:        "claude-sonnet-4",
		AuthType:     "api_key",
		APIKeyID:     &key.ID,
		ConnectionID: &connID,
	}
	if err := s.LogRequest(&entry); err != nil {
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
	if got.APIKeyName == nil || *got.APIKeyName != "my-key" {
		t.Fatalf("APIKeyName = %v, want my-key", got.APIKeyName)
	}
	if got.ConnectionName == nil || *got.ConnectionName != "work-account" {
		t.Fatalf("ConnectionName = %v, want work-account", got.ConnectionName)
	}
	if got.ConnectionProvider == nil || *got.ConnectionProvider != "anthropic" {
		t.Fatalf("ConnectionProvider = %v, want anthropic", got.ConnectionProvider)
	}
	if got.AccountEmail == nil || *got.AccountEmail != "user@example.com" {
		t.Fatalf("AccountEmail = %v, want user@example.com", got.AccountEmail)
	}
}

func TestGetUsageNullAPIKeyAndConnectionReturnsNilLabels(t *testing.T) {
	s := openTestStore(t)
	entry := minimalUsageEntry("req-no-key", "openai", "gpt-4o", time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC))
	// No APIKeyID, no ConnectionID → LEFT JOIN yields NULLs.
	if err := s.LogRequest(&entry); err != nil {
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
	if got.APIKeyName != nil {
		t.Fatalf("APIKeyName = %v, want nil", got.APIKeyName)
	}
	if got.ConnectionName != nil {
		t.Fatalf("ConnectionName = %v, want nil", got.ConnectionName)
	}
	if got.ConnectionProvider != nil {
		t.Fatalf("ConnectionProvider = %v, want nil", got.ConnectionProvider)
	}
	if got.AccountEmail != nil {
		t.Fatalf("AccountEmail = %v, want nil", got.AccountEmail)
	}
}

func TestGetUsageFilterByAPIKeyID(t *testing.T) {
	s := openTestStore(t)
	key, _, err := s.CreateAPIKey("filter-key", "testsecret")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	base := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	withKey := minimalUsageEntry("req-keyed", "openai", "gpt-4o", base)
	withKey.APIKeyID = &key.ID
	noKey := minimalUsageEntry("req-nokey", "openai", "gpt-4o", base.Add(time.Minute))
	if err := s.LogRequest(&withKey); err != nil {
		t.Fatalf("LogRequest withKey: %v", err)
	}
	if err := s.LogRequest(&noKey); err != nil {
		t.Fatalf("LogRequest noKey: %v", err)
	}

	entries, err := s.GetUsage(UsageFilter{APIKeyID: &key.ID})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 1 || entries[0].RequestID != "req-keyed" {
		t.Fatalf("entries = %+v, want only req-keyed", entries)
	}

	count, err := s.CountUsage(UsageFilter{APIKeyID: &key.ID})
	if err != nil {
		t.Fatalf("CountUsage: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
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

func TestGetUsageChartDayAggregation(t *testing.T) {
	s := openTestStore(t)
	now := time.Date(2026, 6, 6, 14, 30, 0, 0, time.UTC)

	// Seed entries across 3 days with gaps.
	base := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	entries := []RequestLogEntry{
		makeChartEntry("req-a", base, 10, 5, 0.50),
		makeChartEntry("req-b", base.Add(2*time.Hour), 20, 10, 1.00),
		makeChartEntry("req-c", base.Add(24*time.Hour), 5, 2, 0.25),
		makeChartEntry("req-d", base.Add(48*time.Hour), 15, 8, 0.75),
	}
	logUsageEntries(t, s, entries)

	chart, err := s.GetUsageChart("7d", "day", now)
	if err != nil {
		t.Fatalf("GetUsageChart: %v", err)
	}

	// Buckets from 2026-05-30 through 2026-06-06 (8 days) when using [now-7d, now].
	if len(chart.Buckets) != 8 {
		t.Fatalf("buckets len = %d, want 8; buckets=%v", len(chart.Buckets), chart.Buckets)
	}

	// Find indices for the days with data.
	idx := make(map[string]int)
	for i, b := range chart.Buckets {
		idx[b] = i
	}

	// June 3 has 2 entries: 30 requests in, 15 out, 1.50 cost
	if chart.Requests[idx["2026-06-03"]] != 2 {
		t.Fatalf("june 3 requests = %d, want 2", chart.Requests[idx["2026-06-03"]])
	}
	if chart.TokensInput[idx["2026-06-03"]] != 30 {
		t.Fatalf("june 3 tokens_input = %d, want 30", chart.TokensInput[idx["2026-06-03"]])
	}
	if chart.TokensOutput[idx["2026-06-03"]] != 15 {
		t.Fatalf("june 3 tokens_output = %d, want 15", chart.TokensOutput[idx["2026-06-03"]])
	}
	if chart.Costs[idx["2026-06-03"]] != 1.50 {
		t.Fatalf("june 3 cost = %f, want 1.50", chart.Costs[idx["2026-06-03"]])
	}

	// June 4 has 1 entry.
	if chart.Requests[idx["2026-06-04"]] != 1 {
		t.Fatalf("june 4 requests = %d, want 1", chart.Requests[idx["2026-06-04"]])
	}

	// June 6 has 0 entries (gap) — verify zero-fill.
	if chart.Requests[idx["2026-06-06"]] != 0 {
		t.Fatalf("june 6 requests = %d, want 0", chart.Requests[idx["2026-06-06"]])
	}
	if chart.Costs[idx["2026-06-06"]] != 0 {
		t.Fatalf("june 6 cost = %f, want 0", chart.Costs[idx["2026-06-06"]])
	}

	// Arrays must be aligned.
	if len(chart.Buckets) != len(chart.Requests) || len(chart.Buckets) != len(chart.TokensInput) ||
		len(chart.Buckets) != len(chart.TokensOutput) || len(chart.Buckets) != len(chart.Costs) {
		t.Fatalf("array lengths misaligned: buckets=%d requests=%d tokens_input=%d tokens_output=%d costs=%d",
			len(chart.Buckets), len(chart.Requests), len(chart.TokensInput), len(chart.TokensOutput), len(chart.Costs))
	}
}

func TestGetUsageChartHourAggregation(t *testing.T) {
	s := openTestStore(t)
	now := time.Date(2026, 6, 6, 14, 30, 0, 0, time.UTC)

	base := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	entries := []RequestLogEntry{
		makeChartEntry("req-a", base, 10, 5, 0.50),
		makeChartEntry("req-b", base.Add(30*time.Minute), 20, 10, 1.00),
		makeChartEntry("req-c", base.Add(2*time.Hour), 5, 2, 0.25),
	}
	logUsageEntries(t, s, entries)

	chart, err := s.GetUsageChart("today", "hour", now)
	if err != nil {
		t.Fatalf("GetUsageChart: %v", err)
	}

	// Buckets from 00:00 through 14:00 = 15 buckets.
	if len(chart.Buckets) != 15 {
		t.Fatalf("buckets len = %d, want 15; buckets=%v", len(chart.Buckets), chart.Buckets)
	}

	idx := make(map[string]int)
	for i, b := range chart.Buckets {
		idx[b] = i
	}

	// 10:00 has 2 entries.
	if chart.Requests[idx["2026-06-06T10:00"]] != 2 {
		t.Fatalf("10:00 requests = %d, want 2", chart.Requests[idx["2026-06-06T10:00"]])
	}
	if chart.TokensInput[idx["2026-06-06T10:00"]] != 30 {
		t.Fatalf("10:00 tokens_input = %d, want 30", chart.TokensInput[idx["2026-06-06T10:00"]])
	}

	// 12:00 has 1 entry.
	if chart.Requests[idx["2026-06-06T12:00"]] != 1 {
		t.Fatalf("12:00 requests = %d, want 1", chart.Requests[idx["2026-06-06T12:00"]])
	}

	// 11:00 is a gap — zero-filled.
	if chart.Requests[idx["2026-06-06T11:00"]] != 0 {
		t.Fatalf("11:00 requests = %d, want 0", chart.Requests[idx["2026-06-06T11:00"]])
	}
}

func TestGetUsageChartEmptyTable(t *testing.T) {
	s := openTestStore(t)
	now := time.Date(2026, 6, 6, 14, 30, 0, 0, time.UTC)

	chart, err := s.GetUsageChart("7d", "day", now)
	if err != nil {
		t.Fatalf("GetUsageChart: %v", err)
	}

	// Should return all-zero series, not nil/empty.
	if len(chart.Buckets) == 0 {
		t.Fatalf("buckets should not be empty")
	}
	for i := range chart.Buckets {
		if chart.Requests[i] != 0 || chart.TokensInput[i] != 0 || chart.TokensOutput[i] != 0 || chart.Costs[i] != 0 {
			t.Fatalf("all values should be zero at index %d", i)
		}
	}
}

func TestGetUsageChartInvalidPeriod(t *testing.T) {
	s := openTestStore(t)
	now := time.Date(2026, 6, 6, 14, 30, 0, 0, time.UTC)

	_, err := s.GetUsageChart("invalid", "day", now)
	if err == nil {
		t.Fatal("expected error for invalid period")
	}
}

func TestGetUsageChartInvalidGranularity(t *testing.T) {
	s := openTestStore(t)
	now := time.Date(2026, 6, 6, 14, 30, 0, 0, time.UTC)

	_, err := s.GetUsageChart("7d", "week", now)
	if err == nil {
		t.Fatal("expected error for invalid granularity")
	}
}

func makeChartEntry(requestID string, ts time.Time, inputTokens, outputTokens int, cost float64) RequestLogEntry {
	return RequestLogEntry{
		RequestID:    requestID,
		Timestamp:    ts,
		Provider:     "openai",
		Model:        "gpt-4o",
		AuthType:     "api_key",
		InputTokens:  &inputTokens,
		OutputTokens: &outputTokens,
		CostUSD:      &cost,
	}
}
