package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// AliasRecord is an id-keyed admin model alias mirroring the dashboard UI shape
// {id, alias, provider, model}. It is distinct from the gateway ModelAlias
// (name→target) resolver in aliases.go: this table backs the admin /api/aliases
// CRUD surface while the model_aliases table remains the live resolution source.
type AliasRecord struct {
	ID        string
	Alias     string
	Provider  string
	Model     string
	CreatedAt int64
	UpdatedAt int64
}

// CreateAliasRecord inserts a new admin alias record with a generated id.
func (s *Store) CreateAliasRecord(in *AliasRecord) (*AliasRecord, error) {
	now := time.Now().Unix()
	id, err := newID()
	if err != nil {
		return nil, err
	}
	rec := &AliasRecord{
		ID:        id,
		Alias:     in.Alias,
		Provider:  in.Provider,
		Model:     in.Model,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err = s.db.Exec(
		"INSERT INTO aliases (id, alias, provider, model, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		rec.ID, rec.Alias, rec.Provider, rec.Model, rec.CreatedAt, rec.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert alias record %s: %w", rec.Alias, err)
	}
	return rec, nil
}

// ListAliasRecords returns all admin alias records ordered by creation time.
func (s *Store) ListAliasRecords() ([]*AliasRecord, error) {
	rows, err := s.db.Query(
		"SELECT id, alias, provider, model, created_at, updated_at FROM aliases ORDER BY created_at, id")
	if err != nil {
		return nil, fmt.Errorf("query alias records: %w", err)
	}
	defer rows.Close()

	var out []*AliasRecord
	for rows.Next() {
		rec, err := scanAliasRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate alias records: %w", err)
	}
	return out, nil
}

// GetAliasRecordByID returns the admin alias record with the given id.
func (s *Store) GetAliasRecordByID(id string) (*AliasRecord, error) {
	return scanAliasRecord(s.db.QueryRow(
		"SELECT id, alias, provider, model, created_at, updated_at FROM aliases WHERE id = ?", id))
}

// UpdateAliasRecord persists the mutable fields of the alias record.
func (s *Store) UpdateAliasRecord(in *AliasRecord) error {
	res, err := s.db.Exec(
		"UPDATE aliases SET alias = ?, provider = ?, model = ?, updated_at = ? WHERE id = ?",
		in.Alias, in.Provider, in.Model, time.Now().Unix(), in.ID,
	)
	if err != nil {
		return fmt.Errorf("update alias record %s: %w", in.ID, err)
	}
	return requireRowAffected(res)
}

// DeleteAliasRecord removes the admin alias record with the given id.
func (s *Store) DeleteAliasRecord(id string) error {
	res, err := s.db.Exec("DELETE FROM aliases WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete alias record %s: %w", id, err)
	}
	return requireRowAffected(res)
}

func scanAliasRecord(row rowScanner) (*AliasRecord, error) {
	var rec AliasRecord
	err := row.Scan(&rec.ID, &rec.Alias, &rec.Provider, &rec.Model, &rec.CreatedAt, &rec.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan alias record: %w", err)
	}
	return &rec, nil
}
