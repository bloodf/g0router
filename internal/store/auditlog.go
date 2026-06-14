package store

import (
	"fmt"
)

// AuditEntry is a single recorded administrative action.
type AuditEntry struct {
	ID        string
	Timestamp string // RFC3339
	Actor     string
	Action    string
	Target    string
	Details   string
}

// InsertAuditEntry persists a single audit entry.
func (s *Store) InsertAuditEntry(e AuditEntry) error {
	_, err := s.db.Exec(
		"INSERT INTO audit_log (id, timestamp, actor, action, target, details) VALUES (?, ?, ?, ?, ?, ?)",
		e.ID, e.Timestamp, e.Actor, e.Action, e.Target, e.Details,
	)
	if err != nil {
		return fmt.Errorf("insert audit entry: %w", err)
	}
	return nil
}

// ListAuditEntries returns up to limit entries, newest first.
func (s *Store) ListAuditEntries(limit int) ([]AuditEntry, error) {
	rows, err := s.db.Query(
		"SELECT id, timestamp, actor, action, target, details FROM audit_log ORDER BY timestamp DESC LIMIT ?", limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query audit entries: %w", err)
	}
	defer rows.Close()

	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Actor, &e.Action, &e.Target, &e.Details); err != nil {
			return nil, fmt.Errorf("scan audit entry: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit entries: %w", err)
	}
	return out, nil
}

// CountAuditEntries returns the total number of audit entries.
func (s *Store) CountAuditEntries() (int, error) {
	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM audit_log").Scan(&n); err != nil {
		return 0, fmt.Errorf("count audit entries: %w", err)
	}
	return n, nil
}
