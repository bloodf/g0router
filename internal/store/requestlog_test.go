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

func TestAggregateEntryToDay(t *testing.T) {
	day := map[string]any{
		"requests":           0,
		"promptTokens":       0,
		"completionTokens":   0,
		"cost":               0.0,
		"byProvider":         map[string]any{},
		"byModel":            map[string]any{},
		"byAccount":          map[string]any{},
		"byApiKey":           map[string]any{},
		"byEndpoint":         map[string]any{},
	}

	aggregateEntryToDay(day, &RequestLogEntry{
		Provider:       "anthropic",
		Model:          "claude-sonnet-4",
		ConnectionID:   "conn-a",
		APIKey:         "ak",
		Endpoint:       "/messages",
		PromptTokens:   10,
		CompletionTokens: 20,
		Cost:           0.123,
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

	// Second entry accumulates.
	aggregateEntryToDay(day, &RequestLogEntry{
		Provider:       "anthropic",
		Model:          "claude-sonnet-4",
		PromptTokens:   5,
		CompletionTokens: 5,
		Cost:           0.1,
	})
	if got := day["requests"].(float64); got != 2 {
		t.Errorf("requests after second = %v, want 2", got)
	}
	if got := day["promptTokens"].(float64); got != 15 {
		t.Errorf("promptTokens after second = %v, want 15", got)
	}

	// Missing provider falls back to model-only key.
	byModel := day["byModel"].(map[string]any)
	if _, ok := byModel["claude-sonnet-4|anthropic"]; !ok {
		t.Errorf("expected byModel key claude-sonnet-4|anthropic")
	}

	// Missing apiKey uses local-no-key fallback.
	day2 := map[string]any{
		"requests":           0,
		"promptTokens":       0,
		"completionTokens":   0,
		"cost":               0.0,
		"byProvider":         map[string]any{},
		"byModel":            map[string]any{},
		"byAccount":          map[string]any{},
		"byApiKey":           map[string]any{},
		"byEndpoint":         map[string]any{},
	}
	aggregateEntryToDay(day2, &RequestLogEntry{
		Provider:       "openai",
		Model:          "gpt-4o",
		PromptTokens:   1,
		CompletionTokens: 2,
	})
	byApiKey := day2["byApiKey"].(map[string]any)
	if _, ok := byApiKey["local-no-key|gpt-4o|openai"]; !ok {
		t.Errorf("expected byApiKey local-no-key fallback")
	}
	byEndpoint := day2["byEndpoint"].(map[string]any)
	if _, ok := byEndpoint["Unknown|gpt-4o|openai"]; !ok {
		t.Errorf("expected byEndpoint Unknown fallback")
	}
}
