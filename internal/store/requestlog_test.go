package store

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSaveUsageTransactional(t *testing.T) {
	st := newTestStore(t)

	e := &RequestLogEntry{
		Timestamp:      "2026-06-12T10:00:00Z",
		Provider:       "openai",
		Model:          "gpt-4o",
		ConnectionID:   "conn-1",
		APIKey:         "key-1",
		Endpoint:       "/chat/completions",
		PromptTokens:   100,
		CompletionTokens: 50,
		Status:         "ok",
		Tokens:         map[string]int64{"prompt_tokens": 100, "completion_tokens": 50},
	}

	if err := st.SaveUsage(e); err != nil {
		t.Fatalf("SaveUsage: %v", err)
	}

	var reqCount int
	if err := st.DB().QueryRow("SELECT COUNT(*) FROM request_log").Scan(&reqCount); err != nil {
		t.Fatalf("count request_log: %v", err)
	}
	if reqCount != 1 {
		t.Errorf("request_log count = %d, want 1", reqCount)
	}

	var dailyCount int
	if err := st.DB().QueryRow("SELECT COUNT(*) FROM usage_daily").Scan(&dailyCount); err != nil {
		t.Fatalf("count usage_daily: %v", err)
	}
	if dailyCount != 1 {
		t.Errorf("usage_daily count = %d, want 1", dailyCount)
	}

	var data string
	if err := st.DB().QueryRow("SELECT data FROM usage_daily WHERE date_key = ?", "2026-06-12").Scan(&data); err != nil {
		t.Fatalf("select usage_daily data: %v", err)
	}

	var day map[string]any
	if err := json.Unmarshal([]byte(data), &day); err != nil {
		t.Fatalf("unmarshal day data: %v", err)
	}
	if got := int(day["requests"].(float64)); got != 1 {
		t.Errorf("day.requests = %d, want 1", got)
	}
	if got := int(day["promptTokens"].(float64)); got != 100 {
		t.Errorf("day.promptTokens = %d, want 100", got)
	}
	if got := int(day["completionTokens"].(float64)); got != 50 {
		t.Errorf("day.completionTokens = %d, want 50", got)
	}

	byProvider, ok := day["byProvider"].(map[string]any)
	if !ok {
		t.Fatalf("byProvider missing or wrong type")
	}
	openai, ok := byProvider["openai"].(map[string]any)
	if !ok {
		t.Fatalf("byProvider.openai missing")
	}
	if got := int(openai["requests"].(float64)); got != 1 {
		t.Errorf("byProvider.openai.requests = %d, want 1", got)
	}

	byModel, ok := day["byModel"].(map[string]any)
	if !ok {
		t.Fatalf("byModel missing or wrong type")
	}
	modelKey := "gpt-4o|openai"
	modelEntry, ok := byModel[modelKey].(map[string]any)
	if !ok {
		t.Fatalf("byModel[%q] missing", modelKey)
	}
	if got := int(modelEntry["requests"].(float64)); got != 1 {
		t.Errorf("byModel[%q].requests = %d, want 1", modelKey, got)
	}
	if modelEntry["rawModel"] != "gpt-4o" {
		t.Errorf("byModel[%q].rawModel = %v, want gpt-4o", modelKey, modelEntry["rawModel"])
	}
	if modelEntry["provider"] != "openai" {
		t.Errorf("byModel[%q].provider = %v, want openai", modelKey, modelEntry["provider"])
	}

	counter, err := st.GetKV("meta", "total_requests_lifetime")
	if err != nil {
		t.Fatalf("GetKV lifetime counter: %v", err)
	}
	if counter != "1" {
		t.Errorf("lifetime counter = %q, want 1", counter)
	}

	// Second save same day accumulates.
	e2 := &RequestLogEntry{
		Timestamp:      "2026-06-12T11:00:00Z",
		Provider:       "openai",
		Model:          "gpt-4o",
		PromptTokens:   200,
		CompletionTokens: 100,
		Tokens:         map[string]int64{"prompt_tokens": 200, "completion_tokens": 100},
	}
	if err := st.SaveUsage(e2); err != nil {
		t.Fatalf("SaveUsage second: %v", err)
	}

	if err := st.DB().QueryRow("SELECT data FROM usage_daily WHERE date_key = ?", "2026-06-12").Scan(&data); err != nil {
		t.Fatalf("select usage_daily data second: %v", err)
	}
	if err := json.Unmarshal([]byte(data), &day); err != nil {
		t.Fatalf("unmarshal day data second: %v", err)
	}
	if got := int(day["requests"].(float64)); got != 2 {
		t.Errorf("day.requests after second save = %d, want 2", got)
	}
	if got := int(day["promptTokens"].(float64)); got != 300 {
		t.Errorf("day.promptTokens after second save = %d, want 300", got)
	}

	counter, err = st.GetKV("meta", "total_requests_lifetime")
	if err != nil {
		t.Fatalf("GetKV lifetime counter second: %v", err)
	}
	if counter != "2" {
		t.Errorf("lifetime counter after second save = %q, want 2", counter)
	}
}

