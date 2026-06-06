package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type RequestLogEntry struct {
	ID                 int64
	RequestID          string
	Timestamp          time.Time
	Provider           string
	Model              string
	ConnectionID       *string
	AuthType           string
	InputTokens        *int
	OutputTokens       *int
	CacheReadTokens    *int
	CacheWriteTokens   *int
	TotalTokens        *int
	CostUSD            *float64
	LatencyMS          *int
	StatusCode         *int
	Error              *string
	SourceFormat       *string
	TargetFormat       *string
	RTKEnabled         *bool
	RTKBytesSaved      *int
	CavemanEnabled     *bool
	ComboName          *string
	APIKeyID           *string
	ClientTool         *string
	APIKeyName         *string
	ConnectionName     *string
	ConnectionProvider *string
	AccountEmail       *string
}

const (
	defaultUsageLimit = 50
	maxUsageLimit     = 200
)

// Status class filter values for UsageFilter.StatusClass.
const (
	StatusClassSuccess     = "success"
	StatusClassClientError = "client_error"
	StatusClassServerError = "server_error"
)

type UsageFilter struct {
	Provider     *string
	Model        *string
	AuthType     *string
	APIKeyID     *string
	SourceFormat *string
	StatusClass  string
	Search       string
	From         *time.Time
	To           *time.Time
	Start        *time.Time
	End          *time.Time
	Limit        int
	Offset       int
}

type UsageSummary struct {
	RequestCount int64
	TotalTokens  int64
	TotalCostUSD float64
}

type UsageChart struct {
	Buckets      []string
	Requests     []int64
	TokensInput  []int64
	TokensOutput []int64
	Costs        []float64
}

func (s *Store) LogRequest(entry *RequestLogEntry) error {
	timestamp := entry.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	_, err := s.db.Exec(
		`INSERT INTO request_log (
			request_id, timestamp, provider, model, connection_id, auth_type,
			input_tokens, output_tokens, cache_read_tokens, cache_write_tokens,
			total_tokens, cost_usd, latency_ms, status_code, error,
			source_format, target_format, rtk_enabled, rtk_bytes_saved,
			caveman_enabled, combo_name, api_key_id, client_tool
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.RequestID,
		timestamp.Format(time.RFC3339),
		entry.Provider,
		entry.Model,
		entry.ConnectionID,
		entry.AuthType,
		entry.InputTokens,
		entry.OutputTokens,
		entry.CacheReadTokens,
		entry.CacheWriteTokens,
		entry.TotalTokens,
		entry.CostUSD,
		entry.LatencyMS,
		entry.StatusCode,
		entry.Error,
		entry.SourceFormat,
		entry.TargetFormat,
		nullableBoolInt(entry.RTKEnabled),
		entry.RTKBytesSaved,
		nullableBoolInt(entry.CavemanEnabled),
		entry.ComboName,
		entry.APIKeyID,
		entry.ClientTool,
	)
	if err != nil {
		return fmt.Errorf("insert request log: %w", err)
	}

	return nil
}

func clampUsageLimit(limit int) int {
	if limit <= 0 {
		return defaultUsageLimit
	}
	if limit > maxUsageLimit {
		return maxUsageLimit
	}
	return limit
}

func (s *Store) GetUsage(filter UsageFilter) ([]RequestLogEntry, error) {
	where, args := usageWhere(filter)
	query := `SELECT
		rl.id, rl.request_id, rl.timestamp, rl.provider, rl.model, rl.connection_id, rl.auth_type,
		rl.input_tokens, rl.output_tokens, rl.cache_read_tokens, rl.cache_write_tokens,
		rl.total_tokens, rl.cost_usd, rl.latency_ms, rl.status_code, rl.error,
		rl.source_format, rl.target_format, rl.rtk_enabled, rl.rtk_bytes_saved,
		rl.caveman_enabled, rl.combo_name, rl.api_key_id, rl.client_tool,
		ak.name, c.name, c.provider, c.email
		FROM request_log rl
		LEFT JOIN api_keys ak ON rl.api_key_id = ak.id
		LEFT JOIN connections c ON rl.connection_id = c.id` + where + ` ORDER BY rl.timestamp DESC, rl.id DESC`
	query += " LIMIT ?"
	args = append(args, clampUsageLimit(filter.Limit))
	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query usage: %w", err)
	}
	defer rows.Close()

	var entries []RequestLogEntry
	for rows.Next() {
		entry, err := scanRequestLogEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate usage: %w", err)
	}

	return entries, nil
}

func (s *Store) GetUsageSummary(filter UsageFilter) (*UsageSummary, error) {
	where, args := usageWhere(filter)

	var summary UsageSummary
	if err := s.db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(rl.total_tokens), 0), COALESCE(SUM(rl.cost_usd), 0)
		FROM request_log rl
		LEFT JOIN api_keys ak ON rl.api_key_id = ak.id
		LEFT JOIN connections c ON rl.connection_id = c.id`+where,
		args...,
	).Scan(&summary.RequestCount, &summary.TotalTokens, &summary.TotalCostUSD); err != nil {
		return nil, fmt.Errorf("query usage summary: %w", err)
	}

	return &summary, nil
}

