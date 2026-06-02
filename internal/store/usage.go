package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type RequestLogEntry struct {
	ID               int64
	RequestID        string
	Timestamp        time.Time
	Provider         string
	Model            string
	ConnectionID     *string
	AuthType         string
	InputTokens      *int
	OutputTokens     *int
	CacheReadTokens  *int
	CacheWriteTokens *int
	TotalTokens      *int
	CostUSD          *float64
	LatencyMS        *int
	StatusCode       *int
	Error            *string
	SourceFormat     *string
	TargetFormat     *string
	RTKEnabled       *bool
	RTKBytesSaved    *int
	CavemanEnabled   *bool
	ComboName        *string
	APIKeyID         *string
	ClientTool       *string
}

type UsageFilter struct {
	Provider *string
	Model    *string
	AuthType *string
	From     *time.Time
	To       *time.Time
	Limit    int
	Offset   int
}

type UsageSummary struct {
	RequestCount int64
	TotalTokens  int64
	TotalCostUSD float64
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

func (s *Store) GetUsage(filter UsageFilter) ([]RequestLogEntry, error) {
	where, args := usageWhere(filter)
	query := `SELECT
		id, request_id, timestamp, provider, model, connection_id, auth_type,
		input_tokens, output_tokens, cache_read_tokens, cache_write_tokens,
		total_tokens, cost_usd, latency_ms, status_code, error,
		source_format, target_format, rtk_enabled, rtk_bytes_saved,
		caveman_enabled, combo_name, api_key_id, client_tool
		FROM request_log` + where + ` ORDER BY timestamp DESC, id DESC`
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
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
		`SELECT COUNT(*), COALESCE(SUM(total_tokens), 0), COALESCE(SUM(cost_usd), 0) FROM request_log`+where,
		args...,
	).Scan(&summary.RequestCount, &summary.TotalTokens, &summary.TotalCostUSD); err != nil {
		return nil, fmt.Errorf("query usage summary: %w", err)
	}

	return &summary, nil
}

func usageWhere(filter UsageFilter) (string, []any) {
	var clauses []string
	var args []any

	if filter.Provider != nil {
		clauses = append(clauses, "provider = ?")
		args = append(args, *filter.Provider)
	}
	if filter.Model != nil {
		clauses = append(clauses, "model = ?")
		args = append(args, *filter.Model)
	}
	if filter.AuthType != nil {
		clauses = append(clauses, "auth_type = ?")
		args = append(args, *filter.AuthType)
	}
	if filter.From != nil {
		clauses = append(clauses, "timestamp >= ?")
		args = append(args, filter.From.Format(time.RFC3339))
	}
	if filter.To != nil {
		clauses = append(clauses, "timestamp <= ?")
		args = append(args, filter.To.Format(time.RFC3339))
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
