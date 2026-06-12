package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// RequestDetailRow is a single persisted request detail.
type RequestDetailRow struct {
	ID           string
	Timestamp    string
	Provider     string
	Model        string
	ConnectionID string
	Status       string
	Data         json.RawMessage
}

// Pagination is returned with filtered list queries.
type Pagination struct {
	Page       int
	PageSize   int
	TotalItems int
	TotalPages int
	HasNext    bool
	HasPrev    bool
}

// RequestDetailsFilter selects and paginates request details.
type RequestDetailsFilter struct {
	Provider     string
	Model        string
	ConnectionID string
	Status       string
	StartDate    string
	EndDate      string
	Page         int
	PageSize     int
}

// SaveRequestDetails atomically upserts detail rows and enforces retention.
func (s *Store) SaveRequestDetails(items []*RequestDetailRow, maxRecords int) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin request details tx: %w", err)
	}
	defer tx.Rollback()

	for _, item := range items {
		if _, err := tx.Exec(
			`INSERT INTO request_details (id, timestamp, provider, model, connection_id, status, data)
			 VALUES (?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT(id) DO UPDATE SET
			   timestamp = excluded.timestamp,
			   provider = excluded.provider,
			   model = excluded.model,
			   connection_id = excluded.connection_id,
			   status = excluded.status,
			   data = excluded.data`,
			item.ID, item.Timestamp, nullIfEmpty(item.Provider), nullIfEmpty(item.Model),
			nullIfEmpty(item.ConnectionID), nullIfEmpty(item.Status), string(item.Data),
		); err != nil {
			return fmt.Errorf("upsert request detail %s: %w", item.ID, err)
		}
	}

	if maxRecords > 0 {
		var count int
		if err := tx.QueryRow("SELECT COUNT(*) FROM request_details").Scan(&count); err != nil {
			return fmt.Errorf("count request details: %w", err)
		}
		if count > maxRecords {
			if _, err := tx.Exec(
				`DELETE FROM request_details WHERE id IN (
				 SELECT id FROM request_details ORDER BY timestamp ASC LIMIT ?
				)`,
				count-maxRecords,
			); err != nil {
				return fmt.Errorf("delete oldest request details: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit request details tx: %w", err)
	}
	return nil
}

// QueryRequestDetails returns filtered, paginated raw data blobs ordered newest first.
func (s *Store) QueryRequestDetails(f RequestDetailsFilter) ([]json.RawMessage, Pagination, error) {
	page := f.Page
	if page <= 0 {
		page = 1
	}
	pageSize := f.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}

	var conds []string
	var params []any

	if f.Provider != "" {
		conds = append(conds, "provider = ?")
		params = append(params, f.Provider)
	}
	if f.Model != "" {
		conds = append(conds, "model = ?")
		params = append(params, f.Model)
	}
	if f.ConnectionID != "" {
		conds = append(conds, "connection_id = ?")
		params = append(params, f.ConnectionID)
	}
	if f.Status != "" {
		conds = append(conds, "status = ?")
		params = append(params, f.Status)
	}
	if f.StartDate != "" {
		conds = append(conds, "timestamp >= ?")
		params = append(params, f.StartDate)
	}
	if f.EndDate != "" {
		conds = append(conds, "timestamp <= ?")
		params = append(params, f.EndDate)
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + joinConds(conds)
	}

	var totalItems int
	countStmt := "SELECT COUNT(*) FROM request_details " + where
	if err := s.db.QueryRow(countStmt, params...).Scan(&totalItems); err != nil {
		return nil, Pagination{}, fmt.Errorf("count request details: %w", err)
	}

	totalPages := totalItems / pageSize
	if totalItems%pageSize > 0 {
		totalPages++
	}
	offset := (page - 1) * pageSize

	queryStmt := "SELECT data FROM request_details " + where + " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	queryParams := append(params, pageSize, offset)

	rows, err := s.db.Query(queryStmt, queryParams...)
	if err != nil {
		return nil, Pagination{}, fmt.Errorf("query request details: %w", err)
	}
	defer rows.Close()

	// Always return a non-nil slice so an empty result marshals as []
	// (matching the reference) rather than null.
	out := make([]json.RawMessage, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, Pagination{}, fmt.Errorf("scan request detail data: %w", err)
		}
		out = append(out, json.RawMessage(data))
	}
	if err := rows.Err(); err != nil {
		return nil, Pagination{}, fmt.Errorf("iterate request details: %w", err)
	}

	pg := Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
	return out, pg, nil
}

// GetRequestDetailByID returns the raw data blob for the given id.
// Not found returns (nil, nil) per store conventions.
func (s *Store) GetRequestDetailByID(id string) (json.RawMessage, error) {
	var data string
	err := s.db.QueryRow("SELECT data FROM request_details WHERE id = ?", id).Scan(&data)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get request detail %s: %w", id, err)
	}
	return json.RawMessage(data), nil
}

func joinConds(conds []string) string {
	out := ""
	for i, c := range conds {
		if i > 0 {
			out += " AND "
		}
		out += c
	}
	return out
}