func (s *Store) GetUsageChart(period, granularity string, now time.Time) (*UsageChart, error) {
	start, end, err := chartTimeRange(period, now)
	if err != nil {
		return nil, err
	}

	var bucketExpr string
	switch granularity {
	case "day":
		bucketExpr = "strftime('%Y-%m-%d', timestamp)"
	case "hour":
		bucketExpr = "strftime('%Y-%m-%dT%H:00', timestamp)"
	default:
		return nil, fmt.Errorf("invalid granularity: %q", granularity)
	}

	query := fmt.Sprintf(
		`SELECT %s as bucket,
			COUNT(*) as requests,
			COALESCE(SUM(input_tokens), 0) as tokens_input,
			COALESCE(SUM(output_tokens), 0) as tokens_output,
			COALESCE(SUM(cost_usd), 0) as cost
		FROM request_log
		WHERE timestamp >= ? AND timestamp <= ?
		GROUP BY bucket
		ORDER BY bucket`,
		bucketExpr,
	)

	rows, err := s.db.Query(query, start.Format(time.RFC3339), end.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("query usage chart: %w", err)
	}
	defer rows.Close()

	data := make(map[string]struct {
		requests     int64
		tokensInput  int64
		tokensOutput int64
		cost         float64
	})
	for rows.Next() {
		var bucket string
		var requests, tokensInput, tokensOutput int64
		var cost float64
		if err := rows.Scan(&bucket, &requests, &tokensInput, &tokensOutput, &cost); err != nil {
			return nil, fmt.Errorf("scan usage chart: %w", err)
		}
		data[bucket] = struct {
			requests     int64
			tokensInput  int64
			tokensOutput int64
			cost         float64
		}{requests, tokensInput, tokensOutput, cost}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate usage chart: %w", err)
	}

	buckets := generateBuckets(start, end, granularity)
	chart := &UsageChart{
		Buckets:      buckets,
		Requests:     make([]int64, len(buckets)),
		TokensInput:  make([]int64, len(buckets)),
		TokensOutput: make([]int64, len(buckets)),
		Costs:        make([]float64, len(buckets)),
	}
	for i, b := range buckets {
		if d, ok := data[b]; ok {
			chart.Requests[i] = d.requests
			chart.TokensInput[i] = d.tokensInput
			chart.TokensOutput[i] = d.tokensOutput
			chart.Costs[i] = d.cost
		}
	}

	return chart, nil
}

func chartTimeRange(period string, now time.Time) (time.Time, time.Time, error) {
	now = now.UTC()
	switch period {
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		return start, now, nil
	case "24h":
		return now.Add(-24 * time.Hour), now, nil
	case "7d":
		return now.Add(-7 * 24 * time.Hour), now, nil
	case "30d":
		return now.Add(-30 * 24 * time.Hour), now, nil
	case "60d":
		return now.Add(-60 * 24 * time.Hour), now, nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("invalid period: %q", period)
	}
}

func generateBuckets(start, end time.Time, granularity string) []string {
	start = start.UTC()
	end = end.UTC()
	var buckets []string
	switch granularity {
	case "day":
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
		end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
		for !start.After(end) {
			buckets = append(buckets, start.Format("2006-01-02"))
			start = start.Add(24 * time.Hour)
		}
	case "hour":
		start = start.Truncate(time.Hour)
		end = end.Truncate(time.Hour)
		for !start.After(end) {
			buckets = append(buckets, start.Format("2006-01-02T15:00"))
			start = start.Add(time.Hour)
		}
	}
	return buckets
}