func TestListRecentRequestLogsNullColumns(t *testing.T) {
	st := newTestStore(t)

	// Insert a row directly so that nullable TEXT columns are NULL, simulating
	// rows imported from a 9router database.
	if _, err := st.DB().Exec(
		`INSERT INTO request_log (
			timestamp, provider, model, connection_id, api_key, endpoint,
			prompt_tokens, completion_tokens, cost, status, tokens, meta
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"2026-06-12T10:00:00Z", nil, nil, nil, nil, nil, 1, 2, 0.1, nil, "{}", "{}",
	); err != nil {
		t.Fatalf("insert null row: %v", err)
	}

	logs, err := st.ListRecentRequestLogs(10)
	if err != nil {
		t.Fatalf("ListRecentRequestLogs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("len(logs) = %d, want 1", len(logs))
	}
	e := logs[0]
	if e.Provider != "" || e.Model != "" || e.ConnectionID != "" || e.APIKey != "" || e.Endpoint != "" || e.Status != "" {
		t.Errorf("nullable columns = %+v, want empty strings", e)
	}
	if e.PromptTokens != 1 || e.CompletionTokens != 2 || e.Cost != 0.1 {
		t.Errorf("numeric columns = %+v, want 1/2/0.1", e)
	}
}

func TestSaveUsageRollsBackTogether(t *testing.T) {
	st := newTestStore(t)

	if _, err := st.DB().Exec("DROP TABLE kv"); err != nil {
		t.Fatalf("DROP TABLE kv: %v", err)
	}

	e := &RequestLogEntry{
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Provider:       "openai",
		Model:          "gpt-4o",
		PromptTokens:   10,
		CompletionTokens: 5,
		Tokens:         map[string]int64{"prompt_tokens": 10, "completion_tokens": 5},
	}

	if err := st.SaveUsage(e); err == nil {
		t.Fatal("SaveUsage expected error after dropping kv table, got nil")
	}

	var reqCount int
	if err := st.DB().QueryRow("SELECT COUNT(*) FROM request_log").Scan(&reqCount); err != nil {
		t.Fatalf("count request_log: %v", err)
	}
	if reqCount != 0 {
		t.Errorf("request_log count = %d, want 0 (rolled back)", reqCount)
	}

	var dailyCount int
	if err := st.DB().QueryRow("SELECT COUNT(*) FROM usage_daily").Scan(&dailyCount); err != nil {
		t.Fatalf("count usage_daily: %v", err)
	}
	if dailyCount != 0 {
		t.Errorf("usage_daily count = %d, want 0 (rolled back)", dailyCount)
	}
}

func freshDay() map[string]any {
	return map[string]any{
		"requests":         0,
		"promptTokens":     0,
		"completionTokens": 0,
		"cost":             0.0,
		"byProvider":       map[string]any{},
		"byModel":          map[string]any{},
		"byAccount":        map[string]any{},
		"byApiKey":         map[string]any{},
		"byEndpoint":       map[string]any{},
	}
}

func requireCounter(t *testing.T, m map[string]any, key string) map[string]any {
	t.Helper()
	c, ok := m[key].(map[string]any)
	if !ok {
		t.Fatalf("counter %q missing or wrong type", key)
	}
	return c
}

func TestLoadDailyRange(t *testing.T) {
	st := newTestStore(t)

	// Insert 4 daily rows out of order.
	rows := []struct {
		dateKey string
		data    string
	}{
		{"2026-06-09", `{"requests":1}`},
		{"2026-06-10", `{"requests":2}`},
		{"2026-06-11", `{"requests":3}`},
		{"2026-06-12", `{"requests":4}`},
	}
	for _, r := range rows {
		if _, err := st.DB().Exec("INSERT INTO usage_daily (date_key, data) VALUES (?, ?)", r.dateKey, r.data); err != nil {
			t.Fatalf("insert usage_daily %s: %v", r.dateKey, err)
		}
	}

	// maxDays=2 should include today and yesterday only (dateKey >= today-1).
	got, err := st.LoadDailyRange(2)
	if err != nil {
		t.Fatalf("LoadDailyRange(2): %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("LoadDailyRange(2) len = %d, want 2", len(got))
	}
	if got[0].DateKey != "2026-06-11" || got[1].DateKey != "2026-06-12" {
		t.Errorf("LoadDailyRange(2) keys = %v, want [2026-06-11 2026-06-12]", []string{got[0].DateKey, got[1].DateKey})
	}

	// nil equivalent: maxDays=0 returns all rows (caller uses zero to mean unlimited here).
	got, err = st.LoadDailyRange(0)
	if err != nil {
		t.Fatalf("LoadDailyRange(0): %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("LoadDailyRange(0) len = %d, want 4", len(got))
	}
}

func TestRangeRequestLogs(t *testing.T) {
	st := newTestStore(t)

	entries := []*RequestLogEntry{
		{Timestamp: "2026-06-12T08:59:59Z", Provider: "openai", Model: "gpt-4o", PromptTokens: 1, CompletionTokens: 1},
		{Timestamp: "2026-06-12T09:00:00Z", Provider: "openai", Model: "gpt-4o", PromptTokens: 2, CompletionTokens: 2},
		{Timestamp: "2026-06-12T09:30:00Z", Provider: "openai", Model: "gpt-4o", PromptTokens: 3, CompletionTokens: 3},
		{Timestamp: "2026-06-12T10:00:00Z", Provider: "openai", Model: "gpt-4o", PromptTokens: 4, CompletionTokens: 4},
		{Timestamp: "2026-06-12T10:00:01Z", Provider: "openai", Model: "gpt-4o", PromptTokens: 5, CompletionTokens: 5},
	}
	for _, e := range entries {
		if err := st.SaveUsage(e); err != nil {
			t.Fatalf("SaveUsage: %v", err)
		}
	}

	// Inclusive bounds: [09:00:00, 10:00:00] should include exactly 09:00:00, 09:30:00, 10:00:00.
	got, err := st.RangeRequestLogs("2026-06-12T09:00:00Z", "2026-06-12T10:00:00Z")
	if err != nil {
		t.Fatalf("RangeRequestLogs: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("RangeRequestLogs len = %d, want 3", len(got))
	}
	sum := int64(0)
	for _, e := range got {
		sum += e.PromptTokens
	}
	if sum != 2+3+4 {
		t.Errorf("prompt tokens sum = %d, want 9", sum)
	}
}

func TestAggregateEntryToDay(t *testing.T) {
	// Full entry: exact key shapes and meta preservation.
	day := freshDay()
	aggregateEntryToDay(day, &RequestLogEntry{
		Provider:         "anthropic",
		Model:            "claude-sonnet-4",
		ConnectionID:     "conn-a",
		APIKey:           "ak",
		Endpoint:         "/messages",
		PromptTokens:     10,
		CompletionTokens: 20,
		Cost:             0.123,
	})

	if got := day["requests"].(float64); got != 1 {
		t.Errorf("requests = %v, want 1", got)
	}
	if got := day["promptTokens"].(float64); got != 10 {
		t.Errorf("promptTokens = %v, want 10", got)
	}
	if got := day["completionTokens"].(float64); got != 20 {
		t.Errorf("completionTokens = %v, want 20", got)
	}
	if got := day["cost"].(float64); got != 0.123 {
		t.Errorf("cost = %v, want 0.123", got)
	}

	byProvider := day["byProvider"].(map[string]any)
	if _, ok := byProvider["anthropic"]; !ok {
		t.Errorf("expected byProvider key anthropic")
	}

	byModel := day["byModel"].(map[string]any)
	modelEntry := requireCounter(t, byModel, "claude-sonnet-4|anthropic")
	if modelEntry["rawModel"] != "claude-sonnet-4" {
		t.Errorf("byModel meta rawModel = %v, want claude-sonnet-4", modelEntry["rawModel"])
	}
	if modelEntry["provider"] != "anthropic" {
		t.Errorf("byModel meta provider = %v, want anthropic", modelEntry["provider"])
	}

	byAccount := day["byAccount"].(map[string]any)
	accountEntry := requireCounter(t, byAccount, "conn-a")
	if accountEntry["rawModel"] != "claude-sonnet-4" {
		t.Errorf("byAccount meta rawModel = %v, want claude-sonnet-4", accountEntry["rawModel"])
	}
	if accountEntry["provider"] != "anthropic" {
		t.Errorf("byAccount meta provider = %v, want anthropic", accountEntry["provider"])
	}

	byApiKey := day["byApiKey"].(map[string]any)
	apiKeyEntry := requireCounter(t, byApiKey, "ak|claude-sonnet-4|anthropic")
	if apiKeyEntry["rawModel"] != "claude-sonnet-4" {
		t.Errorf("byApiKey meta rawModel = %v, want claude-sonnet-4", apiKeyEntry["rawModel"])
	}
	if apiKeyEntry["provider"] != "anthropic" {
		t.Errorf("byApiKey meta provider = %v, want anthropic", apiKeyEntry["provider"])
	}
	if apiKeyEntry["apiKey"] != "ak" {
		t.Errorf("byApiKey meta apiKey = %v, want ak", apiKeyEntry["apiKey"])
	}

	byEndpoint := day["byEndpoint"].(map[string]any)
	endpointEntry := requireCounter(t, byEndpoint, "/messages|claude-sonnet-4|anthropic")
	if endpointEntry["endpoint"] != "/messages" {
		t.Errorf("byEndpoint meta endpoint = %v, want /messages", endpointEntry["endpoint"])
	}
	if endpointEntry["rawModel"] != "claude-sonnet-4" {
		t.Errorf("byEndpoint meta rawModel = %v, want claude-sonnet-4", endpointEntry["rawModel"])
	}
	if endpointEntry["provider"] != "anthropic" {
		t.Errorf("byEndpoint meta provider = %v, want anthropic", endpointEntry["provider"])
	}

	// Second entry accumulates.
	aggregateEntryToDay(day, &RequestLogEntry{
		Provider:         "anthropic",
		Model:            "claude-sonnet-4",
		PromptTokens:     5,
		CompletionTokens: 5,
		Cost:             0.1,
	})
	if got := day["requests"].(float64); got != 2 {
		t.Errorf("requests after second = %v, want 2", got)
	}
	if got := day["promptTokens"].(float64); got != 15 {
		t.Errorf("promptTokens after second = %v, want 15", got)
	}

	// (a) Entry without provider: byModel key is bare model; byApiKey/byEndpoint use
	// "unknown" provider segment; no byProvider entry.
	dayNoProvider := freshDay()
	aggregateEntryToDay(dayNoProvider, &RequestLogEntry{
		Model:            "gpt-4o",
		APIKey:           "k",
		Endpoint:         "/v1",
		PromptTokens:     1,
		CompletionTokens: 2,
		Cost:             0.05,
	})
	if _, ok := dayNoProvider["byProvider"].(map[string]any)["unknown"]; ok {
		t.Errorf("byProvider should not contain a fallback key for missing provider")
	}
	byModel = dayNoProvider["byModel"].(map[string]any)
	if _, ok := byModel["gpt-4o"]; !ok {
		t.Errorf("expected bare byModel key gpt-4o")
	}
	if _, ok := byModel["gpt-4o|unknown"]; ok {
		t.Errorf("byModel key must not add provider suffix when provider is missing")
	}
	byApiKey = dayNoProvider["byApiKey"].(map[string]any)
	requireCounter(t, byApiKey, "k|gpt-4o|unknown")
	byEndpoint = dayNoProvider["byEndpoint"].(map[string]any)
	requireCounter(t, byEndpoint, "/v1|gpt-4o|unknown")

	// (b) Entry without connectionId: no byAccount entry.
	dayNoConn := freshDay()
	aggregateEntryToDay(dayNoConn, &RequestLogEntry{
		Provider:         "openai",
		Model:            "gpt-4o",
		APIKey:           "k",
		Endpoint:         "/v1",
		PromptTokens:     1,
		CompletionTokens: 2,
	})
	byAccount = dayNoConn["byAccount"].(map[string]any)
	if len(byAccount) != 0 {
		t.Errorf("byAccount = %v, want empty when connectionId is missing", byAccount)
	}

	// (c) Entry without apiKey: byApiKey uses local-no-key fallback; endpoint uses
	// Unknown fallback.
	dayNoKey := freshDay()
	aggregateEntryToDay(dayNoKey, &RequestLogEntry{
		Provider:         "openai",
		Model:            "gpt-4o",
		PromptTokens:     1,
		CompletionTokens: 2,
	})
	byApiKey = dayNoKey["byApiKey"].(map[string]any)
	apiKeyEntry = requireCounter(t, byApiKey, "local-no-key|gpt-4o|openai")
	if apiKeyEntry["apiKey"] != "" {
		t.Errorf("byApiKey meta apiKey = %v, want empty", apiKeyEntry["apiKey"])
	}
	byEndpoint = dayNoKey["byEndpoint"].(map[string]any)
	endpointEntry = requireCounter(t, byEndpoint, "Unknown|gpt-4o|openai")
	if endpointEntry["endpoint"] != "Unknown" {
		t.Errorf("byEndpoint meta endpoint = %v, want Unknown", endpointEntry["endpoint"])
	}
}
