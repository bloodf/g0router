package store

import (
	"fmt"
	"strings"
	"time"
)

// AuditEntry is a single admin audit-log record. Details must be a sanitized,
// non-secret summary supplied by the caller: the store never inspects it and
// never strips secrets, so callers must not pass request bodies.
type AuditEntry struct {
	ID            int64
	Timestamp     time.Time
	ActorAPIKeyID string
	Action        string
	Target        string
	Details       string
}

// AuditFilter selects and paginates audit-log records. Newest records are
// returned first. Action and Actor are optional exact-match filters.
type AuditFilter struct {
	Action *string
	Actor  *string
	Limit  int
	Offset int
}

const (
	defaultAuditLimit = 50
	maxAuditLimit     = 200
)

func clampAuditLimit(limit int) int {
	if limit <= 0 {
		return defaultAuditLimit
	}
	if limit > maxAuditLimit {
		return maxAuditLimit
	}
	return limit
}

// AppendAudit inserts an audit-log record. The timestamp defaults to now (UTC)
// when zero.
func (s *Store) AppendAudit(entry AuditEntry) error {
	timestamp := entry.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	_, err := s.db.Exec(
		`INSERT INTO audit_log (timestamp, actor_api_key_id, action, target, details)
			VALUES (?, ?, ?, ?, ?)`,
		timestamp.Format(time.RFC3339),
		entry.ActorAPIKeyID,
		entry.Action,
		entry.Target,
		entry.Details,
	)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

// ListAudit returns audit-log records matching the filter, newest first, plus
// the total count of matching rows (ignoring limit/offset) for pagination.
func (s *Store) ListAudit(filter AuditFilter) ([]AuditEntry, int, error) {
	where, args := auditWhere(filter)

	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM audit_log"+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit log: %w", err)
	}

	query := `SELECT id, timestamp, actor_api_key_id, action, target, details
		FROM audit_log` + where + ` ORDER BY id DESC LIMIT ?`
	args = append(args, clampAuditLimit(filter.Limit))
	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query audit log: %w", err)
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var (
			entry     AuditEntry
			timestamp string
		)
		if err := rows.Scan(&entry.ID, &timestamp, &entry.ActorAPIKeyID, &entry.Action, &entry.Target, &entry.Details); err != nil {
			return nil, 0, fmt.Errorf("scan audit log: %w", err)
		}
		parsed, err := time.Parse(time.RFC3339, timestamp)
		if err != nil {
			return nil, 0, fmt.Errorf("parse audit timestamp: %w", err)
		}
		entry.Timestamp = parsed
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate audit log: %w", err)
	}

	return entries, total, nil
}

func auditWhere(filter AuditFilter) (string, []any) {
	var clauses []string
	var args []any
	if filter.Action != nil {
		clauses = append(clauses, "action = ?")
		args = append(args, *filter.Action)
	}
	if filter.Actor != nil {
		clauses = append(clauses, "actor_api_key_id = ?")
		args = append(args, *filter.Actor)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}