// CountUsage returns the total number of rows matching the filter, ignoring
// the filter's Limit and Offset (used for pagination totals).
func (s *Store) CountUsage(filter UsageFilter) (int, error) {
	where, args := usageWhere(filter)

	var count int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM request_log rl
		LEFT JOIN api_keys ak ON rl.api_key_id = ak.id
		LEFT JOIN connections c ON rl.connection_id = c.id`+where,
		args...,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count usage: %w", err)
	}

	return count, nil
}

// DeleteRequestLogsOlderThan deletes request_log rows whose timestamp is
// strictly older than cutoff and returns the number of rows removed.
func (s *Store) DeleteRequestLogsOlderThan(cutoff time.Time) (int64, error) {
	result, err := s.db.Exec(
		"DELETE FROM request_log WHERE timestamp < ?",
		cutoff.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return 0, fmt.Errorf("delete request logs older than %s: %w", cutoff.Format(time.RFC3339), err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}

	return affected, nil
}

func usageWhere(filter UsageFilter) (string, []any) {
	var clauses []string
	var args []any

	if filter.Provider != nil {
		clauses = append(clauses, "rl.provider = ?")
		args = append(args, *filter.Provider)
	}
	if filter.Model != nil {
		clauses = append(clauses, "rl.model = ?")
		args = append(args, *filter.Model)
	}
	if filter.AuthType != nil {
		clauses = append(clauses, "rl.auth_type = ?")
		args = append(args, *filter.AuthType)
	}
	if filter.APIKeyID != nil {
		clauses = append(clauses, "rl.api_key_id = ?")
		args = append(args, *filter.APIKeyID)
	}
	if filter.SourceFormat != nil {
		clauses = append(clauses, "rl.source_format = ?")
		args = append(args, *filter.SourceFormat)
	}
	switch filter.StatusClass {
	case StatusClassSuccess:
		clauses = append(clauses, "rl.status_code < 400")
	case StatusClassClientError:
		clauses = append(clauses, "rl.status_code >= 400 AND rl.status_code < 500")
	case StatusClassServerError:
		clauses = append(clauses, "rl.status_code >= 500")
	}
	if filter.Search != "" {
		escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToLower(filter.Search))
		pattern := "%" + escaped + "%"
		clauses = append(clauses, "(LOWER(rl.request_id) LIKE ? ESCAPE '\\' OR LOWER(rl.model) LIKE ? ESCAPE '\\' OR LOWER(COALESCE(rl.error, '')) LIKE ? ESCAPE '\\')")
		args = append(args, pattern, pattern, pattern)
	}
	// from/to are aliases for start/end; prefer start/end when both are set.
	start := filter.Start
	if start == nil {
		start = filter.From
	}
	end := filter.End
	if end == nil {
		end = filter.To
	}
	if start != nil {
		clauses = append(clauses, "rl.timestamp >= ?")
		args = append(args, start.Format(time.RFC3339))
	}
	if end != nil {
		clauses = append(clauses, "rl.timestamp <= ?")
		args = append(args, end.Format(time.RFC3339))
	}
	if len(clauses) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(clauses, " AND "), args
}

func scanRequestLogEntry(rows *sql.Rows) (RequestLogEntry, error) {
	var entry RequestLogEntry
	var timestamp string
	var connectionID, errorMessage, sourceFormat, targetFormat sql.NullString
	var comboName, apiKeyID, clientTool sql.NullString
	var apiKeyName, connectionName, connectionProvider, accountEmail sql.NullString
	var inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens sql.NullInt64
	var totalTokens, latencyMS, statusCode, rtkBytesSaved sql.NullInt64
	var costUSD sql.NullFloat64
	var rtkEnabled, cavemanEnabled sql.NullInt64

	err := rows.Scan(
		&entry.ID,
		&entry.RequestID,
		&timestamp,
		&entry.Provider,
		&entry.Model,
		&connectionID,
		&entry.AuthType,
		&inputTokens,
		&outputTokens,
		&cacheReadTokens,
		&cacheWriteTokens,
		&totalTokens,
		&costUSD,
		&latencyMS,
		&statusCode,
		&errorMessage,
		&sourceFormat,
		&targetFormat,
		&rtkEnabled,
		&rtkBytesSaved,
		&cavemanEnabled,
		&comboName,
		&apiKeyID,
		&clientTool,
		&apiKeyName,
		&connectionName,
		&connectionProvider,
		&accountEmail,
	)
	if err != nil {
		return RequestLogEntry{}, fmt.Errorf("scan usage: %w", err)
	}

	parsed, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return RequestLogEntry{}, fmt.Errorf("parse usage timestamp: %w", err)
	}
	entry.Timestamp = parsed
	entry.ConnectionID = nullStringPtr(connectionID)
	entry.InputTokens = nullIntPtr(inputTokens)
	entry.OutputTokens = nullIntPtr(outputTokens)
	entry.CacheReadTokens = nullIntPtr(cacheReadTokens)
	entry.CacheWriteTokens = nullIntPtr(cacheWriteTokens)
	entry.TotalTokens = nullIntPtr(totalTokens)
	entry.CostUSD = nullFloatPtr(costUSD)
	entry.LatencyMS = nullIntPtr(latencyMS)
	entry.StatusCode = nullIntPtr(statusCode)
	entry.Error = nullStringPtr(errorMessage)
	entry.SourceFormat = nullStringPtr(sourceFormat)
	entry.TargetFormat = nullStringPtr(targetFormat)
	entry.RTKEnabled = nullBoolPtr(rtkEnabled)
	entry.RTKBytesSaved = nullIntPtr(rtkBytesSaved)
	entry.CavemanEnabled = nullBoolPtr(cavemanEnabled)
	entry.ComboName = nullStringPtr(comboName)
	entry.APIKeyID = nullStringPtr(apiKeyID)
	entry.ClientTool = nullStringPtr(clientTool)
	entry.APIKeyName = nullStringPtr(apiKeyName)
	entry.ConnectionName = nullStringPtr(connectionName)
	entry.ConnectionProvider = nullStringPtr(connectionProvider)
	entry.AccountEmail = nullStringPtr(accountEmail)

	return entry, nil
}

func nullableBoolInt(value *bool) any {
	if value == nil {
		return nil
	}
	if *value {
		return 1
	}
	return 0
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func nullIntPtr(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	converted := int(value.Int64)
	return &converted
}

func nullFloatPtr(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	return &value.Float64
}

func nullBoolPtr(value sql.NullInt64) *bool {
	if !value.Valid {
		return nil
	}
	converted := value.Int64 != 0
	return &converted
}
