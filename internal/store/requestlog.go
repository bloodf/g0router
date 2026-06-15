package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// UsageDailyRow is a single persisted daily rollup row.
type UsageDailyRow struct {
	DateKey string
	Data    string
}

// RequestLogEntry is a single persisted usage record.
type RequestLogEntry struct {
	Timestamp        string
	Provider         string
	Model            string
	ConnectionID     string
	APIKey           string
	Endpoint         string
	PromptTokens     int64
	CompletionTokens int64
	Cost             float64
	Status           string
	Tokens           map[string]int64
	Meta             map[string]string
}

// SaveUsage atomically inserts a request log row, updates the daily rollup,
// and increments the lifetime request counter.
func (s *Store) SaveUsage(e *RequestLogEntry) error {
	if e.Timestamp == "" {
		e.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin save usage tx: %w", err)
	}
	defer tx.Rollback()

	tokensJSON, err := json.Marshal(e.Tokens)
	if err != nil {
		return fmt.Errorf("marshal tokens: %w", err)
	}
	metaJSON, err := json.Marshal(e.Meta)
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}

	if _, err := tx.Exec(
		`INSERT INTO request_log (
			timestamp, provider, model, connection_id, api_key, endpoint,
			prompt_tokens, completion_tokens, cost, status, tokens, meta
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.Timestamp, nullIfEmpty(e.Provider), nullIfEmpty(e.Model),
		nullIfEmpty(e.ConnectionID), nullIfEmpty(e.APIKey), nullIfEmpty(e.Endpoint),
		e.PromptTokens, e.CompletionTokens, e.Cost, e.Status, string(tokensJSON), string(metaJSON),
	); err != nil {
		return fmt.Errorf("insert request_log: %w", err)
	}

	dateKey := localDateKey(e.Timestamp)

	var data string
	row := tx.QueryRow("SELECT data FROM usage_daily WHERE date_key = ?", dateKey)
	if err := row.Scan(&data); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("select usage_daily: %w", err)
	}

	day := map[string]any{
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
	if data != "" {
		if err := json.Unmarshal([]byte(data), &day); err != nil {
			return fmt.Errorf("unmarshal usage_daily data: %w", err)
		}
	}

	aggregateEntryToDay(day, e)

	dayJSON, err := json.Marshal(day)
	if err != nil {
		return fmt.Errorf("marshal usage_daily data: %w", err)
	}
	if _, err := tx.Exec(
		`INSERT INTO usage_daily (date_key, data) VALUES (?, ?)
		 ON CONFLICT(date_key) DO UPDATE SET data = excluded.data`,
		dateKey, string(dayJSON),
	); err != nil {
		return fmt.Errorf("upsert usage_daily: %w", err)
	}

	var cur string
	if err := tx.QueryRow("SELECT value FROM kv WHERE scope = 'meta' AND key = 'total_requests_lifetime'").Scan(&cur); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("select lifetime counter: %w", err)
	}
	next := parseCounter(cur) + 1
	if _, err := tx.Exec(
		`INSERT INTO kv (scope, key, value) VALUES ('meta', 'total_requests_lifetime', ?)
		 ON CONFLICT(scope, key) DO UPDATE SET value = excluded.value`,
		fmt.Sprintf("%d", next),
	); err != nil {
		return fmt.Errorf("upsert lifetime counter: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit save usage tx: %w", err)
	}
	return nil
}

// ListRecentRequestLogs returns the most recent request log entries.
func (s *Store) ListRecentRequestLogs(limit int) ([]*RequestLogEntry, error) {
	rows, err := s.db.Query(
		`SELECT timestamp, provider, model, connection_id, api_key, endpoint,
		        prompt_tokens, completion_tokens, cost, status, tokens, meta
		 FROM request_log ORDER BY id DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list recent request logs: %w", err)
	}
	defer rows.Close()

	var out []*RequestLogEntry
	for rows.Next() {
		var e RequestLogEntry
		var provider, model, connectionID, apiKey, endpoint, status sql.NullString
		var tokensJSON, metaJSON string
		if err := rows.Scan(
			&e.Timestamp, &provider, &model, &connectionID, &apiKey, &endpoint,
			&e.PromptTokens, &e.CompletionTokens, &e.Cost, &status, &tokensJSON, &metaJSON,
		); err != nil {
			return nil, fmt.Errorf("scan request log: %w", err)
		}
		e.Provider = provider.String
		e.Model = model.String
		e.ConnectionID = connectionID.String
		e.APIKey = apiKey.String
		e.Endpoint = endpoint.String
		e.Status = status.String
		if tokensJSON != "" {
			_ = json.Unmarshal([]byte(tokensJSON), &e.Tokens)
		}
		if metaJSON != "" {
			_ = json.Unmarshal([]byte(metaJSON), &e.Meta)
		}
		out = append(out, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate request logs: %w", err)
	}
	return out, nil
}

