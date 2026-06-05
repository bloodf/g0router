package store

import (
	"fmt"
	"time"
)

// ModelStat holds aggregated telemetry for a single provider/model pair over a
// query window. Only successful requests (status_code < 400) are included.
type ModelStat struct {
	AvgLatencyMS float64
	AvgCostUSD   float64
	Requests     int
}

// ProviderModelStats returns per-provider/model aggregates for successful
// requests logged since the given time. The map key is "provider/model".
// Rows with status_code >= 400 are excluded. Only rows where latency_ms IS NOT
// NULL are included in the latency average (cost likewise).
func (s *Store) ProviderModelStats(since time.Time) (map[string]ModelStat, error) {
	rows, err := s.db.Query(
		`SELECT provider, model,
			AVG(CAST(latency_ms AS REAL)),
			AVG(CAST(cost_usd AS REAL)),
			COUNT(*)
		FROM request_log
		WHERE timestamp >= ?
		  AND (status_code IS NULL OR status_code < 400)
		GROUP BY provider, model`,
		since.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("query provider model stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]ModelStat)
	for rows.Next() {
		var provider, model string
		var avgLatency, avgCost *float64
		var count int
		if err := rows.Scan(&provider, &model, &avgLatency, &avgCost, &count); err != nil {
			return nil, fmt.Errorf("scan provider model stat: %w", err)
		}
		stat := ModelStat{Requests: count}
		if avgLatency != nil {
			stat.AvgLatencyMS = *avgLatency
		}
		if avgCost != nil {
			stat.AvgCostUSD = *avgCost
		}
		stats[provider+"/"+model] = stat
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider model stats: %w", err)
	}

	return stats, nil
}