// LoadDailyRange returns usage_daily rows with date_key on or after the cutoff.
// maxDays <= 0 returns all rows. The cutoff is computed from the injected now
// so callers can pin the window deterministically (e.g. tests, time-travel).
func (s *Store) LoadDailyRange(maxDays int, now time.Time) ([]*UsageDailyRow, error) {
	var rows *sql.Rows
	var err error
	if maxDays > 0 {
		cutoff := now.UTC().AddDate(0, 0, -(maxDays - 1)).Format("2006-01-02")
		rows, err = s.db.Query("SELECT date_key, data FROM usage_daily WHERE date_key >= ? ORDER BY date_key ASC", cutoff)
	} else {
		rows, err = s.db.Query("SELECT date_key, data FROM usage_daily ORDER BY date_key ASC")
	}
	if err != nil {
		return nil, fmt.Errorf("load daily range: %w", err)
	}
	defer rows.Close()

	var out []*UsageDailyRow
	for rows.Next() {
		var r UsageDailyRow
		if err := rows.Scan(&r.DateKey, &r.Data); err != nil {
			return nil, fmt.Errorf("scan daily row: %w", err)
		}
		out = append(out, &r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate daily range: %w", err)
	}
	return out, nil
}

// RangeRequestLogs returns request_log rows with timestamp in the inclusive
// [sinceISO, untilISO] window, ordered newest first.
func (s *Store) RangeRequestLogs(sinceISO, untilISO string) ([]*RequestLogEntry, error) {
	rows, err := s.db.Query(
		`SELECT timestamp, provider, model, connection_id, api_key, endpoint,
		        prompt_tokens, completion_tokens, cost, status, tokens, meta
		 FROM request_log WHERE timestamp >= ? AND timestamp <= ? ORDER BY timestamp DESC`,
		sinceISO, untilISO,
	)
	if err != nil {
		return nil, fmt.Errorf("range request logs: %w", err)
	}
	defer rows.Close()

	var out []*RequestLogEntry
	for rows.Next() {
		var e RequestLogEntry
		var provider, model, connectionID, apiKey, endpoint, status sql.NullString
		var tokensJSON, metaJSON string
		if err := rows.Scan(
			&e.Timestamp, &provider, &model, &connectionID, &apiKey, &endpoint,
			&e.PromptTokens, &e.CompletionTokens, &e.Cost, &status, &tokensJSON, &metaJSON,
		); err != nil {
			return nil, fmt.Errorf("scan request log: %w", err)
		}
		e.Provider = provider.String
		e.Model = model.String
		e.ConnectionID = connectionID.String
		e.APIKey = apiKey.String
		e.Endpoint = endpoint.String
		e.Status = status.String
		if tokensJSON != "" {
			_ = json.Unmarshal([]byte(tokensJSON), &e.Tokens)
		}
		if metaJSON != "" {
			_ = json.Unmarshal([]byte(metaJSON), &e.Meta)
		}
		out = append(out, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate range request logs: %w", err)
	}
	return out, nil
}

// SumCostByAPIKey returns the sum of cost for request_log rows attributed to
// the given api_key with timestamp >= sinceISO. It is used by the quota engine
// to enforce per-key budget windows (PAR-ROUTE-031).
func (s *Store) SumCostByAPIKey(key, sinceISO string) (float64, error) {
	var total sql.NullFloat64
	err := s.db.QueryRow(
		"SELECT SUM(cost) FROM request_log WHERE api_key = ? AND timestamp >= ?",
		key, sinceISO,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum cost by api key: %w", err)
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Float64, nil
}

// SumCostByTeam returns the sum of cost for request_log rows attributed to any
// virtual key owning team teamID (joined via virtual_keys.team_id) with
// timestamp >= sinceISO. It is the live team-budget aggregate used by the quota
// engine's 2-level hierarchy check (bf-gov-1, D8); it does not read the
// display-only teams.budget_used_usd accumulator.
func (s *Store) SumCostByTeam(teamID, sinceISO string) (float64, error) {
	var total sql.NullFloat64
	err := s.db.QueryRow(
		`SELECT SUM(cost) FROM request_log
		 WHERE api_key IN (SELECT key FROM virtual_keys WHERE team_id = ?)
		   AND timestamp >= ?`,
		teamID, sinceISO,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum cost by team: %w", err)
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Float64, nil
}

// SumTokensByAPIKey returns SUM(prompt_tokens + completion_tokens) over
// request_log rows attributed to the given api_key with timestamp >= sinceISO.
// It is the live token-dimension aggregate used by the quota engine's
// dual-dimension rate limit (bf-gov-3, D1); a NULL sum (no rows) returns 0.
func (s *Store) SumTokensByAPIKey(key, sinceISO string) (int64, error) {
	var total sql.NullInt64
	err := s.db.QueryRow(
		"SELECT SUM(prompt_tokens + completion_tokens) FROM request_log WHERE api_key = ? AND timestamp >= ?",
		key, sinceISO,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum tokens by api key: %w", err)
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Int64, nil
}

// SumRequestsByAPIKey returns COUNT(*) over request_log rows attributed to the
// given api_key with timestamp >= sinceISO. It is the live request-dimension
// aggregate used by the quota engine's dual-dimension rate limit (bf-gov-3, D3).
func (s *Store) SumRequestsByAPIKey(key, sinceISO string) (int64, error) {
	var total int64
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM request_log WHERE api_key = ? AND timestamp >= ?",
		key, sinceISO,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("count requests by api key: %w", err)
	}
	return total, nil
}

func localDateKey(timestamp string) string {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		// Fallback: treat as local date if already YYYY-MM-DD or malformed.
		if idx := strings.Index(timestamp, "T"); idx > 0 {
			return timestamp[:idx]
		}
		return timestamp
	}
	return t.Format("2006-01-02")
}

func nullIfEmpty(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func parseCounter(s string) int64 {
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}

func aggregateEntryToDay(day map[string]any, e *RequestLogEntry) {
	promptTokens := e.PromptTokens
	completionTokens := e.CompletionTokens
	cost := e.Cost

	day["requests"] = toFloat64(day["requests"]) + 1
	day["promptTokens"] = toFloat64(day["promptTokens"]) + float64(promptTokens)
	day["completionTokens"] = toFloat64(day["completionTokens"]) + float64(completionTokens)
	day["cost"] = toFloat64(day["cost"]) + cost

	ensureMap(day, "byProvider")
	ensureMap(day, "byModel")
	ensureMap(day, "byAccount")
	ensureMap(day, "byApiKey")
	ensureMap(day, "byEndpoint")

	vals := counterValues{promptTokens: promptTokens, completionTokens: completionTokens, cost: cost}

	if e.Provider != "" {
		addToCounter(day["byProvider"].(map[string]any), e.Provider, vals, nil)
	}

	modelKey := e.Model
	if e.Provider != "" {
		modelKey = e.Model + "|" + e.Provider
	}
	addToCounter(day["byModel"].(map[string]any), modelKey, vals, map[string]any{
		"rawModel": e.Model,
		"provider": e.Provider,
	})

	if e.ConnectionID != "" {
		addToCounter(day["byAccount"].(map[string]any), e.ConnectionID, vals, map[string]any{
			"rawModel": e.Model,
			"provider": e.Provider,
		})
	}

	apiKeyVal := e.APIKey
	if apiKeyVal == "" {
		apiKeyVal = "local-no-key"
	}
	apiKeyKey := fmt.Sprintf("%s|%s|%s", apiKeyVal, e.Model, providerOrUnknown(e.Provider))
	addToCounter(day["byApiKey"].(map[string]any), apiKeyKey, vals, map[string]any{
		"rawModel": e.Model,
		"provider": e.Provider,
		"apiKey":   e.APIKey,
	})

	endpoint := e.Endpoint
	if endpoint == "" {
		endpoint = "Unknown"
	}
	endpointKey := fmt.Sprintf("%s|%s|%s", endpoint, e.Model, providerOrUnknown(e.Provider))
	addToCounter(day["byEndpoint"].(map[string]any), endpointKey, vals, map[string]any{
		"endpoint": endpoint,
		"rawModel": e.Model,
		"provider": e.Provider,
	})
}

type counterValues struct {
	requests         float64
	promptTokens     int64
	completionTokens int64
	cost             float64
}

func addToCounter(target map[string]any, key string, vals counterValues, meta map[string]any) {
	counter, ok := target[key].(map[string]any)
	if !ok {
		counter = map[string]any{
			"requests":         0.0,
			"promptTokens":     0.0,
			"completionTokens": 0.0,
			"cost":             0.0,
		}
		target[key] = counter
	}
	counter["requests"] = toFloat64(counter["requests"]) + 1
	counter["promptTokens"] = toFloat64(counter["promptTokens"]) + float64(vals.promptTokens)
	counter["completionTokens"] = toFloat64(counter["completionTokens"]) + float64(vals.completionTokens)
	counter["cost"] = toFloat64(counter["cost"]) + vals.cost
	if meta != nil {
		for k, v := range meta {
			counter[k] = v
		}
	}
}

func ensureMap(day map[string]any, key string) {
	if _, ok := day[key].(map[string]any); !ok {
		day[key] = map[string]any{}
	}
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	}
	return 0
}

func providerOrUnknown(p string) string {
	if p == "" {
		return "unknown"
	}
	return p
}
